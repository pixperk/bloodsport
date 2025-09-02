package server

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
)

type UDPServer struct {
	ListenAddr string
	conn       *net.UDPConn

	mu         sync.RWMutex
	running    bool
	packetLoss float64 // 0.0 to 1.0 (0% to 100% loss)
}

func NewUDPServer(addr string) *UDPServer {
	return &UDPServer{
		ListenAddr: addr,
	}
}

func (s *UDPServer) Start(ctx context.Context) error {
	addr, err := net.ResolveUDPAddr("udp", s.ListenAddr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	s.conn = conn
	s.running = true

	//single goroutine reads all packets
	return s.listen(ctx)
}

func (s *UDPServer) listen(ctx context.Context) error {
	buf := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		n, clientAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}

		go s.handlePacket(buf[:n], clientAddr)
	}
}

func (s *UDPServer) simulatePacketLoss() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.packetLoss <= 0.0 {
		return false
	}

	loss := rand.Float64()
	s.setPacketLoss(loss)
	return loss < s.packetLoss
}

func (s *UDPServer) setPacketLoss(loss float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.packetLoss = loss
}

func (s *UDPServer) handlePacket(data []byte, clientAddr *net.UDPAddr) {
	if s.simulatePacketLoss() {
		fmt.Printf("Simulated packet loss from %v\n", clientAddr)
		return
	}

	response := "ECHO : " + string(data)
	s.conn.WriteToUDP([]byte(response), clientAddr)
}
