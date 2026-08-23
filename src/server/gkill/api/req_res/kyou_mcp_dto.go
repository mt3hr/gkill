package req_res

import (
	"encoding/json"
	"time"
)

// KyouMCPDTO は MCP レスポンス用の軽量 Kyou DTO。
//
// ID / RepName は v2 から常時付与する。AIクライアントの追撃クエリ
// (query.ids / query.reps / プラグイン本文取得 / 更新系ツール) は両方を前提とし、
// 旧 v1 の「要求フラグを立てないと載らない」既定が往復を1回増やしていた（外部監査 C1 ほか）。
// サイズ増(1件あたり数十バイト)は v2 で max_size_mb が厳密上限になったので上限自体が吸収する。
type KyouMCPDTO struct {
	ID       string `json:"id"`
	DataType string `json:"data_type"`
	// RepName はKyouの取得元リポジトリ名。query.reps 絞り込みの起点になる。
	RepName     string    `json:"rep_name"`
	RelatedTime time.Time `json:"related_time"`
	// IsDeleted / UpdateTime は query.include_deleted_data で削除済みを混ぜたときに
	// 「どれが削除済みか」「どちらが新しいか」を判別するために要る。
	// omitempty は付けない。false / ゼロ値のときに黙って消えると、
	// 「生きている」と「フィールドが無い」の区別が付かなくなる
	// （IsZip / Addition / Deletion と同じ理由。外部監査 C4 / C5）。
	IsDeleted     bool                 `json:"is_deleted"`
	UpdateTime    time.Time            `json:"update_time"`
	Tags          []string             `json:"tags,omitempty"`
	Texts         []string             `json:"texts,omitempty"`
	Notifications []NotificationMCPDTO `json:"notifications,omitempty"`
	TimeIs        []TimeIsMCPDTO       `json:"timeis,omitempty"`
	Payload       any                  `json:"payload,omitempty"`
}

// TimeIsMCPDTO は attached TimeIs（Plaing TimeIs）用DTO
type TimeIsMCPDTO struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

type NotificationMCPDTO struct {
	Content          string    `json:"content"`
	NotificationTime time.Time `json:"notification_time,omitempty"`
	IsNotificated    bool      `json:"is_notificated"`
}

type TimeIsPayloadMCPDTO struct {
	Kind      string     `json:"kind"` // "timeis"
	Title     string     `json:"title"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

type KmemoPayloadMCPDTO struct {
	Kind    string `json:"kind"` // "kmemo"
	Content string `json:"content"`
}

type KCPayloadMCPDTO struct {
	Kind     string      `json:"kind"` // "kc"
	Title    string      `json:"title"`
	NumValue json.Number `json:"num_value"`
}

// URLogPayloadMCPDTO はブックマークのペイロード。
// FaviconImage / ThumbnailImage はbase64画像で巨大なうえAIクライアントで
// 使えないため載せない（ThumbnailImageはDBから読む段階でも外している）。
type URLogPayloadMCPDTO struct {
	Kind        string `json:"kind"` // "urlog"
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type NlogPayloadMCPDTO struct {
	Kind   string      `json:"kind"` // "nlog"
	Title  string      `json:"title"`
	Shop   string      `json:"shop,omitempty"`
	Amount json.Number `json:"amount"`
}

// MiPayloadMCPDTO はタスクのペイロード。
// CreateTimeを持つのは、Kyou側のrelated_timeがdata_typeの射影ごとに
// 意味が変わる（mi_limitなら期限、mi_startなら開始予定…）ため、
// ペイロードに入れないと作成日時を復元できないから。
type MiPayloadMCPDTO struct {
	Kind              string     `json:"kind"` // "mi"
	Title             string     `json:"title"`
	IsChecked         bool       `json:"is_checked"`
	BoardName         string     `json:"board_name,omitempty"`
	CreateTime        time.Time  `json:"create_time"`
	LimitTime         *time.Time `json:"limit_time,omitempty"`
	EstimateStartTime *time.Time `json:"estimate_start_time,omitempty"`
	EstimateEndTime   *time.Time `json:"estimate_end_time,omitempty"`
}

// MiReKyouPayloadMCPDTO は既存Kyouをタスク化したもの（MiReKyou）のペイロード。
// MiReKyou自身はタイトルを持たず、表示はTargetIDの指すKyouをそのまま描くので、
// 中身が要るときはTargetIDをquery.idsに渡して引き直す。
type MiReKyouPayloadMCPDTO struct {
	Kind              string     `json:"kind"` // "mirekyou"
	TargetID          string     `json:"target_id"`
	IsChecked         bool       `json:"is_checked"`
	BoardName         string     `json:"board_name,omitempty"`
	CreateTime        time.Time  `json:"create_time"`
	LimitTime         *time.Time `json:"limit_time,omitempty"`
	EstimateStartTime *time.Time `json:"estimate_start_time,omitempty"`
	EstimateEndTime   *time.Time `json:"estimate_end_time,omitempty"`
}

// ReKyouPayloadMCPDTO は既存Kyouの再投稿（ReKyou）のペイロード。
// 本文はTargetIDの指すKyouが持つので、ここではTargetIDだけを返す。
type ReKyouPayloadMCPDTO struct {
	Kind     string `json:"kind"` // "rekyou"
	TargetID string `json:"target_id"`
}

type LantanaPayloadMCPDTO struct {
	Kind string `json:"kind"` // "lantana"
	Mood int    `json:"mood"`
}

type IDFPayloadMCPDTO struct {
	Kind     string `json:"kind"` // "idf"
	FileName string `json:"file_name"`
	IsImage  bool   `json:"is_image"`
	IsVideo  bool   `json:"is_video"`
	IsAudio  bool   `json:"is_audio"`
	// IsZip は .zip / .cbz のように中身を一覧できるアーカイブかどうか。
	// omitempty を付けてはいけない: false で消えると、スキーマに載っているのに
	// 実レスポンスに一度も現れないフィールドになり、呼び出し側は
	// 「false なのか未実装なのか」を判別できない（外部監査 C4）。
	IsZip    bool   `json:"is_zip"`
	RepName  string `json:"rep_name"`
	MimeType string `json:"mime_type,omitempty"`
	// FileSize はファイルサイズ(バイト)。include_file_size:true のリクエストで、
	// ページ内の行に対する os.Stat が成功したときだけ入る(失敗はフィールド欠落)。
	FileSize *int64 `json:"file_size,omitempty"`
	// FilePath はファイルの絶対パス。同一マシンからのリクエストのときだけ埋める。
	FilePath string `json:"file_path,omitempty"`
}

// PluginPayloadMCPDTO はプラグインが提供するKyouのペイロード。
// プラグインKyouの本文はgkill側に保存されておらず、プラグインから
// コンテンツHTMLとして取り出すしかないので、その取得に必要な
// rep_name / kyou_id をペイロードに含める（idfのrep_name/file_nameと同じ考え方）。
type PluginPayloadMCPDTO struct {
	Kind        string `json:"kind"` // "plugin"
	DataType    string `json:"data_type"`
	RepName     string `json:"rep_name"`
	KyouID      string `json:"kyou_id"`
	PluginName  string `json:"plugin_name,omitempty"`
	Description string `json:"description,omitempty"`
}

type GitPayloadMCPDTO struct {
	Kind string `json:"kind"` // "git_commit_log"
	// CommitHash はコミットハッシュ。Kyou の id と同値だが、ペイロード単体でも
	// 自明になるよう明示する(重複排除・突合の鍵。外部監査 C2)。
	CommitHash    string `json:"commit_hash"`
	CommitMessage string `json:"commit_message"`
	// Addition / Deletion に omitempty を付けてはいけない:
	// int の 0 はキーごと消え、「差分0行のコミット」と「値が取れなかった」を
	// 呼び出し側が区別できなくなる（外部監査 C5）。
	Addition int `json:"addition"`
	Deletion int `json:"deletion"`
}
