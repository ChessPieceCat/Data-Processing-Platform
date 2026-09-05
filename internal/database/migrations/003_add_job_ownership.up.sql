DELETE FROM jobs;

ALTER TABLE jobs
    ADD COLUMN user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN session_id BIGINT REFERENCES sessions(id) ON DELETE CASCADE;

ALTER TABLE jobs
    ADD CONSTRAINT jobs_exactly_one_owner
    CHECK (
        (user_id IS NOT NULL AND session_id IS NULL)
        OR
        (user_id IS NULL AND session_id IS NOT NULL)
    );