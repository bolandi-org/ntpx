package pool

import (
	"testing"
)

func TestPool(t *testing.T) {
	buf := Get()
	if len(*buf) != BufferSize {
		t.Errorf("Expected buffer of size %d, got %d", BufferSize, len(*buf))
	}
	(*buf)[0] = 42
	Put(buf)

	buf2 := Get()
	if len(*buf2) != BufferSize {
		t.Errorf("Expected buffer of size %d, got %d", BufferSize, len(*buf2))
	}
	Put(buf2)
}
