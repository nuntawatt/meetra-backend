-- Migration: 000002_add_events_soft_delete.down.sql

DROP INDEX IF EXISTS idx_events_deleted_at;
DROP INDEX IF EXISTS idx_events_status;

CREATE INDEX IF NOT EXISTS idx_events_status ON events (status);

ALTER TABLE events
    DROP COLUMN IF EXISTS deleted_at;
