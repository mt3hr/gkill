// ˅
package req_res

// ˄

type GetLantanasResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Lantanas []*Lantana `json:"lantanas"`

	// ˅

	// ˄
}

// ˅

// ˄
