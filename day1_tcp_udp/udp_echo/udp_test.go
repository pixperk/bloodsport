package main

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/pixperk/bloodsport/day1_tcp_udp/udp_echo/client"
	"github.com/pixperk/bloodsport/day1_tcp_udp/udp_echo/server"
)

func TestUDPBasicEchoWithoutPacketLoss(t *testing.T) {

	srv := server.NewUDPServer("localhost:9999")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Start(ctx)
	time.Sleep(200 * time.Millisecond) // Give more time to start

	cli, err := client.NewUDPClient("localhost:9999", 3*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test single message
	testMsg := "Hello UDP Server"

	if err := cli.SendMessage(testMsg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	response, err := cli.ReceiveMessage()
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	t.Logf("Received response: %q", response)

	expected := "ECHO : " + testMsg
	if response != expected {
		t.Errorf("Expected %q, got %q", expected, response)
	}
}

func TestUDPManualConnection(t *testing.T) {

	srv := server.NewUDPServer("127.0.0.1:9997")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go srv.Start(ctx)
	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("udp", "127.0.0.1:9997")
	if err != nil {
		t.Fatalf("Failed to dial UDP: %v", err)
	}
	defer conn.Close()

	// Send message
	message := "manual test"
	_, err = conn.Write([]byte(message))
	if err != nil {
		t.Fatalf("Failed to send: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	response := string(buffer[:n])
	expected := "ECHO : " + message
	if response != expected {
		t.Errorf("Expected %q, got %q", expected, response)
	} else {
		t.Logf("Manual UDP test passed: %q", response)
	}
}
