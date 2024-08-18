// ˅
package req_res

// ˄

type GetReKyouResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	ReKyouHistories []*ReKyou `json:"rekyou_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
