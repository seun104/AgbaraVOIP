package api

import "github.com/user/agbaravoip_golang/internal/domain"

// CreateAccountRequest defines the request payload for creating an account.
type CreateAccountRequest struct {
	FriendlyName string `json:"friendly_name"`
	AuthToken    string `json:"auth_token" binding:"required"`
}

// UpdateAccountRequest defines the request payload for updating an account.
type UpdateAccountRequest struct {
	FriendlyName *string `json:"friendly_name"`
	Status       *string `json:"status"`
}

// AccountResponse defines the standard API response for an account.
type AccountResponse struct {
	SID          string               `json:"sid"`
	ParentSID    *string              `json:"parent_sid,omitempty"`
	FriendlyName string               `json:"friendly_name"`
	PhoneNumber  string               `json:"phone_number,omitempty"`
	Type         domain.AccountType   `json:"type"`
	Status       domain.AccountStatus `json:"status"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
}

func ToAccountResponse(acc *domain.Account) AccountResponse {
	var parentSID *string
	if acc.ParentSID != nil {
		parentSID = acc.ParentSID
	}
	return AccountResponse{
		SID:          acc.SID,
		ParentSID:    parentSID,
		FriendlyName: acc.FriendlyName,
		PhoneNumber:  acc.PhoneNumber,
		Type:         acc.Type,
		Status:       acc.Status,
		CreatedAt:    acc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    acc.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ToAccountResponseList(accounts []*domain.Account) []AccountResponse {
	responses := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		responses[i] = ToAccountResponse(acc)
	}
	return responses
}

type GenericErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
