// ˅
package req_res

// ˄

type GetReKyousResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	ReKyous []*ReKyou `json:"rekyous"`

	// ˅

	// ˄
}

// ˅

// ˄
