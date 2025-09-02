package client

import (
	"net"
	"sync"
	"time"
)

type UDPClient struct {
	SrvAddr string
	conn    *net.UDPConn
	timeout time.Duration
	mu      sync.Mutex
}

func NewUDPClient(addr string, timeout time.Duration) (*UDPClient, error) {

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, err
	}

	return &UDPClient{
		SrvAddr: addr,
		conn:    conn,
		timeout: timeout,
	}, nil
}

func (c *UDPClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	srvAddr, err := net.ResolveUDPAddr("udp", c.SrvAddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, srvAddr)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *UDPClient) SendMessage(msg string) error {
	_, err := c.conn.Write([]byte(msg))
	return err
}

func (c *UDPClient) ReceiveMessage() (string, error) {
	// Set read timeout
	c.conn.SetReadDeadline(time.Now().Add(c.timeout))

	buffer := make([]byte, 1024)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return "", err
	}

	return string(buffer[:n]), nil
}
