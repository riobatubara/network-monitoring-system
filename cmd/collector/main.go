package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"network-monitoring-system/internal/collector"
	"network-monitoring-system/internal/config"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Collector: %s pointing to server: %s", cfg.Collector.ID, cfg.Collector.ServerAddress)

	// Context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Instantiate Configured Concurrency Engine
	engine := collector.NewPollingEngine(
		cfg.Collector.WorkerCount,
		cfg.Collector.BufferSize,
		time.Duration(cfg.Collector.RateLimitMs)*time.Millisecond,
	)
	engine.Start(ctx)

	// Instantiate Configured Scheduler
	sched := collector.NewScheduler(engine)
	go sched.Run(ctx, time.Duration(cfg.Collector.PollingIntervalSec)*time.Second)

	// Wait for termination signal
	<-ctx.Done()
	log.Println("Shutting down collector gracefully...")
	engine.Stop()
}
