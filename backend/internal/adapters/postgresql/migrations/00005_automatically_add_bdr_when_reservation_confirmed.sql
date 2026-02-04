-- +goose Up
-- +goose StatementBegin
-- Create a rule to auto insert a BDR when a reservation is confirmed
-- TODO when res logic is made
-- Intentionally left as a no-op until reservation logic is implemented
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Matching no-op down migration
SELECT 1;
-- +goose StatementEnd
