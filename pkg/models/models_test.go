package models

import (
	"testing"
	"time"
)

func TestHTTPMethodValidate(t *testing.T) {
	tests := []struct {
		name     string
		method   HTTPMethod
		expected bool
	}{
		{
			name:     "valid GET",
			method:   HTTPMethodGET,
			expected: true,
		},
		{
			name:     "valid POST",
			method:   HTTPMethodPost,
			expected: true,
		},
		{
			name:     "valid lowercase get",
			method:   HTTPMethod("get"),
			expected: true,
		},
		{
			name:     "valid lowercase post",
			method:   HTTPMethod("post"),
			expected: true,
		},
		{
			name:     "valid mixed case Get",
			method:   HTTPMethod("Get"),
			expected: true,
		},
		{
			name:     "valid mixed case Post",
			method:   HTTPMethod("Post"),
			expected: true,
		},
		{
			name:     "empty string",
			method:   HTTPMethod(""),
			expected: true,
		},
		{
			name:     "invalid method PUT",
			method:   HTTPMethod("PUT"),
			expected: false,
		},
		{
			name:     "invalid method DELETE",
			method:   HTTPMethod("DELETE"),
			expected: false,
		},
		{
			name:     "invalid method PATCH",
			method:   HTTPMethod("PATCH"),
			expected: false,
		},
		{
			name:     "invalid random string",
			method:   HTTPMethod("invalid"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method.Validate()
			if result != tt.expected {
				t.Errorf("HTTPMethod.Validate() = %v, want %v for method %q", result, tt.expected, tt.method)
			}
		})
	}
}

func TestApplicationRequestToAppModel(t *testing.T) {
	tests := []struct {
		name     string
		request  ApplicationRequest
		expected Application
	}{
		{
			name: "basic application request",
			request: ApplicationRequest{
				FriendlyName: "Test App",
				VoiceUrl:     "http://example.com/voice",
				VoiceMethod:  HTTPMethod("get"),
			},
			expected: Application{
				FriendlyName: "Test App",
				VoiceUrl:     "http://example.com/voice",
				VoiceMethod:  HTTPMethodGET,
			},
		},
		{
			name: "complete application request",
			request: ApplicationRequest{
				FriendlyName:            "Complete App",
				VoiceUrl:                "http://example.com/voice",
				VoiceMethod:             HTTPMethod("post"),
				VoiceFallbackUrl:        "http://example.com/fallback",
				VoiceFallbackMethod:     HTTPMethod("get"),
				StatusCallback:          "http://example.com/status",
				StatusCallbackMethod:    HTTPMethod("post"),
				SmsUrl:                  "http://example.com/sms",
				SmsMethod:               HTTPMethod("post"),
				SmsFallbackUrl:          "http://example.com/sms-fallback",
				SmsFallbackMethod:       HTTPMethod("get"),
				SmsStatusCallback:       "http://example.com/sms-status",
				SmsStatusCallbackMethod: HTTPMethod("post"),
				HeartbeatUrl:            "http://example.com/heartbeat",
			},
			expected: Application{
				FriendlyName:            "Complete App",
				VoiceUrl:                "http://example.com/voice",
				VoiceMethod:             HTTPMethodPost,
				VoiceFallbackUrl:        "http://example.com/fallback",
				VoiceFallbackMethod:     HTTPMethodGET,
				StatusCallback:          "http://example.com/status",
				StatusCallbackMethod:    HTTPMethodPost,
				SmsUrl:                  "http://example.com/sms",
				SmsMethod:               HTTPMethodPost,
				SmsFallbackUrl:          "http://example.com/sms-fallback",
				SmsFallbackMethod:       HTTPMethodGET,
				SmsStatusCallback:       "http://example.com/sms-status",
				SmsStatusCallbackMethod: HTTPMethodPost,
				HeartbeatUrl:            "http://example.com/heartbeat",
			},
		},
		{
			name: "mixed case methods",
			request: ApplicationRequest{
				FriendlyName:         "Mixed Case App",
				VoiceUrl:             "http://example.com/voice",
				VoiceMethod:          HTTPMethod("Post"),
				VoiceFallbackMethod:  HTTPMethod("Get"),
				StatusCallbackMethod: HTTPMethod("POST"),
				SmsMethod:            HTTPMethod("get"),
			},
			expected: Application{
				FriendlyName:         "Mixed Case App",
				VoiceUrl:             "http://example.com/voice",
				VoiceMethod:          HTTPMethodPost,
				VoiceFallbackMethod:  HTTPMethodGET,
				StatusCallbackMethod: HTTPMethodPost,
				SmsMethod:            HTTPMethodGET,
			},
		},
		{
			name: "empty methods",
			request: ApplicationRequest{
				FriendlyName: "Empty Methods App",
				VoiceUrl:     "http://example.com/voice",
			},
			expected: Application{
				FriendlyName: "Empty Methods App",
				VoiceUrl:     "http://example.com/voice",
				VoiceMethod:  HTTPMethod(""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.request.ToAppModel()
			
			// Compare all fields
			if result.FriendlyName != tt.expected.FriendlyName {
				t.Errorf("FriendlyName = %v, want %v", result.FriendlyName, tt.expected.FriendlyName)
			}
			if result.VoiceUrl != tt.expected.VoiceUrl {
				t.Errorf("VoiceUrl = %v, want %v", result.VoiceUrl, tt.expected.VoiceUrl)
			}
			if result.VoiceMethod != tt.expected.VoiceMethod {
				t.Errorf("VoiceMethod = %v, want %v", result.VoiceMethod, tt.expected.VoiceMethod)
			}
			if result.VoiceFallbackUrl != tt.expected.VoiceFallbackUrl {
				t.Errorf("VoiceFallbackUrl = %v, want %v", result.VoiceFallbackUrl, tt.expected.VoiceFallbackUrl)
			}
			if result.VoiceFallbackMethod != tt.expected.VoiceFallbackMethod {
				t.Errorf("VoiceFallbackMethod = %v, want %v", result.VoiceFallbackMethod, tt.expected.VoiceFallbackMethod)
			}
			if result.StatusCallback != tt.expected.StatusCallback {
				t.Errorf("StatusCallback = %v, want %v", result.StatusCallback, tt.expected.StatusCallback)
			}
			if result.StatusCallbackMethod != tt.expected.StatusCallbackMethod {
				t.Errorf("StatusCallbackMethod = %v, want %v", result.StatusCallbackMethod, tt.expected.StatusCallbackMethod)
			}
			if result.SmsUrl != tt.expected.SmsUrl {
				t.Errorf("SmsUrl = %v, want %v", result.SmsUrl, tt.expected.SmsUrl)
			}
			if result.SmsMethod != tt.expected.SmsMethod {
				t.Errorf("SmsMethod = %v, want %v", result.SmsMethod, tt.expected.SmsMethod)
			}
			if result.SmsFallbackUrl != tt.expected.SmsFallbackUrl {
				t.Errorf("SmsFallbackUrl = %v, want %v", result.SmsFallbackUrl, tt.expected.SmsFallbackUrl)
			}
			if result.SmsFallbackMethod != tt.expected.SmsFallbackMethod {
				t.Errorf("SmsFallbackMethod = %v, want %v", result.SmsFallbackMethod, tt.expected.SmsFallbackMethod)
			}
			if result.SmsStatusCallback != tt.expected.SmsStatusCallback {
				t.Errorf("SmsStatusCallback = %v, want %v", result.SmsStatusCallback, tt.expected.SmsStatusCallback)
			}
			if result.SmsStatusCallbackMethod != tt.expected.SmsStatusCallbackMethod {
				t.Errorf("SmsStatusCallbackMethod = %v, want %v", result.SmsStatusCallbackMethod, tt.expected.SmsStatusCallbackMethod)
			}
			if result.HeartbeatUrl != tt.expected.HeartbeatUrl {
				t.Errorf("HeartbeatUrl = %v, want %v", result.HeartbeatUrl, tt.expected.HeartbeatUrl)
			}
		})
	}
}

func TestAccountTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    AccountType
		expected string
	}{
		{
			name:     "trial account type",
			value:    AccountTypeTrial,
			expected: "trial",
		},
		{
			name:     "full account type",
			value:    AccountTypeFull,
			expected: "full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("AccountType = %v, want %v", string(tt.value), tt.expected)
			}
		})
	}
}

func TestAccountStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    AccountStatus
		expected string
	}{
		{
			name:     "active account status",
			value:    AccountStatusActive,
			expected: "active",
		},
		{
			name:     "suspended account status",
			value:    AccountStatusSuspended,
			expected: "suspended",
		},
		{
			name:     "closed account status",
			value:    AccountStatusClosed,
			expected: "closed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("AccountStatus = %v, want %v", string(tt.value), tt.expected)
			}
		})
	}
}

func TestCallStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    CallStatus
		expected string
	}{
		{
			name:     "queued call status",
			value:    CallStatusQueued,
			expected: "queued",
		},
		{
			name:     "initiating call status",
			value:    CallStatusInitiating,
			expected: "initiating",
		},
		{
			name:     "ringing call status",
			value:    CallStatusRinging,
			expected: "ringing",
		},
		{
			name:     "in-progress call status",
			value:    CallStatusInProgress,
			expected: "in-progress",
		},
		{
			name:     "completed call status",
			value:    CallStatusCompleted,
			expected: "completed",
		},
		{
			name:     "failed call status",
			value:    CallStatusFailed,
			expected: "failed",
		},
		{
			name:     "busy call status",
			value:    CallStatusBusy,
			expected: "busy",
		},
		{
			name:     "no-answer call status",
			value:    CallStatusNoAnswer,
			expected: "no-answer",
		},
		{
			name:     "canceled call status",
			value:    CallStatusCanceled,
			expected: "canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("CallStatus = %v, want %v", string(tt.value), tt.expected)
			}
		})
	}
}

func TestHTTPMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    HTTPMethod
		expected string
	}{
		{
			name:     "GET method",
			value:    HTTPMethodGET,
			expected: "GET",
		},
		{
			name:     "POST method",
			value:    HTTPMethodPost,
			expected: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("HTTPMethod = %v, want %v", string(tt.value), tt.expected)
			}
		})
	}
}

func TestAccountModel(t *testing.T) {
	now := time.Now()
	account := Account{
		Sid:          "AC123456789",
		ParentSid:    "AC987654321",
		FriendlyName: "Test Account",
		PhoneNumber:  "+1234567890",
		DateCreated:  now,
		DateUpdated:  now,
		Type:         AccountTypeTrial,
		Status:       AccountStatusActive,
		AuthToken:    "secret-token",
		DefaultOutboundGateway: "gateway1",
		GatewaySelectionScript: "select_gateway.lua",
	}

	// Test that all fields are set correctly
	if account.Sid != "AC123456789" {
		t.Errorf("Expected Sid to be 'AC123456789', got %v", account.Sid)
	}
	if account.Type != AccountTypeTrial {
		t.Errorf("Expected Type to be AccountTypeTrial, got %v", account.Type)
	}
	if account.Status != AccountStatusActive {
		t.Errorf("Expected Status to be AccountStatusActive, got %v", account.Status)
	}
}

func TestCallModel(t *testing.T) {
	now := time.Now()
	call := Call{
		Sid:              "CA123456789",
		AccountSid:       "AC123456789",
		CallerId:         "+1234567890",
		CallTo:           "+0987654321",
		AnswerUrl:        "http://example.com/answer",
		ApplicationSid:   "AP123456789",
		Status:           CallStatusInProgress,
		Timeout:          "30",
		Direction:        "outbound",
		Duration:         120,
		Price:            0.05,
		StartTime:        now,
		EndTime:          now.Add(2 * time.Minute),
		DateCreated:      now,
		DateUpdated:      now,
		AnsweredBy:       "human",
		FreeswitchCallID: "uuid-123-456-789",
	}

	// Test that all fields are set correctly
	if call.Sid != "CA123456789" {
		t.Errorf("Expected Sid to be 'CA123456789', got %v", call.Sid)
	}
	if call.Status != CallStatusInProgress {
		t.Errorf("Expected Status to be CallStatusInProgress, got %v", call.Status)
	}
	if call.Duration != 120 {
		t.Errorf("Expected Duration to be 120, got %v", call.Duration)
	}
	if call.Price != 0.05 {
		t.Errorf("Expected Price to be 0.05, got %v", call.Price)
	}
}

func TestApplicationModel(t *testing.T) {
	now := time.Now()
	app := Application{
		Sid:                     "AP123456789",
		AccountSid:              "AC123456789",
		FriendlyName:            "Test Application",
		VoiceUrl:                "http://example.com/voice",
		VoiceMethod:             HTTPMethodPost,
		VoiceFallbackUrl:        "http://example.com/fallback",
		VoiceFallbackMethod:     HTTPMethodGET,
		StatusCallback:          "http://example.com/status",
		StatusCallbackMethod:    HTTPMethodPost,
		SmsUrl:                  "http://example.com/sms",
		SmsMethod:               HTTPMethodPost,
		SmsFallbackUrl:          "http://example.com/sms-fallback",
		SmsFallbackMethod:       HTTPMethodGET,
		SmsStatusCallback:       "http://example.com/sms-status",
		SmsStatusCallbackMethod: HTTPMethodPost,
		HeartbeatUrl:            "http://example.com/heartbeat",
		DateCreated:             now,
		DateUpdated:             now,
	}

	// Test that all fields are set correctly
	if app.Sid != "AP123456789" {
		t.Errorf("Expected Sid to be 'AP123456789', got %v", app.Sid)
	}
	if app.VoiceMethod != HTTPMethodPost {
		t.Errorf("Expected VoiceMethod to be HTTPMethodPost, got %v", app.VoiceMethod)
	}
	if app.VoiceFallbackMethod != HTTPMethodGET {
		t.Errorf("Expected VoiceFallbackMethod to be HTTPMethodGET, got %v", app.VoiceFallbackMethod)
	}
}

func TestRequestModels(t *testing.T) {
	// Test CreateAccountRequest
	createReq := CreateAccountRequest{
		FriendlyName: "New Account",
	}
	if createReq.FriendlyName != "New Account" {
		t.Errorf("Expected FriendlyName to be 'New Account', got %v", createReq.FriendlyName)
	}

	// Test ChangeAccountStatusRequest
	statusReq := ChangeAccountStatusRequest{
		Status: AccountStatusSuspended,
	}
	if statusReq.Status != AccountStatusSuspended {
		t.Errorf("Expected Status to be AccountStatusSuspended, got %v", statusReq.Status)
	}

	// Test CallRequest
	callReq := CallRequest{
		From:           "+1234567890",
		To:             "+0987654321",
		ApplicationSid: "AP123456789",
		AnswerUrl:      "http://example.com/answer",
		Method:         "POST",
		TimeLimit:      "300",
	}
	if callReq.To != "+0987654321" {
		t.Errorf("Expected To to be '+0987654321', got %v", callReq.To)
	}

	// Test CallPlayRequest
	playReq := CallPlayRequest{
		PlayUrl: "http://example.com/audio.mp3",
		Loop:    "2",
		Legs:    "both",
	}
	if playReq.PlayUrl != "http://example.com/audio.mp3" {
		t.Errorf("Expected PlayUrl to be 'http://example.com/audio.mp3', got %v", playReq.PlayUrl)
	}

	// Test CallSpeakRequest
	speakReq := CallSpeakRequest{
		Text: "Hello World",
		Loop: "1",
	}
	if speakReq.Text != "Hello World" {
		t.Errorf("Expected Text to be 'Hello World', got %v", speakReq.Text)
	}

	// Test CallDigitRequest
	digitReq := CallDigitRequest{
		Digits: "1234#",
	}
	if digitReq.Digits != "1234#" {
		t.Errorf("Expected Digits to be '1234#', got %v", digitReq.Digits)
	}
}

func TestResponseModels(t *testing.T) {
	// Test CallResponse
	callResp := CallResponse{
		Message:      "Call initiated successfully",
		IsSuccessful: true,
		CallSid:      "CA123456789",
	}
	if !callResp.IsSuccessful {
		t.Errorf("Expected IsSuccessful to be true, got %v", callResp.IsSuccessful)
	}

	// Test CallPlayResponse
	playResp := CallPlayResponse{
		CallSid: "CA123456789",
		Message: "Audio playing",
	}
	if playResp.CallSid != "CA123456789" {
		t.Errorf("Expected CallSid to be 'CA123456789', got %v", playResp.CallSid)
	}

	// Test CallRecordResponse
	recordResp := CallRecordResponse{
		Success: true,
		Message: "Recording started",
	}
	if !recordResp.Success {
		t.Errorf("Expected Success to be true, got %v", recordResp.Success)
	}

	// Test CallSpeakResponse
	speakResp := CallSpeakResponse{
		Success: true,
		Message: "Text-to-speech started",
	}
	if !speakResp.Success {
		t.Errorf("Expected Success to be true, got %v", speakResp.Success)
	}

	// Test CallDigitResponse
	digitResp := CallDigitResponse{
		Success: true,
		Message: "Digits sent",
	}
	if !digitResp.Success {
		t.Errorf("Expected Success to be true, got %v", digitResp.Success)
	}
}

func TestUpdateAccountSettingsRequest(t *testing.T) {
	friendlyName := "Updated Name"
	phoneNumber := "+1234567890"
	accountType := AccountTypeFull
	gateway := "gateway2"
	script := "new_script.lua"

	updateReq := UpdateAccountSettingsRequest{
		FriendlyName:           &friendlyName,
		PhoneNumber:            &phoneNumber,
		Type:                   &accountType,
		DefaultOutboundGateway: &gateway,
		GatewaySelectionScript: &script,
	}

	if updateReq.FriendlyName == nil || *updateReq.FriendlyName != "Updated Name" {
		t.Errorf("Expected FriendlyName to be 'Updated Name', got %v", updateReq.FriendlyName)
	}
	if updateReq.Type == nil || *updateReq.Type != AccountTypeFull {
		t.Errorf("Expected Type to be AccountTypeFull, got %v", updateReq.Type)
	}
}