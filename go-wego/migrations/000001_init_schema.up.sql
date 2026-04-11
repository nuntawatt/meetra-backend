-- Migration: 000001_init_schema.up.sql
-- Creates the core tables with proper indexes for the WeGo platform.

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ——— Users ——————————————————————————————————————————————————————————————————

CREATE TABLE IF NOT EXISTS users (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    VARCHAR(50) NOT NULL,
    email       VARCHAR(255) NOT NULL,
    password    VARCHAR(255) NOT NULL,
    role        VARCHAR(20)  NOT NULL DEFAULT 'user',
    avatar_url  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

-- Unique index on email (only for non-deleted users)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users (email)
    WHERE deleted_at IS NULL;

-- Index for listing/filtering active users
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- ——— Events —————————————————————————————————————————————————————————————————

CREATE TABLE IF NOT EXISTS events (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    host_id       UUID         NOT NULL REFERENCES users(id),
    title         VARCHAR(200) NOT NULL,
    description   TEXT         NOT NULL,
    location      VARCHAR(500) NOT NULL,
    image_url     TEXT,
    max_capacity  INT          NOT NULL DEFAULT 100,
    status        VARCHAR(20)  NOT NULL DEFAULT 'published',
    starts_at     TIMESTAMPTZ  NOT NULL,
    ends_at       TIMESTAMPTZ  NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Index for filtering by status (most common query)
CREATE INDEX IF NOT EXISTS idx_events_status ON events (status);

-- Index for filtering by location
CREATE INDEX IF NOT EXISTS idx_events_location ON events (location);

-- Index for time-range queries
CREATE INDEX IF NOT EXISTS idx_events_starts_at ON events (starts_at DESC);

-- Index for fetching events by host
CREATE INDEX IF NOT EXISTS idx_events_host_id ON events (host_id);

-- Full-text search index on title and description
CREATE INDEX IF NOT EXISTS idx_events_search
    ON events USING gin(to_tsvector('english', title || ' ' || description));

-- ——— Event Participants ——————————————————————————————————————————————————————

CREATE TABLE IF NOT EXISTS event_participants (
    event_id   UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, user_id)  -- naturally prevents duplicates
);

CREATE INDEX IF NOT EXISTS idx_ep_user_id ON event_participants (user_id);

-- ——— Notifications ——————————————————————————————————————————————————————————

CREATE TABLE IF NOT EXISTS notifications (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id    UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    type        VARCHAR(50) NOT NULL,
    message     TEXT        NOT NULL,
    is_read     BOOLEAN     NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id
    ON notifications (user_id, created_at DESC);
