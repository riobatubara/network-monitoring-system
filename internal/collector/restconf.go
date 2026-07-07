package collector

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ExecuteRestconfQuery issues a secure API request to fetch target device configurations.
// It returns the execution latency, the raw response payload body, and any connection errors.
func ExecuteRestconfQuery(ctx context.Context, target string, timeout time.Duration) (time.Duration, string, error) {
	// 1. Construct the standardized RESTCONF operational datastore resource URL path
	url := fmt.Sprintf("https://%s/restconf/data/ietf-interfaces:interfaces-state", target)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to generate RESTCONF HTTP request context: %w", err)
	}

	// 2. Apply standardized programmatic RESTCONF request headers
	req.Header.Set("Accept", "application/yang-data+json")
	req.Header.Set("Content-Type", "application/yang-data+json")

	// 3. Initialize an isolated, safe HTTP operational transport client
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// In network monitoring topologies, we often allow self-signed certificates on local nodes
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	start := time.Now()

	// 4. Dispatch the HTTP network frame query across the wire
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("RESTCONF endpoint communication failed: %w", err)
	}
	defer resp.Body.Close()

	// 5. Read the incoming payload data stream safely
	latency := time.Since(start)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return latency, "", fmt.Errorf("failed parsing downstream payload body: %w", err)
	}

	// 6. Evaluate structural HTTP feedback properties
	if resp.StatusCode != http.StatusOK {
		return latency, string(bodyBytes), fmt.Errorf("device rejected transaction with HTTP status: %d", resp.StatusCode)
	}

	return latency, string(bodyBytes), nil
}
