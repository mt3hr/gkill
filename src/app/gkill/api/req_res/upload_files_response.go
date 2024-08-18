// ˅
package req_res

// ˄

type UploadFilesResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	UploadedKyous []*Kyou `json:"uploaded_kyous"`

	// ˅

	// ˄
}

// ˅

// ˄
