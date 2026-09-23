package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type GetSkillResponse struct {
	Messages message.GkillMessages `json:"messages"`
	Errors   message.GkillErrors   `json:"errors"`
	// Skill は path を省いたときに入る。
	Skill *SkillDetail `json:"skill"`
	// File は path を指定したときに入る。
	File *SkillFileContent `json:"file"`
}
