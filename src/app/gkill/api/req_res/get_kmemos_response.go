// ˅
package req_res

// ˄

type GetKmemosResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Kmemos []*Kmemo `json:"kmemos"`

	// ˅

	// ˄
}

// ˅

// ˄
