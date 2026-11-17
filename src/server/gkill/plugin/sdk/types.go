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

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// Query はgkillからプラグインへの検索条件。
//
// Words〜Limit は昔からある欄。gkill が受けた検索条件の全体は FindQuery に入っており（ADR-0318）、
// そのうちよく使うものを IDs / Reps / IncludeEndTimeIs に写してある。
// 古い gkill（FindQuery を送らない版）からは FindQuery が nil で、写した欄もゼロ値（IDs・Reps は nil）になる。
//
// gkill は返ってきた Kyou を IDs・期間・IncludeEndTimeIs で必ず絞り直すので、
// プラグインがこれらを見ずに多めに返しても結果は変わらない（読む量が増えるだけ）。
// 見るなら、IDs は MatchID、Reps は MatchRep で判定すると nil（指定なし）と [] （0件）を取り違えない。
type Query struct {
	Words             []string
	NotWords          []string
	WordsAnd          bool
	Tags              []string
	NotTags           []string
	TagsAnd           bool
	CalendarStartDate *time.Time
	CalendarEndDate   *time.Time
	// IsDeleted は「削除済み**だけ**を対象にする」。gkill 本体からは常に false。
	// FindQuery.IncludeDeletedData（削除済みも含める）とは意味が違うので混ぜないこと。
	IsDeleted      bool
	OnlyLatestData bool
	Limit          int

	// WordsSkipIDMatch は true のとき、肯定語（Words）で Kyou の ID の前方一致を見ない。
	// gkill が除外語を肯定語として引き直すときに立つ。MatchText / Matcher が自動で効かせる。
	WordsSkipIDMatch bool

	// IDs は FindQuery.IDs の写し。nil は指定なし、[] は0件。
	IDs []string
	// Reps は FindQuery.Reps の写し（rep 名）。nil は指定なし、[] は0件。
	// 複数の rep 名を返すプラグインが、読む rep を絞るのに使える。
	Reps []string
	// IncludeEndTimeIs は FindQuery.IncludeEndTimeIs の写し。true のとき、終了した TimeIs の
	// 終了の Kyou（data_type "timeis_end"）も返す（TimeIs の説明を参照）。
	IncludeEndTimeIs bool

	// FindQuery は gkill が受けた検索条件の全体。読むだけにすること。古い gkill からは nil。
	FindQuery *find.FindQuery
}

// MatchID は id が Query.IDs の指定に入るかを返す。IDs が nil（指定なし）なら常に真、[] なら常に偽。
// IDs を毎回なめるので、一覧をループで判定するときは IDMatcher を使う。
func (q Query) MatchID(id string) bool {
	if q.IDs == nil {
		return true
	}
	for _, want := range q.IDs {
		if want == id {
			return true
		}
	}
	return false
}

// IDMatcher は Query.IDs を集合にして持ち、多くの ID を判定するのに使う。意味は Query.MatchID と同じ。
type IDMatcher struct {
	// ids が nil なら指定なし（常に真）。
	ids map[string]struct{}
}

// IDMatcher はこの Query の IDs から IDMatcher を作る。
func (q Query) IDMatcher() IDMatcher {
	if q.IDs == nil {
		return IDMatcher{}
	}
	ids := make(map[string]struct{}, len(q.IDs))
	for _, id := range q.IDs {
		ids[id] = struct{}{}
	}
	return IDMatcher{ids: ids}
}

// Match は id が IDs の指定に入るかを返す。
func (m IDMatcher) Match(id string) bool {
	if m.ids == nil {
		return true
	}
	_, ok := m.ids[id]
	return ok
}

// MatchRep は rep 名が Query.Reps の指定に入るかを返す。Reps が nil（指定なし）なら常に真、[] なら常に偽。
func (q Query) MatchRep(repName string) bool {
	if q.Reps == nil {
		return true
	}
	for _, want := range q.Reps {
		if want == repName {
			return true
		}
	}
	return false
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
	// GitCommitLog は Git のコミット。data_typeは"git_commit_log"にすること。
	GitCommitLog *GitCommitLog `json:"git_commit_log,omitempty"`
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
// data_typeは"timeis_start"にすること。終了した TimeIs は、Query.IncludeEndTimeIs が true のときだけ、
// 同じ ID・同じ TimeIs で data_type "timeis_end"・RelatedTime を終了時刻にした Kyou も返す（native と同じ）。
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

// GitCommitLog は Git のコミット1件（コミットメッセージ・追加行数・削除行数）。
// data_typeは"git_commit_log"にすること。ID はコミットハッシュ、
// RelatedTime / CreateTime / UpdateTime はコミッタ日時、CreateUser / UpdateUser は author 名にすると
// native の git rep と同じ形になり、同じコミットが稼働中リポジトリにもあれば1件に畳まれる。
type GitCommitLog struct {
	CommitMessage string `json:"commit_message"`
	Addition      int    `json:"addition"`
	Deletion      int    `json:"deletion"`
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
