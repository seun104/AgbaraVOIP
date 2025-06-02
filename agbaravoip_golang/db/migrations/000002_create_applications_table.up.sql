CREATE TABLE IF NOT EXISTS applications (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL,
    account_sid VARCHAR(64) NOT NULL,
    friendly_name VARCHAR(255),
    voice_url TEXT,
    voice_method VARCHAR(10),
    voice_fallback_url TEXT,
    voice_fallback_method VARCHAR(10),
    status_callback_url TEXT,
    status_callback_method VARCHAR(10),
    sms_url TEXT,
    sms_method VARCHAR(10),
    sms_fallback_url TEXT,
    sms_fallback_method VARCHAR(10),
    sms_status_callback_url TEXT,
    sms_status_callback_method VARCHAR(10),
    heartbeat_url TEXT,
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
    CONSTRAINT fk_applications_account
        FOREIGN KEY(account_sid) 
        REFERENCES accounts(sid)
        ON DELETE CASCADE -- If an account is deleted, its applications are also deleted
);

CREATE INDEX IF NOT EXISTS idx_applications_sid ON applications(sid);
CREATE INDEX IF NOT EXISTS idx_applications_account_sid ON applications(account_sid);

-- Reuse the trigger function for updated_at if it's generic enough
-- or create a specific one if needed. Assuming update_updated_at_column can be reused.
CREATE TRIGGER update_applications_updated_at
BEFORE UPDATE ON applications
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
