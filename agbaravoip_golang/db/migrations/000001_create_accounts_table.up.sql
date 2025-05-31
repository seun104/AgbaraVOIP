CREATE TYPE account_type AS ENUM ('trial', 'full');
CREATE TYPE account_status AS ENUM ('active', 'suspended', 'closed');

CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL,
    parent_sid VARCHAR(64),
    friendly_name VARCHAR(255),
    phone_number VARCHAR(64),
    auth_token VARCHAR(255) NOT NULL, -- Store hashed token
    type account_type DEFAULT 'trial',
    status account_status DEFAULT 'active',
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC'),
    CONSTRAINT fk_parent_account
        FOREIGN KEY(parent_sid)
        REFERENCES accounts(sid)
        ON DELETE SET NULL -- Or RESTRICT, depending on desired behavior
);

CREATE INDEX IF NOT EXISTS idx_accounts_sid ON accounts(sid);
CREATE INDEX IF NOT EXISTS idx_accounts_parent_sid ON accounts(parent_sid);
CREATE INDEX IF NOT EXISTS idx_accounts_auth_token ON accounts(auth_token); -- For login

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW() AT TIME ZONE 'UTC';
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_accounts_updated_at
BEFORE UPDATE ON accounts
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
