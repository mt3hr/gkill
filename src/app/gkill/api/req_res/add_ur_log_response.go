// ˅
package req_res

// ˄

type AddURLogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	AddedURLog *URLog `json:"added_urlog"`

	AddedURLogKyou *Kyou `json:"added_urlog_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
