package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Mode represents the running mode of the application (Client or Server)
type Mode string

const (
	ModeClient Mode = "client"
	ModeServer Mode = "server"
)

// Config holds the configuration for nptx
type Config struct {
	Mode     Mode
	Local    string
	Remote   string
	Streams  int
	Password string
	Routes   map[int]int // Local port -> Remote target port map (for Mux)
}

// ParseRoutes parses a route string like "7305:25566,7306:51820" into a map
func ParseRoutes(routeStr string) (map[int]int, error) {
	routes := make(map[int]int)
	if routeStr == "" {
		return routes, nil
	}

	pairs := strings.Split(routeStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid route format: %s", pair)
		}

		localPort, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid local port in route: %s", parts[0])
		}

		remotePort, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid remote port in route: %s", parts[1])
		}

		routes[localPort] = remotePort
	}
	return routes, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Mode != ModeClient && c.Mode != ModeServer {
		return errors.New("mode must be either 'client' or 'server'")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	if c.Mode == ModeClient && c.Remote == "" {
		return errors.New("remote address is required for client mode")
	}
	if c.Streams <= 0 {
		return errors.New("streams must be greater than 0")
	}
	return nil
}
