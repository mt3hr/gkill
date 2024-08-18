// ˅
package req_res

// ˄

type GetTagsByTargetIDResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Tags []*Tag `json:"tags"`

	// ˅

	// ˄
}

// ˅

// ˄
