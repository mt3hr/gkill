package req_res

type GetPluginListRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`

	// TargetUserID はプラグインを見る対象の利用者。空なら自分自身。
	// 空でない指定は管理者だけが通る（アカウント管理画面からの代行用）。
	TargetUserID string `json:"target_user_id"`
}
