package req_res

import (
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
)

func TestLoginRequest_JSONRoundTrip(t *testing.T) {
	original := LoginRequest{
		UserID:         "testuser",
		PasswordSha256: "abc123def456",
		LocaleName:     "ja",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded LoginRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.UserID != original.UserID {
		t.Errorf("UserID = %q, want %q", decoded.UserID, original.UserID)
	}
	if decoded.PasswordSha256 != original.PasswordSha256 {
		t.Errorf("PasswordSha256 = %q, want %q", decoded.PasswordSha256, original.PasswordSha256)
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
}

func TestAddKmemoRequest_JSONRoundTrip(t *testing.T) {
	txid := "tx-001"
	original := AddKmemoRequest{
		SessionID:        "session-abc",
		TXID:             &txid,
		LocaleName:       "en",
		WantResponseKyou: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded AddKmemoRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.TXID == nil {
		t.Fatal("TXID should not be nil after unmarshal")
	}
	if *decoded.TXID != txid {
		t.Errorf("TXID = %q, want %q", *decoded.TXID, txid)
	}
	if decoded.WantResponseKyou != true {
		t.Errorf("WantResponseKyou = %v, want true", decoded.WantResponseKyou)
	}
}

func TestGetKyousResponse_JSONRoundTrip(t *testing.T) {
	original := GetKyousResponse{
		Messages: nil,
		Errors:   nil,
		Kyous:    nil,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded GetKyousResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// nil slices should marshal as null and unmarshal back to nil
	if decoded.Messages != nil {
		t.Error("expected nil Messages")
	}
}

func TestLoginRequest_JSONFieldNames(t *testing.T) {
	req := LoginRequest{
		UserID:         "user1",
		PasswordSha256: "hash",
		LocaleName:     "ja",
	}
	data, _ := json.Marshal(req)
	s := string(data)

	// Verify JSON field names match expected format
	for _, field := range []string{`"user_id"`, `"password_sha256"`, `"locale_name"`} {
		if !contains(s, field) {
			t.Errorf("JSON missing field %s in %s", field, s)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAddMiRequest_JSONRoundTrip(t *testing.T) {
	txid := "tx-mi-001"
	original := AddMiRequest{
		SessionID:        "session-mi",
		TXID:             &txid,
		LocaleName:       "ja",
		WantResponseKyou: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded AddMiRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.TXID == nil {
		t.Fatal("TXID should not be nil after unmarshal")
	}
	if *decoded.TXID != txid {
		t.Errorf("TXID = %q, want %q", *decoded.TXID, txid)
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
	if decoded.WantResponseKyou != true {
		t.Errorf("WantResponseKyou = %v, want true", decoded.WantResponseKyou)
	}
}

func TestAddLantanaRequest_JSONRoundTrip(t *testing.T) {
	txid := "tx-lantana-001"
	original := AddLantanaRequest{
		SessionID:        "session-lantana",
		TXID:             &txid,
		LocaleName:       "en",
		WantResponseKyou: false,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded AddLantanaRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.TXID == nil {
		t.Fatal("TXID should not be nil after unmarshal")
	}
	if *decoded.TXID != txid {
		t.Errorf("TXID = %q, want %q", *decoded.TXID, txid)
	}
	if decoded.WantResponseKyou != false {
		t.Error("WantResponseKyou should be false")
	}
}

func TestAddNlogRequest_JSONRoundTrip(t *testing.T) {
	txid := "tx-nlog-001"
	original := AddNlogRequest{
		SessionID:        "session-nlog",
		TXID:             &txid,
		LocaleName:       "ja",
		WantResponseKyou: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded AddNlogRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.TXID == nil {
		t.Fatal("TXID should not be nil after unmarshal")
	}
	if *decoded.TXID != txid {
		t.Errorf("TXID = %q, want %q", *decoded.TXID, txid)
	}
	if decoded.WantResponseKyou != true {
		t.Errorf("WantResponseKyou = %v, want true", decoded.WantResponseKyou)
	}
}

func TestUpdateKmemoRequest_JSONRoundTrip(t *testing.T) {
	txid := "tx-update-kmemo-001"
	original := UpdateKmemoRequest{
		SessionID:        "session-update-kmemo",
		TXID:             &txid,
		LocaleName:       "en",
		WantResponseKyou: true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded UpdateKmemoRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.TXID == nil {
		t.Fatal("TXID should not be nil after unmarshal")
	}
	if *decoded.TXID != txid {
		t.Errorf("TXID = %q, want %q", *decoded.TXID, txid)
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
	if decoded.WantResponseKyou != true {
		t.Errorf("WantResponseKyou = %v, want true", decoded.WantResponseKyou)
	}
}

func TestGetMiBoardResponse_JSONRoundTrip(t *testing.T) {
	original := GetMiBoardResponse{
		Boards: []string{"board_a", "board_b", "board_c"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded GetMiBoardResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.Boards) != len(original.Boards) {
		t.Fatalf("Boards length = %d, want %d", len(decoded.Boards), len(original.Boards))
	}
	for i, b := range original.Boards {
		if decoded.Boards[i] != b {
			t.Errorf("Boards[%d] = %q, want %q", i, decoded.Boards[i], b)
		}
	}
}

func TestGetAllTagNamesResponse_JSONRoundTrip(t *testing.T) {
	original := GetAllTagNamesResponse{
		TagNames: []string{"tag_x", "tag_y", "tag_z"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded GetAllTagNamesResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(decoded.TagNames) != len(original.TagNames) {
		t.Fatalf("TagNames length = %d, want %d", len(decoded.TagNames), len(original.TagNames))
	}
	for i, tn := range original.TagNames {
		if decoded.TagNames[i] != tn {
			t.Errorf("TagNames[%d] = %q, want %q", i, decoded.TagNames[i], tn)
		}
	}
}

func TestAddShareKyouListInfoRequest_JSONRoundTrip(t *testing.T) {
	original := AddShareKyouListInfoRequest{
		SessionID: "session-share",
		ShareKyouListInfo: &ShareKyouListInfo{
			ShareID:       "share-001",
			UserID:        "user1",
			Device:        "device1",
			ShareTitle:    "my share",
			FindQueryJSON: share_kyou_info.JSONString(`{"use_words":true}`),
			ViewType:      "list",
		},
		LocaleName: "ja",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded AddShareKyouListInfoRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.ShareKyouListInfo == nil {
		t.Fatal("ShareKyouListInfo should not be nil")
	}
	if decoded.ShareKyouListInfo.ShareID != "share-001" {
		t.Errorf("ShareID = %q, want %q", decoded.ShareKyouListInfo.ShareID, "share-001")
	}
	if decoded.ShareKyouListInfo.ShareTitle != "my share" {
		t.Errorf("ShareTitle = %q, want %q", decoded.ShareKyouListInfo.ShareTitle, "my share")
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
}

func TestCommitTxRequest_JSONRoundTrip(t *testing.T) {
	original := CommitTxRequest{
		SessionID:  "session-commit",
		TXID:       "tx-commit-001",
		LocaleName: "en",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded CommitTxRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.TXID != original.TXID {
		t.Errorf("TXID = %q, want %q", decoded.TXID, original.TXID)
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
}

func TestGetApplicationConfigResponse_JSONRoundTrip(t *testing.T) {
	original := GetApplicationConfigResponse{
		ApplicationConfig: nil,
		Messages:          nil,
		Errors:            nil,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded GetApplicationConfigResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ApplicationConfig != nil {
		t.Error("ApplicationConfig should be nil")
	}
	if decoded.Messages != nil {
		t.Error("Messages should be nil")
	}
	if decoded.Errors != nil {
		t.Error("Errors should be nil")
	}
}

func TestUpdateServerConfigsRequest_JSONRoundTrip(t *testing.T) {
	original := UpdateServerConfigsRequest{
		SessionID:     "session-server-cfg",
		ServerConfigs: nil,
		LocaleName:    "ja",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded UpdateServerConfigsRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.ServerConfigs != nil {
		t.Error("ServerConfigs should be nil")
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
}

func TestSubmitKFTLTextRequest_JSONRoundTrip(t *testing.T) {
	original := SubmitKFTLTextRequest{
		SessionID:  "session-kftl",
		KFTLText:   "kmemo hoge\nfuga\n.",
		LocaleName: "ja",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded SubmitKFTLTextRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.KFTLText != original.KFTLText {
		t.Errorf("KFTLText = %q, want %q", decoded.KFTLText, original.KFTLText)
	}
	if decoded.LocaleName != original.LocaleName {
		t.Errorf("LocaleName = %q, want %q", decoded.LocaleName, original.LocaleName)
	}
}

func TestGetKyousMCPRequest_JSONRoundTrip(t *testing.T) {
	includeTimeIs := false
	original := GetKyousMCPRequest{
		SessionID:       "session-mcp",
		LocaleName:      "en",
		Limit:           25,
		Cursor:          "2025-01-01T00:00:00Z",
		MaxSizeMB:       2.5,
		IsIncludeTimeIs: &includeTimeIs,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded GetKyousMCPRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %q, want %q", decoded.SessionID, original.SessionID)
	}
	if decoded.Limit != original.Limit {
		t.Errorf("Limit = %d, want %d", decoded.Limit, original.Limit)
	}
	if decoded.Cursor != original.Cursor {
		t.Errorf("Cursor = %q, want %q", decoded.Cursor, original.Cursor)
	}
	if decoded.MaxSizeMB != original.MaxSizeMB {
		t.Errorf("MaxSizeMB = %f, want %f", decoded.MaxSizeMB, original.MaxSizeMB)
	}
	if decoded.IsIncludeTimeIs == nil || *decoded.IsIncludeTimeIs != false {
		t.Error("IsIncludeTimeIs should be false")
	}
	if decoded.ShouldIncludeTimeIs() != false {
		t.Error("ShouldIncludeTimeIs() should return false")
	}
}

func TestIDFPayloadMCPDTO_JSONRoundTrip(t *testing.T) {
	original := IDFPayloadMCPDTO{
		Kind:     "idf",
		FileName: "photo.jpg",
		IsImage:  true,
		IsVideo:  false,
		IsAudio:  false,
		RepName:  "images_repo",
		MimeType: "image/jpeg",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded IDFPayloadMCPDTO
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Kind != original.Kind {
		t.Errorf("Kind = %q, want %q", decoded.Kind, original.Kind)
	}
	if decoded.FileName != original.FileName {
		t.Errorf("FileName = %q, want %q", decoded.FileName, original.FileName)
	}
	if decoded.IsImage != original.IsImage {
		t.Errorf("IsImage = %v, want %v", decoded.IsImage, original.IsImage)
	}
	if decoded.RepName != original.RepName {
		t.Errorf("RepName = %q, want %q", decoded.RepName, original.RepName)
	}
	if decoded.MimeType != original.MimeType {
		t.Errorf("MimeType = %q, want %q", decoded.MimeType, original.MimeType)
	}

	// Verify JSON field names
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}
	for _, field := range []string{"kind", "file_name", "is_image", "is_video", "is_audio", "rep_name", "mime_type"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("JSON missing expected field %q", field)
		}
	}
}

func TestIDFPayloadMCPDTO_OmitsEmptyMimeType(t *testing.T) {
	original := IDFPayloadMCPDTO{
		Kind:     "idf",
		FileName: "data.bin",
		RepName:  "files_repo",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}

	if _, ok := raw["mime_type"]; ok {
		t.Error("mime_type should be omitted when empty")
	}
}
