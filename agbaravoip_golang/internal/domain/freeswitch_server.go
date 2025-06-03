package domain

import (
	"time"
)

// FreeswitchServer represents a configured Freeswitch instance.
type FreeswitchServer struct {
	ID              int64     `db:"id" json:"-"`
	SID             string    `db:"sid" json:"sid"` // Public SID (e.g., FS...)
	Host            string    `db:"host" json:"host"`
	Port            int       `db:"port" json:"port"`
	Password        string    `db:"password" json:"-"` // Password for ESL, not exposed in JSON by default
	OutboundAddress string    `db:"outbound_address" json:"outbound_address,omitempty"` // e.g., IP:Port Freeswitch uses for outbound connections from this server
	IsActive        bool      `db:"is_active" json:"is_active"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}
