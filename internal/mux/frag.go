package mux

import (
	"sync"
	"time"
)

// SafeUDPPayloadSize is the maximum safe UDP MTU payload across the internet to bypass fragmentation
// 1200 - 48 (NTP) - 16 (Poly1305 Mac) - 12 (Nonce) = 1124 bytes
const SafeUDPPayloadSize = 1124
const MaxMuxPayloadSize = SafeUDPPayloadSize - HeaderSize

// FragmentPayload breaks a large payload into multiple Frames safely.
func FragmentPayload(streamID uint16, seq uint32, payload []byte) []*Frame {
	total := len(payload)
	if total <= MaxMuxPayloadSize {
		return []*Frame{{
			StreamID:  streamID,
			Sequence:  seq,
			FragIndex: 0,
			FragCount: 1,
			Payload:   payload,
		}}
	}

	var frames []*Frame
	offset := 0
	count := byte((total + MaxMuxPayloadSize - 1) / MaxMuxPayloadSize)
	var index byte = 0

	for offset < total {
		chunkSize := MaxMuxPayloadSize
		if offset+chunkSize > total {
			chunkSize = total - offset
		}

		frames = append(frames, &Frame{
			StreamID:  streamID,
			Sequence:  seq,
			FragIndex: index,
			FragCount: count,
			Payload:   payload[offset : offset+chunkSize],
		})
		offset += chunkSize
		index++
	}

	return frames
}

// Reassembler handles putting out-of-order fragments back together.
type Reassembler struct {
	mu        sync.Mutex
	fragments map[uint32]*packetBuffer
}

type packetBuffer struct {
	lastTime time.Time
	parts    map[byte][]byte // map chunk index to payload slice
	count    byte            // expected chunks
}

func NewReassembler() *Reassembler {
	r := &Reassembler{
		fragments: make(map[uint32]*packetBuffer),
	}
	// Background GC to clean up dropped chunks
	go r.gcLoop()
	return r
}

// Push adds a frame. Returns assembled bytes if complete, or nil.
func (r *Reassembler) Push(f *Frame) []byte {
	// Fast path for unfragmented packets
	if f.FragCount == 1 {
		return f.Payload
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	buf, exists := r.fragments[f.Sequence]
	if !exists {
		buf = &packetBuffer{
			lastTime: time.Now(),
			parts:    make(map[byte][]byte),
			count:    f.FragCount,
		}
		r.fragments[f.Sequence] = buf
	}

	buf.lastTime = time.Now()

	payloadCopy := make([]byte, len(f.Payload))
	copy(payloadCopy, f.Payload)
	buf.parts[f.FragIndex] = payloadCopy

	if byte(len(buf.parts)) == buf.count {
		// Assembled!
		var assembledLen int
		for _, part := range buf.parts {
			assembledLen += len(part)
		}

		assembled := make([]byte, assembledLen)
		offset := 0
		for i := byte(0); i < buf.count; i++ {
			part := buf.parts[i]
			copy(assembled[offset:], part)
			offset += len(part)
		}

		delete(r.fragments, f.Sequence)
		return assembled
	}

	return nil
}

// gcLoop cleans up stale parts allowing for a massive sliding window for Starlink handoffs (e.g. 20s)
func (r *Reassembler) gcLoop() {
	ticker := time.NewTicker(4 * time.Second)
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for seq, buf := range r.fragments {
			// Extremely generous 20-second out-of-order latency threshold (Satellite Handoff)
			if now.Sub(buf.lastTime) > 20*time.Second {
				delete(r.fragments, seq)
			}
		}
		r.mu.Unlock()
	}
}
