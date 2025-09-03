package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
)

func getClientPayloadBytes(payload any) ([]byte, error) {
	switch p := payload.(type) {
	case []byte:
		return p, nil
	case string:
		// Try to base64 decode first (this is what JSON does with []byte fields)
		if decoded, err := base64.StdEncoding.DecodeString(p); err == nil {
			return decoded, nil
		}
		// If base64 decode fails, treat as regular string
		return []byte(p), nil
	default:
		return json.Marshal(p)
	}
}

type TodoClient struct {
	conn *quic.Conn
}

func NewTodoClient() *TodoClient {
	return &TodoClient{}
}

func (c *TodoClient) Connect(addr string) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"todo-quic"},
	}

	conn, err := quic.DialAddr(context.Background(), addr, tlsConfig, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

func (c *TodoClient) Close() error {
	if c.conn != nil {
		return c.conn.CloseWithError(0, "Client closing")
	}
	return nil
}

func (c *TodoClient) sendRequest(msg *Message) (*Message, error) {
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

func (c *TodoClient) Ping() error {
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

func (c *TodoClient) CreateTodo(title string) (*Todo, error) {
	req := CreateTodoRequest{Title: title}
	payload, _ := json.Marshal(req)

	msg := &Message{
		Type:    CreateTodo,
		Payload: payload,
	}

	response, err := c.sendRequest(msg)
	if err != nil {
		return nil, err
	}

	if response.Type == Error {
		return nil, fmt.Errorf(response.Error)
	}

	return nil, nil
}

func (c *TodoClient) ReadTodo(id int) (*Todo, error) {
	req := ReadTodoRequest{Id: id}
	payload, _ := json.Marshal(req)

	msg := &Message{
		Type:    ReadTodo,
		Payload: payload,
	}

	response, err := c.sendRequest(msg)
	if err != nil {
		return nil, err
	}

	if response.Type == Error {
		return nil, fmt.Errorf(response.Error)
	}

	var todo Todo
	payloadBytes, err := getClientPayloadBytes(response.Payload)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadBytes, &todo); err != nil {
		return nil, err
	}

	return &todo, nil
}

func (c *TodoClient) UpdateTodo(id int, title string, done bool) error {
	req := UpdateTodoRequest{
		Id:    id,
		Title: title,
		Done:  done,
	}
	payload, _ := json.Marshal(req)

	msg := &Message{
		Type:    UpdateTodo,
		Payload: payload,
	}

	response, err := c.sendRequest(msg)
	if err != nil {
		return err
	}

	if response.Type == Error {
		return fmt.Errorf(response.Error)
	}

	return nil
}

func (c *TodoClient) DeleteTodo(id int) error {
	req := DeleteTodoRequest{Id: id}
	payload, _ := json.Marshal(req)

	msg := &Message{
		Type:    DeleteTodo,
		Payload: payload,
	}

	response, err := c.sendRequest(msg)
	if err != nil {
		return err
	}

	if response.Type == Error {
		return fmt.Errorf(response.Error)
	}

	return nil
}

func (c *TodoClient) ListTodos() ([]*Todo, error) {
	msg := &Message{Type: ListTodos}

	response, err := c.sendRequest(msg)
	if err != nil {
		return nil, err
	}

	if response.Type == Error {
		return nil, fmt.Errorf(response.Error)
	}

	var listResp ListTodosResponse
	payloadBytes, err := getClientPayloadBytes(response.Payload)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(payloadBytes, &listResp); err != nil {
		return nil, err
	}

	return listResp.Todos, nil
}

func (c *TodoClient) UploadFile(todoID int, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	stream, err := c.conn.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}
	defer stream.Close()

	req := FileUploadRequest{
		TodoID:   todoID,
		FileName: fileInfo.Name(),
		FileSize: fileInfo.Size(),
	}

	msg := &Message{
		Type:    UploadFile,
		ReqId:   uuid.New().String(),
		Payload: mustMarshal(req),
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(msg); err != nil {
		return err
	}

	decoder := json.NewDecoder(stream)
	var response Message
	if err := decoder.Decode(&response); err != nil {
		return err
	}

	if response.Type == Error {
		return fmt.Errorf(response.Error)
	}

	_, err = io.Copy(stream, file)
	return err
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
