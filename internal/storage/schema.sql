-- WaveTrack Database Schema

-- Tabela de dispositivos detectados
CREATE TABLE IF NOT EXISTS devices (
    mac_address TEXT PRIMARY KEY,
    vendor TEXT NOT NULL DEFAULT 'Unknown',
    type TEXT NOT NULL DEFAULT 'unknown',
    first_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    signal_strength INTEGER DEFAULT 0,
    frequency INTEGER DEFAULT 0,
    channel INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1
);

-- Tabela de funcionários
CREATE TABLE IF NOT EXISTS employees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mac_address TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    department TEXT,
    custom_device_type TEXT,
    custom_vendor TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mac_address) REFERENCES devices(mac_address)
);

-- Tabela de eventos (logs estruturados)
-- Histórico preservado mesmo quando colaborador é removido
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL, -- 'arrival', 'departure', 'unknown_device'
    mac_address TEXT NOT NULL,
    employee_name TEXT,
    signal_strength INTEGER DEFAULT 0,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT, -- JSON para dados adicionais
    FOREIGN KEY (mac_address) REFERENCES devices(mac_address)
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);
CREATE INDEX IF NOT EXISTS idx_devices_active ON devices(is_active);
CREATE INDEX IF NOT EXISTS idx_employees_mac ON employees(mac_address);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_mac ON events(mac_address);

-- View para facilitar consultas (device + employee)
CREATE VIEW IF NOT EXISTS device_details AS
SELECT 
    d.mac_address,
    d.vendor,
    d.type,
    d.first_seen,
    d.last_seen,
    d.signal_strength,
    d.is_active,
    e.name as employee_name,
    e.department as employee_department
FROM devices d
LEFT JOIN employees e ON d.mac_address = e.mac_address;
