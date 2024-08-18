// ˅
package req_res

// ˄

type GetURLogsResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	URLogs []*URLog `json:"urlogs"`

	// ˅

	// ˄
}

// ˅

// ˄
