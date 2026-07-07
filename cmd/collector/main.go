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
	log.Println("[Collector-Main] Booting regional monitoring daemon...")

	// 1. Load system properties from central root configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("[Collector-Main] Configuration error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Initialize the concurrent processing engine
	engine := collector.NewPollingEngine(
		cfg.Collector.WorkerCount,
		cfg.Collector.BufferSize,
		time.Duration(cfg.Collector.RateLimitMs)*time.Millisecond,
	)

	// 3. Fire up worker pool and connect the gRPC network stream back to Scheduler
	engine.Start(ctx, cfg.Collector.SchedulerAddress)

	// 4. Pass the engine straight into the listener (syslog uses engine.PushResultDirectly)
	syslogListener := collector.NewSyslogListener("0.0.0.0:514", engine)
	go syslogListener.Start(ctx)

	// 5. Fire up local scheduler emulation loop
	localSched := collector.NewLocalScheduler(engine)
	go localSched.Run(ctx, time.Duration(cfg.Collector.PollingIntervalSec)*time.Second)

	log.Printf("[Collector-Main] Edge Node %s fully initialized and tracking metrics.", cfg.Collector.ID)

	// Keep alive until OS signal catch
	<-ctx.Done()
	log.Println("[Collector-Main] Shutting down collector components safely...")
}
