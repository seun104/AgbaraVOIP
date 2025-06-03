package domain

import (
	"time"
)

// Recording represents a call or conference recording.
type Recording struct {
	ID               int64     `db:"id" json:"-"` // Internal DB ID
	SID              string    `db:"sid" json:"sid"` // Public SID (e.g., RE sanitaria)
	AccountSID       string    `db:"account_sid" json:"account_sid"`
	CallSID          *string   `db:"call_sid" json:"call_sid,omitempty"`     // Nullable, if it's a call recording
	ConferenceSID    *string   `db:"conference_sid" json:"conference_sid,omitempty"` // Nullable, if it's a conference recording
	DurationSeconds  uint32    `db:"duration_seconds" json:"duration_seconds"`
	FilePath         string    `db:"file_path" json:"file_path"` // Path or URL to the recording file
	Format           string    `db:"format" json:"format"`       // e.g., "wav", "mp3"
	SizeBytes        int64     `db:"size_bytes" json:"size_bytes"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
	// Status string `db:"status" json:"status"` // Optional: e.g., "processing", "available", "failed"
}
