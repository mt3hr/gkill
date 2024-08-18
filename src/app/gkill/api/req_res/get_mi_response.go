// ˅
package req_res

// ˄

type GetMiResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	MiHistories []*Mi `json:"mi_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
