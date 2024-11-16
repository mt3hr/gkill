package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateApplicationConfigRequest struct {
	SessionID string `json:"session_id"`

	ApplicationConfig user_config.ApplicationConfig `json:"application_config"`
}
