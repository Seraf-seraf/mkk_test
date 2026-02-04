-- +goose Up
INSERT INTO users (id, email, password_hash, created_at, updated_at)
VALUES (UUID(), 'admin@example.com', '$2a$12$v7rg0CDi75./mLodtKm/Ju64JOn8UN6GNO/0UWMc2vpsJvQ0gZ2KC', NOW(), NOW()); -- password: admin123

-- +goose Down
DELETE FROM users WHERE email = 'admin@example.com';
