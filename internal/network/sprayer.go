package network

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"nptx/pkg/logger"
)

// PacerQueue acts as a custom token-bucket/pacing mechanism to prevent Starlink burst drops.
func pacerLoop(conn *net.UDPConn, queue chan []byte) {
	// Simple pacing: max 10000 packets per second per socket (~80Mbps per socket at 1KB MTU)
	// Smooth out bursts.
	ticker := time.NewTicker(100 * time.Microsecond)
	defer ticker.Stop()

	for payload := range queue {
		<-ticker.C // Wait for token (pacing)
		conn.Write(payload)
	}
}

// Sprayer handles UDP packet transmission across multiple sockets (The Rain Technique).
type Sprayer struct {
	remoteAddr *net.UDPAddr
	sockets    []*managedSocket
	numStreams int
	rrIndex    uint64
	rxChan     chan []byte
}

type managedSocket struct {
	mu         sync.RWMutex
	conn       *net.UDPConn
	lastRx     time.Time
	remoteAddr *net.UDPAddr
	txQueue    chan []byte
}

func NewClientSprayer(remote string, streams int) (*Sprayer, error) {
	addr, err := net.ResolveUDPAddr("udp", remote)
	if err != nil {
		return nil, err
	}

	s := &Sprayer{
		remoteAddr: addr,
		numStreams: streams,
		sockets:    make([]*managedSocket, streams),
		rxChan:     make(chan []byte, 4096), // generous buffer
	}

	for i := 0; i < streams; i++ {
		ms, err := createSocket(addr)
		if err != nil {
			return nil, err
		}
		s.sockets[i] = ms
		go s.readLoop(ms)
	}
	return s, nil
}

func createSocket(remoteAddr *net.UDPAddr) (*managedSocket, error) {
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		return nil, err
	}

	ms := &managedSocket{
		conn:       conn,
		lastRx:     time.Now(),
		remoteAddr: remoteAddr,
		txQueue:    make(chan []byte, 1024), // burst queue
	}
	go pacerLoop(conn, ms.txQueue)

	return ms, nil
}

func (s *Sprayer) readLoop(ms *managedSocket) {
	for {
		ms.mu.RLock()
		conn := ms.conn
		ms.mu.RUnlock()

		if conn == nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		buf := make([]byte, 2048)
		n, err := conn.Read(buf)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		ms.mu.Lock()
		ms.lastRx = time.Now()
		ms.mu.Unlock()

		select {
		case s.rxChan <- buf[:n]:
		default:
		}
	}
}

func (s *Sprayer) Write(payload []byte) error {
	idx := atomic.AddUint64(&s.rrIndex, 1) % uint64(s.numStreams)
	ms := s.sockets[idx]

	select {
	case ms.txQueue <- payload:
		return nil
	default:
		return errors.New("socket tx queue full (severe network congestion)")
	}
}

// Broadcast sends the identical payload across all N sockets simultaneously (Used for Aggressive NAT Keep-Alive)
func (s *Sprayer) Broadcast(payload []byte) {
	for i := 0; i < s.numStreams; i++ {
		select {
		case s.sockets[i].txQueue <- payload:
		default:
		}
	}
}

func (s *Sprayer) Read() ([]byte, error) {
	buf := <-s.rxChan
	return buf, nil
}

func (s *Sprayer) StartHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for i := 0; i < s.numStreams; i++ {
				ms := s.sockets[i]
				ms.mu.Lock()
				// Generous disconnect timeout given Starlink's satellite handoffs
				if now.Sub(ms.lastRx) > 8*time.Second {
					logger.ErrorLog().Warn("Socket stalled, cycling port to force NAT holepunch", "index", i)
					ms.conn.Close()
					close(ms.txQueue)
					newMs, err := createSocket(ms.remoteAddr)
					if err == nil {
						ms.conn = newMs.conn
						ms.txQueue = newMs.txQueue
						ms.lastRx = time.Now()
						go s.readLoop(ms)
					}
				}
				ms.mu.Unlock()
			}
		}
	}
}

type ServerSprayer struct {
	mu          sync.RWMutex
	conn        *net.UDPConn
	activeAddrs []*net.UDPAddr
	addrLastRx  map[string]time.Time
	rrIndex     uint64
	txQueue     chan asyncFrame
}

type asyncFrame struct {
	payload []byte
	addr    *net.UDPAddr
}

func NewServerSprayer(localAddr string) (*ServerSprayer, error) {
	addr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	s := &ServerSprayer{
		conn:        conn,
		activeAddrs: make([]*net.UDPAddr, 0),
		addrLastRx:  make(map[string]time.Time),
		txQueue:     make(chan asyncFrame, 2048),
	}

	// Server-side Pacer
	go func() {
		ticker := time.NewTicker(50 * time.Microsecond) // Very fast but paced
		for f := range s.txQueue {
			<-ticker.C
			s.conn.WriteToUDP(f.payload, f.addr)
		}
	}()

	// Server-side Route Garbage Collector
	go func() {
		for {
			time.Sleep(3 * time.Second)
			s.mu.Lock()
			now := time.Now()
			var valid []*net.UDPAddr
			for _, a := range s.activeAddrs {
				addrStr := a.String()
				if now.Sub(s.addrLastRx[addrStr]) < 30*time.Second {
					valid = append(valid, a)
				} else {
					delete(s.addrLastRx, addrStr)
				}
			}
			s.activeAddrs = valid
			s.mu.Unlock()
		}
	}()

	return s, nil
}

// TrackAddr sets a dynamic return route IP. Only call upon DE-CRYPTING an authenticated packet.
func (s *ServerSprayer) TrackAddr(addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	addrStr := addr.String()
	_, exists := s.addrLastRx[addrStr]
	s.addrLastRx[addrStr] = time.Now()

	if !exists {
		// New IP (CGNAT Roaming Detected)
		s.activeAddrs = append(s.activeAddrs, addr)
		logger.ErrorLog().Info("CGNAT Route dynamically updated", "client_ip", addrStr)
	}
}

func (s *ServerSprayer) Write(payload []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.activeAddrs) == 0 {
		return errors.New("no active client routes")
	}

	idx := atomic.AddUint64(&s.rrIndex, 1) % uint64(len(s.activeAddrs))
	target := s.activeAddrs[idx]

	select {
	case s.txQueue <- asyncFrame{payload, target}:
		return nil
	default:
		return errors.New("server tx queue full")
	}
}

func (s *ServerSprayer) ReadFrom() ([]byte, *net.UDPAddr, error) {
	buf := make([]byte, 2048)
	n, addr, err := s.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, err
	}
	// DO NOT TRACK HERE anymore. Wait for Decrypt in core/server.go
	return buf[:n], addr, nil
}
