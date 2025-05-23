-- Schema for the 'accounts' table in PostgreSQL

-- Drop table if it exists, useful for development and reapplying schema
-- Consider commenting this out for production or using proper migration tools
DROP TABLE IF EXISTS accounts CASCADE; 

CREATE TABLE IF NOT EXISTS accounts (
    sid VARCHAR(255) PRIMARY KEY,
    parent_sid VARCHAR(255) REFERENCES accounts(sid) ON DELETE SET NULL, -- Foreign key for sub-accounts, ON DELETE SET NULL means if parent is deleted, sub-account's parent_sid becomes NULL (it becomes a master account)
    friendly_name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(50),
    type VARCHAR(50) NOT NULL DEFAULT 'trial',      -- e.g., 'trial', 'full'
    status VARCHAR(50) NOT NULL DEFAULT 'active',   -- e.g., 'active', 'suspended', 'closed'
    auth_token TEXT,                                -- Should be stored hashed
    date_created TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    date_updated TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_accounts_parent_sid ON accounts(parent_sid);
CREATE INDEX IF NOT EXISTS idx_accounts_auth_token ON accounts(auth_token); -- If lookups by token are done

-- Assuming the function update_modified_column() was created by 001_create_calls_table.sql
-- If not, or if running this independently, the function needs to be defined:
/*
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.date_updated = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';
*/

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
