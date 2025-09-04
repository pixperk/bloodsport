package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
)

type CodeJudgeClient struct {
	conn *quic.Conn
}

func NewCodeJudgeClient() *CodeJudgeClient {
	return &CodeJudgeClient{}
}

func (c *CodeJudgeClient) Connect(addr string) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"code-judge"},
	}

	conn, err := quic.DialAddr(context.Background(), addr, tlsConfig, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *CodeJudgeClient) Close() error {
	if c.conn != nil {
		return c.conn.CloseWithError(0, "Client closing")
	}
	return nil
}

func (c *CodeJudgeClient) sendRequest(msg *Message) (*Message, error) {
	stream, err := c.conn.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	msg.ReqId = uuid.New().String()

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(stream)
	var response Message
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *CodeJudgeClient) Ping() error {
	msg := &Message{Type: Ping}
	response, err := c.sendRequest(msg)
	if err != nil {
		return err
	}

	if response.Type != Pong {
		return fmt.Errorf("unexpected response type: %d", response.Type)
	}

	return nil
}

func (c *CodeJudgeClient) SubmitCode(submission CodeSubmission) (*CodeExecutionResult, error) {
	payload, _ := json.Marshal(submission)

	msg := &Message{
		Type:    SubmitCode,
		Payload: payload,
	}

	response, err := c.sendRequest(msg)
	if err != nil {
		return nil, err
	}

	if response.Type == Error {
		return nil, fmt.Errorf(response.Error)
	}

	var result CodeExecutionResult
	payloadBytes, err := getClientPayloadBytes(response.Payload)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadBytes, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *CodeJudgeClient) GetProblem(problemID string) (*Problem, error) {
	req := map[string]string{"problem_id": problemID}
	payload, _ := json.Marshal(req)

	msg := &Message{
		Type:    GetProblem,
		Payload: payload,
	}

	response, err := c.sendRequest(msg)
	if err != nil {
		return nil, err
	}

	if response.Type == Error {
		return nil, fmt.Errorf(response.Error)
	}

	var problem Problem
	payloadBytes, err := getClientPayloadBytes(response.Payload)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadBytes, &problem); err != nil {
		return nil, err
	}

	return &problem, nil
}

func (c *CodeJudgeClient) ListProblems() ([]Problem, error) {
	msg := &Message{Type: ListProblems}

	response, err := c.sendRequest(msg)
	if err != nil {
		return nil, err
	}

	if response.Type == Error {
		return nil, fmt.Errorf(response.Error)
	}

	var problems []Problem
	payloadBytes, err := getClientPayloadBytes(response.Payload)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadBytes, &problems); err != nil {
		return nil, err
	}

	return problems, nil
}

func (c *CodeJudgeClient) UploadFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
