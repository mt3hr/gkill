package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateRepStructRequest struct {
	SessionID string `json:"session_id"`

	RepStruct []*user_config.RepStruct `json:"rep_struct"`
}
