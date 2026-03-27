package core

import (
	"context"
	"fmt"

	"nptx/internal/crypto"
)

// Start initializes the nptx node based on config.
func Start(ctx context.Context, cfg *Config) error {
	// 1. Derived cryptographic key
	key, err := crypto.DeriveKey(cfg.Password)
	if err != nil {
		return fmt.Errorf("failed to derive key: %w", err)
	}

	// 2. Create Cipher
	cipher, err := crypto.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// 3. Branch by Mode
	if cfg.Mode == ModeClient {
		return StartClient(ctx, cfg, cipher)
	} else {
		return StartServer(ctx, cfg, cipher)
	}
}
