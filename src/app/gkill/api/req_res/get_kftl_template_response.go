package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
)

type GetKFTLTemplateResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	KFTLTemplates []*user_config.KFTLTemplate `json:"kftl_templates"`
}
