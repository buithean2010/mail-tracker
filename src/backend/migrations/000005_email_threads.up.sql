CREATE TYPE summary_status AS ENUM ('pending', 'processing', 'done', 'failed', 'no_key');

CREATE TABLE email_threads (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id  TEXT NOT NULL,
    subject          TEXT NOT NULL DEFAULT '',
    from_address     TEXT NOT NULL DEFAULT '',
    received_at      TIMESTAMPTZ NOT NULL,
    last_message_at  TIMESTAMPTZ NOT NULL,
    message_count    INT NOT NULL DEFAULT 1,
    raw_messages     JSONB NOT NULL DEFAULT '[]',

    -- Tier 2 fields (populated after AI summary)
    summary          TEXT NOT NULL DEFAULT '',
    priority         TEXT NOT NULL DEFAULT '' CHECK (priority IN ('', 'high', 'medium', 'low')),
    action_required  BOOLEAN NOT NULL DEFAULT false,
    action_detail    TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'done', 'archived')),
    notes            TEXT NOT NULL DEFAULT '',
    summary_status   summary_status NOT NULL DEFAULT 'pending',

    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, conversation_id)
);
