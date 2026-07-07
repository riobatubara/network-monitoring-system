package scheduler

import (
	"context"
	"io"
	"log"

	pb "network-monitoring-system/api/proto/collector"
	"network-monitoring-system/internal/model"
	"network-monitoring-system/internal/storage/postgres"
	"network-monitoring-system/internal/storage/timeseries"
)

// TaskProcessor implements the compiled gRPC MonitoringServiceServer interface
type TaskProcessor struct {
	pb.UnimplementedMonitoringServiceServer
	pgClient *postgres.StorageClient
	tsClient *timeseries.MetricsClient
}

// NewTaskProcessor initializes our gRPC logic pipeline
func NewTaskProcessor(pg *postgres.StorageClient, ts *timeseries.MetricsClient) *TaskProcessor {
	return &TaskProcessor{
		pgClient: pg,
		tsClient: ts,
	}
}

// StreamJobs handles pushing tasks to collectors (Simulated for this milestone)
func (p *TaskProcessor) StreamJobs(req *pb.StreamJobsRequest, stream pb.MonitoringService_StreamJobsServer) error {
	log.Printf("[Scheduler] New collector client registered: %s. Initiating job stream.", req.CollectorId)

	// Keep stream alive or push configurations from target lookup tables
	<-stream.Context().Done()
	return stream.Context().Err()
}

// func (p *TaskProcessor) StreamResults(stream pb.MonitoringService_StreamResultsServer) error {
// 	log.Println("[Scheduler] Results stream started")

// 	for {
// 		pbEvent, err := stream.Recv()
// 		if err == io.EOF {
// 			return stream.SendAndClose(&pb.StreamResultsResponse{
// 				Success:        true,
// 				ProcessedCount: 0, // Ingestion tracking index marker
// 			})
// 		}
// 		if err != nil {
// 			log.Printf("[Scheduler Error] Ingestion stream disrupted: %v", err)
// 			return err
// 		}

// 		eventTime := pbEvent.Timestamp.AsTime()
// 		localEvent := model.UnifiedEvent{
// 			JobID:     pbEvent.JobId,
// 			Target:    pbEvent.Target,
// 			Protocol:  pbEvent.Protocol.String(),
// 			Status:    pbEvent.Status,
// 			LatencyMs: pbEvent.LatencyMs,
// 			Payload:   pbEvent.Payload,
// 			Timestamp: eventTime,
// 		}

// 		go p.saveToStorage(stream.Context(), localEvent)
// 	}
// }

// StreamResults reads coming metrics from collectors and routes them to databases
func (p *TaskProcessor) StreamResults(stream pb.MonitoringService_StreamResultsServer) error {
	log.Println("[Scheduler] Results stream started")

	var processed int32

	for {
		pbEvent, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.StreamResultsResponse{
				Success:        true,
				ProcessedCount: processed,
			})
		}
		if err != nil {
			log.Printf("[Scheduler Error] Stream error: %v", err)
			return err
		}

		localEvent := model.UnifiedEvent{
			JobID:     pbEvent.JobId,
			Target:    pbEvent.Target,
			Protocol:  pbEvent.Protocol.String(),
			Status:    pbEvent.Status,
			LatencyMs: pbEvent.LatencyMs,
			Payload:   pbEvent.Payload,
			Timestamp: pbEvent.Timestamp.AsTime(),
		}

		processed++

		// Asynchronously pass data straight to databases to prevent blocking the gRPC stream
		go p.saveToStorage(stream.Context(), localEvent)
	}
}

func (p *TaskProcessor) saveToStorage(ctx context.Context, e model.UnifiedEvent) {
	// PostgreSQL (only failures)
	if p.pgClient != nil && e.Status == "FAILED" {
		query := `INSERT INTO active_alerts (job_id, target, protocol, issue_description, detected_at) 
		          VALUES ($1, $2, $3, $4, $5)`

		if _, err := p.pgClient.DB.ExecContext(ctx, query,
			e.JobID, e.Target, e.Protocol, e.Payload, e.Timestamp,
		); err != nil {
			log.Printf("[Scheduler Error] PostgreSQL: Failed to write critical incident log: %v", err)
		}
	}

	// Timeseries (VictoriaMetrics)
	if p.tsClient != nil && (e.Protocol == "ICMP" || e.Protocol == "RESTCONF") {
		if err := p.tsClient.WriteMetric(ctx, e); err != nil {
			log.Printf("[Scheduler Error] VictoriaMetrics: Failed to export target metric: %v", err)
		}
	}
}
