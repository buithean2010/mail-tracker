CREATE TABLE user_api_keys (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    ai_mode         TEXT NOT NULL CHECK (ai_mode IN ('byok', 'power_automate')),
    provider        TEXT NOT NULL DEFAULT '',   -- openai | openrouter (byok only)
    api_key         TEXT NOT NULL DEFAULT '',   -- AES-256 encrypted (byok only)
    model           TEXT NOT NULL DEFAULT '',   -- byok only
    pa_webhook_url  TEXT NOT NULL DEFAULT '',   -- AES-256 encrypted (power_automate only)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
