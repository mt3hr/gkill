// ˅
package req_res

// ˄

type GetNlogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	NlogHistories []*Nlog `json:"nlog_histories"`

	// ˅

	// ˄
}

// ˅

// ˄
