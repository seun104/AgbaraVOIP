CREATE TABLE IF NOT EXISTS conference_participants (
    id BIGSERIAL PRIMARY KEY,
    sid VARCHAR(64) UNIQUE NOT NULL, -- e.g., CP...
    conference_sid VARCHAR(64) NOT NULL REFERENCES conferences(sid) ON DELETE CASCADE,
    call_sid VARCHAR(64) NOT NULL UNIQUE, -- Agbara Call SID of the participant, unique constraint implies a call can only be in one conference at a time as a participant.
    account_sid VARCHAR(64) NOT NULL REFERENCES accounts(sid) ON DELETE CASCADE, -- Denormalized for easier lookup
    is_muted BOOLEAN NOT NULL DEFAULT FALSE,
    is_moderator BOOLEAN NOT NULL DEFAULT FALSE,
    join_time TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT (NOW() AT TIME ZONE 'UTC'),
    leave_time TIMESTAMP WITHOUT TIME ZONE -- Nullable, set when participant leaves
    -- status VARCHAR(32) -- e.g., 'joined', 'left', 'talking', 'muted' (can be dynamic, might not store)
);

CREATE INDEX IF NOT EXISTS idx_conference_participants_conference_sid ON conference_participants(conference_sid);
CREATE INDEX IF NOT EXISTS idx_conference_participants_call_sid ON conference_participants(call_sid);
CREATE INDEX IF NOT EXISTS idx_conference_participants_account_sid ON conference_participants(account_sid);
-- No updated_at trigger for participants as they are usually append-once then updated for leave_time.
-- If other fields become mutable (e.g. is_muted, is_moderator can change during call), add the trigger.
-- For now, assuming these are set on join and don't change, or changes are not tracked via updated_at.
