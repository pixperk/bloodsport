package main

import "time"

type MessageType int

const (
	SubmitCode MessageType = iota
	ExecutionResult
	GetProblem
	ListProblems
	Error
)

type CodeSubmission struct {
	ProblemID string `json:"problem_id"`
	Language  string `json:"language"`
	Code      string `json:"code"`
}

type CodeExecutionResult struct {
	Status        string        `json:"status"`
	PassedTests   int           `json:"passed_tests"`
	TotalTests    int           `json:"total_tests"`
	ExecutionTime time.Duration `json:"execution_time_ms"`
}

type TestResult struct {
	Input         string        `json:"input"`
	Expected      string        `json:"expected"`
	Actual        string        `json:"actual"`
	Status        string        `json:"status"`
	ExecutionTime time.Duration `json:"execution_time_ms"`
}

type Problem struct {
	Input       string     `json:"input"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TestCases   []TestCase `json:"test_cases"`
	Template    string     `json:"template"`
}

type TestCase struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
}
