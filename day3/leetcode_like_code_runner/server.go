package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
)

type JudgeServer struct {
	listener   *quic.Listener
	codeRunner *CodeRunner
	problems   map[string]Problem

	mu      sync.RWMutex
	clients map[string]*quic.Conn
}

func NewJudgeServer() (*JudgeServer, error) {
	runner, err := NewCodeRunner()
	if err != nil {
		return nil, err
	}

	problems := map[string]Problem{
		"two-sum": {
			ID:          "two-sum",
			Title:       "Two Sum",
			Description: "Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.",
			TestCases: []TestCase{
				{Input: "2,7,11,15\n9", Expected: "0,1"},
				{Input: "3,2,4\n6", Expected: "1,2"},
				{Input: "3,3\n6", Expected: "0,1"},
			},
			Template: `def two_sum(nums, target):
    # Your code here
    pass

# Read input
line1 = input().split(',')
nums = [int(x) for x in line1]
target = int(input())

# Call function and print result
result = two_sum(nums, target)
print(f"{result[0]},{result[1]}")`,
		},
		"reverse-integer": {
			ID:          "reverse-integer",
			Title:       "Reverse Integer",
			Description: "Given a signed 32-bit integer x, return x with its digits reversed.",
			TestCases: []TestCase{
				{Input: "123", Expected: "321"},
				{Input: "-123", Expected: "-321"},
				{Input: "120", Expected: "21"},
			},
			Template: `def reverse(x):
    # Your code here
    pass

# Read input
x = int(input())

# Call function and print result
result = reverse(x)
print(result)`,
		},
	}

	return &JudgeServer{
		codeRunner: runner,
		problems:   problems,
		clients:    make(map[string]*quic.Conn),
	}, nil
}

func (js *JudgeServer) Start(addr string) error {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return err
	}

	quicConfig := &quic.Config{
		MaxStreamReceiveWindow:     1024 * 1024,
		MaxConnectionReceiveWindow: 4 * 1024 * 1024,
		KeepAlivePeriod:            30 * time.Second,
		MaxIdleTimeout:             5 * time.Minute,
	}

	lis, err := quic.ListenAddr(addr, tlsConfig, quicConfig)
	if err != nil {
		return err
	}

	js.listener = lis
	log.Printf("Code Judge server listening on %s", addr)

	js.acceptConns()
	return nil
}

func (js *JudgeServer) acceptConns() {
	for {
		conn, err := js.listener.Accept(context.Background())
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		log.Printf("New client connected: %s", conn.RemoteAddr().String())
		go js.handleConn(conn)
	}
}

func (js *JudgeServer) handleConn(conn *quic.Conn) {
	clientId := uuid.New().String()

	js.mu.Lock()
	js.clients[clientId] = conn
	js.mu.Unlock()

	defer func() {
		js.mu.Lock()
		delete(js.clients, clientId)
		js.mu.Unlock()
	}()

	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("Failed to accept stream: %v", err)
			return
		}

		go js.handleStream(clientId, stream)
	}
}

func (js *JudgeServer) handleStream(clientId string, stream *quic.Stream) {
	defer stream.Close()

	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode message from client %s: %v", clientId, err)
		return
	}

	log.Printf("Received message from client %s: type=%d", clientId, msg.Type)

	resp := js.processMessage(&msg, stream)

	if err := encoder.Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (js *JudgeServer) processMessage(msg *Message, stream *quic.Stream) *Message {
	switch msg.Type {
	case Ping:
		return &Message{Type: Pong, ReqId: msg.ReqId, Payload: "PONG"}

	case SubmitCode:
		return js.handleCodeSubmission(msg)

	case GetProblem:
		return js.handleGetProblem(msg)

	case ListProblems:
		return js.handleListProblems(msg)

	default:
		return &Message{Type: Error, ReqId: msg.ReqId, Error: "Unknown message type"}
	}
}

func (js *JudgeServer) handleCodeSubmission(msg *Message) *Message {
	var submission CodeSubmission

	payloadBytes, err := getPayloadBytes(msg.Payload)
	if err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid payload format",
		}
	}

	if err := json.Unmarshal(payloadBytes, &submission); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid submission format",
		}
	}

	problem, exists := js.problems[submission.ProblemID]
	if !exists {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Problem not found",
		}
	}

	log.Printf("Executing code for problem %s in %s", submission.ProblemID, submission.Language)

	result, err := js.codeRunner.ExecuteCode(submission, problem)
	if err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	payload, _ := json.Marshal(result)

	return &Message{
		Type:    ExecutionResult,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (js *JudgeServer) handleGetProblem(msg *Message) *Message {
	var req struct {
		ProblemID string `json:"problem_id"`
	}

	payloadBytes, err := getPayloadBytes(msg.Payload)
	if err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid payload format",
		}
	}

	if err := json.Unmarshal(payloadBytes, &req); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid request format",
		}
	}

	problem, exists := js.problems[req.ProblemID]
	if !exists {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Problem not found",
		}
	}

	payload, _ := json.Marshal(problem)

	return &Message{
		Type:    GetProblem,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (js *JudgeServer) handleListProblems(msg *Message) *Message {
	problems := make([]Problem, 0, len(js.problems))
	for _, problem := range js.problems {
		problems = append(problems, problem)
	}

	payload, _ := json.Marshal(problems)

	return &Message{
		Type:    ListProblems,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Code Judge"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"code-judge"},
	}, nil
}
