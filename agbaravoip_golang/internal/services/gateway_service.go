package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/utils" // For SID generation
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // For Preload
)

// ErrGatewayNotFound indicates that a Gateway record was not found.
var ErrGatewayNotFound = errors.New("gateway not found")
// ErrGatewayValidation indicates a validation error for Gateway data.
var ErrGatewayValidation = errors.New("gateway validation failed")
// ErrGatewayAlreadyExists indicates a Gateway with the same unique identifier already exists.
var ErrGatewayAlreadyExists = errors.New("gateway already exists") // If unique constraints are added

type gatewayService struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewGatewayService creates a new IGatewayService.
func NewGatewayService(db *gorm.DB, logger *logrus.Logger) IGatewayService {
	return &gatewayService{
		db:     db,
		logger: logger.WithField("service", "gateway"),
	}
}

func (s *gatewayService) CreateGateway(
	accountSID string,
	fsServerSID *string,
	friendlyName string,
	gatewayString string,
	codecs []string,
	retryCount *int,
	timeoutSeconds *int,
	routes domain.GatewayRoutes,
	isEnabled *bool,
) (*domain.Gateway, error) {
	s.logger.Infof("Attempting to create gateway '%s' for account SID: %s", friendlyName, accountSID)

	if accountSID == "" || friendlyName == "" || gatewayString == "" {
		return nil, fmt.Errorf("%w: account_sid, friendly_name, and gateway_string are required", ErrGatewayValidation)
	}

	actualIsEnabled := true // Default to true
	if isEnabled != nil {
		actualIsEnabled = *isEnabled
	}
	actualRetryCount := 0
	if retryCount != nil {
		actualRetryCount = *retryCount
	}
	actualTimeoutSeconds := 60 // Default timeout
	if timeoutSeconds != nil {
		actualTimeoutSeconds = *timeoutSeconds
	}


	gw := &domain.Gateway{
		SID:                 utils.GenerateSID("GW"), // Assuming "GW" prefix
		AccountSID:          accountSID,
		FreeswitchServerSID: fsServerSID,
		FriendlyName:        friendlyName,
		GatewayString:       gatewayString,
		Codecs:              codecs, // GORM handles []string to text[] or jsonb
		RetryCount:          actualRetryCount,
		TimeoutSeconds:      actualTimeoutSeconds,
		Routes:              routes,
		IsEnabled:           actualIsEnabled,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	if err := s.db.Create(gw).Error; err != nil {
		// TODO: Check for unique constraint errors (e.g. account_sid + friendly_name)
		s.logger.Errorf("Failed to create gateway %s in DB: %v", friendlyName, err)
		return nil, fmt.Errorf("could not create gateway: %w", err)
	}

	s.logger.Infof("Gateway created successfully with SID: %s", gw.SID)
	return gw, nil
}

// GetGatewayBySID retrieves a gateway for a specific account.
func (s *gatewayService) GetGatewayBySID(accountSID string, sid string) (*domain.Gateway, error) {
	s.logger.Infof("Attempting to retrieve gateway with SID: %s for account SID: %s", sid, accountSID)
	var gw domain.Gateway
	if err := s.db.Where("account_sid = ? AND sid = ?", accountSID, sid).First(&gw).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warnf("Gateway with SID %s for account %s not found", sid, accountSID)
			return nil, ErrGatewayNotFound
		}
		s.logger.Errorf("Error retrieving gateway %s for account %s from DB: %v", sid, accountSID, err)
		return nil, err
	}
	return &gw, nil
}

// ListGateways lists gateways for a specific account.
func (s *gatewayService) ListGateways(accountSID string, filters map[string]interface{}) ([]*domain.Gateway, error) {
	s.logger.Infof("Listing gateways for account SID: %s", accountSID)
	var gateways []*domain.Gateway
	query := s.db.Model(&domain.Gateway{}).Where("account_sid = ?", accountSID)

	if friendlyName, ok := filters["friendly_name"].(string); ok && friendlyName != "" {
		query = query.Where("friendly_name LIKE ?", "%"+friendlyName+"%")
	}
	if isEnabled, ok := filters["is_enabled"].(bool); ok {
		query = query.Where("is_enabled = ?", isEnabled)
	}
	// Add other filters as needed

	if err := query.Order("created_at DESC").Find(&gateways).Error; err != nil {
		s.logger.Errorf("Error listing gateways for account %s from DB: %v", accountSID, err)
		return nil, err
	}
	return gateways, nil
}

// UpdateGateway updates a gateway for a specific account.
func (s *gatewayService) UpdateGateway(accountSID string, sid string, updates map[string]interface{}) (*domain.Gateway, error) {
	s.logger.Infof("Attempting to update gateway with SID: %s for account SID: %s", sid, accountSID)

	// Ensure gateway exists and belongs to the account
	gw, err := s.GetGatewayBySID(accountSID, sid)
	if err != nil {
		return nil, err // Handles ErrGatewayNotFound
	}

	// Ensure `updated_at` is set
	updates["updated_at"] = time.Now().UTC()

	if err := s.db.Model(&gw).Where("account_sid = ? AND sid = ?", accountSID, sid).Updates(updates).Error; err != nil {
		s.logger.Errorf("Failed to update gateway %s for account %s in DB: %v", sid, accountSID, err)
		// TODO: Check for unique constraint errors
		return nil, err
	}

    // Fetch the updated record to return it
    updatedGw, err := s.GetGatewayBySID(accountSID, sid)
    if err != nil {
        s.logger.Errorf("Failed to retrieve updated gateway %s for account %s: %v", sid, accountSID, err)
        return nil, err
    }

	s.logger.Infof("Gateway %s for account %s updated successfully", sid, accountSID)
	return updatedGw, nil
}

// DeleteGateway deletes a gateway for a specific account.
func (s *gatewayService) DeleteGateway(accountSID string, sid string) error {
	s.logger.Infof("Attempting to delete gateway with SID: %s for account SID: %s", sid, accountSID)

	// Ensure gateway exists and belongs to the account before deleting
	_, err := s.GetGatewayBySID(accountSID, sid)
	if err != nil {
		return err // Handles ErrGatewayNotFound
	}

	if err := s.db.Where("account_sid = ? AND sid = ?", accountSID, sid).Delete(&domain.Gateway{}).Error; err != nil {
		s.logger.Errorf("Failed to delete gateway %s for account %s from DB: %v", sid, accountSID, err)
		return err
	}

	s.logger.Infof("Gateway %s for account %s deleted successfully", sid, accountSID)
	return nil
}


// --- Global/Admin Methods ---

func (s *gatewayService) ListGlobalGateways(filters map[string]interface{}) ([]*domain.Gateway, error) {
	s.logger.Info("Listing all global gateways")
	var gateways []*domain.Gateway
	query := s.db.Model(&domain.Gateway{})

	if friendlyName, ok := filters["friendly_name"].(string); ok && friendlyName != "" {
		query = query.Where("friendly_name LIKE ?", "%"+friendlyName+"%")
	}
    if accountSID, ok := filters["account_sid"].(string); ok && accountSID != "" {
		query = query.Where("account_sid = ?", accountSID)
	}
	if isEnabled, ok := filters["is_enabled"].(bool); ok {
		query = query.Where("is_enabled = ?", isEnabled)
	}
	// Add other filters as needed

	if err := query.Order("account_sid ASC, created_at DESC").Find(&gateways).Error; err != nil {
		s.logger.Errorf("Error listing all global gateways from DB: %v", err)
		return nil, err
	}
	return gateways, nil
}

func (s *gatewayService) GetGlobalGatewayBySID(sid string) (*domain.Gateway, error) {
	s.logger.Infof("Attempting to retrieve global gateway with SID: %s", sid)
	var gw domain.Gateway
	// Preload Account if you want to return account details along with the gateway
	// query := s.db.Preload("Account")
	// For GORM to Preload, you need to define the relationship in domain.Gateway struct
	// e.g. Account domain.Account `gorm:"foreignKey:AccountSID;references:SID"`
	// For now, not preloading.
	if err := s.db.Where("sid = ?", sid).First(&gw).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warnf("Global gateway with SID %s not found", sid)
			return nil, ErrGatewayNotFound
		}
		s.logger.Errorf("Error retrieving global gateway %s from DB: %v", sid, err)
		return nil, err
	}
	return &gw, nil
}

func (s *gatewayService) UpdateGlobalGateway(sid string, updates map[string]interface{}) (*domain.Gateway, error) {
	s.logger.Infof("Attempting to update global gateway with SID: %s", sid)

	// Ensure gateway exists
	gw, err := s.GetGlobalGatewayBySID(sid)
	if err != nil {
		return nil, err // Handles ErrGatewayNotFound
	}

	// Ensure `updated_at` is set
	updates["updated_at"] = time.Now().UTC()
    // AccountSID cannot be changed via this global update method directly for safety.
    // If account_sid is in updates, it should be carefully considered or disallowed.
    if _, ok := updates["account_sid"]; ok {
        s.logger.Warnf("Attempt to update account_sid for gateway %s via global update is ignored.", sid)
        delete(updates, "account_sid")
    }


	if err := s.db.Model(&gw).Where("sid = ?", sid).Updates(updates).Error; err != nil {
		s.logger.Errorf("Failed to update global gateway %s in DB: %v", sid, err)
		return nil, err
	}

    updatedGw, err := s.GetGlobalGatewayBySID(sid)
    if err != nil {
        s.logger.Errorf("Failed to retrieve updated global gateway %s: %v", sid, err)
        return nil, err
    }

	s.logger.Infof("Global gateway %s updated successfully", sid)
	return updatedGw, nil
}

func (s *gatewayService) DeleteGlobalGateway(sid string) error {
	s.logger.Infof("Attempting to delete global gateway with SID: %s", sid)

	_, err := s.GetGlobalGatewayBySID(sid)
	if err != nil {
		return err // Handles ErrGatewayNotFound
	}

	if err := s.db.Where("sid = ?", sid).Delete(&domain.Gateway{}).Error; err != nil {
		s.logger.Errorf("Failed to delete global gateway %s from DB: %v", sid, err)
		return err
	}

	s.logger.Infof("Global gateway %s deleted successfully", sid)
	return nil
}
