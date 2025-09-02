package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/pixperk/bloodsport/day1_tcp_udp/tcp_echo/client"
	"github.com/pixperk/bloodsport/day1_tcp_udp/tcp_echo/server"
)

func getFreePort() int {
	addr, _ := net.ResolveTCPAddr("tcp", "localhost:0")
	l, _ := net.ListenTCP("tcp", addr)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestTCPMultipleClients(t *testing.T) {

	port := getFreePort()
	addr := fmt.Sprintf("localhost:%d", port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := server.NewTCPServer(addr)
	go s.StartWithContext(ctx)

	time.Sleep(100 * time.Millisecond)

	numClients := 5
	errors := make(chan error, numClients)

	for i := range numClients {
		go func(clientID int) {

			c := client.NewTCPClient(addr)
			err := c.Connect()
			if err != nil {
				errors <- fmt.Errorf("client %d failed to connect: %v", clientID, err)
				return
			}
			defer c.Close()

			message := fmt.Sprintf("Hello from client %d", clientID)
			err = c.SendLine(message)
			if err != nil {
				errors <- fmt.Errorf("client %d failed to send message: %v", clientID, err)
				return
			}
			errors <- nil
		}(i)
	}

	for range numClients {
		err := <-errors
		if err != nil {
			t.Errorf("Client error: %v", err)
		}
	}
}
