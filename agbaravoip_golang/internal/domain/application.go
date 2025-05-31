package domain

import "time"

// Application represents a user-defined voice/SMS application.
// It contains URLs that AgbaraVOIP will request to fetch AgbaraXML
// instructions or send status updates.
type Application struct {
	ID                      int64     `json:"-" gorm:"primaryKey"`
	SID                     string    `json:"sid" gorm:"type:varchar(64);uniqueIndex"`
	AccountSID              string    `json:"account_sid" gorm:"type:varchar(64);index"`
	FriendlyName            string    `json:"friendly_name" gorm:"type:varchar(255)"`
	VoiceURL                string    `json:"voice_url,omitempty" gorm:"type:text"`
	VoiceMethod             string    `json:"voice_method,omitempty" gorm:"type:varchar(10)"`
	VoiceFallbackURL        string    `json:"voice_fallback_url,omitempty" gorm:"type:text"`
	VoiceFallbackMethod     string    `json:"voice_fallback_method,omitempty" gorm:"type:varchar(10)"`
	StatusCallbackURL       string    `json:"status_callback_url,omitempty" gorm:"type:text"`
	StatusCallbackMethod    string    `json:"status_callback_method,omitempty" gorm:"type:varchar(10)"`
	SmsURL                  string    `json:"sms_url,omitempty" gorm:"type:text"`
	SmsMethod               string    `json:"sms_method,omitempty" gorm:"type:varchar(10)"`
	SmsFallbackURL          string    `json:"sms_fallback_url,omitempty" gorm:"type:text"`
	SmsFallbackMethod       string    `json:"sms_fallback_method,omitempty" gorm:"type:varchar(10)"`
	SmsStatusCallbackURL    string    `json:"sms_status_callback_url,omitempty" gorm:"type:text"` // As per plan
	SmsStatusCallbackMethod string    `json:"sms_status_callback_method,omitempty" gorm:"type:varchar(10)"` // As per plan
	HeartbeatURL            string    `json:"heartbeat_url,omitempty" gorm:"type:text"` // As per plan
	CreatedAt               time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt               time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	// Foreign key relationship (optional, GORM can infer or you can be explicit)
	// Account Account `json:"-" gorm:"foreignKey:AccountSID;references:SID"`
}
