// ˅
package req_res

// ˄

type UpdateTextResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedText *Text `json:"updated_text"`

	// ˅

	// ˄
}

// ˅

// ˄
