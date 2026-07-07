package collector

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	"network-monitoring-system/internal/model"
)

type SyslogListener struct {
	listenAddr string
	engine     *PollingEngine
}

// NewSyslogListener initializes the background log collector instance
func NewSyslogListener(listenAddr string, engine *PollingEngine) *SyslogListener {
	return &SyslogListener{
		listenAddr: listenAddr,
		engine:     engine,
	}
}

// Start opens the UDP network socket and listens for inbound firewall/switch logs
func (s *SyslogListener) Start(ctx context.Context) {
	addr, err := net.ResolveUDPAddr("udp", s.listenAddr)
	if err != nil {
		log.Printf("[Syslog-Error] Failed to resolve UDP bind address: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("[Syslog-Error] Socket binding on %s failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[Syslog] Server listening for network device push messages on UDP %s", s.listenAddr)

	// Allocate a safe read buffer for inbound packet frames
	buf := make([]byte, 2048)

	// Launch background handler to terminate socket gracefully if upper context cancels
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				// Prevent crashing if the socket is closed during shutdown sequences
				if ctx.Err() != nil {
					return
				}
				log.Printf("[Syslog-Error] Read failure from network layer: %v", err)
				continue
			}

			rawMessage := string(buf[:n])
			targetIP := remoteAddr.IP.String()

			// Normalize the raw data stream into our unified event footprint
			event := model.UnifiedEvent{
				JobID:     "syslog-event-" + strings.ReplaceAll(time.Now().Format("150405.000"), ".", ""),
				Target:    targetIP,
				Protocol:  "SYSLOG",
				Status:    "SUCCESS",
				LatencyMs: 0, // Inbound push streams have no execution round-trip latency
				Payload:   strings.TrimSpace(rawMessage),
				Timestamp: time.Now(),
			}

			// Hand off event directly into the locked engine channel
			s.engine.PushResultDirectly(event)
		}
	}
}
