CREATE TABLE IF NOT EXISTS recordings (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL,
    account_sid VARCHAR(64) NOT NULL REFERENCES accounts(sid) ON DELETE CASCADE,
    call_sid VARCHAR(64) REFERENCES calls(sid) ON DELETE SET NULL,
    conference_sid VARCHAR(64), -- Add REFERENCES conferences(sid) later if conferences table exists
    duration_seconds INT NOT NULL DEFAULT 0,
    file_path VARCHAR(1024) NOT NULL,
    format VARCHAR(16) NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX IF NOT EXISTS idx_recordings_account_sid ON recordings(account_sid);
CREATE INDEX IF NOT EXISTS idx_recordings_call_sid ON recordings(call_sid);
-- CREATE INDEX IF NOT EXISTS idx_recordings_conference_sid ON recordings(conference_sid);

-- Trigger for updated_at
-- Ensure this function is created only if it doesn't exist from a previous migration.
-- For simplicity in this script, using CREATE OR REPLACE.
-- A more robust migration system might check for existence or use a separate shared DDL script.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = (NOW() AT TIME ZONE 'UTC');
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_recordings_updated_at
BEFORE UPDATE ON recordings
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
