package req_res

import "time"

type URLogBookmarkletRequest struct {
	// URL
	URL string `json:"url"`

	// ページタイトル
	Title string `json:"title"`

	// ページのDescription
	Description string `json:"description"`

	// ページのfaviconのURL
	FaviconURL string `json:"favicon_url"`

	// ページに関連したImageのURL
	ImageURL string `json:"image_url"`

	// 記録された日時
	Time time.Time `json:"time"`

	SessionID string `json:"session_id"`
}
