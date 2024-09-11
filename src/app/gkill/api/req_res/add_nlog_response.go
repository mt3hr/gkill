// ˅
package req_res

import (
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

// ˄

type AddNlogResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	AddedNlog *reps.Nlog `json:"added_nlog"`

	AddedNlogKyou *reps.Kyou `json:"added_nlog_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
