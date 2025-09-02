package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	protocol "github.com/pixperk/bloodsport/day1_tcp_udp/tcp_multiclient_chat_n_file"
)

type Client struct {
	conn   net.Conn
	id     string
	name   string
	reader *bufio.Reader
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run client.go <client_name>")
		os.Exit(1)
	}

	clientName := os.Args[1]
	client := &Client{
		id:     fmt.Sprintf("client_%s_%d", clientName, os.Getpid()),
		name:   clientName,
		reader: bufio.NewReader(os.Stdin),
	}

	if err := client.connect("localhost:8080"); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	defer client.conn.Close()

	if err := client.sendInitAck(); err != nil {
		fmt.Printf("Failed to send init ack: %v\n", err)
		return
	}

	go client.receiveMessages()

	client.interactiveMode()
}

func (c *Client) connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn
	fmt.Printf("Connected to server as %s (ID: %s)\n", c.name, c.id)
	return nil
}

func (c *Client) sendInitAck() error {
	msg := &protocol.Message{
		Type: protocol.TypeInitAck,
		InitAck: &protocol.InitAck{
			ID:   c.id,
			Name: c.name,
		},
	}
	return json.NewEncoder(c.conn).Encode(msg)
}

func (c *Client) receiveMessages() {
	decoder := json.NewDecoder(c.conn)
	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			fmt.Printf("\nConnection lost: %v\n", err)
			return
		}

		switch msg.Type {
		case protocol.TypeInitAck:
			if msg.InitAck != nil && msg.InitAck.ID != c.id {
				fmt.Printf("\n[SYSTEM] %s joined the chat\n", msg.InitAck.Name)
				fmt.Print("> ")
			}
		case protocol.TypeChat:
			if msg.Chat != nil {
				if msg.Chat.ToID == "" {
					fmt.Printf("\n[BROADCAST] %s: %s\n", msg.Chat.FromID, msg.Chat.Message)
				} else {
					fmt.Printf("\n[DM] %s: %s\n", msg.Chat.FromID, msg.Chat.Message)
				}
				fmt.Print("> ")
			}
		case protocol.TypeFile:
			if msg.File != nil {
				go c.receiveFile(msg.File)
			}
		}
	}
}

func (c *Client) interactiveMode() {
	fmt.Println("\nChat Commands:")
	fmt.Println("  /dm <user_id> <message>     - Send direct message")
	fmt.Println("  /file <path>                - Broadcast file to all")
	fmt.Println("  /sendfile <user_id> <path>  - Send file to specific user")
	fmt.Println("  /quit                       - Exit")
	fmt.Println("  <message>                   - Broadcast message to all")
	fmt.Print("> ")

	for {
		input, err := c.reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			fmt.Print("> ")
			continue
		}

		if input == "/quit" {
			fmt.Println("Goodbye!")
			break
		}

		if strings.HasPrefix(input, "/dm ") {
			c.handleDirectMessage(input)
		} else if strings.HasPrefix(input, "/file ") {
			c.handleFileCommand(input, "")
		} else if strings.HasPrefix(input, "/sendfile ") {
			c.handleSendFileCommand(input)
		} else {
			c.sendBroadcastMessage(input)
		}
		fmt.Print("> ")
	}
}

func (c *Client) handleDirectMessage(input string) {
	parts := strings.SplitN(input[4:], " ", 2) // Remove "/dm "
	if len(parts) < 2 {
		fmt.Println("Usage: /dm <user_id> <message>")
		return
	}

	toID := parts[0]
	message := parts[1]

	c.sendChatMessage(toID, message)
}

func (c *Client) sendBroadcastMessage(message string) {
	c.sendChatMessage("", message)
}

func (c *Client) sendChatMessage(toID, message string) {
	msg := &protocol.Message{
		Type: protocol.TypeChat,
		Chat: &protocol.Chat{
			FromID:  c.id,
			ToID:    toID,
			Message: message,
		},
	}

	if err := json.NewEncoder(c.conn).Encode(msg); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
	}
}

// File transfer methods
func (c *Client) handleFileCommand(input, toID string) {
	filePath := strings.TrimSpace(input[6:]) // Remove "/file "
	if filePath == "" {
		fmt.Println("Usage: /file <path>")
		return
	}
	c.sendFile(filePath, toID)
}

func (c *Client) handleSendFileCommand(input string) {
	parts := strings.SplitN(input[10:], " ", 2) // Remove "/sendfile "
	if len(parts) < 2 {
		fmt.Println("Usage: /sendfile <user_id> <path>")
		return
	}

	toID := parts[0]
	filePath := parts[1]
	c.sendFile(filePath, toID)
}

func (c *Client) sendFile(filePath, toID string) {
	// Check if file exists and get info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Error accessing file %s: %v\n", filePath, err)
		return
	}

	fileName := filepath.Base(filePath)
	fileSize := fileInfo.Size()
	bufferSize := int64(1024) // 1KB chunks

	fmt.Printf("Sending file: %s (%d bytes) to %s\n", fileName, fileSize,
		func() string {
			if toID == "" {
				return "all users"
			}
			return toID
		}())

	// Send file metadata
	fileMsg := &protocol.Message{
		Type: protocol.TypeFile,
		File: &protocol.File{
			FromID:     c.id,
			ToID:       toID,
			Name:       fileName,
			Size:       fileSize,
			BufferSize: bufferSize,
		},
	}

	if err := json.NewEncoder(c.conn).Encode(fileMsg); err != nil {
		fmt.Printf("Failed to send file metadata: %v\n", err)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, bufferSize)
	var totalSent int64

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			return
		}

		if _, err := c.conn.Write(buffer[:n]); err != nil {
			fmt.Printf("Error sending file data: %v\n", err)
			return
		}

		totalSent += int64(n)
	}

	fmt.Printf("File sent successfully: %d bytes\n", totalSent)
}

func (c *Client) receiveFile(file *protocol.File) {
	if file.ToID == "" {
		fmt.Printf("\n[FILE BROADCAST] %s is sending: %s (%d bytes)\n", file.FromID, file.Name, file.Size)
	} else {
		fmt.Printf("\n[FILE DM] %s is sending: %s (%d bytes)\n", file.FromID, file.Name, file.Size)
	}

	if err := os.MkdirAll("received_files", 0755); err != nil {
		fmt.Printf("Failed to create received_files directory: %v\n", err)
		return
	}

	outputPath := filepath.Join("received_files", fmt.Sprintf("%s_%s", file.FromID, file.Name))

	outFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Failed to create output file: %v\n", err)
		return
	}
	defer outFile.Close()

	buffer := make([]byte, file.BufferSize)
	var totalReceived int64

	for totalReceived < file.Size {
		n, err := c.conn.Read(buffer)
		if err != nil {
			fmt.Printf("Error receiving file data: %v\n", err)
			return
		}

		if _, err := outFile.Write(buffer[:n]); err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
			return
		}

		totalReceived += int64(n)
	}

	fmt.Printf("File received: %s (%d bytes) -> %s\n", file.Name, totalReceived, outputPath)
	fmt.Print("> ")
}
