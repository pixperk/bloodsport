package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func serverMain() {
	var (
		addr    = flag.String("addr", ":8443", "Server address")
		dataDir = flag.String("data", "./data", "Data directory")
	)
	flag.Parse()

	storage := NewFileStorage(*dataDir)
	if err := storage.EnsureDataDir(); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	if file, err := os.Open(storage.todosFile); err == nil {
		storage.Load(file)
		file.Close()
		log.Println("Loaded existing todos")
	}

	server := NewTodoServer(storage)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		log.Println("Saving todos before shutdown...")
		if file, err := os.Create(storage.todosFile); err == nil {
			storage.Save(file)
			file.Close()
		}
		os.Exit(0)
	}()

	log.Fatal(server.Start(*addr))
}
