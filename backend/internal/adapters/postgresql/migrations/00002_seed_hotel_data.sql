-- +goose Up
-- +goose StatementBegin
INSERT INTO users (username, email, password_hash, first_name, last_name, role, is_active) VALUES
('jdoe', 'jdoe@example.com', '$2a$12$KIXQJY1K6u5jHc3Pq5hO8uJ8bG9kF6jF6jF6jF6jF6jF6jF6jF6jF6', 'John', 'Doe', 'admin', true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users WHERE username = 'jdoe';
-- +goose StatementEnd
