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
	RepName string `json:"rep_name"`
	// CreateApp / UpdateApp は書いたアプリ・最後に更新したアプリ。
	// create_apps / update_apps で絞れるようにした以上、結果からも読めないと
	// 呼び出し側が絞り込みの結果を検証できない（omitempty は付けない。
	// 空と「フィールドが無い」が区別できなくなるため）。
	CreateApp   string    `json:"create_app"`
	UpdateApp   string    `json:"update_app"`
	RelatedTime time.Time `json:"related_time"`
	// IsDeleted は削除済みのときだけ載る（omitempty）。query.include_deleted_data で削除済みを
	// 混ぜたときに「どれが削除済みか」を見分けるための欄で、既定の検索では全件 false なので
	// 毎件 `"is_deleted":false` が定常のオーバーヘッドになっていた（2026-09-18 の実利用報告）。
	// 「無ければ生きている」と読ませる（外部監査 C4 の判断を is_deleted に限って覆す。ADR-0629。
	// IsZip / Addition / Deletion は据え置き —— あちらは false / 0 に「値が取れなかった」と紛れる意味がある）。
	// UpdateTime は「どちらが新しいか」の判別に要るので omitempty を付けない。
	IsDeleted     bool                 `json:"is_deleted,omitempty"`
	UpdateTime    time.Time            `json:"update_time"`
	Tags          []string             `json:"tags,omitempty"`
	Texts         []string             `json:"texts,omitempty"`
	Notifications []NotificationMCPDTO `json:"notifications,omitempty"`
	TimeIs        []TimeIsMCPDTO       `json:"timeis,omitempty"`
	// TagEntities / TextEntities は Tags / Texts と同じ内容を ID 付きで返す枠。順序も同じ。
	// 既存の Tags / Texts を置き換えないのは、Web の列と Wear OS が []string を前提に
	// しているため（ワイヤ互換）。**リクエストの include_attached_ids:true のときだけ組む**
	// （既定では二重持ちが毎件並ぶだけだった。ADR-0629）。詳細は AttachedEntityMCPDTO のコメント。
	TagEntities  []AttachedEntityMCPDTO `json:"tag_entities,omitempty"`
	TextEntities []AttachedEntityMCPDTO `json:"text_entities,omitempty"`
	Payload      any                    `json:"payload,omitempty"`
}

// TimeIsMCPDTO は attached TimeIs（Playing TimeIs）用DTO。
//
// ID と時刻が要る。以前は Title と Tags しか無く、同じ題名の打刻が
// 1つの応答に何度も並んでも**区別も特定もできなかった**（実測で lantana 3件に対し
// 付随 TimeIs が90件、うち同題名が4回）。「記録時に何が走っていたか」を知る機能なのに
// 時刻が無いため、実質「その日に存在した打刻の題名一覧」になっていた
// （2026-08-25 の実利用レビュー）。
type TimeIsMCPDTO struct {
	// ID は打刻の実体を引くための識別子。gkill_get_kyou_history や
	// query.ids へそのまま渡せる。
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
	// StartTime / EndTime は打刻の期間。EndTime が無いものは計測中。
	// 応答の他の時刻と同じくローカルタイムゾーンへ揃える。
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// AttachedEntityMCPDTO は付随データ（タグ・テキスト）の実体1件。
//
// tags / texts は表示用の文字列配列で、**注釈を後から直す・消すのに必要な ID を運べない**。
// gkill_update_text と gkill_delete_kyou(data_type:"text") は text 自身の ID を要求するので、
// add_text の応答を手元に持っていない限り編集も削除もできなかった
// （2026-08-25 の実利用レビュー）。同じレビューで TimeIs にだけ ID を足したので、残りも揃える。
type AttachedEntityMCPDTO struct {
	// ID は付随データ自身の識別子。**親の Kyou の ID とは別物**で、
	// gkill_update_text / gkill_delete_kyou / gkill_get_kyou_history へ渡すのはこちら。
	ID string `json:"id"`
	// Value はタグ名またはテキスト本文。Tags / Texts に入るのと同じ文字列。
	Value string `json:"value"`
}

type NotificationMCPDTO struct {
	// ID は通知自身の識別子。gkill_delete_kyou(data_type:"notification") と
	// gkill_get_kyou_history はこれを要求するのに応答へ載っておらず、
	// **MCP から通知の id を得る経路が1つも無かった**
	// （notification は data_type フィルタにも group_by のバケットにも出ない）。
	ID               string    `json:"id"`
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
// コンテンツHTMLとして取り出すしかない。取得に要る rep_name / id は Kyou 側（KyouMCPDTO の
// rep_name / id）にあり、ここへ写すと毎件3欄が二重に並ぶだけだった（data_type も同じ。
// 2026-09-18 の実利用報告。ADR-0629）。MCP の inlinePluginContents は Kyou 側の欄を読む。
type PluginPayloadMCPDTO struct {
	Kind       string `json:"kind"` // "plugin"
	PluginName string `json:"plugin_name,omitempty"`
	// Description はここには入れない。プラグインの説明文は130〜150字あり、
	// Kyou 1件ごとに焼き込むと20件取るだけで同じ文が20回並ぶ
	// （include_plugin_content とは無関係に常に載っていた。2026-08-25 の実利用レビュー）。
	// 応答トップレベルの Plugins（PluginDescriptionMCPDTO）に rep_name ごと1回だけ出す。
}

// PluginDescriptionMCPDTO は応答に出てきたプラグインの説明を1回だけ載せる枠。
// payload.rep_name から引く。plugin_content ブロックと同じ「まとめて末尾」の形。
type PluginDescriptionMCPDTO struct {
	RepName     string `json:"rep_name"`
	PluginName  string `json:"plugin_name,omitempty"`
	Description string `json:"description,omitempty"`
}

// GitPayloadMCPDTO はコミットのペイロード。コミットハッシュは Kyou の id そのもの
// （git_commit_log_repository_local_dir_impl.go の kyou.ID = commit.Hash.String()）なので、
// ここには持たない（外部監査 C2 で足した commit_hash は id と常に同値で、毎件 40 桁が二重に並ぶだけだった。
// 2026-09-18 の実利用報告。ADR-0629）。
type GitPayloadMCPDTO struct {
	Kind          string `json:"kind"` // "git_commit_log"
	CommitMessage string `json:"commit_message"`
	// Addition / Deletion に omitempty を付けてはいけない:
	// int の 0 はキーごと消え、「差分0行のコミット」と「値が取れなかった」を
	// 呼び出し側が区別できなくなる（外部監査 C5）。
	Addition int `json:"addition"`
	Deletion int `json:"deletion"`
}
