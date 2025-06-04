-- Schema for the 'accounts' table in PostgreSQL

DROP TABLE IF EXISTS accounts CASCADE; 

CREATE TABLE IF NOT EXISTS accounts (
    sid VARCHAR(255) PRIMARY KEY,
    parent_sid VARCHAR(255) REFERENCES accounts(sid) ON DELETE SET NULL, 
    friendly_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(50),
    type VARCHAR(50) NOT NULL DEFAULT 'trial',      
    status VARCHAR(50) NOT NULL DEFAULT 'active',   
    auth_token TEXT,                                
    date_created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- New Gateway Settings Columns
    default_outbound_gateway VARCHAR(255) NULL,
    gateway_selection_script VARCHAR(255) NULL
);

CREATE INDEX IF NOT EXISTS idx_accounts_parent_sid ON accounts(parent_sid);
CREATE INDEX IF NOT EXISTS idx_accounts_auth_token ON accounts(auth_token); 

-- Assuming the function update_modified_column() was created by 001_create_calls_table.sql
CREATE TRIGGER update_accounts_modtime
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

COMMENT ON TABLE accounts IS 'Stores information about user or sub-user accounts.';
COMMENT ON COLUMN accounts.sid IS 'Unique identifier for the account, prefixed (e.g., ACxxx).';
COMMENT ON COLUMN accounts.parent_sid IS 'Identifier of the parent account, if this is a sub-account. NULL for master accounts.';
COMMENT ON COLUMN accounts.auth_token IS 'Hashed authentication token for API access.';
COMMENT ON COLUMN accounts.type IS 'Type of the account (e.g., trial, full).';
COMMENT ON COLUMN accounts.status IS 'Status of the account (e.g., active, suspended).';
COMMENT ON COLUMN accounts.default_outbound_gateway IS 'Name of the default FreeSWITCH gateway for outbound calls.';
COMMENT ON COLUMN accounts.gateway_selection_script IS 'Name of a Lua script in FreeSWITCH for dynamic gateway selection.';
