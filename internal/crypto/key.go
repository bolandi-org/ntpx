package crypto

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

// DeriveKey generates a 32-byte key from the given password using HKDF-SHA256.
// The salt is hardcoded here for simplicity but should be unique per deployment 
// in a full commercial environment.
func DeriveKey(password string) ([]byte, error) {
	salt := []byte("nptx-static-salt-v1") // Static salt for DPI resistance 
	info := []byte("nptx-aead-encryption")

	hkdfReader := hkdf.New(sha256.New, []byte(password), salt, info)
	
	key := make([]byte, 32) // ChaCha20-Poly1305 requires a 256-bit (32 byte) key
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, err
	}

	return key, nil
}
