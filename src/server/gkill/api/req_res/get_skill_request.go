package req_res

type GetSkillRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	Name       string `json:"name"`
	// Path を省くと SKILL.md の全文とファイル一覧、指定するとそのファイルの中身を返す。
	Path string `json:"path"`
	// MaxBytes が正でファイルがそれより大きいときは中身を返さない（AI へ渡す量を抑えるため）。
	MaxBytes int64 `json:"max_bytes"`
}
