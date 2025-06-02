-- +migrate Down
DROP TRIGGER IF EXISTS set_calls_updated_at ON calls;
DROP TABLE IF EXISTS calls;
DROP TYPE IF EXISTS call_status;
DROP TYPE IF EXISTS call_direction;

