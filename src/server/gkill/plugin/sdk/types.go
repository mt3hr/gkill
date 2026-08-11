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

import (
	"encoding/json"
	"time"
)

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

	// Typed はこの記録の型別データ。manifest.json の provides に書いた種別だけが採用される。
	Typed *TypedData `json:"typed,omitempty"`

	// Notifications はこの記録に付ける通知。provides に "notification" が要る。
	Notifications []Notification `json:"notifications,omitempty"`
}

// TypedData はKyouに載せる型別データ。非nilにしてよいのは高々1つ。
//
// ID・各種時刻・RepName・DataTypeは持たない。親のKyouからコピーされる。
// gkillのクライアントは「KyouのUpdateTimeと型別データのUpdateTimeが
// 秒精度で一致する版」を選ぶので、別々の更新時刻は持たせられない。
type TypedData struct {
	Kmemo   *Kmemo   `json:"kmemo,omitempty"`
	KC      *KC      `json:"kc,omitempty"`
	URLog   *URLog   `json:"urlog,omitempty"`
	Nlog    *Nlog    `json:"nlog,omitempty"`
	Lantana *Lantana `json:"lantana,omitempty"`
	TimeIs  *TimeIs  `json:"timeis,omitempty"`
	Mi      *Mi      `json:"mi,omitempty"`
}

// Kmemo はテキストメモの本文。data_typeは"kmemo"にすること。
type Kmemo struct {
	Content string `json:"content"`
}

// KC は数値記録のタイトルと数値。data_typeは"kc"にすること。
type KC struct {
	Title    string      `json:"title"`
	NumValue json.Number `json:"num_value"`
}

// URLog はブックマーク。data_typeは"urlog"にすること。
// 画像はbase64（data URIのプレフィックスは付けない）。
type URLog struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FaviconImage   string `json:"favicon_image,omitempty"`
	ThumbnailImage string `json:"thumbnail_image,omitempty"`
}

// Nlog は支出記録。data_typeは"nlog"にすること。
type Nlog struct {
	Shop   string      `json:"shop"`
	Title  string      `json:"title"`
	Amount json.Number `json:"amount"`
}

// Lantana は気分値（0〜10）。data_typeは"lantana"にすること。
type Lantana struct {
	Mood int `json:"mood"`
}

// TimeIs は時間計測。EndTimeがnilなら計測中。
// data_typeは"timeis_start"（終了行を出すなら"timeis_end"）にすること。
type TimeIs struct {
	Title     string     `json:"title"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// Mi はタスク。
// data_typeは"mi_create"/"mi_check"/"mi_limit"/"mi_start"/"mi_end"のいずれかにすること。
type Mi struct {
	Title             string     `json:"title"`
	IsChecked         bool       `json:"is_checked"`
	BoardName         string     `json:"board_name"`
	LimitTime         *time.Time `json:"limit_time,omitempty"`
	EstimateStartTime *time.Time `json:"estimate_start_time,omitempty"`
	EstimateEndTime   *time.Time `json:"estimate_end_time,omitempty"`
}

// Notification はKyouに付ける通知。
type Notification struct {
	Content          string    `json:"content"`
	NotificationTime time.Time `json:"notification_time"`
	IsNotificated    bool      `json:"is_notificated"`
}

// GPSLogQuery はgkillからプラグインへのGPSログ取得条件。期間は両端を含む。
type GPSLogQuery struct {
	// StartTime は取得期間の開始時刻。nilなら下限なし。
	StartTime *time.Time
	// EndTime は取得期間の終了時刻。nilなら上限なし。
	EndTime *time.Time
	// Offset は返し始める位置（0起点）。
	Offset int
	// Limit は返す最大点数。0ならプラグイン側の既定に任せてよい。
	Limit int
}

// GPSLog はプラグインが返すGPSログの1点。
type GPSLog struct {
	RelatedTime time.Time `json:"related_time"`
	Longitude   float64   `json:"longitude"`
	Latitude    float64   `json:"latitude"`
}

// GPSLogPage は GetGPSLogs の返り値。
type GPSLogPage struct {
	// GPSLogs はこのページぶんの点。RelatedTimeの昇順で返すこと。
	GPSLogs []GPSLog
	// HasMore は Offset+len(GPSLogs) 以降にまだ点があることを表す。
	HasMore bool
}

// Config はプラグインの設定。config.json の内容を表す。
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
