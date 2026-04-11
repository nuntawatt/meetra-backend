-- Migration: 000002_add_events_soft_delete.up.sql
-- Adds soft-delete support to the events table.
-- Notifications are hard-deleted (cascade from events/users is correct for audit data).

ALTER TABLE events
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Partial index: enforces unique title per host only among non-deleted events
-- (optional but useful to prevent ghost duplicates)
CREATE INDEX IF NOT EXISTS idx_events_deleted_at
    ON events (deleted_at)
    WHERE deleted_at IS NULL;

-- Existing idx_events_status should also filter deleted rows — recreate as partial
DROP INDEX IF EXISTS idx_events_status;
CREATE INDEX IF NOT EXISTS idx_events_status
    ON events (status)
    WHERE deleted_at IS NULL;
