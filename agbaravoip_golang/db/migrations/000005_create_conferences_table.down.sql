DROP TRIGGER IF EXISTS update_conferences_updated_at ON conferences;

-- Only drop the function if this migration was the sole creator
-- and no other tables depend on it. This is often hard to guarantee.
-- A common strategy is to leave shared functions unless a specific migration
-- is tasked with removing a function known to be unused.
-- For this exercise, we'll comment out the function drop for safety,
-- assuming it might be shared (e.g. with 'recordings' or 'accounts' tables).
-- DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS conferences;
