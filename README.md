# network-monitoring-system

### 1. ICMP (Ping) Collector

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
| SNMP      | OIDs            | metrics          |
| TCP       | host:port       | open/close       |
| gNMI      | path            | telemetry stream |
| Syslog    | UDP port        | log events       |
| Trap      | UDP port        | alerts           |
| RESTCONF  | API endpoint    | JSON state       |