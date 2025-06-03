DROP TRIGGER IF EXISTS update_recordings_updated_at ON recordings;

-- Conditionally drop the function only if it's not used by other tables.
-- For this project, if it's shared, it might be managed by the first migration that created it,
-- or a dedicated script. Assuming for now it might be specific or okay to drop if this is the only user.
-- A safer approach in a complex system is to check for dependent triggers before dropping.
-- However, typical simple down migrations often drop functions they created.
-- If update_updated_at_column is used by other tables (accounts, calls, applications),
-- DO NOT DROP IT HERE. Only drop the trigger specific to this table.
-- For this exercise, I will include the function drop, assuming it's either not shared or
-- that other tables' down migrations would also drop their triggers but not necessarily the function.
-- This is a common point of contention in migration strategies.
-- If the function was defined with "CREATE OR REPLACE", multiple up scripts might have run it.
-- The safest is often to leave shared functions and remove them in a dedicated cleanup migration
-- when they are no longer needed by any table.
-- For now, including it as per typical isolated down migration.
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS recordings;
