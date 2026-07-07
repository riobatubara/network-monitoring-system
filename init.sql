-- Inventory table for monitored network nodes
CREATE TABLE IF NOT EXISTS monitored_devices (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    ip_address VARCHAR(45) NOT NULL, -- IPv4 and IPv6
    protocol_type VARCHAR(20) NOT NULL, -- 'ICMP', 'RESTCONF', 'SYSLOG'
    polling_interval_sec INT DEFAULT 30,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Store encrypted credentials or authentication targets for RESTCONF
CREATE TABLE IF NOT EXISTS device_credentials (
    id SERIAL PRIMARY KEY,
    device_id INT REFERENCES monitored_devices(id) ON DELETE CASCADE,
    username VARCHAR(100) NOT NULL,
    password_hash TEXT NOT NULL,
    auth_token TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Track incidents and active system alerts caught by the pipeline
CREATE TABLE IF NOT EXISTS active_alerts (
    id SERIAL PRIMARY KEY,
    job_id VARCHAR(50) NOT NULL,
    target VARCHAR(45) NOT NULL,
    protocol VARCHAR(20) NOT NULL,
    issue_description TEXT NOT NULL,
    severity VARCHAR(20) DEFAULT 'CRITICAL',
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample seed data for testing
INSERT INTO monitored_devices (name, ip_address, protocol_type, polling_interval_sec) VALUES
('core-switch-01', '192.168.1.1', 'ICMP', 5),
('edge-router-02', '10.0.0.5', 'RESTCONF', 10),
('firewall-cluster', '172.16.5.1', 'SYSLOG', 0) -- 0 means push-based streaming
ON CONFLICT (name) DO NOTHING;
