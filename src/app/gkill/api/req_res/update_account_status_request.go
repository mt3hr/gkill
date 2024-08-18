// ˅
package req_res

// ˄

type UpdateAccountStatusRequest struct {
	// ˅

	// ˄

	SessionID string `json:"session_id"`

	TargetUserID string `json:"target_user_id"`

	Enable bool `json:"enable"`

	// ˅

	// ˄
}

// ˅

// ˄
