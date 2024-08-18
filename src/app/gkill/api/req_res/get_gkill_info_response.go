// ˅
package req_res

// ˄

type GetGkillInfoResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	// ˅

	// ˄
}

// ˅

// ˄
