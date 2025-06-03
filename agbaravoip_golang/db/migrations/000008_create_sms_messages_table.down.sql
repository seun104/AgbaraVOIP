DROP TRIGGER IF EXISTS update_sms_messages_updated_at ON sms_messages;

-- As noted in previous migrations, the update_updated_at_column function
-- should ideally be managed separately if it's shared across many tables.
-- For this project, we've included its creation (if not exists) in each 'up' migration
-- that needs it. Dropping it here might affect other tables if this is the last
-- table using it and its down migration is run.
-- For simplicity, we are not dropping the function here, assuming it might be used by other tables.
-- A more robust system might have a dependency count or a dedicated migration for shared functions.
-- DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS sms_messages;
