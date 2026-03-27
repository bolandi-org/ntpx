package mux

import (
	"bytes"
	"testing"
)

func TestFrameEncodeDecode(t *testing.T) {
	original := &Frame{
		StreamID:  1024,
		Sequence:  42,
		FragIndex: 1,
		FragCount: 5,
		Payload:   []byte("hello world, this is a test payload"),
	}

	buf := make([]byte, HeaderSize+len(original.Payload))
	err := original.Encode(buf)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	decoded := &Frame{}
	err = decoded.Decode(buf)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	if decoded.StreamID != original.StreamID {
		t.Errorf("StreamID mismatch: got %d, want %d", decoded.StreamID, original.StreamID)
	}
	if decoded.Sequence != original.Sequence {
		t.Errorf("Sequence mismatch: got %d, want %d", decoded.Sequence, original.Sequence)
	}
	if decoded.FragIndex != original.FragIndex {
		t.Errorf("FragIndex mismatch: got %d, want %d", decoded.FragIndex, original.FragIndex)
	}
	if decoded.FragCount != original.FragCount {
		t.Errorf("FragCount mismatch: got %d, want %d", decoded.FragCount, original.FragCount)
	}
	if !bytes.Equal(decoded.Payload, original.Payload) {
		t.Errorf("Payload mismatch: got %s, want %s", decoded.Payload, original.Payload)
	}
}

func TestFragmentAndReassemble(t *testing.T) {
	// Create a large payload (3000 bytes)
	largePayload := make([]byte, 3000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	frames := FragmentPayload(1, 100, largePayload)

	// MaxMuxPayloadSize = 1116. 3000 / 1116 = 3 frames
	if len(frames) != 3 {
		t.Fatalf("Expected 3 frames, got %d", len(frames))
	}

	r := NewReassembler()

	// Send out of order
	res := r.Push(frames[2])
	if res != nil {
		t.Fatal("Expected nil on incomplete")
	}

	res = r.Push(frames[0])
	if res != nil {
		t.Fatal("Expected nil on incomplete")
	}

	res = r.Push(frames[1]) // the final piece
	if res == nil {
		t.Fatal("Expected assembled bytes")
	}

	if !bytes.Equal(res, largePayload) {
		t.Fatal("Reassembled payload doesn't match original")
	}
}
