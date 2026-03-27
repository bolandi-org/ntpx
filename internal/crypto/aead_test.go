package crypto

import (
	"bytes"
	"testing"
)

func TestChaCha20Poly1305AEAD(t *testing.T) {
	key, err := DeriveKey("my-super-secret-password")
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}

	cipher, err := NewCipher(key)
	if err != nil {
		t.Fatalf("Failed to create cipher: %v", err)
	}

	plaintext := []byte("nptx high-speed encryption test! 1234567890")

	// Encrypt
	buf := make([]byte, 0, len(plaintext)+28)
	ciphertext, err := cipher.Encrypt(buf, plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Make sure it's longer (nonce + tag overhead = 28 bytes)
	if len(ciphertext) != len(plaintext)+28 {
		t.Errorf("Ciphertext length invalid: got %d, want %d", len(ciphertext), len(plaintext)+28)
	}

	// Decrypt
	decBuf := make([]byte, 0, len(ciphertext))
	decrypted, err := cipher.Decrypt(decBuf, ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted data mismatch: got %s, want %s", string(decrypted), string(plaintext))
	}

	// Test tamper
	ciphertext[15] ^= 0x01
	_, err = cipher.Decrypt(decBuf, ciphertext)
	if err == nil {
		t.Fatal("Expected error on tampered ciphertext, got nil")
	}
}
