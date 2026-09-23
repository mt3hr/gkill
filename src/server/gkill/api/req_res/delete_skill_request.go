package req_res

type DeleteSkillRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	Name       string `json:"name"`
	// Path を省くとスキルを丸ごと消す（画面だけが使う）。指定するとそのファイルだけを消す。
	Path string `json:"path"`
	// Revision はファイルを消すときの楽観ロック（省略可）。
	Revision string `json:"revision"`
}
