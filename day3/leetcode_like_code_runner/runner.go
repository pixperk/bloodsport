package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type CodeRunner struct {
	client    *client.Client
	languages map[string]LangConfig
}

type LangConfig struct {
	Image      string        `json:"image"`
	CompileCmd []string      `json:"compile_cmd"`
	RunCmd     []string      `json:"run_cmd"`
	Extension  string        `json:"extension"`
	Timeout    time.Duration `json:"timeout"`
}

func NewCodeRunner() (*CodeRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	lang := map[string]LangConfig{
		"python": {
			Image:     "python:3.9-slim",
			RunCmd:    []string{"python", "/app/solution.py"},
			Extension: "py",
			Timeout:   5 * time.Second,
		},
		"java": {
			Image:      "openjdk:11-jre-slim",
			CompileCmd: []string{"javac", "/app/Solution.java"},
			RunCmd:     []string{"java", "-cp", "/app", "Solution"},
			Extension:  "java",
			Timeout:    10 * time.Second,
		},
		"cpp": {
			Image:      "gcc:9",
			CompileCmd: []string{"g++", "/app/solution.cpp", "-o", "/app/solution"},
			RunCmd:     []string{"/app/solution"},
			Extension:  "cpp",
			Timeout:    10 * time.Second,
		},
	}

	return &CodeRunner{
		client:    cli,
		languages: lang,
	}, nil
}

func (r *CodeRunner) ExecuteCode(submission CodeSubmission, problem Problem) (*CodeExecutionResult, error) {
	langConfig, exists := r.languages[submission.Language]
	if !exists {
		return &CodeExecutionResult{
			Status: "LANGUAGE_NOT_SUPPORTED",
		}, nil
	}

	contId, err := r.createContainer(langConfig)
	if err != nil {
		return nil, err
	}
	defer r.cleanup(contId)

	if err := r.copyCodeToContainer(contId, submission.Code, langConfig.Extension); err != nil {
		return nil, err
	}

	if langConfig.CompileCmd != nil {
		if err := r.compile(contId, langConfig); err != nil {
			return &CodeExecutionResult{
				Status:       "COMPILE_ERROR",
				CompileError: err.Error(),
			}, nil
		}
	}

	results := make([]TestResult, len(problem.TestCases))
	passed := 0

	for i, testCase := range problem.TestCases {
		res := r.runTestCase(contId, langConfig, testCase)
		results[i] = res
		if res.Status == "ACCEPTED" {
			passed++
		}
	}

	status := "WRONG_ANSWER"
	if passed == len(problem.TestCases) {
		status = "ACCEPTED"
	}

	return &CodeExecutionResult{
		Status:      status,
		PassedTests: passed,
		TotalTests:  len(problem.TestCases),
		TestResults: results,
	}, nil
}

func (r *CodeRunner) createContainer(conf LangConfig) (string, error) {
	ctx := context.Background()

	contConf := &container.Config{
		Image:      conf.Image,
		Cmd:        []string{"sleep", "300"}, //keep container alive
		WorkingDir: "/app",
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   128 * 1024 * 1024, // 128MB
			NanoCPUs: 500000000,         // 0.5 CPU
		},
		NetworkMode: "none", //disable networking
	}

	resp, err := r.client.ContainerCreate(ctx, contConf, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (r *CodeRunner) copyCodeToContainer(containerId, code, extension string) error {
	filename := fmt.Sprintf("solution.%s", extension)
	if extension == "java" {
		filename = "Solution.java" //java needs class name
	}

	//tar archive with code file
	tarContent := createTarWithFile(filename, code)
	return r.client.CopyToContainer(context.Background(), containerId, "/app", tarContent, container.CopyToContainerOptions{})
}

func (r *CodeRunner) compile(containerID string, conf LangConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), conf.Timeout)
	defer cancel()

	execConf := container.ExecOptions{
		Cmd:          conf.CompileCmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := r.client.ContainerExecCreate(ctx, containerID, execConf)
	if err != nil {
		return err
	}

	hijackedResp, err := r.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return err
	}
	defer hijackedResp.Close()

	output, err := io.ReadAll(hijackedResp.Reader)
	if err != nil {
		return err
	}

	inspectResp, err := r.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return err
	}

	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("compilation failed: %s", string(output))
	}

	return nil
}

func (cr *CodeRunner) runTestCase(containerID string, config LangConfig, testCase TestCase) TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	startTime := time.Now()

	// Prepare execution with input
	execConfig := container.ExecOptions{
		Cmd:          config.RunCmd,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := cr.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return TestResult{
			Input:         testCase.Input,
			Expected:      testCase.Expected,
			Status:        "SYSTEM_ERROR",
			ExecutionTime: time.Since(startTime),
		}
	}

	hijackedResp, err := cr.client.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{})
	if err != nil {
		return TestResult{
			Input:         testCase.Input,
			Expected:      testCase.Expected,
			Status:        "SYSTEM_ERROR",
			ExecutionTime: time.Since(startTime),
		}
	}
	defer hijackedResp.Close()

	// Send input
	hijackedResp.Conn.Write([]byte(testCase.Input))
	hijackedResp.CloseWrite()

	// Read output
	output, err := io.ReadAll(hijackedResp.Reader)
	executionTime := time.Since(startTime)

	if ctx.Err() == context.DeadlineExceeded {
		return TestResult{
			Input:         testCase.Input,
			Expected:      testCase.Expected,
			Status:        "TIME_LIMIT_EXCEEDED",
			ExecutionTime: executionTime,
		}
	}

	if err != nil {
		return TestResult{
			Input:         testCase.Input,
			Expected:      testCase.Expected,
			Status:        "RUNTIME_ERROR",
			ExecutionTime: executionTime,
		}
	}

	actual := strings.TrimSpace(string(output))
	expected := strings.TrimSpace(testCase.Expected)

	status := "WRONG_ANSWER"
	if actual == expected {
		status = "ACCEPTED"
	}

	return TestResult{
		Input:         testCase.Input,
		Expected:      expected,
		Actual:        actual,
		Status:        status,
		ExecutionTime: executionTime,
	}
}

func (cr *CodeRunner) cleanup(containerID string) {
	ctx := context.Background()
	cr.client.ContainerKill(ctx, containerID, "SIGKILL")
	cr.client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

func createTarWithFile(filename, content string) io.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	header := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return strings.NewReader("")
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		return strings.NewReader("")
	}

	return &buf
}
