// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
)

// ˄

type GetServerConfigResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	ServerConfig *server_config.ServerConfig `json:"server_config"`

	// ˅

	// ˄
}

// ˅

// ˄
