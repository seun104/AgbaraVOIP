ALTER TABLE recordings
DROP CONSTRAINT IF EXISTS recordings_conference_sid_fkey;

-- The index idx_recordings_conference_sid might have been created by migration 000004
-- or by this migration (000007). Dropping it here is generally safe
-- as the corresponding 'up' ensures it exists.
-- If 000004 definitely created it, this line could be omitted here,
-- and its own down migration (000004...down.sql) would be responsible.
-- However, since 000004 had it commented out and this one has CREATE INDEX IF NOT EXISTS,
-- it's appropriate to drop it here.
DROP INDEX IF EXISTS idx_recordings_conference_sid;

-- Do not drop the 'conference_sid' column itself, as it was defined in migration 000004,
-- not added by this migration (000007). This migration only adds the constraint.
