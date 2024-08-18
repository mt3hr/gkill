// ˅
package req_res

// ˄

type GetAllTagNamesResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	TagNames []*string `json:"tag_names"`

	// ˅

	// ˄
}

// ˅

// ˄
