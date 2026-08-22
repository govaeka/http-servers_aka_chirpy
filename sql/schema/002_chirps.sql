-- +goose Up
CREATE TABLE chirps (
    id  UUID PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    body       VARCHAR(140) NOT NULL,
    user_id     UUID REFERENCES users (id)
);

-- +goose Down
DROP TABLE chirps;