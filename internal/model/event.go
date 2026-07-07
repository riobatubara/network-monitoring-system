package model

import (
	"time"
)

type UnifiedEvent struct {
	JobID     string    `json:"job_id"`
	Target    string    `json:"target"`     // Device IP address or hostname
	Protocol  string    `json:"protocol"`   // "ICMP", "RESTCONF", or "SYSLOG"
	Status    string    `json:"status"`     // "SUCCESS" or "FAILED"
	LatencyMs int64     `json:"latency_ms"` // Execution latency (0 for push-based Syslog)
	Payload   string    `json:"payload"`    // Raw logs, JSON response data, or error strings
	Timestamp time.Time `json:"timestamp"`  // Exact time the event was captured
}
