package req_res

type WriteSkillFileRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	// Revision を省くと新規作成だけを許す。指定すると今の中身と一致したときだけ上書きする。
	Revision string `json:"revision"`
}
