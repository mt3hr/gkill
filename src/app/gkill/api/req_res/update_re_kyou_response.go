// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type UpdateReKyouResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedReKyou *reps.ReKyou `json:"updated_rekyou"`

	UpdatedReKyouKyou *reps.Kyou `json:"updated_rekyou_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
