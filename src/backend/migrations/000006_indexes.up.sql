-- Fast thread listing per user
CREATE INDEX idx_email_threads_user_received ON email_threads (user_id, received_at DESC);

-- Fast summary job: find pending threads
CREATE INDEX idx_email_threads_summary_status ON email_threads (summary_status) WHERE summary_status = 'pending';

-- Fast session lookup by ID
CREATE INDEX idx_sessions_id ON sessions (id);

-- Fast cleanup of expired sessions
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

-- Fast active-user lookup for sync job
CREATE INDEX idx_sessions_user_expires ON sessions (user_id, expires_at DESC);
