DROP TRIGGER IF EXISTS update_accounts_updated_at ON accounts;
DROP FUNCTION IF EXISTS update_updated_at_column(); -- Be cautious if this function is shared

DROP TABLE IF EXISTS accounts;

DROP TYPE IF EXISTS account_status;
DROP TYPE IF EXISTS account_type;
