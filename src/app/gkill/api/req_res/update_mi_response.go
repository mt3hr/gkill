// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type UpdateMiResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	UpdatedMi *reps.Mi `json:"updated_mi"`

	UpdatedMiKyou *reps.Kyou `json:"updated_mi_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
