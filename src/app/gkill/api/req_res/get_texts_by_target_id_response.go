// ˅
package req_res

// ˄

type GetTextsByTargetIDResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Texts []*Text `json:"texts"`

	// ˅

	// ˄
}

// ˅

// ˄
