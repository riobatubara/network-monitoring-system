# network-monitoring-system

#### Architecture
![Architecture Diagram](./network-monitoring-system.png)

#### Project Structure

```
network-monitoring-system/
├── api/
│   └── proto/
│       ├── collector
│       │   └── collector_grpc_pb.go
│       │   └── collector_pb.go
│       ├── scheduler
│       │   └── scheduler_grpc_pb.go
│       │   └── scheduler_pb.go
│       ├── collector.proto       # gRPC definitions for collector registration/heartbeats
│       └── scheduler.proto       # gRPC definitions for job distribution
├── cmd/
│   ├── scheduler/
│   │   └── main.go               # Scheduler entry point
│   └── collector/
│       └── main.go               # Collector entry point
├── internal/
│   ├── scheduler/                # Scheduler logic (job assignment, target registry)
│   ├── collector/                # Core collector engine
│   │   ├── engine.go             # Worker pool management
│   │   ├── icmp.go               # ICMP execution logic
│   │   ├── syslog.go             # Syslog UDP/TCP listener
│   │   └── restconf.go           # RESTCONF/HTTP client
│   ├── model/
│   │   └── event.go              # Unified Normalized Event structs
│   └── storage/                  # Database clients
│       ├── postgres/             # Metadata/Configs
│       ├── redis/                # Rate limiting / caching
│       └── timeseries/           # Prometheus/VictoriaMetrics exporters
├── pkg/                          # Shared utilities (logging, retry helpers)
├── go.mod
└── go.sum
```

---

#### Pipeline 1: Pull-Based Polling Flow (ICMP & RESTCONF)

This flow tracks periodic health checks where the system actively queries network infrastructure.

```
[ PostgreSQL ]
       │  (1. Fetch Inventory & Frequencies)
       ▼
 ┌───────────┐  (2. Stream Jobs via gRPC)   ┌─────────────┐
 │ SCHEDULER │ ───────────────────────────> │  COLLECTOR  │
 └───────────┘                              └──────┬──────┘
       ▲                                           │ (3. Read job queue)
       │                                           ▼
       │                                    ┌─────────────┐
       │ (6. gRPC StreamResults)            │ Worker Pool │
       │                                    └──────┬──────┘
       │                                           │ (4. Execute concurrent I/O)
       │                                           ▼
       │                                    ┌─────────────┐
       └─────────────────────────────────── │ Target Node │
                                            └─────────────┘
```

#### Steps

1. Scheduler queries PostgreSQL for devices and polling intervals  
2. Sends jobs via gRPC (`StreamJobs`)  
3. Collector pushes jobs into `jobQueue`  
4. Worker pool executes with timeout + rate limiting  
5. Performs ICMP / RESTCONF calls  
6. Sends `UnifiedEvent` back via `StreamResults`  


#### Pipeline 2: Push-Based Streaming Flow (Syslog)

This flow handles spontaneous events generated directly by network devices.

```
 ┌─────────────┐  (1. UDP Packet on Port 514)  ┌─────────────┐
 │ Target Node │ ────────────────────────────> │  COLLECTOR  │
 └─────────────┘                               └──────┬──────┘
 (Firewall/Switch)                                    │ (2. Extract payload)
                                                      ▼
                                               ┌─────────────┐
                                               │ Syslog Loop │
                                               └──────┬──────┘
                                                      │ (3. Normalize data struct)
                                                      ▼
 ┌───────────┐         (4. gRPC Stream)        ┌─────────────┐
 │ SCHEDULER │ <────────────────────────────── │ Collector   │
 └─────┬─────┘                                 └─────────────┘
       │
       ├─ (5a. FAILED) ──> PostgreSQL (alerts)
       └─ (5b. METRICS) ─> Timeseries DB
```

#### Steps

1. Device emits syslog event  
2. Collector receives via UDP listener  
3. Normalizes into `UnifiedEvent`  
4. Streams to scheduler via gRPC  


#### Central Processing Layer (Scheduler Routing)
Once the Scheduler receives a `UnifiedEvent`, it routes:

##### PostgreSQL (Alerts)
- Condition: `Status == FAILED`
- Stored in `active_alerts`

##### Timeseries DB (VictoriaMetrics)
- Condition: performance metrics  
- Format: Influx Line Protocol  
- Sent via HTTP POST