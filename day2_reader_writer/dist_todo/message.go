package main

const (
	CreateTodo = iota
	ReadTodo
	UpdateTodo
	DeleteTodo
	ListTodos
	UploadFile
)

type Message struct {
	Type    int `json:"type"`
	Payload any `json:"payload"`
}
