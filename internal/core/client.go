package core

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"nptx/internal/crypto"
	"nptx/internal/mux"
	"nptx/internal/network"
	"nptx/pkg/logger"
)

func StartClient(ctx context.Context, cfg *Config, cipher *crypto.Cipher) error {
	logger.ErrorLog().Info("Initializing Client-Mode UDP Sprayer")

	sprayer, err := network.NewClientSprayer(cfg.Remote, cfg.Streams)
	if err != nil {
		return err
	}
	go sprayer.StartHealthCheck(ctx)

	// Aggressive UDP NAT keep-alive heartbeat loop
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				frame := &mux.Frame{
					StreamID:  0,
					Sequence:  0,
					FragIndex: 0,
					FragCount: 1,
					Payload:   []byte{},
				}
				frameBuf := make([]byte, mux.HeaderSize)
				frame.Encode(frameBuf)

				encBuf := make([]byte, 0, len(frameBuf)+28)
				ciphertext, err := cipher.Encrypt(encBuf, frameBuf)
				if err != nil {
					continue
				}

				packetBuf := make([]byte, network.NTPHeaderSize+len(ciphertext))
				network.WriteDynamicNTPHeader(packetBuf[:network.NTPHeaderSize])
				copy(packetBuf[network.NTPHeaderSize:], ciphertext)

				sprayer.Broadcast(packetBuf)
			}
		}
	}()

	var seqCounter uint32 = 0
	reassembler := mux.NewReassembler()

	type localTarget struct {
		conn *net.UDPConn
		addr *net.UDPAddr
	}
	var localMu sync.RWMutex
	localTargets := make(map[uint16]*localTarget)

	// Background reader from tunnel
	go func() {
		for {
			buf, err := sprayer.Read()
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

			var frame mux.Frame
			if err := frame.Decode(plaintext); err != nil {
				continue
			}

			assembled := reassembler.Push(&frame)
			if assembled == nil {
				continue // wait for more parts
			}

			// Route back to local user application
			localMu.RLock()
			tgt, ok := localTargets[frame.StreamID]
			localMu.RUnlock()

			if ok && tgt.addr != nil {
				tgt.conn.WriteToUDP(assembled, tgt.addr)
			}
		}
	}()

	for localPort, targetPort := range cfg.Routes {
		go func(lPort, tPort int) {
			addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", lPort))
			if err != nil {
				return
			}
			conn, err := net.ListenUDP("udp", addr)
			if err != nil {
				return
			}
			defer conn.Close()

			logger.ErrorLog().Info("Listening", "local_port", lPort, "mapped_to", tPort)
			streamID := uint16(tPort)
			buf := make([]byte, 65535)

			for {
				n, srcAddr, err := conn.ReadFromUDP(buf)
				if err != nil {
					continue
				}

				localMu.Lock()
				localTargets[streamID] = &localTarget{conn: conn, addr: srcAddr}
				localMu.Unlock()

				payload := buf[:n]
				seq := atomic.AddUint32(&seqCounter, 1)

				// Fragment large MTUs
				frames := mux.FragmentPayload(streamID, seq, payload)
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

					sprayer.Write(packetBuf)
				}
			}
		}(localPort, targetPort)
	}

	<-ctx.Done()
	return nil
}
