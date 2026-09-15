package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

type GetApplicationConfigResponse struct {
	Messages message.GkillMessages `json:"messages"`

	Errors message.GkillErrors `json:"errors"`

	ApplicationConfig *user_config.ApplicationConfig `json:"application_config"`
}
