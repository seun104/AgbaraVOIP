CREATE TABLE IF NOT EXISTS freeswitch_servers (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL, -- e.g., FS...
    host VARCHAR(255) NOT NULL,
    port INT NOT NULL DEFAULT 5060,
    password TEXT NOT NULL, -- Store securely (e.g., encrypted or in a vault, TEXT for schema)
    outbound_address VARCHAR(255), -- Optional: Specific IP:Port FS uses for outbound calls from this instance
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX IF NOT EXISTS idx_freeswitch_servers_host_port ON freeswitch_servers(host, port);
CREATE INDEX IF NOT EXISTS idx_freeswitch_servers_is_active ON freeswitch_servers(is_active);

-- Trigger for updated_at (assuming update_updated_at_column function already exists)
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

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_freeswitch_servers_updated_at') THEN
        CREATE TRIGGER update_freeswitch_servers_updated_at
        BEFORE UPDATE ON freeswitch_servers
        FOR EACH ROW
        EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
