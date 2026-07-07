-- +goose Up
-- +goose StatementBegin
-- Schema for internal infrastructure tables (not tenant data — no RLS applied).
CREATE SCHEMA IF NOT EXISTS internal;

CREATE TYPE internal.outbox_event_status AS ENUM(
    'pending',
    'processing',
    'completed',
    'failed'
);

CREATE TABLE internal.outbox_events(
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  event_type TEXT NOT NULL CHECK(CHAR_LENGTH(event_type) <= 100),
  payload JSONB NOT NULL,
  status internal.outbox_event_status NOT NULL DEFAULT 'pending',
  retry_count INT NOT NULL DEFAULT 0,
  last_error TEXT,
  -- process_at doubles as a visibility timeout: on claim it is forwarded
  -- 5 minutes into the future so a crashed worker does not strand the row.
  process_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

-- Covers both pending rows and timed-out processing rows (crash recovery).
-- Poll query: WHERE (status = 'pending' OR status = 'processing') AND process_at <= NOW()
CREATE INDEX idx_outbox_events_pending
ON internal.outbox_events(process_at)
WHERE
  status IN(
    'pending',
    'processing'
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS internal.outbox_events;
DROP TYPE IF EXISTS internal.outbox_event_status;
DROP SCHEMA IF EXISTS internal;
-- +goose StatementEnd
