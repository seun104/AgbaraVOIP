package services

import (
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt" 
)

// Update AccountService interface
type AccountService interface {
	CreateMasterAccount(ctx context.Context, friendlyName string) (*models.Account, error)
	CreateSubAccount(ctx context.Context, friendlyName string, parentSid string) (*models.Account, error)
	GetAccount(ctx context.Context, accountSid string) (*models.Account, error)
	GetAccountBySidAndToken(ctx context.Context, accountSid string, authToken string) (*models.Account, error) 
	ListSubAccounts(ctx context.Context, parentSid string) ([]*models.Account, error)
	ChangeAccountStatus(ctx context.Context, accountSid string, status models.AccountStatus) (*models.Account, error)
	ChangeAccountType(ctx context.Context, accountSid string, accType models.AccountType) (*models.Account, error)
    GenerateAuthToken(ctx context.Context, accountSid string) (string, error)
    UpdateAccountSettings(ctx context.Context, accountSid string, settingsRequest *models.UpdateAccountSettingsRequest) (*models.Account, error) // <-- NEW METHOD
}

// PostgresAccountService (struct definition remains the same)
type PostgresAccountService struct {
	db *sql.DB
}

// NewPostgresAccountService (constructor remains the same)
func NewPostgresAccountService(db *sql.DB) *PostgresAccountService {
	return &PostgresAccountService{db: db}
}

// hashAuthToken (already defined)
func hashAuthToken(token string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil { return "", err }
	return string(hashedBytes), nil
}

// Update scanAccount helper
func scanAccount(scanner interface{ Scan(...interface{}) error }) (*models.Account, error) {
	acc := &models.Account{}
	var parentSid, phoneNumber, defaultGateway, gatewayScript sql.NullString

	// Ensure AuthToken is scanned, even if it's json:"-" in the model for responses.
	// It's needed for GetAccountBySidAndToken.
	err := scanner.Scan(
		&acc.Sid, &parentSid, &acc.FriendlyName, &phoneNumber,
		&acc.Type, &acc.Status, &acc.AuthToken, 
		&acc.DateCreated, &acc.DateUpdated,
		&defaultGateway, &gatewayScript, // Scan new fields
	)
	if err != nil {
		return nil, err
	}

	acc.ParentSid = parentSid.String 
	acc.PhoneNumber = phoneNumber.String
	acc.DefaultOutboundGateway = defaultGateway.String
	acc.GatewaySelectionScript = gatewayScript.String
	return acc, nil
}

// Update CreateMasterAccount to initialize new fields
func (s *PostgresAccountService) CreateMasterAccount(ctx context.Context, friendlyName string) (*models.Account, error) {
	acc := &models.Account{
		Sid:          "AC" + uuid.NewString(),
		FriendlyName: friendlyName,
		Type:         models.AccountTypeTrial,  
		Status:       models.AccountStatusActive, 
		DateCreated:  time.Now().UTC(),
		DateUpdated:  time.Now().UTC(),
        // Initialize new fields as empty or default for DB (NULL)
        DefaultOutboundGateway: "",
        GatewaySelectionScript: "",
	}
	rawToken := uuid.NewString() 
	hashedToken, err := hashAuthToken(rawToken)
	if err != nil { return nil, fmt.Errorf("failed to hash auth token: %w", err) }
	acc.AuthToken = hashedToken 

	query := `
		INSERT INTO accounts (sid, friendly_name, type, status, auth_token, date_created, date_updated, 
		                      parent_sid, phone_number, default_outbound_gateway, gateway_selection_script)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL, NULL, NULL)
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
		          date_created, date_updated, default_outbound_gateway, gateway_selection_script;
	`
	row := s.db.QueryRowContext(ctx, query,
		acc.Sid, acc.FriendlyName, acc.Type, acc.Status, acc.AuthToken, acc.DateCreated, acc.DateUpdated,
	)
	return scanAccount(row)
}

// Update CreateSubAccount to initialize new fields
func (s *PostgresAccountService) CreateSubAccount(ctx context.Context, friendlyName string, parentSid string) (*models.Account, error) {
	_, err := s.GetAccount(ctx, parentSid) // Verify parent
	if err != nil { return nil, fmt.Errorf("parent account with SID %s not found: %w", parentSid, err) }

	acc := &models.Account{
		Sid:          "AC" + uuid.NewString(),
		ParentSid:    parentSid,
		FriendlyName: friendlyName,
		Type:         models.AccountTypeTrial,
		Status:       models.AccountStatusActive,
		DateCreated:  time.Now().UTC(),
		DateUpdated:  time.Now().UTC(),
        DefaultOutboundGateway: "", // Initialize
        GatewaySelectionScript: "", // Initialize
	}
	rawToken := uuid.NewString()
	hashedToken, err := hashAuthToken(rawToken)
	if err != nil { return nil, fmt.Errorf("failed to hash auth token: %w", err) }
	acc.AuthToken = hashedToken

	query := `
		INSERT INTO accounts (sid, parent_sid, friendly_name, type, status, auth_token, date_created, date_updated, 
		                      phone_number, default_outbound_gateway, gateway_selection_script)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULL, NULL, NULL)
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
		          date_created, date_updated, default_outbound_gateway, gateway_selection_script;
	`
	row := s.db.QueryRowContext(ctx, query,
		acc.Sid, acc.ParentSid, acc.FriendlyName, acc.Type, acc.Status, acc.AuthToken, acc.DateCreated, acc.DateUpdated,
	)
	return scanAccount(row)
}

// Implement UpdateAccountSettings method
func (s *PostgresAccountService) UpdateAccountSettings(ctx context.Context, accountSid string, req *models.UpdateAccountSettingsRequest) (*models.Account, error) {
	acc, err := s.GetAccount(ctx, accountSid)
	if err != nil {
		return nil, err // Handles not found
	}

	// Apply updates only for fields provided in the request
	if req.FriendlyName != nil {
		acc.FriendlyName = *req.FriendlyName
	}
	if req.PhoneNumber != nil {
		acc.PhoneNumber = *req.PhoneNumber
	}
	if req.Type != nil {
		acc.Type = *req.Type
	}
	if req.DefaultOutboundGateway != nil {
		acc.DefaultOutboundGateway = *req.DefaultOutboundGateway
	}
	if req.GatewaySelectionScript != nil {
		acc.GatewaySelectionScript = *req.GatewaySelectionScript
	}
	acc.DateUpdated = time.Now().UTC()

	query := `
		UPDATE accounts SET 
			friendly_name = $1, phone_number = $2, type = $3, 
			default_outbound_gateway = $4, gateway_selection_script = $5, date_updated = $6
		WHERE sid = $7
		RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
		          date_created, date_updated, default_outbound_gateway, gateway_selection_script;
	`
	row := s.db.QueryRowContext(ctx, query,
		acc.FriendlyName, sql.NullString{String: acc.PhoneNumber, Valid: acc.PhoneNumber != ""}, acc.Type,
		sql.NullString{String: acc.DefaultOutboundGateway, Valid: acc.DefaultOutboundGateway != ""},
		sql.NullString{String: acc.GatewaySelectionScript, Valid: acc.GatewaySelectionScript != ""},
		acc.DateUpdated, acc.Sid,
	)
	return scanAccount(row)
}


// GetAccount, ListSubAccounts, ChangeAccountStatus, ChangeAccountType, GetAccountBySidAndToken, GenerateAuthToken
// should remain, but ensure they use the updated scanAccount if they weren't already.
// For brevity, only showing the new/modified methods and scanAccount.
// Worker should ensure the rest of the file pkg/services/account_service.go is maintained from its previous correct state,
// and that all methods fetching full account details use the updated scanAccount.

func (s *PostgresAccountService) GetAccount(ctx context.Context, accountSid string) (*models.Account, error) {
	query := `SELECT sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
	                 date_created, date_updated, default_outbound_gateway, gateway_selection_script 
	          FROM accounts WHERE sid = $1;`
	row := s.db.QueryRowContext(ctx, query, accountSid)
	account, err := scanAccount(row) // scanAccount is already updated
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("account with SID %s not found: %w", accountSid, err)
        }
        return nil, fmt.Errorf("failed to get account %s: %w", accountSid, err)
    }
    return account, nil
}

func (s *PostgresAccountService) ListSubAccounts(ctx context.Context, parentSid string) ([]*models.Account, error) {
	query := `SELECT sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
	                 date_created, date_updated, default_outbound_gateway, gateway_selection_script 
	          FROM accounts WHERE parent_sid = $1 ORDER BY date_created DESC;`
	rows, err := s.db.QueryContext(ctx, query, parentSid)
	if err != nil { return nil, fmt.Errorf("failed to query sub-accounts for parent SID %s: %w", parentSid, err) } // Added error context
	defer rows.Close()
	var accounts []*models.Account
	for rows.Next() {
		account, scanErr := scanAccount(rows) // scanAccount is already updated
		if scanErr != nil { return nil, fmt.Errorf("failed to scan sub-account row: %w", scanErr) } // Added error context
		accounts = append(accounts, account)
	}
	if err = rows.Err(); err != nil { return nil, fmt.Errorf("error iterating sub-account rows for parent SID %s: %w", parentSid, err) } // Added error context
    if accounts == nil { accounts = []*models.Account{} }
	return accounts, nil
}

func (s *PostgresAccountService) ChangeAccountStatus(ctx context.Context, accountSid string, status models.AccountStatus) (*models.Account, error) {
	query := `UPDATE accounts SET status = $1, date_updated = $2 WHERE sid = $3
	          RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
	                    date_created, date_updated, default_outbound_gateway, gateway_selection_script;`
	row := s.db.QueryRowContext(ctx, query, status, time.Now().UTC(), accountSid)
	account, err := scanAccount(row) // scanAccount is already updated
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("account %s not found for status update: %w", accountSid, err)
        }
        return nil, fmt.Errorf("failed to update account status for SID %s: %w", accountSid, err) // Added error context
    }
    return account, nil
}

func (s *PostgresAccountService) ChangeAccountType(ctx context.Context, accountSid string, accType models.AccountType) (*models.Account, error) {
	query := `UPDATE accounts SET type = $1, date_updated = $2 WHERE sid = $3
	          RETURNING sid, parent_sid, friendly_name, phone_number, type, status, auth_token, 
	                    date_created, date_updated, default_outbound_gateway, gateway_selection_script;`
	row := s.db.QueryRowContext(ctx, query, accType, time.Now().UTC(), accountSid)
	account, err := scanAccount(row) // scanAccount is already updated
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("account %s not found for type update: %w", accountSid, err)
        }
        return nil, fmt.Errorf("failed to update account type for SID %s: %w", accountSid, err) // Added error context
    }
	return account, nil
}

func (s *PostgresAccountService) GetAccountBySidAndToken(ctx context.Context, accountSid string, plainToken string) (*models.Account, error) {
    account, err := s.GetAccount(ctx, accountSid) // GetAccount uses updated scanAccount
    if err != nil { return nil, err }
    err = bcrypt.CompareHashAndPassword([]byte(account.AuthToken), []byte(plainToken))
    if err != nil { return nil, fmt.Errorf("auth token validation failed: %w", err) }
    return account, nil 
}

func (s *PostgresAccountService) GenerateAuthToken(ctx context.Context, accountSid string) (string, error) {
    plainToken := uuid.NewString() + uuid.NewString() 
    hashedToken, err := hashAuthToken(plainToken)
    if err != nil { return "", fmt.Errorf("failed to hash new token: %w", err) }
    query := `UPDATE accounts SET auth_token = $1, date_updated = $2 WHERE sid = $3 RETURNING sid;`
    var updatedSid string // Not returning the full account, so scanAccount not needed here.
    err = s.db.QueryRowContext(ctx, query, hashedToken, time.Now().UTC(), accountSid).Scan(&updatedSid)
    if err != nil {
        if err == sql.ErrNoRows { return "", fmt.Errorf("account %s not found for token regeneration: %w", accountSid, err) }
        return "", fmt.Errorf("failed to update auth token in db for SID %s: %w", accountSid, err) // Added error context
    }
    return plainToken, nil
}
