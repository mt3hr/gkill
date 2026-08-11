package reps

// プラグインが返した型別データ・付随データのインメモリ索引。
//
// なぜ索引が要るのか（消さないこと）:
//
//   - find_filter.getAllTags は検索1回につき全TagRepsのFindTagsを呼ぶ。
//     findTextsGeneric も全TextRepsのFindTextsを呼ぶ。
//   - handle_get_kyous_mcp は Kyou 1件ごとに GetTagsByTargetID /
//     GetTextsByTargetID / GetNotificationsByTargetID を呼ぶ。
//   - ブラウザは画面上の Kyou ごとに get_kc / get_tags_by_target_id を8並列で投げる。
//
// 一方 pluginRepositoryImpl の呼び出しは容量1のスロットで完全に直列化され、
// 順番待ちの上限は10秒（ErrPluginBusy）、実行の上限は30秒（Process.Kill）。
// アダプタのメソッドが1回でもプラグインへ往復すると、
// 15,000件の一覧をDnoteで舐めた瞬間に15,000回の直列stdio呼び出しになり、
// プラグインプロセスが殺され続ける。
//
// したがってアダプタは必ずメモリから即答する。索引を埋めるのは
// pluginRepositoryImpl.FindKyous の副作用（検索はReps経由で走り、
// goForRep がスレッドプールを迂回するのでそこでブロックしても安全）と、
// UpdateCache からの明示的な再構築だけ。

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

const (
	// pluginIndexBuildTimeout は索引の再構築1回に許す時間。
	// pluginCallTimeout(30秒)より短くして、UpdateCacheが索引の再構築だけで
	// 長時間ブロックしないようにしている。
	pluginIndexBuildTimeout = 20 * time.Second

	// pluginIndexMinRebuildInterval は再構築の最短間隔。
	// UpdateCacheはReps・TagReps・TextReps・NotificationRepsから立て続けに呼ばれるので、
	// これが無いと1回の更新でプラグインを4往復することになる。
	pluginIndexMinRebuildInterval = 30 * time.Second

	// pluginIndexTTL はスナップショットを「そろそろ古い」とみなす時間。
	// 超えていたら読み取りのついでに非同期再構築を予約する（読み取り自体は待たない）。
	pluginIndexTTL = 5 * time.Minute

	// pluginIndexMaxRecords は索引に載せる最大件数。
	// プラグインが際限なく返してきたときにヒープを食い潰さないための保険。
	pluginIndexMaxRecords = 100000
)

// pluginDerivedIDNamespace はプラグイン由来の付随データIDを導出するための名前空間UUID。
//
// この値は永久に変えないこと。変えると既存のタグ・テキスト・通知のIDが全部変わり、
// ユーザがgkill側で付けた削除や編集（同一IDの新しい版）が迷子になる。
var pluginDerivedIDNamespace = uuid.MustParse("6f9b1f8e-7c1a-4d5e-9a3b-0c2d4e6f8a10")

// pluginDerivedID は (repName, kind, targetID, value) から決定的なIDを作ります。
//
// 添字ではなく値（タグ名・本文）を混ぜるのが要点です。
//   - プラグインが返す並び順が変わってもIDが変わらない
//   - 一度消したタグを付け直すと同じIDに戻る（gkill側の削除版が効き続ける）
//   - 別プラグインの同名タグと衝突しない
func pluginDerivedID(repName string, kind string, targetID string, value string) string {
	seed := repName + "\x00" + kind + "\x00" + targetID + "\x00" + value
	return uuid.NewSHA1(pluginDerivedIDNamespace, []byte(seed)).String()
}

// pluginTypedRecord はプラグイン1件ぶんの型別データと付随データを、
// gkill本体の型に変換済みの形で保持します。
//
// Kyous が複数になるのは射影を持つ型（TimeIsのstart/end、Miの5射影）です。
// プラグインが同じIDで data_type 違いの PluginKyou を複数返したものを1レコードにまとめます。
type pluginTypedRecord struct {
	// ID は Kyou.ID。
	ID string
	// UpdateTime はこのレコードの版。型別データ・付随データすべてがこの時刻を共有します。
	UpdateTime time.Time
	// IsDeleted は削除済みフラグ。
	IsDeleted bool

	// Kyous はこのIDについてプラグインが申告したKyou（射影ぶん）。
	Kyous []Kyou

	Kmemo   *Kmemo
	KC      *KC
	URLog   *URLog
	Nlog    *Nlog
	Lantana *Lantana
	TimeIs  *TimeIs
	Mi      *Mi

	Tags          []Tag
	Texts         []Text
	Notifications []Notification
}

// pluginIndexSnapshot は索引の不変スナップショットです。
// 公開後は誰も書き換えないので、読み手はロックなしで参照できます。
type pluginIndexSnapshot struct {
	// builtAt は構築時刻。TTL判定に使います。
	builtAt time.Time
	// ok は構築が成功したかどうか。失敗時は空のスナップショットを ok=false で置きます。
	ok bool

	byID    map[string]*pluginTypedRecord
	records []*pluginTypedRecord

	tagsByTarget          map[string][]Tag
	textsByTarget         map[string][]Text
	notificationsByTarget map[string][]Notification

	tagsByID          map[string]Tag
	textsByID         map[string]Text
	notificationsByID map[string]Notification

	boardNames []string
}

// newEmptyPluginIndexSnapshot は空のスナップショットを返します。
// 未構築のときと構築に失敗したときに使います。
func newEmptyPluginIndexSnapshot(ok bool) *pluginIndexSnapshot {
	return &pluginIndexSnapshot{
		builtAt:               time.Now(),
		ok:                    ok,
		byID:                  map[string]*pluginTypedRecord{},
		records:               []*pluginTypedRecord{},
		tagsByTarget:          map[string][]Tag{},
		textsByTarget:         map[string][]Text{},
		notificationsByTarget: map[string][]Notification{},
		tagsByID:              map[string]Tag{},
		textsByID:             map[string]Text{},
		notificationsByID:     map[string]Notification{},
		boardNames:            []string{},
	}
}

// pluginIndexSource は索引が材料を取ってくる相手です。pluginRepositoryImpl が実装します。
// インターフェースにしているのは索引を単体テストできるようにするためです。
type pluginIndexSource interface {
	// indexRepName はリポジトリ表示名を返します。
	indexRepName() string
	// indexPluginName はプラグイン名を返します（ログ・警告用）。
	indexPluginName() string
	// indexProvidedKinds は manifest.provides の集合を返します。
	indexProvidedKinds() map[gkill_plugin.PluginProvidedKind]struct{}
	// indexFetchAll は全件を1回の find_kyous で取得します。
	indexFetchAll(ctx context.Context) ([]gkill_plugin.PluginKyou, error)
}

// PluginTypedIndex はプラグイン1本ぶんの型別データ・付随データのインメモリ索引です。
//
// 全アダプタ（最大10個）がこの1つを共有します。
// 読み取りメソッドは決してプラグインへ往復しません。理由はファイル冒頭を参照してください。
type PluginTypedIndex struct {
	source pluginIndexSource

	// snapshot は公開中の不変スナップショット。nilなら未構築。
	snapshot atomic.Pointer[pluginIndexSnapshot]

	// buildMu は再構築を1本に絞ります。
	buildMu sync.Mutex
	// building は非同期再構築が走行中かを表します。二重起動を防ぎます。
	building atomic.Bool
	// lastAttemptUnixNano は直近の再構築「試行」時刻。
	// 成功・失敗にかかわらず更新し、最短間隔と失敗時のバックオフの両方に使います。
	lastAttemptUnixNano atomic.Int64
}

// newPluginTypedIndex は索引を作ります。
func newPluginTypedIndex(source pluginIndexSource) *PluginTypedIndex {
	return &PluginTypedIndex{source: source}
}

// Snapshot は現在のスナップショットを返します。未構築なら空のスナップショットを返します。
// 決してブロックしません。
func (i *PluginTypedIndex) Snapshot() *pluginIndexSnapshot {
	if snapshot := i.snapshot.Load(); snapshot != nil {
		return snapshot
	}
	return newEmptyPluginIndexSnapshot(false)
}

// Ensure は索引が無ければ非同期に構築を開始し、待たずに現在のスナップショットを返します。
//
// 待たないのは、この関数が検索経路（getAllTags / findTexts）から
// 1検索につき何度も呼ばれるためです。ここで待つとプラグインの解析時間が
// そのまま全ユーザの検索レイテンシになります。
// 索引はプラグイン自身の FindKyous と GetRepositories 末尾の UpdateCache で埋まるので、
// 実運用で「1回目の検索だけタグが出ない」ことはほとんど起きません。
func (i *PluginTypedIndex) Ensure(ctx context.Context) *pluginIndexSnapshot {
	snapshot := i.Snapshot()
	if !snapshot.ok || time.Since(snapshot.builtAt) > pluginIndexTTL {
		i.kickRebuild(ctx)
	}
	return snapshot
}

// Refresh は索引を同期的に作り直します。UpdateCache から呼びます。
// 直近の試行から pluginIndexMinRebuildInterval 以内なら何もしません。
func (i *PluginTypedIndex) Refresh(ctx context.Context) error {
	if !i.shouldAttempt() {
		return nil
	}
	return i.build(ctx)
}

// NoteMissingID は索引に無いIDが引かれたことを記録し、非同期再構築を予約します。
//
// ここでプラグインに問い合わせてはいけません。1件ずつ聞きに行く実装にすると、
// 索引に無いレコードが多いとき（プラグインを入れ替えた直後など）に
// 画面の行数ぶんの直列stdio呼び出しが発生し、プラグインが殺され続けます。
func (i *PluginTypedIndex) NoteMissingID(ctx context.Context, _ string) {
	i.kickRebuild(ctx)
}

// Invalidate は次の読み取りで作り直させます。
func (i *PluginTypedIndex) Invalidate() {
	i.snapshot.Store(nil)
	i.lastAttemptUnixNano.Store(0)
}

// shouldAttempt は最短間隔を見て、いま再構築してよいかを返します。
func (i *PluginTypedIndex) shouldAttempt() bool {
	last := i.lastAttemptUnixNano.Load()
	if last == 0 {
		return true
	}
	return time.Since(time.Unix(0, last)) >= pluginIndexMinRebuildInterval
}

// kickRebuild は非同期の再構築を1本だけ起こします。
// 呼び出し元のcontextのキャンセルは引き継ぎません
// （検索が打ち切られても、温め直し自体は最後までやらせたいため）。
func (i *PluginTypedIndex) kickRebuild(_ context.Context) {
	if !i.shouldAttempt() {
		return
	}
	if !i.building.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer i.building.Store(false)
		buildCtx, cancel := context.WithTimeout(context.Background(), pluginIndexBuildTimeout)
		defer cancel()
		if err := i.build(buildCtx); err != nil {
			slog.Warn(fmt.Sprintf("plugin typed index rebuild error %q: %q", i.source.indexPluginName(), err))
		}
	}()
}

// build は find_kyous を1回だけ投げてスナップショットを作り直します。
func (i *PluginTypedIndex) build(ctx context.Context) error {
	i.buildMu.Lock()
	defer i.buildMu.Unlock()

	i.lastAttemptUnixNano.Store(time.Now().UnixNano())

	pluginKyous, err := i.source.indexFetchAll(ctx)
	if err != nil {
		// 失敗しても既存のスナップショットは空で潰さない。
		// 一時的にプラグインが混んでいるだけで、画面からタグが全部消えるのを避ける。
		if i.snapshot.Load() == nil {
			i.snapshot.Store(newEmptyPluginIndexSnapshot(false))
		}
		AppendPluginFindWarning(ctx, i.source.indexPluginName())
		return fmt.Errorf("error at fetch all plugin kyous for index: %w", err)
	}

	i.snapshot.Store(i.buildSnapshot(pluginKyous))
	return nil
}

// buildSnapshot は PluginKyou の並びを pluginTypedRecord に畳み込みます。
func (i *PluginTypedIndex) buildSnapshot(pluginKyous []gkill_plugin.PluginKyou) *pluginIndexSnapshot {
	repName := i.source.indexRepName()
	provided := i.source.indexProvidedKinds()

	snapshot := newEmptyPluginIndexSnapshot(true)
	boardNameSet := map[string]struct{}{}

	truncated := false
	for _, pluginKyou := range pluginKyous {
		if pluginKyou.ID == "" {
			continue
		}
		record, exist := snapshot.byID[pluginKyou.ID]
		if !exist {
			if len(snapshot.records) >= pluginIndexMaxRecords {
				truncated = true
				continue
			}
			record = &pluginTypedRecord{ID: pluginKyou.ID}
			snapshot.byID[pluginKyou.ID] = record
			snapshot.records = append(snapshot.records, record)
		}

		// 同じIDで版が違うものが来たら、新しい版だけを残す。
		// 索引は only_latest_data で引いているので通常は起きないが、
		// プラグインが古い版を混ぜてきても壊れないようにしておく。
		if !record.UpdateTime.IsZero() && pluginKyou.UpdateTime.Before(record.UpdateTime) {
			continue
		}
		if pluginKyou.UpdateTime.After(record.UpdateTime) {
			record.Kyous = record.Kyous[:0]
			record.Tags = nil
			record.Texts = nil
			record.Notifications = nil
			record.Kmemo, record.KC, record.URLog = nil, nil, nil
			record.Nlog, record.Lantana, record.TimeIs, record.Mi = nil, nil, nil, nil
		}
		record.UpdateTime = pluginKyou.UpdateTime
		record.IsDeleted = pluginKyou.IsDeleted
		record.Kyous = append(record.Kyous, convertPluginKyouToKyou(pluginKyou))

		i.applyTypedData(record, pluginKyou, provided)
		i.applyAttachedData(snapshot, record, pluginKyou, repName, provided)

		if record.Mi != nil && record.Mi.BoardName != "" {
			boardNameSet[record.Mi.BoardName] = struct{}{}
		}
	}
	if truncated {
		slog.Warn(fmt.Sprintf("plugin %q returned more than %d records, truncated", i.source.indexPluginName(), pluginIndexMaxRecords))
	}

	for boardName := range boardNameSet {
		snapshot.boardNames = append(snapshot.boardNames, boardName)
	}
	return snapshot
}

// applyTypedData は型別データをレコードに載せます。
// providesに宣言していない種別は捨てます。
func (i *PluginTypedIndex) applyTypedData(record *pluginTypedRecord, pluginKyou gkill_plugin.PluginKyou, provided map[gkill_plugin.PluginProvidedKind]struct{}) {
	typed := pluginKyou.Typed
	if typed == nil {
		return
	}
	base := pluginKyou

	// 2つ以上入っていたら Kmemo→KC→URLog→Nlog→Lantana→TimeIs→Mi の順で
	// 最初の1つだけを採用し、残りは警告に落とす。
	// 静かに握り潰すとプラグイン側の実装ミスに気づけない。
	applied := ""
	if typed.Kmemo != nil {
		if _, ok := provided[gkill_plugin.PluginProvidesKmemo]; ok {
			record.Kmemo = &Kmemo{
				IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
				RelatedTime: base.RelatedTime, DataType: base.DataType,
				CreateTime: base.CreateTime, CreateApp: base.CreateApp,
				CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
				UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
				UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
				Content: typed.Kmemo.Content,
			}
		}
		applied = "kmemo"
	}
	if typed.KC != nil {
		if applied == "" {
			if _, ok := provided[gkill_plugin.PluginProvidesKC]; ok {
				record.KC = &KC{
					IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
					RelatedTime: base.RelatedTime, DataType: base.DataType,
					CreateTime: base.CreateTime, CreateApp: base.CreateApp,
					CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
					UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
					UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
					Title: typed.KC.Title, NumValue: typed.KC.NumValue,
				}
			}
			applied = "kc"
		} else {
			i.warnMultipleTyped(base.ID, applied, "kc")
		}
	}
	if typed.URLog != nil {
		if applied == "" {
			if _, ok := provided[gkill_plugin.PluginProvidesURLog]; ok {
				record.URLog = &URLog{
					IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
					RelatedTime: base.RelatedTime, DataType: base.DataType,
					CreateTime: base.CreateTime, CreateApp: base.CreateApp,
					CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
					UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
					UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
					URL: typed.URLog.URL, Title: typed.URLog.Title,
					Description:  typed.URLog.Description,
					FaviconImage: typed.URLog.FaviconImage, ThumbnailImage: typed.URLog.ThumbnailImage,
				}
			}
			applied = "urlog"
		} else {
			i.warnMultipleTyped(base.ID, applied, "urlog")
		}
	}
	if typed.Nlog != nil {
		if applied == "" {
			if _, ok := provided[gkill_plugin.PluginProvidesNlog]; ok {
				record.Nlog = &Nlog{
					IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
					RelatedTime: base.RelatedTime, DataType: base.DataType,
					CreateTime: base.CreateTime, CreateApp: base.CreateApp,
					CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
					UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
					UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
					Shop: typed.Nlog.Shop, Title: typed.Nlog.Title, Amount: typed.Nlog.Amount,
				}
			}
			applied = "nlog"
		} else {
			i.warnMultipleTyped(base.ID, applied, "nlog")
		}
	}
	if typed.Lantana != nil {
		if applied == "" {
			if _, ok := provided[gkill_plugin.PluginProvidesLantana]; ok {
				record.Lantana = &Lantana{
					IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
					RelatedTime: base.RelatedTime, DataType: base.DataType,
					CreateTime: base.CreateTime, CreateApp: base.CreateApp,
					CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
					UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
					UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
					Mood: typed.Lantana.Mood,
				}
			}
			applied = "lantana"
		} else {
			i.warnMultipleTyped(base.ID, applied, "lantana")
		}
	}
	if typed.TimeIs != nil {
		if applied == "" {
			if _, ok := provided[gkill_plugin.PluginProvidesTimeIs]; ok {
				record.TimeIs = &TimeIs{
					IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
					DataType:   base.DataType,
					CreateTime: base.CreateTime, CreateApp: base.CreateApp,
					CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
					UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
					UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
					Title: typed.TimeIs.Title, StartTime: typed.TimeIs.StartTime, EndTime: typed.TimeIs.EndTime,
				}
			}
			applied = "timeis"
		} else {
			i.warnMultipleTyped(base.ID, applied, "timeis")
		}
	}
	if typed.Mi != nil {
		if applied == "" {
			if _, ok := provided[gkill_plugin.PluginProvidesMi]; ok {
				record.Mi = &Mi{
					IsDeleted: base.IsDeleted, ID: base.ID, RepName: base.RepName,
					DataType:   base.DataType,
					CreateTime: base.CreateTime, CreateApp: base.CreateApp,
					CreateDevice: base.CreateDevice, CreateUser: base.CreateUser,
					UpdateTime: base.UpdateTime, UpdateApp: base.UpdateApp,
					UpdateDevice: base.UpdateDevice, UpdateUser: base.UpdateUser,
					Title: typed.Mi.Title, IsChecked: typed.Mi.IsChecked, BoardName: typed.Mi.BoardName,
					LimitTime:         typed.Mi.LimitTime,
					EstimateStartTime: typed.Mi.EstimateStartTime,
					EstimateEndTime:   typed.Mi.EstimateEndTime,
				}
			}
			applied = "mi"
		} else {
			i.warnMultipleTyped(base.ID, applied, "mi")
		}
	}
}

// warnMultipleTyped は型別データが2つ以上入っていたことを警告します。
func (i *PluginTypedIndex) warnMultipleTyped(kyouID string, applied string, ignored string) {
	slog.Warn(fmt.Sprintf("plugin %q kyou %q has multiple typed data; %q was used and %q was ignored",
		i.source.indexPluginName(), kyouID, applied, ignored))
}

// applyAttachedData はタグ・テキスト・通知をレコードとスナップショットに載せます。
// IDは値から決定的に導出するので、並び順が変わっても再走査しても変わりません。
func (i *PluginTypedIndex) applyAttachedData(
	snapshot *pluginIndexSnapshot,
	record *pluginTypedRecord,
	pluginKyou gkill_plugin.PluginKyou,
	repName string,
	provided map[gkill_plugin.PluginProvidedKind]struct{},
) {
	if _, ok := provided[gkill_plugin.PluginProvidesTag]; ok {
		for _, tagName := range pluginKyou.Tags {
			if strings.TrimSpace(tagName) == "" {
				continue
			}
			tag := Tag{
				IsDeleted: pluginKyou.IsDeleted,
				ID:        pluginDerivedID(repName, "tag", pluginKyou.ID, tagName),
				TargetID:  pluginKyou.ID,
				Tag:       tagName,
				RepName:   repName,

				RelatedTime: pluginKyou.RelatedTime,
				CreateTime:  pluginKyou.CreateTime, CreateApp: pluginKyou.CreateApp,
				CreateDevice: pluginKyou.CreateDevice, CreateUser: pluginKyou.CreateUser,
				UpdateTime: pluginKyou.UpdateTime, UpdateApp: pluginKyou.UpdateApp,
				UpdateDevice: pluginKyou.UpdateDevice, UpdateUser: pluginKyou.UpdateUser,
			}
			if _, duplicated := snapshot.tagsByID[tag.ID]; duplicated {
				continue
			}
			snapshot.tagsByID[tag.ID] = tag
			snapshot.tagsByTarget[tag.TargetID] = append(snapshot.tagsByTarget[tag.TargetID], tag)
			record.Tags = append(record.Tags, tag)
		}
	}

	if _, ok := provided[gkill_plugin.PluginProvidesText]; ok {
		for _, textValue := range pluginKyou.Texts {
			if textValue == "" {
				continue
			}
			text := Text{
				IsDeleted: pluginKyou.IsDeleted,
				ID:        pluginDerivedID(repName, "text", pluginKyou.ID, textValue),
				TargetID:  pluginKyou.ID,
				Text:      textValue,
				RepName:   repName,

				RelatedTime: pluginKyou.RelatedTime,
				CreateTime:  pluginKyou.CreateTime, CreateApp: pluginKyou.CreateApp,
				CreateDevice: pluginKyou.CreateDevice, CreateUser: pluginKyou.CreateUser,
				UpdateTime: pluginKyou.UpdateTime, UpdateApp: pluginKyou.UpdateApp,
				UpdateDevice: pluginKyou.UpdateDevice, UpdateUser: pluginKyou.UpdateUser,
			}
			if _, duplicated := snapshot.textsByID[text.ID]; duplicated {
				continue
			}
			snapshot.textsByID[text.ID] = text
			snapshot.textsByTarget[text.TargetID] = append(snapshot.textsByTarget[text.TargetID], text)
			record.Texts = append(record.Texts, text)
		}
	}

	if _, ok := provided[gkill_plugin.PluginProvidesNotification]; ok {
		for _, pluginNotification := range pluginKyou.Notifications {
			if pluginNotification.Content == "" {
				continue
			}
			// 同じ本文で複数の時刻を持てるよう、時刻も種にする。
			seed := pluginNotification.Content + "\x00" + pluginNotification.NotificationTime.UTC().Format(time.RFC3339Nano)
			notification := Notification{
				IsDeleted:        pluginKyou.IsDeleted,
				ID:               pluginDerivedID(repName, "notification", pluginKyou.ID, seed),
				TargetID:         pluginKyou.ID,
				Content:          pluginNotification.Content,
				NotificationTime: pluginNotification.NotificationTime,
				IsNotificated:    pluginNotification.IsNotificated,
				RepName:          repName,

				CreateTime: pluginKyou.CreateTime, CreateApp: pluginKyou.CreateApp,
				CreateDevice: pluginKyou.CreateDevice, CreateUser: pluginKyou.CreateUser,
				UpdateTime: pluginKyou.UpdateTime, UpdateApp: pluginKyou.UpdateApp,
				UpdateDevice: pluginKyou.UpdateDevice, UpdateUser: pluginKyou.UpdateUser,
			}
			if _, duplicated := snapshot.notificationsByID[notification.ID]; duplicated {
				continue
			}
			snapshot.notificationsByID[notification.ID] = notification
			snapshot.notificationsByTarget[notification.TargetID] = append(snapshot.notificationsByTarget[notification.TargetID], notification)
			record.Notifications = append(record.Notifications, notification)
		}
	}
}
