package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "network-monitoring-system/api/proto/collector"
	"network-monitoring-system/internal/model"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Job represents a distinct performance tracking execution task
type Job struct {
	ID       string
	Target   string
	Protocol string
	Timeout  time.Duration
}

type PollingEngine struct {
	workerCount int
	jobQueue    chan Job
	results     chan model.UnifiedEvent
	rateLimit   time.Duration
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPollingEngine allocates structural internal concurrency properties
func NewPollingEngine(workerCount int, bufferSize int, rateLimit time.Duration) *PollingEngine {
	ctx, cancel := context.WithCancel(context.Background())
	return &PollingEngine{
		workerCount: workerCount,
		jobQueue:    make(chan Job, bufferSize),
		results:     make(chan model.UnifiedEvent, bufferSize),
		rateLimit:   rateLimit,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start spawns the local pool routines and binds the gRPC return streaming client pipe
func (pe *PollingEngine) Start(ctx context.Context, schedulerAddr string) {
	go func() {
		<-ctx.Done()
		pe.Stop()
	}()

	// 1. Spawn concurrent workers to drain incoming network checks
	for i := 0; i < pe.workerCount; i++ {
		pe.wg.Add(1)
		go pe.worker()
	}

	// 2. Spawn persistent streaming routine to forward results to the Scheduler via gRPC
	go pe.streamToScheduler(ctx, schedulerAddr)
}

func (pe *PollingEngine) worker() {
	defer pe.wg.Done()
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
			<-limiter.C
			pe.executeJob(job)
		}
	}
}

func (pe *PollingEngine) executeJob(job Job) {
	ctx, cancel := context.WithTimeout(pe.ctx, job.Timeout)
	defer cancel()

	start := time.Now()
	var err error

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-time.After(10 * time.Millisecond):
		if job.Target == "broken-device" {
			err = fmt.Errorf("network host unreachable")
		}
	}

	latency := time.Since(start).Milliseconds()

	status := "SUCCESS"
	payload := ""
	if err != nil {
		status = "FAILED"
		payload = err.Error()
	}

	pe.results <- model.UnifiedEvent{
		JobID:     job.ID,
		Target:    job.Target,
		Protocol:  job.Protocol,
		Status:    status,
		LatencyMs: latency,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}

func (pe *PollingEngine) streamToScheduler(ctx context.Context, serverAddr string) {
	for {
		if ctx.Err() != nil {
			return
		}

		log.Printf("[Collector-Engine] Connecting to %s...", serverAddr)

		conn, err := grpc.DialContext(
			ctx,
			serverAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		client := pb.NewMonitoringServiceClient(conn)
		stream, err := client.StreamResults(ctx)
		if err != nil {
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Println("[Collector-Engine] Stream established")

		reconnect := false

		for !reconnect {
			select {
			case <-ctx.Done():
				_, _ = stream.CloseAndRecv()
				conn.Close()
				return

			case localEvent, ok := <-pe.results:
				if !ok {
					_, _ = stream.CloseAndRecv()
					conn.Close()
					return
				}

				pbEvent := &pb.UnifiedEvent{
					JobId:     localEvent.JobID,
					Target:    localEvent.Target,
					Status:    localEvent.Status,
					LatencyMs: localEvent.LatencyMs,
					Payload:   localEvent.Payload,
					Timestamp: timestamppb.New(localEvent.Timestamp),
				}

				switch localEvent.Protocol {
				case "ICMP":
					pbEvent.Protocol = pb.ProtocolType_ICMP
				case "RESTCONF":
					pbEvent.Protocol = pb.ProtocolType_RESTCONF
				case "SYSLOG":
					pbEvent.Protocol = pb.ProtocolType_SYSLOG
				default:
					pbEvent.Protocol = pb.ProtocolType_PROTOCOL_UNSPECIFIED
				}

				if err := stream.Send(pbEvent); err != nil {
					log.Printf("[Collector-Engine] Send failed, reconnecting: %v", err)
					reconnect = true
				}
			}
		}

		conn.Close()
		time.Sleep(2 * time.Second)
	}
}

func (pe *PollingEngine) QueueJob(job Job) {
	select {
	case pe.jobQueue <- job:
	default:
		log.Println("[Collector-Warning] Engine task channel saturated. Dropping runtime job allocation request.")
	}
}

func (pe *PollingEngine) PushResultDirectly(event model.UnifiedEvent) {
	select {
	case pe.results <- event:
	default:
		log.Println("[Collector-Warning] Results channel saturated. Dropping inbound execution record.")
	}
}

func (pe *PollingEngine) Stop() {
	pe.cancel()
	close(pe.jobQueue)
	pe.wg.Wait()
}

// --- Local Scheduler Loop ---

type LocalScheduler struct {
	engine *PollingEngine
}

func NewLocalScheduler(engine *PollingEngine) *LocalScheduler {
	return &LocalScheduler{engine: engine}
}

func (s *LocalScheduler) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.engine.QueueJob(Job{
				ID:       fmt.Sprintf("job-%d-icmp", time.Now().Unix()),
				Target:   "8.8.8.8",
				Protocol: "ICMP",
				Timeout:  2 * time.Second,
			})
			s.engine.QueueJob(Job{
				ID:       fmt.Sprintf("job-%d-rest", time.Now().Unix()),
				Target:   "broken-device",
				Protocol: "RESTCONF",
				Timeout:  2 * time.Second,
			})
		}
	}
}
