package main

const (
	Ping = iota
	Pong
	Error
	CreateTodo
	ReadTodo
	UpdateTodo
	DeleteTodo
	ListTodos
	UploadFile
	UploadTodos
)

type Message struct {
	Type    int    `json:"type"`
	ReqId   string `json:"req_id"`
	Payload any    `json:"payload"`
	Error   string `json:"error,omitempty"`
}

type Todo struct {
	Id    int     `json:"id"`
	Title string  `json:"title"`
	Done  bool    `json:"done"`
	Files []*File `json:"files,omitempty"`
}

type File struct {
	Path string `json:"path"`
	Data []byte `json:"data,omitempty"`
}

type CreateTodoRequest struct {
	Title string `json:"title"`
}
type UpdateTodoRequest struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type ReadTodoRequest struct {
	Id int `json:"id"`
}

type DeleteTodoRequest struct {
	Id int `json:"id"`
}
