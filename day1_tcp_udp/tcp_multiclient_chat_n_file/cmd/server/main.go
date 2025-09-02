package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pixperk/bloodsport/day1_tcp_udp/tcp_multiclient_chat_n_file/server"
)

func main() {
	srv := server.NewServer(":8080")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown signal received...")
		cancel()
	}()

	fmt.Println("Starting TCP chat server on :8080...")
	if err := srv.Start(ctx); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
