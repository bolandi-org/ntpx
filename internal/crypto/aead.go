package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// Cipher block struct holding the AEAD implementation
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher creates a new ChaCha20-Poly1305 cipher using the derived key
func NewCipher(key []byte) (*Cipher, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt encrypts the plaintext and appends it to dst.
// Requires a 12-byte nonce which is randomly generated and prepended to the ciphertext.
func (c *Cipher) Encrypt(dst, plaintext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize() // 12 bytes
	
	// Create nonce. We append the nonce to the end of dst, or just allocate enough space.
	// Since we want zero-allocation, dst should have enough capacity for nonce + plaintext + overhead.
	offset := len(dst)
	
	// Ensure dst has capacity
	out := append(dst, make([]byte, nonceSize)...) // We will use a pre-allocated pool in practice
	
	nonce := out[offset : offset+nonceSize]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the encrypted data to out
	out = c.aead.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// Decrypt decrypts the ciphertext from src and appends plaintext to dst.
func (c *Cipher) Decrypt(dst, ciphertext []byte) ([]byte, error) {
	nonceSize := c.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]

	out, err := c.aead.Open(dst, nonce, data, nil)
	if err != nil {
		return nil, err
	}

	return out, nil
}
