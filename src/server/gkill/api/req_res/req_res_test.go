package req_res

import (
	"encoding/json"
	"slices"
	"sort"
	"testing"
	"time"
)

// req_res のリクエスト/レスポンス構造体は、TypeScriptクライアントとMCPサーバが
// 直接依存しているワイヤ形式そのもの。Goの型を変えなくても json タグを1つ書き換えれば
// クライアントが黙って壊れるので、ここでは「JSONのフィールド名」を契約として固定する。
//
// 以前はここに構造体 → JSON → 構造体 の往復テストが19本あったが、カスタムMarshaler
// を持たない素の構造体の往復は encoding/json 自体のテストであり、タグ名を変えても
// 落ちなかった（往復では常に一致するため）。実際に壊れる箇所を見るテストに置き換えている。

// marshalKeys は構造体をJSONにしたときのトップレベルのキー一覧を返す。
func marshalKeys(t *testing.T, v any) []string {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TestRequestResponse_JSONFieldNames は、クライアント/MCPが参照するJSONフィールド名を固定する。
// フィールドを増やす分には落ちないが、既存フィールドのリネーム・削除で落ちる。
func TestRequestResponse_JSONFieldNames(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
	}{
		// --- 認証 ---
		{"LoginRequest", LoginRequest{}, []string{"user_id", "password_sha256", "locale_name"}},
		{"LoginResponse", LoginResponse{}, []string{"session_id", "messages", "errors"}},

		// --- 追加/更新（TSクライアントが組み立てる） ---
		{"AddKmemoRequest", AddKmemoRequest{}, []string{"session_id", "kmemo", "tx_id", "locale_name", "added_kyou", "want_response_kyou"}},
		{"UpdateKmemoRequest", UpdateKmemoRequest{}, []string{"session_id", "kmemo", "tx_id", "locale_name", "updated_kyou", "want_response_kyou"}},
		// MiReKyouのペイロードキーは "mirekyou"（"mi_re_kyou" ではない）
		{"AddMiReKyouRequest", AddMiReKyouRequest{}, []string{"session_id", "mirekyou", "tx_id", "locale_name", "added_kyou", "want_response_kyou"}},
		{"UpdateMiReKyouRequest", UpdateMiReKyouRequest{}, []string{"session_id", "mirekyou", "tx_id", "locale_name", "updated_kyou", "want_response_kyou"}},
		{"GetMiReKyouRequest", GetMiReKyouRequest{}, []string{"session_id", "id", "update_time", "rep_name", "locale_name"}},
		{"GetMiReKyouResponse", GetMiReKyouResponse{}, []string{"messages", "errors", "mirekyou_histories"}},

		// --- 検索 ---
		{"GetKyousRequest", GetKyousRequest{}, []string{"session_id", "query", "locale_name"}},
		{"GetKyousResponse", GetKyousResponse{}, []string{"messages", "errors", "kyous"}},
		{"GetSharedKyousRequest", GetSharedKyousRequest{}, []string{"shared_id", "locale_name"}},
		{"GetMiBoardResponse", GetMiBoardResponse{}, []string{"messages", "errors", "boards"}},
		{"GetAllTagNamesResponse", GetAllTagNamesResponse{}, []string{"messages", "errors", "tag_names"}},

		// --- MCPサーバが参照する ---
		{"GetKyousMCPRequest", GetKyousMCPRequest{}, []string{"session_id", "query", "locale_name", "limit", "cursor", "max_size_mb", "include_id", "is_include_timeis"}},

		// --- その他 ---
		{"SubmitKFTLTextRequest", SubmitKFTLTextRequest{}, []string{"session_id", "kftl_text", "locale_name"}},
		{"AddShareKyouListInfoRequest", AddShareKyouListInfoRequest{}, []string{"session_id", "share_kyou_list_info", "locale_name"}},
		{"CommitTxRequest", CommitTxRequest{}, []string{"session_id", "tx_id", "locale_name"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalKeys(t, tt.value)
			for _, want := range tt.want {
				if !slices.Contains(got, want) {
					t.Errorf("フィールド %q が無い（リネーム/削除するとクライアントが壊れる）: got %q", want, got)
				}
			}
		})
	}
}

// TestGetKyousMCPRequest_ShouldIncludeTimeIs は is_include_timeis の三値
// （未指定 / true / false）の解釈を確認する。
// *bool にしているのは「未指定なら true」を表すためで、省略時にTimeIsが
// 落ちるとMCPクライアント側の記録内容が黙って減る。
func TestGetKyousMCPRequest_ShouldIncludeTimeIs(t *testing.T) {
	yes, no := true, false

	tests := []struct {
		name string
		json string
		set  *bool
		want bool
	}{
		{"未指定はtrue", `{"session_id":"s"}`, nil, true},
		{"明示true", `{"session_id":"s","is_include_timeis":true}`, &yes, true},
		{"明示false", `{"session_id":"s","is_include_timeis":false}`, &no, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 構造体を直接組んだ場合
			req := GetKyousMCPRequest{IsIncludeTimeIs: tt.set}
			if got := req.ShouldIncludeTimeIs(); got != tt.want {
				t.Errorf("ShouldIncludeTimeIs() = %v, want %v", got, tt.want)
			}

			// JSONから復元した場合（MCPサーバが実際に通る経路）
			var decoded GetKyousMCPRequest
			if err := json.Unmarshal([]byte(tt.json), &decoded); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got := decoded.ShouldIncludeTimeIs(); got != tt.want {
				t.Errorf("JSON経由の ShouldIncludeTimeIs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMCPPayloadDTO_JSONFieldNames は、MCPサーバの payload.kind による分岐が
// 依存しているフィールド名を固定する。
func TestMCPPayloadDTO_JSONFieldNames(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{"KmemoPayloadMCPDTO", KmemoPayloadMCPDTO{Kind: "kmemo"}, []string{"kind", "content"}},
		{"KCPayloadMCPDTO", KCPayloadMCPDTO{Kind: "kc"}, []string{"kind", "title", "num_value"}},
		{"TimeIsPayloadMCPDTO", TimeIsPayloadMCPDTO{Kind: "timeis"}, []string{"kind", "title", "start_time"}},
		{
			"MiPayloadMCPDTO",
			MiPayloadMCPDTO{Kind: "mi", BoardName: "inbox"},
			[]string{"kind", "title", "is_checked", "board_name", "create_time"},
		},
		{
			"MiReKyouPayloadMCPDTO",
			MiReKyouPayloadMCPDTO{Kind: "mirekyou", TargetID: "target-1", BoardName: "inbox"},
			[]string{"kind", "target_id", "is_checked", "board_name", "create_time"},
		},
		{
			"ReKyouPayloadMCPDTO",
			ReKyouPayloadMCPDTO{Kind: "rekyou", TargetID: "target-1"},
			[]string{"kind", "target_id"},
		},
		{
			"URLogPayloadMCPDTO",
			URLogPayloadMCPDTO{Kind: "urlog", Description: "概要"},
			[]string{"kind", "title", "url", "description"},
		},
		{
			"IDFPayloadMCPDTO",
			IDFPayloadMCPDTO{Kind: "idf", MimeType: "image/jpeg"},
			[]string{"kind", "file_name", "is_image", "is_video", "is_audio", "rep_name", "mime_type"},
		},
		{
			"PluginPayloadMCPDTO",
			PluginPayloadMCPDTO{Kind: "plugin", PluginName: "p", Description: "d"},
			[]string{"kind", "data_type", "rep_name", "kyou_id", "plugin_name", "description"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalKeys(t, tt.value)
			for _, want := range tt.want {
				if !slices.Contains(got, want) {
					t.Errorf("フィールド %q が無い（MCPサーバが参照している）: got %q", want, got)
				}
			}
		})
	}
}

// TestMCPPayloadDTO_OmitsEmptyOptionalFields は omitempty の契約を固定する。
// MCPのレスポンスは1リクエストで数百件返るため、空フィールドが載ると
// max_size_mb の上限にすぐ達してしまう。
func TestMCPPayloadDTO_OmitsEmptyOptionalFields(t *testing.T) {
	tests := []struct {
		name        string
		value       any
		wantOmitted []string
	}{
		{
			"IDFPayloadMCPDTO",
			IDFPayloadMCPDTO{Kind: "idf", FileName: "data.bin", RepName: "files_repo"},
			[]string{"mime_type", "is_zip"},
		},
		{
			"MiPayloadMCPDTO",
			MiPayloadMCPDTO{Kind: "mi", Title: "タスク"},
			[]string{"board_name", "limit_time", "estimate_start_time", "estimate_end_time"},
		},
		{
			"MiReKyouPayloadMCPDTO",
			MiReKyouPayloadMCPDTO{Kind: "mirekyou", TargetID: "target-1"},
			[]string{"board_name", "limit_time", "estimate_start_time", "estimate_end_time"},
		},
		{
			"URLogPayloadMCPDTO",
			URLogPayloadMCPDTO{Kind: "urlog", Title: "ページ", URL: "https://example.com"},
			[]string{"description"},
		},
		{
			"PluginPayloadMCPDTO",
			PluginPayloadMCPDTO{Kind: "plugin", DataType: "claude_code_message", RepName: "Claude Code", KyouID: "id-1"},
			[]string{"plugin_name", "description"},
		},
		{
			"TimeIsPayloadMCPDTO",
			TimeIsPayloadMCPDTO{Kind: "timeis", Title: "作業", StartTime: time.Now()},
			[]string{"end_time"},
		},
		{
			"KyouMCPDTO",
			KyouMCPDTO{DataType: "kmemo", RelatedTime: time.Now()},
			[]string{"id", "rep_name", "tags", "texts", "notifications", "timeis", "payload"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := marshalKeys(t, tt.value)
			for _, omitted := range tt.wantOmitted {
				if slices.Contains(got, omitted) {
					t.Errorf("空の %q が出力されている（omitempty が外れている）: got %q", omitted, got)
				}
			}
		})
	}
}

// TestKyouMCPDTO_CarriesPluginPayload は KyouMCPDTO.Payload が any 型でも
// 具体的なペイロードがそのままネストして出ることを確認する。
// MCPクライアントは payload.kind を見て分岐するので、ここが崩れると
// gkill_get_plugin_content に渡す rep_name / kyou_id が取れなくなる。
func TestKyouMCPDTO_CarriesPluginPayload(t *testing.T) {
	dto := KyouMCPDTO{
		DataType:    "claude_code_message",
		RelatedTime: time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC),
		Payload: PluginPayloadMCPDTO{
			Kind:     "plugin",
			DataType: "claude_code_message",
			RepName:  "Claude Code",
			KyouID:   "id-1",
		},
	}

	data, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var raw struct {
		DataType string              `json:"data_type"`
		Payload  PluginPayloadMCPDTO `json:"payload"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if raw.DataType != "claude_code_message" {
		t.Errorf("data_type = %q, want %q", raw.DataType, "claude_code_message")
	}
	if raw.Payload.Kind != "plugin" {
		t.Errorf("payload.kind = %q, want %q", raw.Payload.Kind, "plugin")
	}
	if raw.Payload.RepName != "Claude Code" {
		t.Errorf("payload.rep_name = %q, want %q", raw.Payload.RepName, "Claude Code")
	}
	if raw.Payload.KyouID != "id-1" {
		t.Errorf("payload.kyou_id = %q, want %q", raw.Payload.KyouID, "id-1")
	}
}
