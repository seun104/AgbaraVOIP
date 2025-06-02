-- +migrate Up
-- SQL in section Up is executed when this migration is applied.

CREATE TYPE call_status AS ENUM (
    queued,
    initiated,
    ringing,
    in-progress,
    completed,
    failed,
    busy,
    no-answer,
    canceled 
);

CREATE TYPE call_direction AS ENUM (
    inbound,
    outbound-api,
    outbound-dial
);

CREATE TABLE IF NOT EXISTS calls (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL,
    account_sid VARCHAR(64) NOT NULL REFERENCES accounts(sid) ON DELETE CASCADE,
    application_sid VARCHAR(64) NULL REFERENCES applications(sid) ON DELETE SET NULL,
    
    from_num VARCHAR(100), 
    to_num VARCHAR(100),   
    
    answer_url TEXT,       
    
    status call_status NOT NULL DEFAULT queued,
    direction call_direction NOT NULL,
    
    duration_seconds INTEGER DEFAULT 0,
    price NUMERIC(10, 5) DEFAULT 0.0,
    
    answered_by VARCHAR(100), 
    timeout_seconds INTEGER,  
    
    hangup_cause VARCHAR(100), 
    forwarded_from VARCHAR(100), 

    start_time TIMESTAMP WITHOUT TIME ZONE NULL, 
    answer_time TIMESTAMP WITHOUT TIME ZONE NULL,
    end_time TIMESTAMP WITHOUT TIME ZONE NULL,
    
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE UTC),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE UTC),
    deleted_at TIMESTAMP WITHOUT TIME ZONE NULL
);

CREATE INDEX IF NOT EXISTS idx_calls_account_sid ON calls(account_sid);
CREATE INDEX IF NOT EXISTS idx_calls_application_sid ON calls(application_sid);
CREATE INDEX IF NOT EXISTS idx_calls_status ON calls(status);
CREATE INDEX IF NOT EXISTS idx_calls_direction ON calls(direction);
CREATE INDEX IF NOT EXISTS idx_calls_created_at ON calls(created_at);
CREATE INDEX IF NOT EXISTS idx_calls_start_time ON calls(start_time);
CREATE INDEX IF NOT EXISTS idx_calls_from_num ON calls(from_num);
CREATE INDEX IF NOT EXISTS idx_calls_to_num ON calls(to_num);

-- This assumes trigger_set_updated_at() was created in 000001 migration.
-- If not, define it here or ensure it is created globally.
CREATE TRIGGER set_calls_updated_at
BEFORE UPDATE ON calls
FOR EACH ROW
EXECUTE FUNCTION trigger_set_updated_at();

-- +migrate Down
-- SQL section Down is executed when this migration is rolled back.
-- Content for down migration is typically in a separate .down.sql file
-- For this combined file approach (if used), ensure Down statements are correctly placed.
-- However, golang-migrate prefers separate files. This Down section will be ignored if a .down.sql file exists.


