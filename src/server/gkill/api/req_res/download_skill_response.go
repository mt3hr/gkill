package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

// DownloadSkillResponse はスキルの zip を base64 で返す（クライアントは JSON 以外の応答を受け付けないため）。
type DownloadSkillResponse struct {
	Messages  message.GkillMessages `json:"messages"`
	Errors    message.GkillErrors   `json:"errors"`
	FileName  string                `json:"file_name"`
	ZipBase64 string                `json:"zip_base64"`
}
