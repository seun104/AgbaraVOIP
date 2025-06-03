-- The 'conference_sid' column (type VARCHAR(64)) should already exist in the 'recordings' table
-- as per migration 000004_create_recordings_table.up.sql.
-- This migration adds the foreign key constraint now that the 'conferences' table is created.

-- It's good practice to ensure the column exists before trying to add a constraint to it,
-- though 000004 should have created it.
-- ALTER TABLE recordings ADD COLUMN IF NOT EXISTS conference_sid VARCHAR(64);

-- Add the foreign key constraint.
-- Wrap in DO block to allow conditional execution if needed, though direct is fine if order is guaranteed.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'recordings_conference_sid_fkey' AND table_name = 'recordings'
    ) THEN
        ALTER TABLE recordings
        ADD CONSTRAINT recordings_conference_sid_fkey
        FOREIGN KEY (conference_sid) REFERENCES conferences(sid) ON DELETE SET NULL;
        -- ON DELETE SET NULL: If a conference is deleted, recordings associated with it
        -- are kept but their conference_sid is set to NULL.
    END IF;
END $$;

-- Add an index if not already present from original recordings migration (000004 had it commented out)
CREATE INDEX IF NOT EXISTS idx_recordings_conference_sid ON recordings(conference_sid);
