package network

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

const NTPHeaderSize = 48

// WriteDynamicNTPHeader generates a 48-byte NTPv4 mock header.
// It bypasses DPI by making fields look like a real, time-synchronized NTP client.
func WriteDynamicNTPHeader(buffer []byte) {
	if len(buffer) < NTPHeaderSize {
		return
	}

	// NTP packet layout:
	// LI/VN/Mode (1 byte) -> LI=0 (No warning), VN=4 (IPv4/v6), Mode=3 (Client)
	buffer[0] = 0x23 // 00 100 011

	// Stratum (1 byte) -> Primary Reference
	buffer[1] = 1

	// Poll Interval (1 byte) -> 6 (64 seconds)
	// Precision (1 byte) -> -20
	// 4 bytes: Root Delay
	// 4 bytes: Root Dispersion
	// 4 bytes: Reference ID
	// 8 bytes: Reference Timestamp
	// 8 bytes: Origin Timestamp
	// 8 bytes: Receive Timestamp
	// 8 bytes: Transmit Timestamp

	now := time.Now()
	// NTP time starts from 1900, Unix from 1970. Diff is 2208988800 seconds.
	ntpSecs := uint32(now.Unix() + 2208988800)
	ntpFrac := uint32((now.Nanosecond() * 1000) / 232) // Approximation

	// Fill random noise for fields that vary per system
	randBytes := make([]byte, 14)
	rand.Read(randBytes)

	buffer[2] = randBytes[0] // Poll
	buffer[3] = randBytes[1] // Precision

	// Root Delay & Dispersion
	copy(buffer[4:12], randBytes[2:10])

	// Reference ID (Pseudo-random to simulate IPs or clock sources)
	copy(buffer[12:16], randBytes[10:14])

	// Reference timestamp
	binary.BigEndian.PutUint32(buffer[16:20], ntpSecs)
	binary.BigEndian.PutUint32(buffer[20:24], ntpFrac)

	// Origin
	binary.BigEndian.PutUint32(buffer[24:28], ntpSecs)
	binary.BigEndian.PutUint32(buffer[28:32], ntpFrac)

	// Receive
	binary.BigEndian.PutUint32(buffer[32:36], ntpSecs)
	binary.BigEndian.PutUint32(buffer[36:40], ntpFrac)

	// Transmit
	binary.BigEndian.PutUint32(buffer[40:44], ntpSecs)
	binary.BigEndian.PutUint32(buffer[44:48], ntpFrac)
}
