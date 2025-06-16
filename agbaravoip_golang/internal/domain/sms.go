package domain

import (
	"database/sql" // For sql.NullString, sql.NullInt32, sql.NullTime
	"time"
)

// SMSDirection defines the direction of an SMS message.
type SMSDirection string

const (
	SMSDirectionOutbound    SMSDirection = "outbound"     // SMS sent from the system (e.g. via AgbaraXML <Sms>)
	SMSDirectionInbound     SMSDirection = "inbound"      // SMS received by the system
	SMSDirectionOutboundAPI SMSDirection = "outbound-api" // SMS sent via API call
)

// SMSStatus defines the status of an SMS message.
type SMSStatus string

const (
	SMSStatusQueued      SMSStatus = "queued"      // Message is queued for sending
	SMSStatusSent        SMSStatus = "sent"        // Message has been successfully sent to the gateway
	SMSStatusFailed      SMSStatus = "failed"      // Message failed to send from our system or gateway
	SMSStatusDelivered   SMSStatus = "delivered"   // Message was successfully delivered to the handset (requires DLR)
	SMSStatusUndelivered SMSStatus = "undelivered" // Message could not be delivered to the handset (requires DLR)
	SMSStatusReceiving   SMSStatus = "receiving"   // For inbound, initial state
	SMSStatusReceived    SMSStatus = "received"    // For inbound, successfully processed
)

// SMSMessage represents an SMS message in the database.
type SMSMessage struct {
	ID                int64          `db:"id" json:"-"`
	SID               string         `db:"sid" json:"sid"` // Public SID (e.g., SM...)
	AccountSID        string         `db:"account_sid" json:"account_sid"`
	To                string         `db:"msg_to" json:"to"`     // Using msg_to to avoid SQL keyword conflict
	From              string         `db:"msg_from" json:"from"`   // Using msg_from
	Body              string         `db:"body" json:"body"`
	Status            SMSStatus      `db:"status" json:"status"`
	Direction         SMSDirection   `db:"direction" json:"direction"`
	Price             sql.NullString `db:"price" json:"price,omitempty"` // Nullable, e.g., "0.00500"
	PriceUnit         sql.NullString `db:"price_unit" json:"price_unit,omitempty"` // Nullable, e.g., "USD"
	ErrorCode         sql.NullInt32  `db:"error_code" json:"error_code,omitempty"` // Nullable, gateway-specific error code
	ErrorMessage      sql.NullString `db:"error_message" json:"error_message,omitempty"` // Nullable
	GatewayMessageSID sql.NullString `db:"gateway_message_sid" json:"gateway_message_sid,omitempty"` // SID from the external SMS gateway
	SentAt            sql.NullTime   `db:"sent_at" json:"sent_at,omitempty"`     // Time SMS was sent by gateway
	DeliveredAt       sql.NullTime   `db:"delivered_at" json:"delivered_at,omitempty"` // Time SMS was delivered to handset
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}
