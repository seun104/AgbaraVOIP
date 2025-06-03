CREATE TABLE IF NOT EXISTS gateways (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL, -- e.g., GW...
    account_sid VARCHAR(64) NOT NULL REFERENCES accounts(sid) ON DELETE CASCADE,
    freeswitch_server_sid VARCHAR(64) REFERENCES freeswitch_servers(sid) ON DELETE SET NULL, -- Optional link
    friendly_name VARCHAR(255) NOT NULL,
    gateway_string TEXT NOT NULL, -- e.g., sofia/gateway/myprovider/
    codecs TEXT[], -- PostgreSQL text array for []string domain.Gateway.Codecs
    retry_count INT NOT NULL DEFAULT 0,
    timeout_seconds INT NOT NULL DEFAULT 30,
    routes JSONB, -- For domain.Gateway.Routes (GatewayRoutes map[string]interface{})
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX IF NOT EXISTS idx_gateways_account_sid ON gateways(account_sid);
CREATE INDEX IF NOT EXISTS idx_gateways_freeswitch_server_sid ON gateways(freeswitch_server_sid);
CREATE INDEX IF NOT EXISTS idx_gateways_is_enabled ON gateways(is_enabled);

-- Trigger for updated_at (assuming update_updated_at_column function already exists)
DO $$
BEGIN
    -- The function update_updated_at_column is assumed to be created by a previous migration (e.g., 000004 or 000005)
    -- Ensure it exists before creating trigger, or include its creation if this could be the first.
    -- For this project, 000004/000005/000009 all ensure it exists.

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_gateways_updated_at') THEN
        CREATE TRIGGER update_gateways_updated_at
        BEFORE UPDATE ON gateways
        FOR EACH ROW
        EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
