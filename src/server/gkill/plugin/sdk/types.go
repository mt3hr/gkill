// Package sdk はgkillプラグイン開発者向けのSDKを提供する。
//
// # 使い方
//
//	func main() {
//	    sdk.Run(sdk.Handler{
//	        FindKyous: func(ctx context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
//	            // 外部APIを叩いてKyouを返す
//	        },
//	        GetContentHTML: func(ctx context.Context, kyouID string, cfg sdk.Config) (string, error) {
//	            // Kyou詳細のHTMLを返す
//	        },
//	        GetConfigHTML: func(ctx context.Context, cfg sdk.Config) (string, error) {
//	            // 設定フォームのHTMLを返す
//	        },
//	        PostConfig: func(ctx context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
//	            // フォームデータを受けて設定を保存する
//	        },
//	        RepName: "MyPlugin",
//	    })
//	}
package sdk

import "time"

// Query はgkillからプラグインへの検索条件。
type Query struct {
	Words             []string
	NotWords          []string
	WordsAnd          bool
	Tags              []string
	NotTags           []string
	TagsAnd           bool
	CalendarStartDate *time.Time
	CalendarEndDate   *time.Time
	IsDeleted         bool
	OnlyLatestData    bool
	Limit             int
}

// Kyou はプラグインが返す記録データ。
type Kyou struct {
	IsDeleted    bool      `json:"is_deleted"`
	ID           string    `json:"id"`
	RepName      string    `json:"rep_name"`
	RelatedTime  time.Time `json:"related_time"`
	DataType     string    `json:"data_type"`
	CreateTime   time.Time `json:"create_time"`
	CreateApp    string    `json:"create_app"`
	CreateDevice string    `json:"create_device"`
	CreateUser   string    `json:"create_user"`
	UpdateTime   time.Time `json:"update_time"`
	UpdateApp    string    `json:"update_app"`
	UpdateDevice string    `json:"update_device"`
	UpdateUser   string    `json:"update_user"`
	ImageSource  string    `json:"image_source,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	Texts        []string  `json:"texts,omitempty"`
}

// Config はプラグインの設定。etc/config.json の内容を表す。
// キーと値の単純なマップ。
type Config map[string]any

// Get は設定から文字列値を取得する。存在しない場合はデフォルト値を返す。
func (c Config) Get(key string, defaultValue string) string {
	if c == nil {
		return defaultValue
	}
	v, ok := c[key]
	if !ok {
		return defaultValue
	}
	s, ok := v.(string)
	if !ok {
		return defaultValue
	}
	return s
}
