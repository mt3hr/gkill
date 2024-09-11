// ˅
package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

// ˄

type AddTimeIsRequest struct {
	// ˅

	// ˄

	SessionID string `json:"session_id"`

	TimeIs *reps.TimeIs `json:"time_is"`

	// ˅

	// ˄
}

// ˅

// ˄
