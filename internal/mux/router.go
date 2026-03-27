package mux

import (
	"fmt"
	"net"
	"sync"
)

// Router maps listening ports or remote targets to StreamIDs
type Router struct {
	mu            sync.RWMutex
	portToStream  map[int]uint16
	streamToAddr  map[uint16]*net.UDPAddr
	nextStreamID  uint16
}

// NewRouter creates a new MUX router
func NewRouter() *Router {
	return &Router{
		portToStream: make(map[int]uint16),
		streamToAddr: make(map[uint16]*net.UDPAddr),
		nextStreamID: 1, // 0 can be reserved for control messages
	}
}

// RegisterRoute maps a local port to a specific StreamID.
func (r *Router) RegisterRoute(port int) uint16 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if id, exists := r.portToStream[port]; exists {
		return id
	}

	id := r.nextStreamID
	r.nextStreamID++
	r.portToStream[port] = id
	return id
}

// GetStreamID returns the StreamID for a given port
func (r *Router) GetStreamID(port int) (uint16, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.portToStream[port]
	if !exists {
		return 0, fmt.Errorf("no route for port %d", port)
	}
	return id, nil
}

// AddAddrMapping sets the destination address for a specific StreamID (used on server to send back replies)
func (r *Router) AddAddrMapping(streamID uint16, addr *net.UDPAddr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.streamToAddr[streamID] = addr
}

// GetAddr returns the destination address for a given StreamID
func (r *Router) GetAddr(streamID uint16) *net.UDPAddr {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.streamToAddr[streamID]
}
