package req_res

// DeleteKyouCascadeRequest は記録を連鎖して論理削除する要求です（ADR-0636）。
type DeleteKyouCascadeRequest struct {
	SessionID string `json:"session_id"`

	// TargetID は起点の記録の ID です。
	TargetID string `json:"target_id"`

	// DataType は起点の記録の種別です。射影名（mi_create / timeis_start / mirekyou_create …）でもかまいません。
	DataType string `json:"data_type"`

	// UpdateApp は削除の版に刻むアプリ名です。空なら "gkill"（Web の画面）。MCP は自分のアプリ名を送ります。
	UpdateApp string `json:"update_app"`

	// UpdateDevice は削除の版に刻む端末名です。空ならサーバの端末名（Web の画面）。MCP は "mcp" を送ります。
	// temp rep のキーには使いません（削除の版の update_device に刻むだけ）。
	UpdateDevice string `json:"update_device,omitempty"`

	LocaleName string `json:"locale_name"`
}
