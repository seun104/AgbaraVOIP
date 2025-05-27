-- Schema for the 'calls' table in PostgreSQL

-- Drop table if it exists, useful for development and reapplying schema
-- Consider commenting this out for production or using proper migration tools
DROP TABLE IF EXISTS calls CASCADE; 

CREATE TABLE IF NOT EXISTS calls (
    sid VARCHAR(255) PRIMARY KEY,          -- Unique Call SID, e.g., CA...
    account_sid VARCHAR(255) NOT NULL,     -- Account SID this call belongs to
    caller_id VARCHAR(255),                -- Caller's number or identifier
    call_to VARCHAR(255),                  -- Number or endpoint being called
    answer_url TEXT,                       -- URL to fetch TwiML/XML when call is answered
    application_sid VARCHAR(255) NULL,     -- << NEW: Optional: Application handling this call
    status VARCHAR(50),                    -- Call status (e.g., queued, ringing, in-progress, completed, failed)
    timeout_seconds INT,                   -- Call timeout in seconds (if applicable)
    direction VARCHAR(50),                 -- Call direction (e.g., inbound, outbound-api, outbound-dial)
    duration_seconds INT DEFAULT 0,        -- Duration of the call in seconds
    price NUMERIC(19, 5) DEFAULT 0.0,      -- Cost of the call
    start_time TIMESTAMPTZ,                -- Time the call started ringing or was initiated
    end_time TIMESTAMPTZ,                  -- Time the call ended
    date_created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Record creation timestamp
    date_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP, -- Record last update timestamp
    answered_by VARCHAR(255),              -- Identifier of who answered (e.g., human, machine)
    freeswitch_call_id VARCHAR(255) NULL   -- << NEW COLUMN: FreeSWITCH Job/Channel UUID
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_calls_account_sid ON calls(account_sid);
CREATE INDEX IF NOT EXISTS idx_calls_application_sid ON calls(application_sid); -- << NEW: Index for new app sid
CREATE INDEX IF NOT EXISTS idx_calls_start_time ON calls(start_time);
CREATE INDEX IF NOT EXISTS idx_calls_date_created ON calls(date_created);
CREATE INDEX IF NOT EXISTS idx_calls_freeswitch_call_id ON calls(freeswitch_call_id); -- << NEW: Index for new fs call id

-- Optional: Trigger to automatically update 'date_updated' timestamp on any row update
-- Function should be defined once. If this is the first schema file, define it.
-- If other schema files also define it, ensure it's idempotent or defined only once.
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.date_updated = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_calls_modtime
    BEFORE UPDATE ON calls
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

COMMENT ON TABLE calls IS 'Stores information about voice calls.';
COMMENT ON COLUMN calls.sid IS 'Unique identifier for the call, prefixed (e.g., CAxxx).';
COMMENT ON COLUMN calls.account_sid IS 'Identifier of the account that initiated or received the call.';
COMMENT ON COLUMN calls.application_sid IS 'SID of the Application used for call flow, if any.'; -- << NEW
COMMENT ON COLUMN calls.status IS 'Current status of the call (e.g., queued, ringing, completed).';
COMMENT ON COLUMN calls.duration_seconds IS 'Total duration of the call in seconds, after it has ended.';
COMMENT ON COLUMN calls.price IS 'Cost associated with the call.';
COMMENT ON COLUMN calls.start_time IS 'Timestamp when the call attempt was initiated or connected.';
COMMENT ON COLUMN calls.end_time IS 'Timestamp when the call was concluded.';
COMMENT ON COLUMN calls.freeswitch_call_id IS 'Internal FreeSWITCH call identifier (Job UUID or Channel UUID).'; -- << NEW
