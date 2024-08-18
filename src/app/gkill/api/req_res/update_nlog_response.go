// ˅
package req_res

// ˄

type UpdateNlogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedNlog *Nlog `json:"updated_nlog"`

	UpdatedNlogKyou *Kyou `json:"updated_nlog_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
