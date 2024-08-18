// ˅
package req_res

// ˄

type AddTimeIsResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedTimeis *TimeIs `json:"added_timeis"`

	AddedTimeisKyou *Kyou `json:"added_timeis_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
