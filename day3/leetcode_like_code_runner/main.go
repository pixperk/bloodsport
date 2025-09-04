package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . server  - Start the judge server")
		fmt.Println("  go run . client  - Start the client")
		os.Exit(1)
	}

	mode := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	switch mode {
	case "server":
		judgeServerMain()
	case "client":
		codeJudgeMain()
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		fmt.Println("Use 'server' or 'client'")
		os.Exit(1)
	}
}

func judgeServerMain() {
	var addr = flag.String("addr", ":8443", "Server address")
	flag.Parse()

	server, err := NewJudgeServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	// Handle graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		log.Println("Shutting down judge server...")
		os.Exit(0)
	}()

	log.Printf("Starting Code Judge server on %s", *addr)
	log.Fatal(server.Start(*addr))
}
