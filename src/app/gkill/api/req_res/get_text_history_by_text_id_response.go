// ˅
package req_res

// ˄

type GetTextHistoryByTextIDResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	TextHistories []*Text `json:"text_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
