-- Seed data for local development / testing
-- Run: psql $DATABASE_URL -f scripts/seed.sql

-- Seed admin user (password: Admin@12345)
INSERT INTO users (id, username, email, password, role)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin',
    'admin@wego.dev',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6o.gyVhHMi',
    'admin'
) ON CONFLICT DO NOTHING;

-- Seed regular user (password: User@12345)
INSERT INTO users (id, username, email, password, role)
VALUES (
    'b0000000-0000-0000-0000-000000000001',
    'alice',
    'alice@wego.dev',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj6o.gyVhHMi',
    'user'
) ON CONFLICT DO NOTHING;

-- Seed sample event
INSERT INTO events (id, host_id, title, description, location, max_capacity, status, starts_at, ends_at)
VALUES (
    'c0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'Bangkok Go Meetup Q2 2026',
    'Monthly Go developer meetup in Bangkok. Topics: clean architecture, generics, and more.',
    'Siam Paragon, Bangkok',
    50,
    'published',
    now() + interval '7 days',
    now() + interval '7 days' + interval '3 hours'
) ON CONFLICT DO NOTHING;
