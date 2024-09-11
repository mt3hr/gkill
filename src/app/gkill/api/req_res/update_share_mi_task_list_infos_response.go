// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"
)

// ˄

type UpdateShareMiTaskListInfosResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage

	Errors []*message.GkillError

	ShareMiTaskListInfos *mi_share_info.MiShareInfo

	// ˅

	// ˄
}

// ˅

// ˄
