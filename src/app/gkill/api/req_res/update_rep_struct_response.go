// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
)

// ˄

type UpdateRepStructResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	ApplicationConfig *user_config.ApplicationConfig `json:"application_config"`

	// ˅

	// ˄
}

// ˅

// ˄
