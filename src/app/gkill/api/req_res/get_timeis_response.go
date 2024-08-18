// ˅
package req_res

// ˄

type GetTimeisResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	TimeisHistories []*TimeIs `json:"timeis_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
