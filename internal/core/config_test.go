package core

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	cfg := Config{
		Mode:     ModeClient,
		Password: "test",
		Remote:   "127.0.0.1:1230",
		Streams:  4,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Valid config failed: %v", err)
	}

	invalidMode := cfg
	invalidMode.Mode = "unknown"
	if invalidMode.Validate() == nil {
		t.Error("Expected error for invalid mode")
	}

	noPass := cfg
	noPass.Password = ""
	if noPass.Validate() == nil {
		t.Error("Expected error for no password")
	}

	noRemoteCli := cfg
	noRemoteCli.Remote = ""
	if noRemoteCli.Validate() == nil {
		t.Error("Expected error for client with no remote")
	}

	serverRemote := cfg
	serverRemote.Mode = ModeServer
	serverRemote.Remote = "" // Server doesn't need remote
	if serverRemote.Validate() != nil {
		t.Error("Server shouldn't require remote")
	}

	zeroStreams := cfg
	zeroStreams.Streams = 0
	if zeroStreams.Validate() == nil {
		t.Error("Expected error for zero streams")
	}
}

func TestParseRoutes(t *testing.T) {
	routes, err := ParseRoutes("7305:25566,7306:51820")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if routes[7305] != 25566 {
		t.Errorf("Expected 25566, got %d", routes[7305])
	}
	if routes[7306] != 51820 {
		t.Errorf("Expected 51820, got %d", routes[7306])
	}

	_, err = ParseRoutes("invalid")
	if err == nil {
		t.Error("Expected error for invalid format")
	}

	_, err = ParseRoutes("abc:def")
	if err == nil {
		t.Error("Expected error for non-numeric ports")
	}

	empty, _ := ParseRoutes("")
	if len(empty) != 0 {
		t.Error("Expected empty map")
	}
}
