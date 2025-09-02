package client

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type TCPClient struct {
	SrvAddr   string
	conn      net.Conn
	reader    *bufio.Reader
	timeout   time.Duration
	mu        sync.Mutex
	connected bool
}

func NewTCPClient(addr string) *TCPClient {
	return &TCPClient{
		SrvAddr: addr,
		timeout: 30 * time.Second,
	}
}

func (c *TCPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.connected {
		return nil
	}
	conn, err := net.DialTimeout("tcp", c.SrvAddr, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	c.connected = true

	return nil
}

func (c *TCPClient) SendLine(line string) error {
	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	if !strings.HasSuffix(line, "\n") {
		line += "\n"
	}

	_, err := c.conn.Write([]byte(line))
	return err
}

func (c *TCPClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.connected {
		return fmt.Errorf("not connected to server")
	}
	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	c.connected = false
	c.conn = nil
	c.reader = nil
	return nil
}
