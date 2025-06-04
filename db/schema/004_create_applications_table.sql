-- Schema for the 'applications' table in PostgreSQL

DROP TABLE IF EXISTS applications CASCADE;

CREATE TABLE IF NOT EXISTS applications (
    sid VARCHAR(255) PRIMARY KEY,
    account_sid VARCHAR(255) NOT NULL, -- No direct FK to accounts for now for simplified schema loading
    friendly_name VARCHAR(255) NOT NULL,
    voice_url TEXT,
    voice_method VARCHAR(10), -- e.g., GET, POST
    voice_fallback_url TEXT,
    voice_fallback_method VARCHAR(10),
    status_callback TEXT,
    status_callback_method VARCHAR(10),
    sms_url TEXT,
    sms_method VARCHAR(10),
    sms_fallback_url TEXT,
    sms_fallback_method VARCHAR(10),
    sms_status_callback TEXT,
    sms_status_callback_method VARCHAR(10),
    heartbeat_url TEXT,
    date_created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_applications_account_sid ON applications(account_sid);

-- Assuming the function update_modified_column() was created by 001_create_calls_table.sql
CREATE TRIGGER update_applications_modtime
    BEFORE UPDATE ON applications
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

COMMENT ON TABLE applications IS 'Stores configurations for voice and SMS application handling (webhooks).';
COMMENT ON COLUMN applications.sid IS 'Unique identifier for the application, prefixed (e.g., APxxx).';
COMMENT ON COLUMN applications.account_sid IS 'Account SID that owns this application.';
COMMENT ON COLUMN applications.friendly_name IS 'User-defined name for the application.';
COMMENT ON COLUMN applications.voice_url IS 'URL to request when a call for this application is received.';
COMMENT ON COLUMN applications.sms_url IS 'URL to request when an SMS for this application is received.';
