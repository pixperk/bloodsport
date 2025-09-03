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

type TodoServer struct {
	listener *quic.Listener
	storage  TodoStorage

	mu      sync.RWMutex
	clients map[string]*quic.Conn
}

func NewTodoServer(storage TodoStorage) *TodoServer {
	return &TodoServer{
		storage: storage,
		clients: make(map[string]*quic.Conn),
	}
}

func (s *TodoServer) Start(addr string) error {
	tlsConfig, err := generateTLSConfig()
	if err != nil {
		return err
	}

	quicConfig := &quic.Config{
		MaxStreamReceiveWindow:     1024 * 1024,     //1mb per stream
		MaxConnectionReceiveWindow: 4 * 1024 * 1024, //4mb per connection
		KeepAlivePeriod:            30 * time.Second,
		MaxIdleTimeout:             5 * time.Minute,
	}

	lis, err := quic.ListenAddr(addr, tlsConfig, quicConfig)
	if err != nil {
		return err
	}

	s.listener = lis
	log.Printf("quic todo server listening on %s\n", addr)

	s.acceptConns()

	return nil

}

func (s *TodoServer) acceptConns() {
	for {
		conn, err := s.listener.Accept(context.Background())
		if err != nil {
			log.Printf("failed to accept connection: %v\n", err)
			continue
		}

		log.Printf("new client connected: %s\n", conn.RemoteAddr().String())
		go s.handleConn(conn)
	}
}

func (s *TodoServer) handleConn(conn *quic.Conn) {
	clientId := uuid.New().String()

	s.mu.Lock()
	s.clients[clientId] = conn
	s.mu.Unlock()

	for {
		stream, err := conn.AcceptStream(context.Background())
		if err != nil {
			log.Printf("failed to accept stream: %v\n", err)
			return
		}

		go s.handleStream(clientId, stream)
	}

}

func (s *TodoServer) handleStream(clientId string, stream *quic.Stream) {
	defer stream.Close()

	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("failed to decode message from client %s: %v\n", clientId, err)
		return
	}

	log.Printf("received message from client %s: %+v\n", clientId, msg)

	resp := s.processMessage(&msg, stream)

	if err := encoder.Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (s *TodoServer) processMessage(msg *Message, stream *quic.Stream) *Message {
	switch msg.Type {
	case Ping:
		return &Message{Type: Pong, ReqId: msg.ReqId, Payload: "PONG"}

	case CreateTodo:
		return s.handleCreateTodo(msg)

	case ReadTodo:
		return s.handleReadTodo(msg)

	case UpdateTodo:
		return s.handleUpdateTodo(msg)

	case DeleteTodo:
		return s.handleDeleteTodo(msg)

	case ListTodos:
		return s.handleListTodos(msg)

	case UploadFile:
		return s.handleFileUpload(msg, stream)

	case UploadTodos:
		return s.handleUploadTodos(msg, stream)

	default:
		return &Message{Type: Error, ReqId: msg.ReqId, Payload: "Unknown message type"}

	}
}

// generate fake ssl cert
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Todo App"},
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
		NextProtos:   []string{"todo-quic"},
	}, nil
}
