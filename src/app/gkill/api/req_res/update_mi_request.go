// ˅
package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

// ˄

type UpdateMiRequest struct {
	// ˅

	// ˄

	SessionID string `json:"session_id"`

	Mi *reps.Mi `json:"mi"`

	// ˅

	// ˄
}

// ˅

// ˄
