package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/server_config"

type UpdateServerConfigRequest struct {
	SessionID string `json:"session_id"`

	ServerConfig server_config.ServerConfig `json:"server_config"`
}
