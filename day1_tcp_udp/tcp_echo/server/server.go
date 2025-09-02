package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

type TCPServer struct {
	ListenAddr  string
	mu          sync.RWMutex
	connections map[net.Conn]bool
}

func NewTCPServer(addr string) *TCPServer {
	return &TCPServer{
		ListenAddr:  addr,
		connections: make(map[net.Conn]bool),
	}
}

func (s *TCPServer) Start() error {
	return s.StartWithContext(context.Background())
}

func (s *TCPServer) StartWithContext(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}

	defer lis.Close()

	fmt.Printf("listening on %v\n", s.ListenAddr)

	s.acceptConns(ctx, lis)
	return nil
}

func (s *TCPServer) acceptConns(ctx context.Context, lis net.Listener) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Server shutting down...")
			s.closeAllConnections()
			return
		default:
		}

		conn, err := lis.Accept()
		if err != nil {
			fmt.Printf("accept error: %v\n", err)
			continue
		}

		s.addConnection(conn)
		go s.handleConn(conn)
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	defer func() {
		s.removeConnection(conn)
		conn.Close()
	}()

	reader := bufio.NewReader(conn)
	for {
		data, err := reader.ReadString('\n')
		if err == io.EOF {
			return
		} else if err != nil {
			fmt.Println("read error:", err)
			return
		}
		s.handleData(conn, data)
	}
}

func (s *TCPServer) handleData(conn net.Conn, data string) {
	_, err := conn.Write([]byte("Echo: " + data))
	if err != nil {
		fmt.Println("write error:", err)
	}
}

func (s *TCPServer) addConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[conn] = true
}

func (s *TCPServer) removeConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connections, conn)
}

func (s *TCPServer) closeAllConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.connections {
		conn.Close()
	}
	s.connections = make(map[net.Conn]bool)
}
