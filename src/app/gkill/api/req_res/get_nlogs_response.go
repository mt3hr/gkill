// ˅
package req_res

// ˄

type GetNlogsResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Nlogs []*Nlog `json:"nlogs"`

	// ˅

	// ˄
}

// ˅

// ˄
