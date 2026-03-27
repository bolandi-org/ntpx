package core

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"nptx/internal/crypto"
	"nptx/internal/mux"
	"nptx/internal/network"
	"nptx/pkg/logger"
)

func StartServer(ctx context.Context, cfg *Config, cipher *crypto.Cipher) error {
	logger.ErrorLog().Info("Initializing Server-Mode listener", "local", cfg.Local)

	serverSprayer, err := network.NewServerSprayer(cfg.Local)
	if err != nil {
		return err
	}

	var mu sync.RWMutex
	targetConns := make(map[uint16]*net.UDPConn)
	reassembler := mux.NewReassembler()
	var seqCounter uint32 = 0

	getTargetConn := func(targetPort uint16) (*net.UDPConn, error) {
		mu.RLock()
		conn, ok := targetConns[targetPort]
		mu.RUnlock()
		if ok {
			return conn, nil
		}

		mu.Lock()
		defer mu.Unlock()
		if conn, ok := targetConns[targetPort]; ok {
			return conn, nil
		}

		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", targetPort))
		if err != nil {
			return nil, err
		}
		newConn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			return nil, err
		}
		targetConns[targetPort] = newConn

		// Read downstream replies (e.g. from Wireguard) and spray back
		go func(c *net.UDPConn, port uint16) {
			b := make([]byte, 65535)
			for {
				n, err := c.Read(b)
				if err != nil {
					return
				}

				payload := b[:n]
				seq := atomic.AddUint32(&seqCounter, 1)

				frames := mux.FragmentPayload(port, seq, payload)
				for _, frame := range frames {
					frameBuf := make([]byte, mux.HeaderSize+len(frame.Payload))
					frame.Encode(frameBuf)

					encBuf := make([]byte, 0, len(frameBuf)+28)
					ciphertext, err := cipher.Encrypt(encBuf, frameBuf)
					if err != nil {
						continue
					}

					packetBuf := make([]byte, network.NTPHeaderSize+len(ciphertext))
					network.WriteDynamicNTPHeader(packetBuf[:network.NTPHeaderSize])
					copy(packetBuf[network.NTPHeaderSize:], ciphertext)

					serverSprayer.Write(packetBuf)
				}
			}
		}(newConn, targetPort)

		return newConn, nil
	}

	go func() {
		for {
			buf, rxAddr, err := serverSprayer.ReadFrom()
			if err != nil {
				continue
			}

			if len(buf) < network.NTPHeaderSize+28+mux.HeaderSize {
				continue
			}

			ciphertext := buf[network.NTPHeaderSize:]
			decBuf := make([]byte, 0, len(ciphertext))
			plaintext, err := cipher.Decrypt(decBuf, ciphertext)
			if err != nil {
				continue
			}

			// SUCCESS: Authentication passed! Safe to dynamically track IP for CGNAT Roaming
			serverSprayer.TrackAddr(rxAddr)

			var frame mux.Frame
			if err := frame.Decode(plaintext); err != nil {
				continue
			}

			// Drop dummy heartbeat frames silently
			if frame.StreamID == 0 && len(frame.Payload) == 0 {
				continue
			}

			assembled := reassembler.Push(&frame)
			if assembled == nil {
				continue
			}

			tConn, err := getTargetConn(frame.StreamID)
			if err == nil {
				tConn.Write(assembled)
			}
		}
	}()

	<-ctx.Done()
	return nil
}
