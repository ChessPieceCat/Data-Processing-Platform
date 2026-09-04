CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);