package pool

import (
	"sync"
)

// Buffers up to 2KB (safe for UDP MTU limits usually around 1200-1500)
const BufferSize = 2048

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, BufferSize)
		return &buf
	},
}

// Get retrieves a pre-allocated byte slice from the pool.
// The returned slice has length BufferSize.
func Get() *[]byte {
	return bufferPool.Get().(*[]byte)
}

// Put returns the byte slice to the pool.
func Put(b *[]byte) {
	// Optional: we don't zero it out to save CPU cycles.
	// The consumer should use slices up to the needed length.
	bufferPool.Put(b)
}
