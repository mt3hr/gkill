package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateKFTLTemplateRequest struct {
	SessionID string `json:"session_id"`

	KFTLTemplates []*user_config.KFTLTemplate `json:"kftl_templates"`
}
