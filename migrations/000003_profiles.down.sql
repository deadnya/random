DROP INDEX IF EXISTS idx_users_public_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS public_id;

ALTER TABLE users
    ADD CONSTRAINT users_username_key UNIQUE (username);