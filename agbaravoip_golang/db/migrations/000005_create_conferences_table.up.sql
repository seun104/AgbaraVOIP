CREATE TABLE IF NOT EXISTS conferences (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL, -- e.g., CF...
    account_sid VARCHAR(64) NOT NULL REFERENCES accounts(sid) ON DELETE CASCADE,
    friendly_name VARCHAR(255) NOT NULL, -- Usually the RoomName from XML
    status VARCHAR(32) NOT NULL DEFAULT 'initializing', -- e.g., initializing, in-progress, completed
    start_time TIMESTAMP WITHOUT TIME ZONE, -- Nullable, set when first participant joins or explicitly started
    end_time TIMESTAMP WITHOUT TIME ZONE,   -- Nullable, set when conference ends
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX IF NOT EXISTS idx_conferences_account_sid ON conferences(account_sid);
CREATE INDEX IF NOT EXISTS idx_conferences_friendly_name ON conferences(friendly_name);
CREATE INDEX IF NOT EXISTS idx_conferences_status ON conferences(status);

-- Trigger for updated_at (assuming update_updated_at_column function already exists from previous migrations)
-- If the function does not exist, it should be created here as in recordings migration (000004).
-- The DO block attempts to create the trigger only if it doesn't exist.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column') THEN
        CREATE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $func$
        BEGIN
           NEW.updated_at = (NOW() AT TIME ZONE 'UTC');
           RETURN NEW;
        END;
        $func$ language 'plpgsql';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_conferences_updated_at') THEN
        CREATE TRIGGER update_conferences_updated_at
        BEFORE UPDATE ON conferences
        FOR EACH ROW
        EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
