// ˅
package req_res

// ˄

type GetKFTLTemplateResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	KFTLTemplates []*KFTLTemplate `json:"kftl_templates"`

	// ˅

	// ˄
}

// ˅

// ˄
