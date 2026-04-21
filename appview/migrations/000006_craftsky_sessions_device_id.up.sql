-- Add per-device correlation column to craftsky_sessions.
-- See docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md §3.3.
ALTER TABLE craftsky_sessions
  ADD COLUMN last_device_id TEXT;
