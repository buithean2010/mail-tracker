CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,              -- random 32-byte hex
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
