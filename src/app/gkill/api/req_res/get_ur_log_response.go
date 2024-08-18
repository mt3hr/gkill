// ˅
package req_res

// ˄

type GetURLogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	URLogHistories []*URLog `json:"urlog_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
