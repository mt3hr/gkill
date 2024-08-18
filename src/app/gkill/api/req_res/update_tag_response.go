// ˅
package req_res

// ˄

type UpdateTagResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedTag *Tag `json:"updated_tag"`

	// ˅

	// ˄
}

// ˅

// ˄
