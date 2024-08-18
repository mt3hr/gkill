// ˅
package req_res

// ˄

type UpdateURLogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UpdatedURLog *URLog `json:"updated_urlog"`

	UpdatedURLogKyou *Kyou `json:"updated_urlog_kyou"`

	// ˅

	// ˄
}

// ˅

// ˄
