package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func codeJudgeMain() {
	client := NewCodeJudgeClient()
	if err := client.Connect("localhost:8443"); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer client.Close()

	fmt.Println("Code Judge - LeetCode Style Runner")
	fmt.Println("Commands:")
	fmt.Println("  ping                        - Test connection")
	fmt.Println("  problems                    - List all problems")
	fmt.Println("  get <problem_id>            - Get problem details")
	fmt.Println("  submit <problem_id> <lang> <file> - Submit solution")
	fmt.Println("  quit                        - Exit")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("judge> ")
		if !scanner.Scan() {
			break
		}

		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "ping":
			if err := client.Ping(); err != nil {
				fmt.Printf("Ping failed: %v\n", err)
			} else {
				fmt.Println("Pong!")
			}

		case "problems":
			problems, err := client.ListProblems()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Printf("Available problems:\n")
			for _, problem := range problems {
				fmt.Printf("  %s: %s\n", problem.ID, problem.Title)
			}

		case "get":
			if len(parts) != 2 {
				fmt.Println("Usage: get <problem_id>")
				continue
			}

			problem, err := client.GetProblem(parts[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Printf("\nProblem: %s\n", problem.Title)
			fmt.Printf("Description: %s\n", problem.Description)
			fmt.Printf("Test cases: %d\n", len(problem.TestCases))
			fmt.Printf("\nTemplate:\n%s\n", problem.Template)

		case "submit":
			if len(parts) != 4 {
				fmt.Println("Usage: submit <problem_id> <language> <file>")
				fmt.Println("Supported languages: python, java, cpp")
				continue
			}

			code, err := client.UploadFile(parts[3])
			if err != nil {
				fmt.Printf("Error reading file: %v\n", err)
				continue
			}

			submission := CodeSubmission{
				ProblemID: parts[1],
				Language:  parts[2],
				Code:      code,
			}

			fmt.Printf("Submitting solution for %s...\n", parts[1])

			result, err := client.SubmitCode(submission)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
			fmt.Printf("RESULT: %s\n", result.Status)
			fmt.Printf("Passed: %d/%d tests\n", result.PassedTests, result.TotalTests)

			if result.CompileError != "" {
				fmt.Printf("\nCompile Error:\n%s\n", result.CompileError)
			}

			fmt.Printf("\nTest Results:\n")
			for i, testResult := range result.TestResults {
				status := testResult.Status
				if status == "ACCEPTED" {
					status = "✓ " + status
				} else {
					status = "✗ " + status
				}

				fmt.Printf("Test %d: %s (%.2fms)\n",
					i+1, status,
					float64(testResult.ExecutionTime.Nanoseconds())/1e6)

				if testResult.Status != "ACCEPTED" {
					fmt.Printf("  Input: %s\n", testResult.Input)
					fmt.Printf("  Expected: %s\n", testResult.Expected)
					fmt.Printf("  Actual: %s\n", testResult.Actual)
				}
			}
			fmt.Println(strings.Repeat("=", 50))

		case "quit":
			return

		default:
			fmt.Println("Unknown command. Type 'quit' to exit.")
		}
	}
}
