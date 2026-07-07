package collector

import (
	"context"
	"log"
	"net"
	"strings"
	"time"
)

// SyslogListener binds to a UDP port to consume pushed network events
type SyslogListener struct {
	addr        string
	resultsChan chan<- UnifiedEvent
	collectorID string
}

func NewSyslogListener(addr string, results chan<- UnifiedEvent, collectorID string) *SyslogListener {
	return &SyslogListener{
		addr:        addr,
		resultsChan: results,
		collectorID: collectorID,
	}
}

// Start opens the network port and handles streaming incoming data packets
func (s *SyslogListener) Start(ctx context.Context) {
	// Parse local address and spin up standard UDP connection listener
	lAddr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP socket address: %v", err)
	}

	conn, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		log.Fatalf("Failed to bind to UDP syslog port: %v", err)
	}
	defer conn.Close()

	log.Printf("Syslog UDP engine running on port %s", s.addr)

	// Buffer payload sizing (standard Syslog packet max payload size is under 2KB)
	buf := make([]byte, 2048)

	// Context cancel close trigger routine
	go func() {
		<-ctx.Done()
		conn.Close() // Forces ReadFromUDP to break immediately
	}()

	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return // Closed normally via context shutdown
			default:
				log.Printf("Error reading Syslog packet: %v", err)
				continue
			}
		}

		rawMessage := string(buf[:n])

		// Concurrently normalize the packet to keep the UDP receiver buffer completely open
		go s.normalizeSyslog(remoteAddr.IP.String(), rawMessage)
	}
}

// Hand-rolled primitive parser to normalize unstructured data
func (s *SyslogListener) normalizeSyslog(senderIP string, rawMessage string) {
	// Cleans trailing line breaks common in device strings
	cleanMsg := strings.TrimSpace(rawMessage)

	// Map straight into your system-wide Normalized Struct format
	event := UnifiedEvent{
		JobID:     "syslog-" + time.Now().Format("20060102150405"),
		Target:    senderIP,
		Protocol:  "SYSLOG",
		Status:    "SUCCESS",
		LatencyMs: 0, // Inbound pushed data has no target latency metrics
		Payload:   cleanMsg,
		Timestamp: time.Now(),
	}

	// Push down standard pipeline channel
	s.resultsChan <- event
}
