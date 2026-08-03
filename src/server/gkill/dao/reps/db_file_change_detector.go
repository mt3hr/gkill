package reps

import (
	"os"
	"sync"
	"time"
)

// dbFileChangeDetector は、rep の実DBファイルが前回のキャッシュ再構築から
// 変わったかどうかを mtime + サイズ で判定します。
//
// キャッシュrep（*_cached_sqlite3_impl.go）の UpdateCache は
// 「DELETE FROM で全消し → 全行を1行ずつ再INSERT」というフルリビルドで、
// その間ずっと13種のcached repで共有している書き込みロックを握ります。
// つまり実際には変わっていないrepまで作り直すと、
// メモを1つ保存しただけで全種類の検索が再構築完了まで止まります。
// （実データでは URLog.db が90MBあり、これを毎回作り直していました）
//
// 下層repが「前回から変わったか」を答えられるようにして、
// 変わっていないrepのフルリビルドを飛ばすために使います。
//
// mtime + サイズでの判定は *_local_cached.go が既に採用している方式です。
//
// ReKyou / MiReKyou には使えません。
// この2つのキャッシュ内容は自分のDBファイルだけでなく、
// 他repのLatestDataRepositoryAddress（ターゲット解決結果）にも依存します。
// GkillRepositories.UpdateCache は
//
//	Phase1: g.Reps (ReKyouのcached repを含む) を更新
//	  ↓
//	アドレス確定
//	  ↓
//	g.ReKyouReps を「もう一度」更新   ← ターゲット解決後のこちらが本番
//
// という順で、意図的に2回更新しています。
// ファイルのmtimeは2回のあいだで変わらないため、
// この判定を入れると2回目が丸ごと飛ばされ、ターゲット未解決の中身が残ります。
// （re_kyou_granular_cache_test.go が検出します）
//
// 各メソッドは nil レシーバでも安全に呼べます（nil のときは常に「変更あり」）。
// temp rep が `type xTempRepositorySQLite3Impl xRepositorySQLite3Impl` と
// 構造体変換でコピーされるため、repのフィールドはポインタで持つ必要があります
// （値で持つと sync.Mutex を含む構造体のコピーになり copylocks になる）。
//
// 基準がまだ無いあいだは「変更あり」を返すので、起動直後は必ずキャッシュが構築されます。
type dbFileChangeDetector struct {
	m sync.Mutex

	// 最後に「取り込みに成功した」時点のファイル状態
	baseModTime time.Time
	baseSize    int64
	hasBaseline bool

	// refresh で観測した最新のファイル状態（commit されるまで基準にはしない）
	pendingModTime time.Time
	pendingSize    int64
	hasPending     bool

	changed bool
}

// refresh は現在のファイル状態を観測し、
// 「最後に commit した時点」から変わったかどうかを記録します。
// 基準は進めないので、再構築が失敗した場合は次回も「変更あり」のままになります。
func (d *dbFileChangeDetector) refresh(filename string) {
	if d == nil {
		return
	}
	d.m.Lock()
	defer d.m.Unlock()

	stat, err := os.Stat(filename)
	if err != nil {
		// statできないときは判定を諦めて安全側（変更あり）に倒す。
		// 基準も進めないので、フルリビルドが走り続けるだけで取りこぼしはしない。
		d.hasPending = false
		d.changed = true
		return
	}

	d.pendingModTime = stat.ModTime()
	d.pendingSize = stat.Size()
	d.hasPending = true

	if !d.hasBaseline {
		d.changed = true
		return
	}
	d.changed = !stat.ModTime().Equal(d.baseModTime) || stat.Size() != d.baseSize
}

// commit はキャッシュの再構築に成功したときに呼び、基準を進めます。
// これを呼ばないかぎり changed は下がらないので、
// 再構築が途中で失敗してもキャッシュが古いまま放置されることはありません。
func (d *dbFileChangeDetector) commit() {
	if d == nil {
		return
	}
	d.m.Lock()
	defer d.m.Unlock()

	if !d.hasPending {
		return
	}
	d.baseModTime = d.pendingModTime
	d.baseSize = d.pendingSize
	d.hasBaseline = true
	d.changed = false
}

// lastChanged は直近の refresh 時点で「変更あり」と判定されたかを返します。
func (d *dbFileChangeDetector) lastChanged() bool {
	if d == nil {
		// 変更検知に未対応のrep（temp repなど）は、今までどおり毎回再構築させる
		return true
	}
	d.m.Lock()
	defer d.m.Unlock()

	if !d.hasBaseline {
		return true
	}
	return d.changed
}

// cacheRebuildCommitter は、キャッシュ再構築の成功を下層repへ伝えられるrepが
// 実装する任意インタフェースです。
//
// 既存の Repository インタフェースを増やさずに済むよう、型アサーションで使います。
// 実装していないrepは今までどおり毎回フルリビルドされるだけなので、
// 段階的に適用できます。
type cacheRebuildCommitter interface {
	CommitCacheRebuild()
}

// commitCacheRebuildIfSupported は rep が対応していれば再構築成功を通知します。
func commitCacheRebuildIfSupported(rep any) {
	if c, ok := rep.(cacheRebuildCommitter); ok {
		c.CommitCacheRebuild()
	}
}
