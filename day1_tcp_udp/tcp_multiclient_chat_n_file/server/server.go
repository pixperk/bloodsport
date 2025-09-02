package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"

	protocol "github.com/pixperk/bloodsport/day1_tcp_udp/tcp_multiclient_chat_n_file"
)

type Server struct {
	ListenAddr string

	mu      sync.RWMutex
	clients map[*Client]bool
}

type Client struct {
	Conn net.Conn
	ID   string //can also serve as file prefix
	Name string
}

func NewServer(listenAddr string) *Server {
	return &Server{
		ListenAddr: listenAddr,
		clients:    make(map[*Client]bool),
	}
}

func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.ListenAddr, err)
	}

	defer lis.Close()

	fmt.Printf("chat and file transfer server listening on %s\n", s.ListenAddr)

	s.acceptConns(ctx, lis)

	return nil
}

func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	clients := make([]*Client, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}

	s.mu.Unlock()

	for _, c := range clients {
		c.Conn.Close()
		s.removeClient(c) // This will acquire its own lock
	}
}

func (s *Server) acceptConns(ctx context.Context, lis net.Listener) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Server shutting down...")
			s.Close()
			return
		default:
		}
		conn, err := lis.Accept()
		if err != nil {
			fmt.Printf("accept error: %v\n", err)
			continue
		}

		go s.handleNewConnection(conn)
	}
}

func (s *Server) handleNewConnection(conn net.Conn) {
	//Unique ID and name are generated on the client side
	client := &Client{
		Conn: conn,
	}

	defer func() {
		conn.Close()
		s.removeClient(client)
	}()

	decoder := json.NewDecoder(conn)

	for {
		var msg protocol.Message

		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				fmt.Printf("Client %s disconnected\n", client.ID)
				return
			}
			fmt.Println("failed to decode message:", err)
			continue
		}
		s.handleMessage(client, &msg)
	}
}

func (s *Server) handleMessage(client *Client, msg *protocol.Message) {
	switch msg.Type {
	case protocol.TypeInitAck:
		if msg.InitAck != nil {
			s.handleInitAck(client, msg.InitAck)
		}
	case protocol.TypeChat:
		if msg.Chat != nil {
			s.handleChat(client, msg.Chat)
		}
	case protocol.TypeFile:
		if msg.File != nil {
			s.handleFile(client, msg.File)
		}
	default:
		fmt.Printf("Unknown message type %d from client %s\n", msg.Type, client.ID)
	}
}

func (s *Server) handleInitAck(client *Client, initAck *protocol.InitAck) {
	client.ID = initAck.ID
	client.Name = initAck.Name

	s.addClient(client)

	fmt.Printf("Client registered: ID=%s, Name=%s\n", client.ID, client.Name)

	ackMsg := &protocol.Message{
		Type: protocol.TypeInitAck,
		InitAck: &protocol.InitAck{
			ID:   client.ID,
			Name: client.Name,
		},
	}

	s.broadcastToAll(ackMsg)
}

func (s *Server) handleChat(client *Client, chat *protocol.Chat) {
	if chat.FromID != client.ID {
		fmt.Printf("Mismatched FromID in chat message: expected %s, got %s\n", client.ID, chat.FromID)
		return
	}

	chatMsg := &protocol.Message{
		Type: protocol.TypeChat,
		Chat: chat,
	}

	if chat.ToID == "" {
		// Broadcast message
		s.broadcastToAll(chatMsg)
	} else {
		// Direct message
		receiver, ok := s.getClientByID(chat.ToID)
		if !ok {
			fmt.Printf("Unknown recipient ID %s\n", chat.ToID)
			return
		}
		s.sendTo(receiver, chatMsg)
	}
}

// this just plays with the metadata
func (s *Server) handleFile(client *Client, file *protocol.File) {
	if file.FromID != client.ID {
		fmt.Printf("Mismatched FromID in file transfer: expected %s, got %s\n", client.ID, file.FromID)
		return
	}

	fileMsg := &protocol.Message{
		Type: protocol.TypeFile,
		File: file,
	}

	if file.ToID == "" {
		s.broadcastToAll(fileMsg)
	} else {
		receiver, ok := s.getClientByID(file.ToID)
		if !ok {
			fmt.Printf("Unknown recipient for file: %s\n", file.ToID)
			return
		}
		s.sendTo(receiver, fileMsg)
	}

	s.handleFileData(client, file)

}

func (s *Server) handleFileData(client *Client, file *protocol.File) {
	buffer := make([]byte, file.BufferSize)
	var totalReceived int64

	for totalReceived < file.Size {
		n, err := client.Conn.Read(buffer)
		if err != nil {
			fmt.Printf("Error reading file data: %v\n", err)
			return
		}

		if file.ToID == "" {
			s.broadcastFileChunk(client, buffer[:n])
		} else {
			receiver, ok := s.getClientByID(file.ToID)
			if ok {
				receiver.Conn.Write(buffer[:n])
			}
		}

		totalReceived += int64(n)
	}

	fmt.Printf("File transfer complete: %s (%d bytes)\n", file.Name, totalReceived)
}

func (s *Server) getClientByID(id string) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client.ID == id {
			return client, true
		}
	}
	return nil, false
}

func (s *Server) sendTo(client *Client, msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := json.NewEncoder(client.Conn).Encode(msg); err != nil {
		fmt.Printf("failed to send message to client %s: %v\n", client.ID, err)
	}
}

func (s *Server) broadcastToAll(msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if msg.Type == protocol.TypeChat && msg.Chat != nil && client.ID == msg.Chat.FromID {
			continue // Skip sender
		}
		if err := json.NewEncoder(client.Conn).Encode(msg); err != nil {
			fmt.Printf("failed to send message to client %s: %v\n", client.ID, err)
		}
	}
}

func (s *Server) broadcastFileChunk(sender *Client, chunk []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client.ID == sender.ID {
			continue // Skip sender
		}
		if _, err := client.Conn.Write(chunk); err != nil {
			fmt.Printf("failed to send file chunk to client %s: %v\n", client.ID, err)
		}
	}
}

func (s *Server) addClient(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = true
}

func (s *Server) removeClient(c *Client) {
	delete(s.clients, c)
}
