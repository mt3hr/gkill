package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type UploadSkillResponse struct {
	Messages message.GkillMessages `json:"messages"`
	Errors   message.GkillErrors   `json:"errors"`
	Plan     *SkillReplacePlan     `json:"plan"`
	// Applied は実際に置き換えたか（dry_run なら false）。
	Applied bool `json:"applied"`
}
