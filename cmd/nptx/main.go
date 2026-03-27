package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"nptx/internal/core"
	"nptx/pkg/logger"
)

type ConfigJSON struct {
	Mode     string `json:"mode"`
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Streams  int    `json:"streams"`
	Password string `json:"password"`
	Routes   string `json:"routes"`
}

func main() {
	logger.Init()

	configPtr := flag.String("config", "", "Path to config.json (Overrides other CLI flags)")
	modePtr := flag.String("mode", "", "client or server (required)")
	localPtr := flag.String("local", "0.0.0.0:7305", "Local listen address")
	remotePtr := flag.String("remote", "", "Remote IP:Port destination (required for client)")
	streamsPtr := flag.Int("streams", 16, "Number of parallel UDP sockets")
	passwordPtr := flag.String("password", "", "Encryption password (required)")
	routesPtr := flag.String("routes", "", "Port mappings for MUX (e.g., 7305:25566)")

	flag.Parse()

	var c ConfigJSON
	if *configPtr != "" {
		data, err := os.ReadFile(*configPtr)
		if err != nil {
			slog.Error("Failed to read config file", "error", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(data, &c); err != nil {
			slog.Error("Failed to parse config file", "error", err)
			os.Exit(1)
		}
	} else {
		// Use CLI flags
		c.Mode = *modePtr
		c.Local = *localPtr
		c.Remote = *remotePtr
		c.Streams = *streamsPtr
		c.Password = *passwordPtr
		c.Routes = *routesPtr
	}

	routesMap, err := core.ParseRoutes(c.Routes)
	if err != nil {
		slog.Error("Failed to parse routes", "error", err)
		os.Exit(1)
	}

	config := core.Config{
		Mode:     core.Mode(c.Mode),
		Local:    c.Local,
		Remote:   c.Remote,
		Streams:  c.Streams,
		Password: c.Password,
		Routes:   routesMap,
	}

	if err := config.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	slog.Info("Starting nptx", "mode", config.Mode, "local", config.Local, "remote", config.Remote)

	ctx := context.Background()
	if err := core.Start(ctx, &config); err != nil {
		slog.Error("Core failed", "error", err)
		os.Exit(1)
	}
}
