-- name: CreateOutboxEvent :one
INSERT INTO internal.outbox_events (event_type, payload, process_at)
VALUES (@event_type, @payload, @process_at)
RETURNING id;
