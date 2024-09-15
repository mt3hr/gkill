package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateRepTypeStructRequest struct {
	SessionID string `json:"session_id"`

	RepTypeStruct []*user_config.RepTypeStruct `json:"rep_type_struct"`
}
