package main

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type MessageType int

const (
	SubmitCode MessageType = iota
	ExecutionResult
	GetProblem
	ListProblems
	Error
	Ping
	Pong
)

type Message struct {
	Type    MessageType `json:"type"`
	ReqId   string      `json:"req_id"`
	Payload interface{} `json:"payload"`
	Error   string      `json:"error,omitempty"`
}

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
	MemoryUsage   int64         `json:"memory_usage"`
	CompileError  string        `json:"compile_error,omitempty"`
	TestResults   []TestResult  `json:"test_results"`
}

type TestResult struct {
	Input         string        `json:"input"`
	Expected      string        `json:"expected"`
	Actual        string        `json:"actual"`
	Status        string        `json:"status"`
	ExecutionTime time.Duration `json:"execution_time_ms"`
}

type Problem struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	TestCases   []TestCase `json:"test_cases"`
	Template    string     `json:"template"`
}

type TestCase struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
}

func getPayloadBytes(payload interface{}) ([]byte, error) {
	switch p := payload.(type) {
	case []byte:
		return p, nil
	case string:
		// Try base64 decode first (for JSON marshaled []byte)
		if decoded, err := base64.StdEncoding.DecodeString(p); err == nil {
			return decoded, nil
		}
		// If not base64, treat as regular string
		return []byte(p), nil
	default:
		return json.Marshal(p)
	}
}

func getClientPayloadBytes(payload interface{}) ([]byte, error) {
	switch p := payload.(type) {
	case []byte:
		return p, nil
	case string:
		// Try base64 decode first
		if decoded, err := base64.StdEncoding.DecodeString(p); err == nil {
			return decoded, nil
		}
		return []byte(p), nil
	default:
		return json.Marshal(p)
	}
}
