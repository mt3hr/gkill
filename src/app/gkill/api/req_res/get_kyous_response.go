// ˅
package req_res

// ˄

type GetKyousResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	Kyous []*Kyou `json:"kyous"`

	// ˅

	// ˄
}

// ˅

// ˄
