package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Job represents a polling task sent from the scheduler
type Job struct {
	ID       string
	Target   string
	Protocol string
	Timeout  time.Duration
}

// UnifiedEvent is the normalized format for storage
type UnifiedEvent struct {
	JobID     string    `json:"job_id"`
	Target    string    `json:"target"`
	Protocol  string    `json:"protocol"`
	Status    string    `json:"status"`
	LatencyMs int64     `json:"latency_ms"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

type PollingEngine struct {
	workerCount int
	jobQueue    chan Job
	results     chan UnifiedEvent
	rateLimit   time.Duration
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPollingEngine matches the signature called in main.go
func NewPollingEngine(workerCount int, bufferSize int, rateLimit time.Duration) *PollingEngine {
	ctx, cancel := context.WithCancel(context.Background())
	return &PollingEngine{
		workerCount: workerCount,
		jobQueue:    make(chan Job, bufferSize),
		results:     make(chan UnifiedEvent, bufferSize),
		rateLimit:   rateLimit,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start launches the worker pool and a results consumer
func (pe *PollingEngine) Start(ctx context.Context) {
	// Link the engine lifecycle to the passed context
	go func() {
		<-ctx.Done()
		pe.Stop()
	}()

	for i := 0; i < pe.workerCount; i++ {
		pe.wg.Add(1)
		go pe.worker(i)
	}

	// Start a background routine to consume results so the channel doesn't block
	go pe.consumeResults()
}

func (pe *PollingEngine) worker(workerID int) {
	defer pe.wg.Done()

	// Create a local ticker for local worker rate-limiting
	limiter := time.NewTicker(pe.rateLimit)
	defer limiter.Stop()

	for {
		select {
		case <-pe.ctx.Done():
			return
		case job, ok := <-pe.jobQueue:
			if !ok {
				return
			}
			<-limiter.C // Wait for the rate-limiting tick
			pe.executeJobWithRetry(job)
		}
	}
}

func (pe *PollingEngine) executeJobWithRetry(job Job) {
	maxRetries := 3
	var event UnifiedEvent

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(pe.ctx, job.Timeout)

		start := time.Now()
		err := pe.pollTarget(ctx, job)
		latency := time.Since(start).Milliseconds()
		cancel()

		if err == nil {
			event = UnifiedEvent{
				JobID:     job.ID,
				Target:    job.Target,
				Protocol:  job.Protocol,
				Status:    "SUCCESS",
				LatencyMs: latency,
				Timestamp: time.Now(),
			}
			break
		}

		if attempt == maxRetries {
			event = UnifiedEvent{
				JobID:     job.ID,
				Target:    job.Target,
				Protocol:  job.Protocol,
				Status:    "FAILED",
				Payload:   err.Error(),
				Timestamp: time.Now(),
			}
		} else {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}

	pe.results <- event
}

func (pe *PollingEngine) pollTarget(ctx context.Context, job Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		if job.Target == "broken-device" {
			return fmt.Errorf("connection refused")
		}
		return nil
	}
}

func (pe *PollingEngine) consumeResults() {
	for event := range pe.results {
		log.Printf("[Engine Result] Job %s for %s (%s) finished with status: %s",
			event.JobID, event.Target, event.Protocol, event.Status)
	}
}

func (pe *PollingEngine) Stop() {
	pe.cancel()
	pe.wg.Wait()
}

type Scheduler struct {
	engine *PollingEngine
}

func NewScheduler(engine *PollingEngine) *Scheduler {
	return &Scheduler{engine: engine}
}

// Run simulates periodic target generation and queues them into the engine
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Println("[Scheduler] Ticking... dispatching periodic monitoring jobs.")

			// Mock generating 2 tracking tasks per tick interval
			s.engine.jobQueue <- Job{
				ID:       fmt.Sprintf("job-%d-icmp", time.Now().Unix()),
				Target:   "8.8.8.8",
				Protocol: "ICMP",
				Timeout:  2 * time.Second,
			}
			s.engine.jobQueue <- Job{
				ID:       fmt.Sprintf("job-%d-rest", time.Now().Unix()),
				Target:   "broken-device", // This will trigger our retry simulation logic
				Protocol: "RESTCONF",
				Timeout:  2 * time.Second,
			}
		}
	}
}
