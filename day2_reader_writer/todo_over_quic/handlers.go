package main

import (
	"encoding/json"
	"log"

	"github.com/quic-go/quic-go"
)

type Response struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type ListTodosResponse struct {
	Todos []*Todo `json:"todos"`
}

type FileUploadRequest struct {
	TodoID   int    `json:"todo_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

type FileUploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func (s *TodoServer) handleCreateTodo(msg *Message) *Message {
	var req CreateTodoRequest
	if err := json.Unmarshal(msg.Payload.([]byte), &req); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid request payload",
		}
	}

	todo := &Todo{
		Title: req.Title,
		Done:  false,
		Files: make([]*File, 0),
	}

	if err := s.storage.Create(todo); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	response := Response{
		Ok: true,
	}
	payload, _ := json.Marshal(response)

	return &Message{
		Type:    CreateTodo,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (s *TodoServer) handleReadTodo(msg *Message) *Message {
	var req ReadTodoRequest
	if err := json.Unmarshal(msg.Payload.([]byte), &req); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid request payload",
		}
	}

	todo, err := s.storage.Read(req.Id)
	if err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	payload, _ := json.Marshal(todo)

	return &Message{
		Type:    ReadTodo,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (s *TodoServer) handleUpdateTodo(msg *Message) *Message {
	var req UpdateTodoRequest
	if err := json.Unmarshal(msg.Payload.([]byte), &req); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid request payload",
		}
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	updates["done"] = req.Done

	if err := s.storage.Update(req.Id, updates); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	payload := Response{
		Ok: true,
	}
	payloadData, _ := json.Marshal(payload)

	return &Message{
		Type:    UpdateTodo,
		ReqId:   msg.ReqId,
		Payload: payloadData,
	}
}

func (s *TodoServer) handleDeleteTodo(msg *Message) *Message {
	var req DeleteTodoRequest
	if err := json.Unmarshal(msg.Payload.([]byte), &req); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid request payload",
		}
	}

	if err := s.storage.Delete(req.Id); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	response := Response{
		Ok: true,
	}
	payload, _ := json.Marshal(response)

	return &Message{
		Type:    DeleteTodo,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (s *TodoServer) handleListTodos(msg *Message) *Message {
	todos, err := s.storage.List()
	if err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	response := ListTodosResponse{
		Todos: todos,
	}
	payload, _ := json.Marshal(response)

	return &Message{
		Type:    ListTodos,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (s *TodoServer) handleFileUpload(msg *Message, stream *quic.Stream) *Message {
	var req FileUploadRequest
	if err := json.Unmarshal(msg.Payload.([]byte), &req); err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: "Invalid file upload request",
		}
	}

	go func() {
		err := s.storage.SaveFile(req.TodoID, req.FileName, stream, req.FileSize)
		if err != nil {
			log.Printf("File upload failed: %v", err)
		}
	}()

	response := FileUploadResponse{
		Success: true,
		Message: "Upload initiated",
	}
	payload, _ := json.Marshal(response)

	return &Message{
		Type:    UploadFile,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}

func (s *TodoServer) handleUploadTodos(msg *Message, stream *quic.Stream) *Message {
	err := s.storage.Save(stream)
	if err != nil {
		return &Message{
			Type:  Error,
			ReqId: msg.ReqId,
			Error: err.Error(),
		}
	}

	response := Response{
		Ok: true,
	}
	payload, _ := json.Marshal(response)

	return &Message{
		Type:    UploadTodos,
		ReqId:   msg.ReqId,
		Payload: payload,
	}
}
