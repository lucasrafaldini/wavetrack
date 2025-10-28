-- Migração para preservar histórico ao deletar colaboradores
-- Remove ON DELETE CASCADE das foreign keys

-- 1. Backup das tabelas
CREATE TABLE IF NOT EXISTS employees_backup AS SELECT * FROM employees;
CREATE TABLE IF NOT EXISTS events_backup AS SELECT * FROM events;

-- 2. Dropar tabelas com constraint
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS events;

-- 3. Recriar tabela de funcionários SEM CASCADE
CREATE TABLE employees (
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

-- 4. Recriar tabela de eventos SEM CASCADE
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL, -- 'arrival', 'departure', 'unknown_device'
    mac_address TEXT NOT NULL,
    employee_name TEXT,
    signal_strength INTEGER DEFAULT 0,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT, -- JSON para dados adicionais
    FOREIGN KEY (mac_address) REFERENCES devices(mac_address)
);

-- 5. Restaurar dados
INSERT INTO employees SELECT * FROM employees_backup;
INSERT INTO events SELECT * FROM events_backup;

-- 6. Recriar índices
CREATE INDEX IF NOT EXISTS idx_employees_mac ON employees(mac_address);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_mac ON events(mac_address);

-- 7. Limpar backups
DROP TABLE IF EXISTS employees_backup;
DROP TABLE IF EXISTS events_backup;

-- 8. Recriar view
DROP VIEW IF EXISTS device_details;
CREATE VIEW device_details AS
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
