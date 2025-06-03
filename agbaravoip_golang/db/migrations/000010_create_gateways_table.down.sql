DROP TRIGGER IF EXISTS update_gateways_updated_at ON gateways;

-- Do not drop update_updated_at_column function here as it's shared.

DROP TABLE IF EXISTS gateways;
