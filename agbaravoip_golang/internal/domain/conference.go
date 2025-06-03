package domain

import (
	"time"
)

// ConferenceStatus defines the possible statuses of a conference room.
type ConferenceStatus string

const (
	ConferenceStatusInit       ConferenceStatus = "initializing"
	ConferenceStatusInProgress ConferenceStatus = "in-progress"
	ConferenceStatusCompleted  ConferenceStatus = "completed"
)

// Conference represents a conference room session in the database.
type Conference struct {
	ID             int64            `db:"id" json:"-"`
	SID            string           `db:"sid" json:"sid"` // Public SID (e.g., CF...)
	AccountSID     string           `db:"account_sid" json:"account_sid"`
	FriendlyName   string           `db:"friendly_name" json:"friendly_name"` // Usually same as RoomName from XML
	Status         ConferenceStatus `db:"status" json:"status"`
	StartTime      *time.Time       `db:"start_time" json:"start_time,omitempty"` // Nullable
	EndTime        *time.Time       `db:"end_time" json:"end_time,omitempty"`   // Nullable
	CreatedAt      time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time        `db:"updated_at" json:"updated_at"`
	// MaxParticipants int           `db:"max_participants" json:"max_participants"` // From ConferenceElement
	// EndReason string `db:"end_reason" json:"end_reason,omitempty"` // e.g., last_member_left, api_request
}

// ConferenceParticipant represents a participant in a conference.
type ConferenceParticipant struct {
	ID            int64      `db:"id" json:"-"`
	SID           string     `db:"sid" json:"sid"` // Public SID (e.g., CP...)
	ConferenceSID string     `db:"conference_sid" json:"conference_sid"`
	CallSID       string     `db:"call_sid" json:"call_sid"` // Agbara Call SID of the participant
	AccountSID    string     `db:"account_sid" json:"account_sid"`
	IsMuted       bool       `db:"is_muted" json:"is_muted"`
	IsModerator   bool       `db:"is_moderator" json:"is_moderator"` // Or 'Coach' in FS terms
	JoinTime      time.Time  `db:"join_time" json:"join_time"`
	LeaveTime     *time.Time `db:"leave_time" json:"leave_time,omitempty"` // Nullable
	// Status string `db:"status" json:"status"` // e.g., 'joined', 'left', 'talking', 'muted'
	// HoldMusic bool `db:"hold_music" json:"hold_music"` // If participant is hearing MoH (e.g. waiting for conference to start)
}
