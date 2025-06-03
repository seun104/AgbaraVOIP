DROP TRIGGER IF EXISTS update_freeswitch_servers_updated_at ON freeswitch_servers;

-- Do not drop update_updated_at_column function here if it's shared by other tables.
-- Only drop if it was uniquely created for this table and not shared.
-- The up script uses "CREATE FUNCTION IF NOT EXISTS" logic via DO block,
-- so the function is likely shared and should not be dropped here.
-- DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS freeswitch_servers;
