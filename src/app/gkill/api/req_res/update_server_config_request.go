package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/server_config"

type UpdateServerConfigsRequest struct {
	SessionID string `json:"session_id"`

	ServerConfigs []*server_config.ServerConfig `json:"server_configs"`
}
