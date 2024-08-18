// ˅
package req_res

// ˄

type GetSharedMiTasksResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	MiKyous []*Kyou `json:"mi_kyous"`

	// ˅

	// ˄
}

// ˅

// ˄
