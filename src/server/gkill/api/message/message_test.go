package message

import (
	"encoding/json"
	"regexp"
	"testing"
)

func TestErrorCodes_NonEmpty(t *testing.T) {
	errorCodes := map[string]string{
		"AccountInvalidLoginRequestDataError":  AccountInvalidLoginRequestDataError,
		"AccountNotFoundError":                 AccountNotFoundError,
		"AccountIsNotEnableError":              AccountIsNotEnableError,
		"AccountInvalidPasswordError":          AccountInvalidPasswordError,
		"AccountLoginInternalServerError":      AccountLoginInternalServerError,
		"AccountSessionNotFoundError":          AccountSessionNotFoundError,
		"RepositoriesGetError":                 RepositoriesGetError,
		"AddTagError":                          AddTagError,
		"GetTagError":                          GetTagError,
		"AddKmemoError":                        AddKmemoError,
		"GetKmemoError":                        GetKmemoError,
		"AddURLogError":                        AddURLogError,
		"AddTimeIsError":                       AddTimeIsError,
		"AddLantanaError":                      AddLantanaError,
		"AddMiError":                           AddMiError,
		"GetMiError":                           GetMiError,
		"AddReKyouError":                       AddReKyouError,
		"GetKyouError":                         GetKyouError,
		"AddKCError":                           AddKCError,
		"GetKCError":                           GetKCError,
		"SubmitKFTLTextError":                  SubmitKFTLTextError,
		"NotImplementsError":                   NotImplementsError,
		"InvalidURLogBookmarkletRequestDataError": InvalidURLogBookmarkletRequestDataError,
		"UpdateCacheError":                     UpdateCacheError,
		"CommitTxGetKmemoError":                CommitTxGetKmemoError,
		"AccountSessionExpiredError":           AccountSessionExpiredError,
		"LoginRateLimitError":                  LoginRateLimitError,
	}

	for name, code := range errorCodes {
		if code == "" {
			t.Errorf("Error code %s is empty", name)
		}
	}
}

func TestErrorCodes_UniqueValues(t *testing.T) {
	codes := []string{
		AccountInvalidLoginRequestDataError,
		AccountNotFoundError,
		AccountIsNotEnableError,
		AccountInvalidPasswordError,
		AccountLoginInternalServerError,
		AccountSessionNotFoundError,
		RepositoriesGetError,
		AddTagError,
		GetTagError,
		AddKmemoError,
		GetKmemoError,
		AddURLogError,
		AddTimeIsError,
		AddLantanaError,
		AddMiError,
		GetMiError,
		AddReKyouError,
		GetKyouError,
		AddKCError,
		GetKCError,
		SubmitKFTLTextError,
		NotImplementsError,
		UpdateCacheError,
		CommitTxGetKmemoError,
		AccountSessionExpiredError,
		LoginRateLimitError,
	}

	seen := make(map[string]bool)
	for _, code := range codes {
		if seen[code] {
			t.Errorf("Duplicate error code value: %s", code)
		}
		seen[code] = true
	}
}

func TestErrorCodes_Format(t *testing.T) {
	pattern := regexp.MustCompile(`^ERR\d{6}$`)

	codes := map[string]string{
		"AccountInvalidLoginRequestDataError": AccountInvalidLoginRequestDataError,
		"AccountNotFoundError":                AccountNotFoundError,
		"AccountInvalidPasswordError":         AccountInvalidPasswordError,
		"RepositoriesGetError":                RepositoriesGetError,
		"AddKmemoError":                       AddKmemoError,
		"GetKmemoError":                       GetKmemoError,
		"AddMiError":                          AddMiError,
		"AddKCError":                          AddKCError,
		"SubmitKFTLTextError":                 SubmitKFTLTextError,
		"UpdateCacheError":                    UpdateCacheError,
		"NotImplementsError":                  NotImplementsError,
		"CommitTxDeleteURLogError":            CommitTxDeleteURLogError,
		"NotFoundTLSCertFileError":            NotFoundTLSCertFileError,
		"NotFoundTLSKeyFileError":             NotFoundTLSKeyFileError,
		"GetAccountSessionsError":             GetAccountSessionsError,
		"AddURLogLoginSessionError":           AddURLogLoginSessionError,
		"InvalidURLogBookmarkletRequestDataError": InvalidURLogBookmarkletRequestDataError,
		"UpdateTagError":                      UpdateTagError,
		"UpdateNotificationError":             UpdateNotificationError,
		"GetKyousMCPError":                    GetKyousMCPError,
		"AccountSessionExpiredError":          AccountSessionExpiredError,
		"LoginRateLimitError":                 LoginRateLimitError,
	}

	for name, code := range codes {
		if !pattern.MatchString(code) {
			t.Errorf("Error code %s = %q does not match format ERR + 6 digits", name, code)
		}
	}
}

func TestMessageCodes_NonEmpty(t *testing.T) {
	messageCodes := map[string]string{
		"LoginSuccessMessage":                    LoginSuccessMessage,
		"LogoutSuccessMessage":                   LogoutSuccessMessage,
		"PasswordResetSuccessMessage":            PasswordResetSuccessMessage,
		"SetNewPasswordSuccessMessage":           SetNewPasswordSuccessMessage,
		"AddTagSuccessMessage":                   AddTagSuccessMessage,
		"AddKmemoSuccessMessage":                 AddKmemoSuccessMessage,
		"AddURLogSuccessMessage":                 AddURLogSuccessMessage,
		"AddNlogSuccessMessage":                  AddNlogSuccessMessage,
		"AddTimeIsSuccessMessage":                AddTimeIsSuccessMessage,
		"AddLantanaSuccessMessage":               AddLantanaSuccessMessage,
		"AddMiSuccessMessage":                    AddMiSuccessMessage,
		"UpdateMiSuccessMessage":                 UpdateMiSuccessMessage,
		"GetKyousSuccessMessage":                 GetKyousSuccessMessage,
		"GetApplicationConfigSuccessMessage":     GetApplicationConfigSuccessMessage,
		"AddAccountSuccessMessage":               AddAccountSuccessMessage,
		"SubmitKFTLTextSuccessMessage":           SubmitKFTLTextSuccessMessage,
		"GetKyousMCPSuccessMessage":              GetKyousMCPSuccessMessage,
		"UpdateCacheSuccessMessage":              UpdateCacheSuccessMessage,
		"CommitTxSuccessMessage":                 CommitTxSuccessMessage,
		"RegisterGkillNotificationSuccessMessage": RegisterGkillNotificationSuccessMessage,
	}

	for name, code := range messageCodes {
		if code == "" {
			t.Errorf("Message code %s is empty", name)
		}
	}
}

func TestMessageCodes_Format(t *testing.T) {
	pattern := regexp.MustCompile(`^MSG\d{6}$`)

	codes := map[string]string{
		"LoginSuccessMessage":                LoginSuccessMessage,
		"LogoutSuccessMessage":               LogoutSuccessMessage,
		"AddKmemoSuccessMessage":             AddKmemoSuccessMessage,
		"AddMiSuccessMessage":                AddMiSuccessMessage,
		"GetKyousSuccessMessage":             GetKyousSuccessMessage,
		"GetApplicationConfigSuccessMessage": GetApplicationConfigSuccessMessage,
		"AddAccountSuccessMessage":           AddAccountSuccessMessage,
		"SubmitKFTLTextSuccessMessage":       SubmitKFTLTextSuccessMessage,
		"GetKyousMCPSuccessMessage":          GetKyousMCPSuccessMessage,
		"UpdateCacheSuccessMessage":          UpdateCacheSuccessMessage,
		"CommitTxSuccessMessage":             CommitTxSuccessMessage,
		"TLSFileCreateSuccessMessage":        TLSFileCreateSuccessMessage,
		"AddKCSuccessMessage":                AddKCSuccessMessage,
		"UpdateKCSuccessMessage":             UpdateKCSuccessMessage,
		"RebootingMessage":                   RebootingMessage,
		"OpenDirectorySuccessMessage":        OpenDirectorySuccessMessage,
		"OpenFileSuccessMessage":             OpenFileSuccessMessage,
		"ReloadRepositoriesSuccessMessage":   ReloadRepositoriesSuccessMessage,
		"GetPlaingTimeIsSuccessMessage":      GetPlaingTimeIsSuccessMessage,
		"RegisterGkillNotificationSuccessMessage": RegisterGkillNotificationSuccessMessage,
	}

	for name, code := range codes {
		if !pattern.MatchString(code) {
			t.Errorf("Message code %s = %q does not match format MSG + 6 digits", name, code)
		}
	}
}

func TestGkillError_JSONRoundTrip(t *testing.T) {
	original := GkillError{
		ErrorCode:    AccountNotFoundError,
		ErrorMessage: "Account was not found",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored GkillError
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.ErrorCode != original.ErrorCode {
		t.Errorf("ErrorCode: got %q, want %q", restored.ErrorCode, original.ErrorCode)
	}
	if restored.ErrorMessage != original.ErrorMessage {
		t.Errorf("ErrorMessage: got %q, want %q", restored.ErrorMessage, original.ErrorMessage)
	}

	// Verify JSON field names
	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	if err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	if _, ok := raw["error_code"]; !ok {
		t.Error("JSON should contain 'error_code' field")
	}
	if _, ok := raw["error_message"]; !ok {
		t.Error("JSON should contain 'error_message' field")
	}
}

func TestGkillMessage_JSONRoundTrip(t *testing.T) {
	original := GkillMessage{
		MessageCode: LoginSuccessMessage,
		Message:     "Login successful",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored GkillMessage
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if restored.MessageCode != original.MessageCode {
		t.Errorf("MessageCode: got %q, want %q", restored.MessageCode, original.MessageCode)
	}
	if restored.Message != original.Message {
		t.Errorf("Message: got %q, want %q", restored.Message, original.Message)
	}

	// Verify JSON field names
	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	if err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}
	if _, ok := raw["message_code"]; !ok {
		t.Error("JSON should contain 'message_code' field")
	}
	if _, ok := raw["message"]; !ok {
		t.Error("JSON should contain 'message' field")
	}
}
