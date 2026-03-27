package network

import (
	"testing"
)

func TestWriteDynamicNTPHeader(t *testing.T) {
	buf := make([]byte, 48)
	WriteDynamicNTPHeader(buf)

	if buf[0] != 0x23 {
		t.Errorf("Expected 0x23 for LI/VN/Mode, got %x", buf[0])
	}
	if buf[1] != 1 {
		t.Errorf("Expected stratum 1, got %d", buf[1])
	}

	// Make sure no panic on short buffer
	shortBuf := make([]byte, 10)
	WriteDynamicNTPHeader(shortBuf) // Should return early and not panic
}
