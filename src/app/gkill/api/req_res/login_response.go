// ˅
package req_res

// ˄

type LoginResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	SessionID string `json:"session_id"`

	// ˅

	// ˄
}

// ˅

// ˄
