// ˅
package req_res

// ˄

type AddNlogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedNlog *Nlog `json:"added_nlog"`

	AddedNlogKyou *Kyou `json:"added_nlog_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
