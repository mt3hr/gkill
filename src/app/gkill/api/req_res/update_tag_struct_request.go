package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateTagStructRequest struct {
	SessionID string `json:"session_id"`

	TagStruct []*user_config.TagStruct `json:"tag_struct"`
}
