package reps

// プラグインが返した型別データを、既存のリポジトリ契約に見せかける薄いアダプタ。
//
// 実体は pluginTypedIndex の中にしかない。したがって全アダプタで共通に、
//   - 読み取りは索引から即答し、決してプラグインへ往復しない（理由は plugin_typed_index.go 冒頭）
//   - 書き込み（AddXxxInfo）は必ずエラー
//   - Close は何もしない（プロセスの寿命は pluginRepositoryImpl と PluginManager が握っている）
//   - GetLatestDataRepositoryAddress は空
//
// 最後の1つは意図的な選択。型別repは UpdateCache の getAddrTargets に含まれないので
// そもそも呼ばれないが、仮に実データを返すと replaceLatestKyouInfos の対象になり、
// プラグインが UpdateTime をわずかに揺らしただけでレコードごと検索結果から消える
// （find_filter.go の「アドレス表に載らない rep 種は素通し」を意図的に使っている）。
// 付随データ（Tag/Text/Notification）はこれと逆で、実データを返す必要がある。
// plugin_attached_adapters.go を参照。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// pluginAdapterBase は全プラグインアダプタが共有する定型部分。
type pluginAdapterBase struct {
	plugin PluginRepository
	index  *PluginTypedIndex
	// kind はこのアダプタが担当する種別。
	kind gkill_plugin.PluginProvidedKind
}

// repName はプラグインのリポジトリ表示名を返す。
func (b *pluginAdapterBase) repName() string {
	return b.plugin.GetManifest().RepName
}

// GetPath はプラグインディレクトリを返す。idは無視する。
func (b *pluginAdapterBase) GetPath(ctx context.Context, _ string) (string, error) {
	return b.plugin.GetPath(ctx, "")
}

// GetRepName はプラグインのリポジトリ表示名を返す。
// プラグイン本体と同じ名前を返すのは、クライアントが Kyou.rep_name で問い合わせるため。
func (b *pluginAdapterBase) GetRepName(_ context.Context) (string, error) {
	return b.repName(), nil
}

// UpdateCache は索引を作り直す。実際にプラグインを叩くかは索引側の最短間隔が決める。
func (b *pluginAdapterBase) UpdateCache(ctx context.Context) error {
	return b.index.Refresh(ctx)
}

// LastUpdateCacheChanged は常にfalseを返す。
// trueを返すと、同じ集約に同居しているSQLiteキャッシュ実装に
// 無関係なフルリビルドを促してしまう。
func (b *pluginAdapterBase) LastUpdateCacheChanged() bool { return false }

// Close は何もしない。
// プラグインプロセスを閉じるのは pluginRepositoryImpl.Close と PluginManager.CloseAll だけ。
// ここで転送すると、GkillRepositories.Close が種別ぶん（最大10回）closeを送り、
// そのたびに実行スロットを最大30秒待つことになる。
func (b *pluginAdapterBase) Close(_ context.Context) error { return nil }

// GetLatestDataRepositoryAddress は空を返す。理由はファイル冒頭。
func (b *pluginAdapterBase) GetLatestDataRepositoryAddress(_ context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return []gkill_cache.LatestDataRepositoryAddress{}, nil
}

// findKinds はこのアダプタが FindKyous で返すべき種別を返す。
//
// findCtx.MatchReps は rep 名がキーなので、同じプラグインの
// KCアダプタとKmemoアダプタは片方しか入らない。
// そのため rep 種別が指定されているときは「自分の種別」ではなく
// 「provides ∩ 指定された種別」を返す。こうすると同じプラグインの
// どのアダプタが選ばれても同じ結果になる。
//
// RepTypes が nil のときはその型のrepリストしか候補にならないので、
// 自分の種別だけを返せば足りる。
func (b *pluginAdapterBase) findKinds(query *find.FindQuery) map[gkill_plugin.PluginProvidedKind]struct{} {
	own := map[gkill_plugin.PluginProvidedKind]struct{}{b.kind: {}}
	if query == nil || query.RepTypes == nil {
		return own
	}
	provided := b.plugin.GetManifest().ProvidedKinds()
	kinds := map[gkill_plugin.PluginProvidedKind]struct{}{}
	for _, repType := range query.RepTypes {
		kind := gkill_plugin.PluginProvidedKind(repType)
		if _, ok := provided[kind]; ok && kind.IsTyped() {
			kinds[kind] = struct{}{}
		}
	}
	if len(kinds) == 0 {
		return own
	}
	return kinds
}

// findKyous は索引から Kyou を集める。プラグインへは往復しない。
func (b *pluginAdapterBase) findKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	snapshot := b.index.Ensure(ctx)
	kinds := b.findKinds(query)

	kyous := []Kyou{}
	for _, record := range snapshot.records {
		if !recordHasAnyKind(record, kinds) {
			continue
		}
		for _, kyou := range record.Kyous {
			if pluginKyouMatchesQuery(kyou, query) {
				kyous = append(kyous, kyou)
			}
		}
	}
	return map[string][]Kyou{b.repName(): kyous}, nil
}

// getKyou は索引から Kyou を1件返す。無ければ (nil, nil)。
func (b *pluginAdapterBase) getKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	record := b.lookup(ctx, id)
	if record == nil {
		return nil, nil
	}
	// 自分の種別を持たないレコードは、そのIDが他の種別のものなので返さない。
	if !recordHasAnyKind(record, map[gkill_plugin.PluginProvidedKind]struct{}{b.kind: {}}) {
		return nil, nil
	}
	for _, kyou := range record.Kyous {
		if updateTime != nil && !sameSecond(kyou.UpdateTime, *updateTime) {
			continue
		}
		found := kyou
		return &found, nil
	}
	return nil, nil
}

// getKyouHistories は索引が最新版しか持たないので0件か1件を返す。
func (b *pluginAdapterBase) getKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyou, err := b.getKyou(ctx, id, nil)
	if err != nil || kyou == nil {
		return []Kyou{}, err
	}
	return []Kyou{*kyou}, nil
}

// lookup は索引からレコードを引く。無ければ非同期の温め直しを予約してnilを返す。
func (b *pluginAdapterBase) lookup(ctx context.Context, id string) *pluginTypedRecord {
	snapshot := b.index.Ensure(ctx)
	record, exist := snapshot.byID[id]
	if !exist {
		b.index.NoteMissingID(ctx, id)
		return nil
	}
	return record
}

// readOnlyError は書き込みメソッドが返すエラー。
func (b *pluginAdapterBase) readOnlyError(method string) error {
	return fmt.Errorf("plugin repository %s is read only: %s is not supported", b.repName(), method)
}

// recordHasAnyKind はレコードが指定の種別のいずれかを持つかを返す。
func recordHasAnyKind(record *pluginTypedRecord, kinds map[gkill_plugin.PluginProvidedKind]struct{}) bool {
	for kind := range kinds {
		switch kind {
		case gkill_plugin.PluginProvidesKmemo:
			if record.Kmemo != nil {
				return true
			}
		case gkill_plugin.PluginProvidesKC:
			if record.KC != nil {
				return true
			}
		case gkill_plugin.PluginProvidesURLog:
			if record.URLog != nil {
				return true
			}
		case gkill_plugin.PluginProvidesNlog:
			if record.Nlog != nil {
				return true
			}
		case gkill_plugin.PluginProvidesLantana:
			if record.Lantana != nil {
				return true
			}
		case gkill_plugin.PluginProvidesTimeIs:
			if record.TimeIs != nil {
				return true
			}
		case gkill_plugin.PluginProvidesMi:
			if record.Mi != nil {
				return true
			}
		}
	}
	return false
}

// sameSecond は2つの時刻が秒単位で一致するかを返す。
// クライアントの型別データ突き合わせと同じ精度にそろえてある。
func sameSecond(a time.Time, b time.Time) bool {
	return a.Unix() == b.Unix()
}

// pluginMatchWords はワード検索の共通判定。
// 対象列とIDを連結した文字列に対する大小無視の部分一致で、
// WordsAndがtrueなら全語、falseならいずれか1語。NotWordsは常に除外。
func pluginMatchWords(target string, id string, query *find.FindQuery) bool {
	if query == nil || !query.HasWordFilter() {
		return true
	}
	haystack := strings.ToLower(target + "\x00" + id)

	for _, notWord := range query.NotWords {
		if notWord == "" {
			continue
		}
		if strings.Contains(haystack, strings.ToLower(notWord)) {
			return false
		}
	}
	if len(query.Words) == 0 {
		return true
	}
	matchedAny := false
	for _, word := range query.Words {
		if word == "" {
			continue
		}
		matched := strings.Contains(haystack, strings.ToLower(word))
		if query.WordsAnd && !matched {
			return false
		}
		if matched {
			matchedAny = true
		}
	}
	if query.WordsAnd {
		return true
	}
	return matchedAny
}

// pluginMatchIDs は query.IDs による絞り込み。nilなら素通し。
func pluginMatchIDs(id string, query *find.FindQuery) bool {
	if query == nil || query.IDs == nil {
		return true
	}
	for _, queryID := range query.IDs {
		if queryID == id {
			return true
		}
	}
	return false
}

// pluginMatchCalendar は関連時刻の期間フィルタ。両端を含む。
func pluginMatchCalendar(relatedTime time.Time, query *find.FindQuery) bool {
	if query == nil {
		return true
	}
	if query.CalendarStartDate != nil && relatedTime.Before(*query.CalendarStartDate) {
		return false
	}
	if query.CalendarEndDate != nil && relatedTime.After(*query.CalendarEndDate) {
		return false
	}
	return true
}

// ---- Kmemo ----

type pluginKmemoRepositoryImpl struct{ pluginAdapterBase }

var _ KmemoRepository = (*pluginKmemoRepositoryImpl)(nil)

func (p *pluginKmemoRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginKmemoRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginKmemoRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginKmemoRepositoryImpl) FindKmemo(ctx context.Context, query *find.FindQuery) ([]Kmemo, error) {
	snapshot := p.index.Ensure(ctx)
	kmemos := []Kmemo{}
	for _, record := range snapshot.records {
		if record.Kmemo == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) || !pluginMatchCalendar(record.Kmemo.RelatedTime, query) {
			continue
		}
		if !pluginMatchWords(record.Kmemo.Content, record.ID, query) {
			continue
		}
		kmemos = append(kmemos, *record.Kmemo)
	}
	return kmemos, nil
}
func (p *pluginKmemoRepositoryImpl) GetKmemo(ctx context.Context, id string, updateTime *time.Time) (*Kmemo, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.Kmemo == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.Kmemo.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.Kmemo
	return &found, nil
}
func (p *pluginKmemoRepositoryImpl) GetKmemoHistories(ctx context.Context, id string) ([]Kmemo, error) {
	kmemo, err := p.GetKmemo(ctx, id, nil)
	if err != nil || kmemo == nil {
		return []Kmemo{}, err
	}
	return []Kmemo{*kmemo}, nil
}
func (p *pluginKmemoRepositoryImpl) AddKmemoInfo(_ context.Context, _ Kmemo) error {
	return p.readOnlyError("AddKmemoInfo")
}
func (p *pluginKmemoRepositoryImpl) UnWrapTyped() ([]KmemoRepository, error) {
	return []KmemoRepository{p}, nil
}
func (p *pluginKmemoRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// ---- KC ----

type pluginKCRepositoryImpl struct{ pluginAdapterBase }

var _ KCRepository = (*pluginKCRepositoryImpl)(nil)

func (p *pluginKCRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginKCRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginKCRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginKCRepositoryImpl) FindKC(ctx context.Context, query *find.FindQuery) ([]KC, error) {
	snapshot := p.index.Ensure(ctx)
	kcs := []KC{}
	for _, record := range snapshot.records {
		if record.KC == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) || !pluginMatchCalendar(record.KC.RelatedTime, query) {
			continue
		}
		// キーワードの対象列はTITLE。数値は検索対象にしない（ネイティブと同じ）。
		if !pluginMatchWords(record.KC.Title, record.ID, query) {
			continue
		}
		kcs = append(kcs, *record.KC)
	}
	return kcs, nil
}
func (p *pluginKCRepositoryImpl) GetKC(ctx context.Context, id string, updateTime *time.Time) (*KC, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.KC == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.KC.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.KC
	return &found, nil
}
func (p *pluginKCRepositoryImpl) GetKCHistories(ctx context.Context, id string) ([]KC, error) {
	kc, err := p.GetKC(ctx, id, nil)
	if err != nil || kc == nil {
		return []KC{}, err
	}
	return []KC{*kc}, nil
}
func (p *pluginKCRepositoryImpl) AddKCInfo(_ context.Context, _ KC) error {
	return p.readOnlyError("AddKCInfo")
}
func (p *pluginKCRepositoryImpl) UnWrapTyped() ([]KCRepository, error) {
	return []KCRepository{p}, nil
}
func (p *pluginKCRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// ---- URLog ----

type pluginURLogRepositoryImpl struct{ pluginAdapterBase }

var _ URLogRepository = (*pluginURLogRepositoryImpl)(nil)

func (p *pluginURLogRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginURLogRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginURLogRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginURLogRepositoryImpl) FindURLog(ctx context.Context, query *find.FindQuery) ([]URLog, error) {
	snapshot := p.index.Ensure(ctx)
	urlogs := []URLog{}
	for _, record := range snapshot.records {
		if record.URLog == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) || !pluginMatchCalendar(record.URLog.RelatedTime, query) {
			continue
		}
		if !pluginMatchWords(record.URLog.URL+"\x00"+record.URLog.Title+"\x00"+record.URLog.Description, record.ID, query) {
			continue
		}
		urlog := *record.URLog
		// 索引のレコードは書き換えず、値のコピーからサムネイルだけを外す。
		if query != nil && query.ExcludeURLogThumbnailImage {
			urlog.ThumbnailImage = ""
		}
		urlogs = append(urlogs, urlog)
	}
	return urlogs, nil
}
func (p *pluginURLogRepositoryImpl) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.URLog == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.URLog.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.URLog
	return &found, nil
}
func (p *pluginURLogRepositoryImpl) GetURLogHistories(ctx context.Context, id string) ([]URLog, error) {
	urlog, err := p.GetURLog(ctx, id, nil)
	if err != nil || urlog == nil {
		return []URLog{}, err
	}
	return []URLog{*urlog}, nil
}
func (p *pluginURLogRepositoryImpl) AddURLogInfo(_ context.Context, _ URLog) error {
	return p.readOnlyError("AddURLogInfo")
}
func (p *pluginURLogRepositoryImpl) UnWrapTyped() ([]URLogRepository, error) {
	return []URLogRepository{p}, nil
}
func (p *pluginURLogRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// ---- Nlog ----

type pluginNlogRepositoryImpl struct{ pluginAdapterBase }

var _ NlogRepository = (*pluginNlogRepositoryImpl)(nil)

func (p *pluginNlogRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginNlogRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginNlogRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginNlogRepositoryImpl) FindNlog(ctx context.Context, query *find.FindQuery) ([]Nlog, error) {
	snapshot := p.index.Ensure(ctx)
	nlogs := []Nlog{}
	for _, record := range snapshot.records {
		if record.Nlog == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) || !pluginMatchCalendar(record.Nlog.RelatedTime, query) {
			continue
		}
		if !pluginMatchWords(record.Nlog.Title+"\x00"+record.Nlog.Shop, record.ID, query) {
			continue
		}
		nlogs = append(nlogs, *record.Nlog)
	}
	return nlogs, nil
}
func (p *pluginNlogRepositoryImpl) GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.Nlog == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.Nlog.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.Nlog
	return &found, nil
}
func (p *pluginNlogRepositoryImpl) GetNlogHistories(ctx context.Context, id string) ([]Nlog, error) {
	nlog, err := p.GetNlog(ctx, id, nil)
	if err != nil || nlog == nil {
		return []Nlog{}, err
	}
	return []Nlog{*nlog}, nil
}
func (p *pluginNlogRepositoryImpl) AddNlogInfo(_ context.Context, _ Nlog) error {
	return p.readOnlyError("AddNlogInfo")
}
func (p *pluginNlogRepositoryImpl) UnWrapTyped() ([]NlogRepository, error) {
	return []NlogRepository{p}, nil
}
func (p *pluginNlogRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// ---- Lantana ----

type pluginLantanaRepositoryImpl struct{ pluginAdapterBase }

var _ LantanaRepository = (*pluginLantanaRepositoryImpl)(nil)

func (p *pluginLantanaRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginLantanaRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginLantanaRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginLantanaRepositoryImpl) FindLantana(ctx context.Context, query *find.FindQuery) ([]Lantana, error) {
	snapshot := p.index.Ensure(ctx)
	lantanas := []Lantana{}
	// Lantanaに検索対象の文字列列は無い。ネイティブと同じくワード検索が有効なら0件にする。
	if query != nil && query.HasWordFilter() {
		return lantanas, nil
	}
	for _, record := range snapshot.records {
		if record.Lantana == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) || !pluginMatchCalendar(record.Lantana.RelatedTime, query) {
			continue
		}
		lantanas = append(lantanas, *record.Lantana)
	}
	return lantanas, nil
}
func (p *pluginLantanaRepositoryImpl) GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.Lantana == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.Lantana.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.Lantana
	return &found, nil
}
func (p *pluginLantanaRepositoryImpl) GetLantanaHistories(ctx context.Context, id string) ([]Lantana, error) {
	lantana, err := p.GetLantana(ctx, id, nil)
	if err != nil || lantana == nil {
		return []Lantana{}, err
	}
	return []Lantana{*lantana}, nil
}
func (p *pluginLantanaRepositoryImpl) AddLantanaInfo(_ context.Context, _ Lantana) error {
	return p.readOnlyError("AddLantanaInfo")
}
func (p *pluginLantanaRepositoryImpl) UnWrapTyped() ([]LantanaRepository, error) {
	return []LantanaRepository{p}, nil
}
func (p *pluginLantanaRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// ---- TimeIs ----

type pluginTimeIsRepositoryImpl struct{ pluginAdapterBase }

var _ TimeIsRepository = (*pluginTimeIsRepositoryImpl)(nil)

func (p *pluginTimeIsRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginTimeIsRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginTimeIsRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginTimeIsRepositoryImpl) FindTimeIs(ctx context.Context, query *find.FindQuery) ([]TimeIs, error) {
	snapshot := p.index.Ensure(ctx)
	timeiss := []TimeIs{}
	for _, record := range snapshot.records {
		if record.TimeIs == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) {
			continue
		}
		if !pluginMatchWords(record.TimeIs.Title, record.ID, query) {
			continue
		}
		// 期間は開始時刻で判定する。終了時刻が無い（計測中）ものも拾う。
		if !pluginMatchCalendar(record.TimeIs.StartTime, query) {
			continue
		}
		timeiss = append(timeiss, *record.TimeIs)
	}
	return timeiss, nil
}
func (p *pluginTimeIsRepositoryImpl) GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.TimeIs == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.TimeIs.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.TimeIs
	return &found, nil
}
func (p *pluginTimeIsRepositoryImpl) GetTimeIsHistories(ctx context.Context, id string) ([]TimeIs, error) {
	timeis, err := p.GetTimeIs(ctx, id, nil)
	if err != nil || timeis == nil {
		return []TimeIs{}, err
	}
	return []TimeIs{*timeis}, nil
}
func (p *pluginTimeIsRepositoryImpl) AddTimeIsInfo(_ context.Context, _ TimeIs) error {
	return p.readOnlyError("AddTimeIsInfo")
}
func (p *pluginTimeIsRepositoryImpl) UnWrapTyped() ([]TimeIsRepository, error) {
	return []TimeIsRepository{p}, nil
}
func (p *pluginTimeIsRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// ---- Mi ----

type pluginMiRepositoryImpl struct{ pluginAdapterBase }

var _ MiRepository = (*pluginMiRepositoryImpl)(nil)

func (p *pluginMiRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	return p.findKyous(ctx, query)
}
func (p *pluginMiRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	return p.getKyou(ctx, id, updateTime)
}
func (p *pluginMiRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	return p.getKyouHistories(ctx, id)
}
func (p *pluginMiRepositoryImpl) FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error) {
	snapshot := p.index.Ensure(ctx)
	mis := []Mi{}
	for _, record := range snapshot.records {
		if record.Mi == nil {
			continue
		}
		if !pluginMatchIDs(record.ID, query) {
			continue
		}
		if !pluginMatchWords(record.Mi.Title, record.ID, query) {
			continue
		}
		// 板名はnilが「すべて」。
		if query != nil && query.MiBoardName != nil && *query.MiBoardName != record.Mi.BoardName {
			continue
		}
		mis = append(mis, *record.Mi)
	}
	return mis, nil
}
func (p *pluginMiRepositoryImpl) GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error) {
	record := p.lookup(ctx, id)
	if record == nil || record.Mi == nil {
		return nil, nil
	}
	if updateTime != nil && !sameSecond(record.Mi.UpdateTime, *updateTime) {
		return nil, nil
	}
	found := *record.Mi
	return &found, nil
}
func (p *pluginMiRepositoryImpl) GetMiHistories(ctx context.Context, id string) ([]Mi, error) {
	mi, err := p.GetMi(ctx, id, nil)
	if err != nil || mi == nil {
		return []Mi{}, err
	}
	return []Mi{*mi}, nil
}
func (p *pluginMiRepositoryImpl) AddMiInfo(_ context.Context, _ Mi) error {
	return p.readOnlyError("AddMiInfo")
}
func (p *pluginMiRepositoryImpl) GetBoardNames(ctx context.Context) ([]string, error) {
	snapshot := p.index.Ensure(ctx)
	return append([]string{}, snapshot.boardNames...), nil
}
func (p *pluginMiRepositoryImpl) UnWrapTyped() ([]MiRepository, error) {
	return []MiRepository{p}, nil
}
func (p *pluginMiRepositoryImpl) UnWrap() ([]Repository, error) { return []Repository{p}, nil }

// PluginTypedRepositories はプラグイン1本ぶんのアダプタ一式です。
// manifest.json の provides に無い種別は nil になります。
type PluginTypedRepositories struct {
	Kmemo        KmemoRepository
	KC           KCRepository
	URLog        URLogRepository
	Nlog         NlogRepository
	Lantana      LantanaRepository
	TimeIs       TimeIsRepository
	Mi           MiRepository
	Tag          TagRepository
	Text         TextRepository
	Notification NotificationRepository
}

// NewPluginTypedRepositories はプラグインの provides に応じてアダプタを作ります。
// provides が空、または索引を持たないプラグインではゼロ値（全部 nil）を返します。
func NewPluginTypedRepositories(plugin PluginRepository) PluginTypedRepositories {
	adapters := PluginTypedRepositories{}
	index := plugin.TypedIndex()
	if index == nil {
		return adapters
	}
	provided := plugin.GetManifest().ProvidedKinds()
	base := func(kind gkill_plugin.PluginProvidedKind) pluginAdapterBase {
		return pluginAdapterBase{plugin: plugin, index: index, kind: kind}
	}

	if _, ok := provided[gkill_plugin.PluginProvidesKmemo]; ok {
		adapters.Kmemo = &pluginKmemoRepositoryImpl{base(gkill_plugin.PluginProvidesKmemo)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesKC]; ok {
		adapters.KC = &pluginKCRepositoryImpl{base(gkill_plugin.PluginProvidesKC)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesURLog]; ok {
		adapters.URLog = &pluginURLogRepositoryImpl{base(gkill_plugin.PluginProvidesURLog)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesNlog]; ok {
		adapters.Nlog = &pluginNlogRepositoryImpl{base(gkill_plugin.PluginProvidesNlog)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesLantana]; ok {
		adapters.Lantana = &pluginLantanaRepositoryImpl{base(gkill_plugin.PluginProvidesLantana)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesTimeIs]; ok {
		adapters.TimeIs = &pluginTimeIsRepositoryImpl{base(gkill_plugin.PluginProvidesTimeIs)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesMi]; ok {
		adapters.Mi = &pluginMiRepositoryImpl{base(gkill_plugin.PluginProvidesMi)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesTag]; ok {
		adapters.Tag = &pluginTagRepositoryImpl{base(gkill_plugin.PluginProvidesTag)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesText]; ok {
		adapters.Text = &pluginTextRepositoryImpl{base(gkill_plugin.PluginProvidesText)}
	}
	if _, ok := provided[gkill_plugin.PluginProvidesNotification]; ok {
		adapters.Notification = &pluginNotificationRepositoryImpl{base(gkill_plugin.PluginProvidesNotification)}
	}
	return adapters
}
