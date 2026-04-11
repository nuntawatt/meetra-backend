-- Migration: 000001_init_schema.down.sql
-- Drops all tables created by the up migration.

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS event_participants;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";
