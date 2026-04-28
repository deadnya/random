ALTER TABLE users
    ADD COLUMN IF NOT EXISTS public_id TEXT;

UPDATE users
SET public_id = 'legacy-' || id
WHERE public_id IS NULL;

ALTER TABLE users
    ALTER COLUMN public_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_public_id
    ON users(public_id);

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_username_key;