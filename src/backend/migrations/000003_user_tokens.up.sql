CREATE TABLE user_tokens (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_token  TEXT NOT NULL,              -- AES-256 encrypted
    refresh_token TEXT NOT NULL,             -- AES-256 encrypted
    expires_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
