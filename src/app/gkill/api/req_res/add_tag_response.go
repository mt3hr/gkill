// ˅
package req_res

// ˄

type AddTagResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedTag *Tag `json:"added_tag"`

	// ˅

	// ˄
}

// ˅

// ˄
