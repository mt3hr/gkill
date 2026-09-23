package req_res

type UploadSkillRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	// ZipBase64 は zip の中身。data URI の接頭辞（data:application/zip;base64,）が付いていてもよい。
	ZipBase64 string `json:"zip_base64"`
	// DryRun が true なら置き換えずに、何が起きるかだけを返す（画面の確認の1段目）。
	DryRun bool `json:"dry_run"`
}
