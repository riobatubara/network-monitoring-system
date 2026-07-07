package timeseries

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"network-monitoring-system/internal/model"
)

type MetricsClient struct {
	writeURL   string
	httpClient *http.Client
}

// NewClient timeseries exporter layer
func NewClient(writeURL string) *MetricsClient {
	return &MetricsClient{
		writeURL: writeURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// WriteMetric exports an event latency payload using Influx line protocol
func (c *MetricsClient) WriteMetric(ctx context.Context, event model.UnifiedEvent) error {
	var sb strings.Builder
	sb.WriteString("netmon_latency,target=")
	sb.WriteString(event.Target)
	sb.WriteString(",protocol=")
	sb.WriteString(event.Protocol)
	sb.WriteString(" status=\"")
	sb.WriteString(event.Status)
	sb.WriteString("\",latency_ms=")
	sb.WriteString(fmt.Sprintf("%di ", event.LatencyMs))
	sb.WriteString(fmt.Sprintf("%d\n", event.Timestamp.UnixNano()))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.writeURL, strings.NewReader(sb.String()))
	if err != nil {
		return fmt.Errorf("failed to create metric request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metric to backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("metric write failed with status code: %d", resp.StatusCode)
	}

	return nil
}
