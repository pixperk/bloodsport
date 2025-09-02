package protocol

import "io"

type MessageType int

const (
	TypeInitAck MessageType = iota
	TypeChat
	TypeFile
)

// Main message wrapper - this is what gets sent over the network
type Message struct {
	Type MessageType `json:"type"`

	// Only one of these will be populated based on Type
	InitAck *InitAck `json:"init_ack,omitempty"`
	Chat    *Chat    `json:"chat,omitempty"`
	File    *File    `json:"file,omitempty"`
}

type InitAck struct { //Client sends ID and Name
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Chat struct {
	FromID  string `json:"from_id"`         // Who sent this message
	ToID    string `json:"to_id,omitempty"` // Empty = broadcast, otherwise DM
	Message string `json:"message"`
}

type File struct {
	FromID     string `json:"from_id"`         // Who sent this file
	ToID       string `json:"to_id,omitempty"` // Empty = broadcast, otherwise DM
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	BufferSize int64  `json:"buffer_size"`
	Reader     io.Reader
}
