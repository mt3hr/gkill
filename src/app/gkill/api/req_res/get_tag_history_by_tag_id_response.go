// ˅
package req_res

// ˄

type GetTagHistoryByTagIDResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	TagHistories []*Tag `json:"tag_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
