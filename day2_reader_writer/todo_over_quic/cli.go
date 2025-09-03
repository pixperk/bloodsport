package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func clientMain() {
	var addr = flag.String("addr", "localhost:8443", "Server address")
	flag.Parse()

	client := NewTodoClient()
	if err := client.Connect(*addr); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer client.Close()

	fmt.Println("Connected to QUIC todo server!")
	fmt.Println("Commands:")
	fmt.Println("  ping                         - Test connection")
	fmt.Println("  create <title>               - Create a new todo")
	fmt.Println("  read <id>                    - Read a todo by ID")
	fmt.Println("  update <id> <title> <done>   - Update todo (done: true/false)")
	fmt.Println("  delete <id>                  - Delete a todo")
	fmt.Println("  list                         - List all todos")
	fmt.Println("  upload <todo_id> <file_path> - Upload file to todo")
	fmt.Println("  quit                         - Exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]

		switch cmd {
		case "ping":
			if err := client.Ping(); err != nil {
				fmt.Printf("Ping failed: %v\n", err)
			} else {
				fmt.Println("Pong!")
			}

		case "create":
			if len(parts) < 2 {
				fmt.Println("Usage: create <title>")
				continue
			}
			title := strings.Join(parts[1:], " ")
			_, err := client.CreateTodo(title)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Created todo: %s\n", title)
			}

		case "read":
			if len(parts) != 2 {
				fmt.Println("Usage: read <id>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid todo ID")
				continue
			}
			todo, err := client.ReadTodo(id)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				status := "incomplete"
				if todo.Done {
					status = "complete"
				}
				fmt.Printf("Todo %d: %s [%s]\n", todo.Id, todo.Title, status)
				if len(todo.Files) > 0 {
					fmt.Printf("Files: %d attached\n", len(todo.Files))
				}
			}

		case "update":
			if len(parts) < 4 {
				fmt.Println("Usage: update <id> <title> <done>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid todo ID")
				continue
			}
			title := parts[2]
			done, err := strconv.ParseBool(parts[3])
			if err != nil {
				fmt.Println("Invalid done value (use true/false)")
				continue
			}
			if err := client.UpdateTodo(id, title, done); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Updated todo %d\n", id)
			}

		case "delete":
			if len(parts) != 2 {
				fmt.Println("Usage: delete <id>")
				continue
			}
			id, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid todo ID")
				continue
			}
			if err := client.DeleteTodo(id); err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("Deleted todo %d\n", id)
			}

		case "list":
			todos, err := client.ListTodos()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				if len(todos) == 0 {
					fmt.Println("No todos found")
				} else {
					fmt.Printf("Found %d todos:\n", len(todos))
					for _, todo := range todos {
						status := "incomplete"
						if todo.Done {
							status = "complete"
						}
						fmt.Printf("  %d: %s [%s]\n", todo.Id, todo.Title, status)
					}
				}
			}

		case "upload":
			if len(parts) != 3 {
				fmt.Println("Usage: upload <todo_id> <file_path>")
				continue
			}
			todoID, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid todo ID")
				continue
			}
			if err := client.UploadFile(todoID, parts[2]); err != nil {
				fmt.Printf("Upload failed: %v\n", err)
			} else {
				fmt.Println("File uploaded successfully!")
			}

		case "quit":
			return

		default:
			fmt.Println("Unknown command. Type 'quit' to exit.")
		}
	}
}
