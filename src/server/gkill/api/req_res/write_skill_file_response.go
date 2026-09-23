package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type WriteSkillFileResponse struct {
	Messages message.GkillMessages `json:"messages"`
	Errors   message.GkillErrors   `json:"errors"`
	Path     string                `json:"path"`
	Revision string                `json:"revision"`
}
