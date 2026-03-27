package mux

import (
	"encoding/binary"
	"errors"
)

// Frame constants
const (
	HeaderSize = 8 // StreamID (2) + Sequence (4) + FragIndex (1) + FragCount (1)
)

// Frame represents a single multiplexed or fragmented packet
type Frame struct {
	StreamID  uint16
	Sequence  uint32
	FragIndex byte
	FragCount byte
	Payload   []byte
}

// Encode marshals the Frame into the target byte slice.
// target must have length >= HeaderSize + len(Payload).
func (f *Frame) Encode(target []byte) error {
	if len(target) < HeaderSize+len(f.Payload) {
		return errors.New("target buffer too small for frame")
	}

	binary.BigEndian.PutUint16(target[0:2], f.StreamID)
	binary.BigEndian.PutUint32(target[2:6], f.Sequence)
	target[6] = f.FragIndex
	target[7] = f.FragCount

	copy(target[8:], f.Payload)
	return nil
}

// Decode unmarshals a Frame from the source byte slice.
// Payload points to the slice within src (no allocation).
func (f *Frame) Decode(src []byte) error {
	if len(src) < HeaderSize {
		return errors.New("source buffer too small for header")
	}

	f.StreamID = binary.BigEndian.Uint16(src[0:2])
	f.Sequence = binary.BigEndian.Uint32(src[2:6])
	f.FragIndex = src[6]
	f.FragCount = src[7]
	f.Payload = src[8:]

	return nil
}
