CREATE TABLE IF NOT EXISTS sms_messages (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL, -- e.g., SM...
    account_sid VARCHAR(64) NOT NULL REFERENCES accounts(sid) ON DELETE CASCADE,
    msg_to VARCHAR(64) NOT NULL, -- Recipient
    msg_from VARCHAR(64) NOT NULL, -- Sender
    body TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued', -- Using ENUMs defined in domain: queued, sent, failed, delivered, undelivered, receiving, received
    direction VARCHAR(32) NOT NULL, -- outbound, inbound
    price VARCHAR(16), -- Nullable, e.g., "0.00500"
    price_unit VARCHAR(8), -- Nullable, e.g., "USD"
    error_code INT, -- Nullable, gateway-specific error code
    error_message TEXT, -- Nullable
    gateway_message_sid VARCHAR(255), -- Nullable, SID from the external SMS gateway
    sent_at TIMESTAMP WITHOUT TIME ZONE, -- Nullable, time SMS was sent by gateway
    delivered_at TIMESTAMP WITHOUT TIME ZONE, -- Nullable, time SMS was delivered to handset
    created_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    updated_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC')
);

CREATE INDEX IF NOT EXISTS idx_sms_messages_account_sid ON sms_messages(account_sid);
CREATE INDEX IF NOT EXISTS idx_sms_messages_status ON sms_messages(status);
CREATE INDEX IF NOT EXISTS idx_sms_messages_direction ON sms_messages(direction);
CREATE INDEX IF NOT EXISTS idx_sms_messages_msg_to ON sms_messages(msg_to); -- For looking up inbound messages
CREATE INDEX IF NOT EXISTS idx_sms_messages_msg_from ON sms_messages(msg_from);
CREATE INDEX IF NOT EXISTS idx_sms_messages_gateway_message_sid ON sms_messages(gateway_message_sid);
CREATE INDEX IF NOT EXISTS idx_sms_messages_created_at ON sms_messages(created_at);


-- Trigger for updated_at (assuming update_updated_at_column function already exists from previous migrations)
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

    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_sms_messages_updated_at') THEN
        CREATE TRIGGER update_sms_messages_updated_at
        BEFORE UPDATE ON sms_messages
        FOR EACH ROW
        EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
