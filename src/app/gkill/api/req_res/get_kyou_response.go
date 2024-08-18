// ˅
package req_res

// ˄

type GetKyouResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	KyouHistories []*Kyou `json:"kyou_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
