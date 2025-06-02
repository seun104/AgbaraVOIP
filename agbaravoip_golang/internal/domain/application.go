package domain

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Application represents a user-defined voice or SMS application.
// It defines URLs that AgbaraVOIP will request to get instructions (AgbaraXML or similar)
// when calls or messages for associated phone numbers arrive.
type Application struct {
	ID        uint   `gorm:"primaryKey"`
	SID       string `gorm:"type:varchar(64);uniqueIndex;not null"` // Publicly visible SID, e.g., APxxxxxxxx
	AccountSID string `gorm:"type:varchar(64);index;not null"`    // Belongs to an Account SID

	FriendlyName string `gorm:"type:varchar(255)"`

	// Voice Application Settings
	VoiceURL            string `gorm:"type:text"`
	VoiceMethod         string `gorm:"type:varchar(10);default:POST"` // e.g., GET, POST
	VoiceFallbackURL    string `gorm:"type:text"`
	VoiceFallbackMethod string `gorm:"type:varchar(10);default:POST"`

	// SMS Application Settings
	SmsURL            string `gorm:"type:text"`
	SmsMethod         string `gorm:"type:varchar(10);default:POST"`
	SmsFallbackURL    string `gorm:"type:text"`
	SmsFallbackMethod string `gorm:"type:varchar(10);default:POST"`

	// Common Status Callbacks
	StatusCallbackURL    string `gorm:"type:text"`
	StatusCallbackMethod string `gorm:"type:varchar(10);default:POST"`
	// HeartbeatURL         string `gorm:"type:text"` // As per original design, can add if needed

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft deletes
}

// BeforeCreate is a GORM hook that runs before a new record is created.
func (app *Application) BeforeCreate(tx *gorm.DB) (err error) {
	if app.SID == "" {
		app.SID = "AP" + uuid.NewString()
	}
	// Default methods if not provided
	if app.VoiceMethod == "" {
		app.VoiceMethod = "POST"
	}
	if app.VoiceFallbackMethod == "" {
		app.VoiceFallbackMethod = "POST"
	}
	if app.SmsMethod == "" {
		app.SmsMethod = "POST"
	}
	if app.SmsFallbackMethod == "" {
		app.SmsFallbackMethod = "POST"
	}
	if app.StatusCallbackMethod == "" {
		app.StatusCallbackMethod = "POST"
	}
	return
}

