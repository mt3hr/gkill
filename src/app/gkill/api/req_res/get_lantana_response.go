// ˅
package req_res

// ˄

type GetLantanaResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	LantanaHistories []*Lantana `json:"lantana_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
