// ˅
package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/message"

// ˄

type LoginResponse struct {
	// ˅

	// ˄

	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	SessionID string `json:"session_id"`

	// ˅

	// ˄
}

// ˅

// ˄
