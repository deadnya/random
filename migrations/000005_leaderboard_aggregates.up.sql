CREATE TABLE IF NOT EXISTS user_aggregates (
    user_id BIGINT PRIMARY KEY,
    username TEXT NOT NULL,
    best_score INTEGER NOT NULL DEFAULT 0,
    total_value INTEGER NOT NULL DEFAULT 0,
    roll_count INTEGER NOT NULL DEFAULT 0,
    best_number INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
