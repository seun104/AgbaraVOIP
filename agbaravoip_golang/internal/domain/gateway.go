package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
	// "github.com/lib/pq" // Not used if GORM handles []string or if using JSONB for codecs as well
)

// GatewayRoutes can be stored as JSONB in PostgreSQL.
// This custom type helps handle JSON marshalling/unmarshalling.
type GatewayRoutes map[string]interface{} // Flexible, could be more structured

func (gr GatewayRoutes) Value() (driver.Value, error) {
	if gr == nil {
		return nil, nil
	}
	return json.Marshal(gr)
}

func (gr *GatewayRoutes) Scan(value interface{}) error {
	if value == nil {
		*gr = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for GatewayRoutes")
	}
	return json.Unmarshal(b, gr)
}


// Gateway represents an outbound VOIP gateway configuration.
type Gateway struct {
	ID                 int64         `db:"id" json:"-"`
	SID                string        `db:"sid" json:"sid"` // Public SID (e.g., GW...)
	AccountSID         string        `db:"account_sid" json:"account_sid"` // Gateways are typically account-specific
	FreeswitchServerSID *string       `db:"freeswitch_server_sid" json:"freeswitch_server_sid,omitempty"` // Optional: Link to a specific FS server
	FriendlyName       string        `db:"friendly_name" json:"friendly_name"`
	GatewayString      string        `db:"gateway_string" json:"gateway_string"` // e.g., sofia/gateway/myprovider/
	Codecs             []string      `db:"codecs" json:"codecs,omitempty"`       // Stored as text[] or jsonb in PG. GORM handles []string.
	RetryCount         int           `db:"retry_count" json:"retry_count"`
	TimeoutSeconds     int           `db:"timeout_seconds" json:"timeout_seconds"`
	Routes             GatewayRoutes `db:"routes" json:"routes,omitempty"` // Stored as JSONB
	IsEnabled          bool          `db:"is_enabled" json:"is_enabled"`
	CreatedAt          time.Time     `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time     `db:"updated_at" json:"updated_at"`
}
