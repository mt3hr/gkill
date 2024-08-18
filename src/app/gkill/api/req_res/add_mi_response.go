// ˅
package req_res

// ˄

type AddMiResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedMi *Mi `json:"added_mi"`

	AddedMiKyou *Kyou `json:"added_mi_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
