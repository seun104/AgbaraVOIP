-- Schemas for 'conferences' and 'participants' tables in PostgreSQL

-- Drop tables if they exist, useful for development and reapplying schema
DROP TABLE IF EXISTS participants CASCADE;
DROP TABLE IF EXISTS conferences CASCADE; 

-- Conferences Table
CREATE TABLE IF NOT EXISTS conferences (
    sid VARCHAR(255) PRIMARY KEY,
    account_sid VARCHAR(255) NOT NULL, -- Removed direct FK to accounts(sid) for now to simplify schema application order, can be added back if accounts table is guaranteed to exist first. Consider application-level checks or later ALTER TABLE.
    friendly_name VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'initializing', -- e.g., 'initializing', 'in-progress', 'completed'
    date_created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_conferences_account_sid ON conferences(account_sid);

-- Assuming the function update_modified_column() was created by 001_create_calls_table.sql
CREATE TRIGGER update_conferences_modtime
    BEFORE UPDATE ON conferences
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

COMMENT ON TABLE conferences IS 'Stores information about conference rooms.';
COMMENT ON COLUMN conferences.sid IS 'Unique identifier for the conference, prefixed (e.g., COxxx).';
COMMENT ON COLUMN conferences.status IS 'Current status of the conference.';

-- Participants Table
CREATE TABLE IF NOT EXISTS participants (
    call_sid VARCHAR(255) NOT NULL, -- SID of the call leg acting as participant
    conference_sid VARCHAR(255) NOT NULL REFERENCES conferences(sid) ON DELETE CASCADE,
    account_sid VARCHAR(255) NOT NULL, -- Denormalized, should match conference.account_sid. No direct FK to accounts(sid) for now for simplicity.
    friendly_name VARCHAR(255),
    muted BOOLEAN DEFAULT FALSE,
    start_conference_on_enter BOOLEAN DEFAULT FALSE,
    end_conference_on_exit BOOLEAN DEFAULT FALSE,
    date_created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (conference_sid, call_sid) -- Composite primary key
);

CREATE INDEX IF NOT EXISTS idx_participants_call_sid ON participants(call_sid);
CREATE INDEX IF NOT EXISTS idx_participants_conference_sid ON participants(conference_sid); -- Added for lookups by conference

-- Assuming the function update_modified_column() was created by 001_create_calls_table.sql
CREATE TRIGGER update_participants_modtime
    BEFORE UPDATE ON participants
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

COMMENT ON TABLE participants IS 'Stores information about participants in a conference.';
COMMENT ON COLUMN participants.call_sid IS 'Call SID of the participant leg.';
COMMENT ON COLUMN participants.conference_sid IS 'Conference SID the participant belongs to.';
COMMENT ON COLUMN participants.muted IS 'Whether the participant is currently muted.';
