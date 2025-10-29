-- Migração: Adicionar campos de ambiguidade
-- Execute este script no banco existente para adicionar suporte a tipos ambíguos

-- Adiciona campos para informações de ambiguidade
ALTER TABLE devices ADD COLUMN is_ambiguous BOOLEAN DEFAULT 0;
ALTER TABLE devices ADD COLUMN possible_types TEXT DEFAULT NULL; -- JSON array dos tipos possíveis

-- Índice para consultas de dispositivos ambíguos
CREATE INDEX IF NOT EXISTS idx_devices_ambiguous ON devices(is_ambiguous);
