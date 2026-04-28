CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roll_state (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    rolls_available SMALLINT NOT NULL DEFAULT 10
        CHECK (rolls_available >= 0 AND rolls_available <= 10),
    last_refill_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rolls (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rolled_number INTEGER NOT NULL
        CHECK (rolled_number >= 0 AND rolled_number <= 999999),
    total_score INTEGER NOT NULL CHECK (total_score >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roll_specs (
    id BIGSERIAL PRIMARY KEY,
    roll_id BIGINT NOT NULL REFERENCES rolls(id) ON DELETE CASCADE,
    spec_key TEXT NOT NULL,
    spec_value TEXT NOT NULL,
    spec_score INTEGER NOT NULL CHECK (spec_score >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (roll_id, spec_key)
);

CREATE INDEX IF NOT EXISTS idx_rolls_user_created_at
    ON rolls(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_rolls_total_score_desc
    ON rolls(total_score DESC);

CREATE INDEX IF NOT EXISTS idx_roll_specs_key
    ON roll_specs(spec_key);
