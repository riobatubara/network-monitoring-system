package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "network-monitoring-system/api/proto/collector"
	"network-monitoring-system/internal/config"
	"network-monitoring-system/internal/scheduler"
	"network-monitoring-system/internal/storage/postgres"
	"network-monitoring-system/internal/storage/timeseries"

	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("[Scheduler-Main Boot Error] Failed parsing properties configuration: %v", err)
	}

	// Postgres
	pgClient, err := postgres.NewClient(cfg.Scheduler.DB.PostgresDSN)
	if err != nil {
		log.Fatalf("[Scheduler-Main Boot Error] DB init failure: %v", err)
	}
	defer pgClient.Close()

	// Timeseries (VictoriaMetrics)
	tsClient := timeseries.NewClient(cfg.Scheduler.DB.VictoriaMetricsURL)

	// Listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Scheduler.GRPCPort))
	if err != nil {
		log.Fatalf("[Scheduler Boot Error] Failed to bind interface port: %v", err)
	}

	// gRPC server
	grpcServer := grpc.NewServer()
	processor := scheduler.NewTaskProcessor(pgClient, tsClient)
	pb.RegisterMonitoringServiceServer(grpcServer, processor)

	// Graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopChan
		log.Println("[Scheduler-Main] Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("[Scheduler-Main] Enginer active on port %d", cfg.Scheduler.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("[Scheduler-Main] Server crashed: %v", err)
	}
}
