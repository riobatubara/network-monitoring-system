# network-monitoring-system

#### Architecture
![Alt text](./network-monitoring-system.png.png?raw=true "Optional Title")

<!-- ### 1. ICMP (Ping) Collector

#### Input
```
{
  "type": "icmp",
  "target": "192.168.1.1",
  "interval_sec": 10,
  "timeout_ms": 1000,
  "retries": 2
}
```

or
```
{
  "id": "evt-1001",
  "timestamp": "2026-06-23T10:00:00Z",
  "source": "router-1",
  "protocol": "ICMP",
  "payload": {
    "host": "8.8.8.8",
    "latency_ms": 12,
    "packet_loss": 0
  }
}
```

#### Internal Behavior
- Goroutine per target
- Context with timeout
- Retry loop
- Rate-limited worker pool

#### Output
```
{
  "target": "192.168.1.1",
  "status": "up",
  "latency_ms": 12,
  "packet_loss": 0,
  "timestamp": "2026-06-22T10:00:00Z"
}
```

### 2. SNMP Polling (CPU, Interface)
#### Input
```
{
  "type": "snmp",
  "target": "192.168.1.10",
  "community": "public",
  "version": "v2c",
  "oids": [
    "1.3.6.1.2.1.1.3.0",
    "1.3.6.1.2.1.2.2.1.10.1"
  ],
  "timeout_ms": 2000,
  "retries": 3
}
```

or
```
{
  "id": "evt-1002",
  "timestamp": "2026-06-23T10:00:01Z",
  "source": "switch-3",
  "protocol": "SNMP",
  "payload": {
    "oids": [
      {
        "oid": "1.3.6.1.2.1.1.5.0",
        "value": "core-switch-01"
      },
      {
        "oid": "1.3.6.1.2.1.2.2.1.10.1",
        "value": 12345678
      }
    ]
  }
}
```

#### Internal Behavior
- Worker pool executes SNMP GET/WALK
- Mutex for shared metrics cache
- Channel for results aggregation

#### Output
```
{
  "target": "192.168.1.10",
  "metrics": {
    "sysUptime": 12345678,
    "ifInOctets_1": 987654321
  },
  "status": "success",
  "timestamp": "2026-06-22T10:00:05Z"
}
```

### 3. TCP Port Check
#### Input
```
{
  "type": "tcp",
  "target": "example.com:443",
  "timeout_ms": 1500,
  "retries": 2
}
```

#### Internal Behavior
- net.DialContext() with timeout
- Retry on failure
- Connection pooling (optional)

#### Output
```
{
  "target": "example.com:443",
  "status": "open",
  "response_time_ms": 45,
  "timestamp": "2026-06-22T10:00:03Z"
}
```

### 4. gNMI Telemetry (Streaming)
#### Input
```
{
  "type": "gnmi",
  "target": "router1",
  "subscription": [
    "/interfaces/interface/state/counters"
  ],
  "mode": "stream",
  "sample_interval_ms": 5000
}
```

or
```
{
  "id": "evt-1003",
  "timestamp": "2026-06-23T10:00:02Z",
  "source": "router-2",
  "protocol": "gNMI",
  "payload": {
    "path": "/interfaces/interface/state/counters",
    "values": {
      "in_octets": 998877,
      "out_octets": 887766
    }
  }
}
```

#### Internal Behavior
- Persistent gRPC connection
- Goroutine listener per stream
- Channel fan-in to aggregator

#### Output
```
{
  "target": "router1",
  "updates": [
    {
      "interface": "eth0",
      "in_octets": 123456,
      "out_octets": 654321
    }
  ],
  "timestamp": "2026-06-22T10:00:10Z"
}
```

### 5. Syslog Collector
#### Input
```
{
  "type": "syslog",
  "listen_port": 514,
  "protocol": "udp"
}
```

or
```
{
  "id": "evt-1004",
  "timestamp": "2026-06-23T10:00:03Z",
  "source": "firewall-1",
  "protocol": "SYSLOG",
  "payload": {
    "severity": "warning",
    "message": "CPU usage high",
    "facility": "system"
  }
}
```

#### Internal Behavior
- UDP listener (non-blocking)
- Goroutine per packet (or batch)
- Channel queue → parser → storage

#### Output
```
{
  "device": "firewall-1",
  "severity": "warning",
  "message": "Blocked connection from 10.0.0.5",
  "timestamp": "2026-06-22T10:00:12Z"
}
```


### 6. SNMP Trap Receiver
#### Input
```
{
  "type": "snmp_trap",
  "listen_port": 162
}
```

or
```
{
  "id": "evt-1005",
  "timestamp": "2026-06-23T10:00:04Z",
  "source": "switch-5",
  "protocol": "SNMP_TRAP",
  "payload": {
    "trap_oid": "1.3.6.1.6.3.1.1.5.3",
    "varbinds": [
      {
        "oid": "1.3.6.1.2.1.2.2.1.7.2",
        "value": "down"
      }
    ]
  }
}
```

#### Internal Behavior
- UDP listener
- Decode ASN.1 traps
- Push to event pipeline

#### Output
```
{
  "source": "192.168.1.20",
  "trap_oid": "1.3.6.1.6.3.1.1.5.3",
  "description": "Link Down",
  "timestamp": "2026-06-22T10:00:15Z"
}
```

### 7. RESTCONF / NETCONF Polling
#### Input
```
{
  "type": "restconf",
  "url": "https://router/api/interfaces",
  "method": "GET",
  "auth": {
    "username": "admin",
    "password": "password"
  },
  "timeout_ms": 3000
}
```

or
```
{
  "id": "evt-1006",
  "timestamp": "2026-06-23T10:00:05Z",
  "source": "router-3",
  "protocol": "NETCONF",
  "payload": {
    "operation": "get-config",
    "format": "xml",
    "data": "<interfaces><interface><name>eth0</name></interface></interfaces>"
  }
}
```

#### Internal Behavior
- HTTP client with timeout + retry
- Rate-limited requests
- Context cancellation

#### Output
```
{
  "target": "router",
  "interfaces": [
    {
      "name": "Gig0/0",
      "status": "up",
      "traffic_in": 100000
    }
  ],
  "status": "success",
  "timestamp": "2026-06-22T10:00:20Z"
}
```


### 8. Worker Pool Job Queue (Internal Input)
#### Input (Job Queue)
```
{
  "job_id": "job-123",
  "collector_type": "snmp",
  "target": "192.168.1.10",
  "scheduled_at": "2026-06-22T10:00:00Z",
  "priority": "high"
}
```

#### Internal Behavior
- Buffered channel queue
- Fixed worker pool (e.g., 50 workers)
- Rate limiter (token bucket)
- Mutex for shared state

#### Output
```
{
  "job_id": "job-123",
  "status": "completed",
  "duration_ms": 180,
  "result_ref": "metrics/snmp/192.168.1.10"
}
```


### Summary (Quick Mapping)

| Collector | Input           | Output           |
| --------- | --------------- | ---------------- |
| ICMP      | target, timeout | latency, status  |
| TCP       | host:port       | open/close       |
| Syslog    | UDP port        | log events       |
| RESTCONF  | API endpoint    | JSON state       |


network-monitoring-system/
│
├── cmd/
│   ├── ingestion-rest/
│   ├── ingestion-grpc/
│   ├── syslog-receiver/
│   ├── snmp-trap-receiver/
│   │
│   ├── scheduler-service/
│   ├── worker-node/
│   ├── collector-icmp/
│   ├── collector-restconf/
│   │
│   ├── processing-core/
│   ├── alerting-service/
│   ├── api-backend/
│
├── internal/
│   ├── ingestion/
│   │   ├── rest/
│   │   ├── grpc/
│   │   ├── syslog/
│   │   ├── parser/
│   │   └── normalizer/
│   │
│   ├── scheduler/
│   │   ├── job/
│   │   ├── allocator/
│   │   ├── retry/
│   │   └── heartbeat/
│   │
│   ├── worker/
│   │   ├── pool/
│   │   ├── executor/
│   │   ├── timeout/
│   │   ├── ratelimit/
│   │   └── dispatcher/
│   │
│   ├── collectors/
│   │   ├── icmp/
│   │   └── restconf/
│   │
│   ├── processing/
│   │   ├── pipeline/
│   │   ├── enrichment/
│   │   ├── filter/
│   │   ├── aggregator/
│   │   └── dedup/
│   │
│   ├── eventbus/
│   │   ├── kafka/
│   │   ├── nats/
│   │   ├── redisstream/
│   │   └── memory/
│   │
│   ├── storage/
│   │   ├── postgres/
│   │   ├── victoriametrics/
│   │   ├── redis/
│   │   └── interface/
│   │
│   ├── alerting/
│   │   ├── engine/
│   │   ├── rules/
│   │   ├── evaluator/
│   │   └── notifier/
│   │
│   ├── api/
│   │   ├── http/
│   │   ├── handlers/
│   │   ├── middleware/
│   │   └── grpc/
│   │
│   ├── config/
│   ├── logger/
│   ├── metrics/
│   ├── tracing/
│   ├── errors/
│   └── utils/
│
├── pkg/
│   ├── models/
│   │   ├── event.go
│   │   ├── device.go
│   │   ├── metric.go
│   │   └── job.go
│   │
│   ├── proto/
│   │   ├── scheduler.proto
│   │   ├── worker.proto
│   │   ├── ingestion.proto
│   │   └── collector.proto
│   │
│   ├── constants/
│   └── types/
│
├── deployments/
│   ├── docker/
│   ├── kubernetes/
│   ├── helm/
│   └── compose/
│
├── scripts/
│   ├── build.sh
│   ├── run-local.sh
│   ├── migrate.sh
│   └── load-test.sh
│
├── configs/
│   ├── dev.yaml
│   ├── staging.yaml
│   ├── prod.yaml
│   └── collectors.yaml
│
├── migrations/
│   └── postgres/
│
├── docs/
│   ├── architecture.md
│   ├── protocols.md
│   ├── api.md
│   └── diagrams/
│
├── go.mod
├── go.sum
└── Makefile -->