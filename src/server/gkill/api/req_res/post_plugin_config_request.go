package req_res

type PostPluginConfigRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`

	// TargetUserID は設定を書き換える対象の利用者。空なら自分自身。
	// このAPIは空でも管理者だけが通る。
	TargetUserID string `json:"target_user_id"`
	// RepName はKyouのrep_name。フロントはKyou.rep_nameをそのまま渡す。
	RepName  string            `json:"rep_name"`
	FormData map[string]string `json:"form_data"`
}
