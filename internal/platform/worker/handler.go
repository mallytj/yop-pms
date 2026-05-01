package worker

import (
	"context"
	"encoding/json"
)

// Handler processes a single outbox event. The payload is the raw JSONB from
// the database; handlers unmarshal it into their own typed struct.
// Returning an error triggers a retry; returning nil marks the event completed.
type Handler func(ctx context.Context, payload json.RawMessage) error
