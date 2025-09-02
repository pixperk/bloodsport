package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
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
		}
	}
}

func (c *Client) interactiveMode() {
	fmt.Println("\nChat Commands:")
	fmt.Println("  /dm <user_id> <message>  - Send direct message")
	fmt.Println("  /quit                    - Exit")
	fmt.Println("  <message>                - Broadcast message to all")
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
