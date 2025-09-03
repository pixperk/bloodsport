package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run . server [options]  - Start the server")
		fmt.Println("  go run . client [options]  - Start the client")
		os.Exit(1)
	}

	mode := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	switch mode {
	case "server":
		serverMain()
	case "client":
		clientMain()
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
		fmt.Println("Use 'server' or 'client'")
		os.Exit(1)
	}
}
