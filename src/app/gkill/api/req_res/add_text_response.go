// ˅
package req_res

// ˄

type AddTextResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedText *Text `json:"added_text"`

	// ˅

	// ˄
}

// ˅

// ˄
