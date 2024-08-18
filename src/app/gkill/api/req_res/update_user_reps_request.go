// ˅
package req_res

// ˄

type UpdateUserRepsRequest struct {
	// ˅

	// ˄

	SessionID string `json:"session_id"`

	TargetUserID string `json:"target_user_id"`

	UpdatedReps []*Repository `json:"updated_reps"`

	// ˅

	// ˄
}

// ˅

// ˄
