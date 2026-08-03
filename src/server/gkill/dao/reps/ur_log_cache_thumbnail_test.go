package reps

// URLog のサムネイルをインメモリキャッシュに載せないことの回帰テスト。
//
// THUMBNAIL_IMAGE は base64 で1行あたり平均406KB・最大10MBあり、
// 実データ227行の合計が90MBに達する。これをキャッシュ表に持つと
// --cache_in_memory（既定true）でそのままメモリに常駐してしまう。

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"
)

// countingURLogRep は GetURLog の呼び出し回数を数えるラッパ。
// サムネイルの取り直しが「その版を持つrepだけ」に限定されているかの確認に使う。
type countingURLogRep struct {
	URLogRepository
	repName       string
	getURLogCalls *int
}

func (c *countingURLogRep) GetRepName(_ context.Context) (string, error) {
	return c.repName, nil
}

func (c *countingURLogRep) UnWrapTyped() ([]URLogRepository, error) {
	return []URLogRepository{c}, nil
}

func (c *countingURLogRep) GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error) {
	*c.getURLogCalls++
	return c.URLogRepository.GetURLog(ctx, id, updateTime)
}

// キャッシュ表に THUMBNAIL_IMAGE 列が無いこと。
// これが戻ると「メモリに載せない」がまた失われる。
func TestURLogCacheTable_HasNoThumbnailColumn(t *testing.T) {
	ctx := context.Background()
	baseRepo := newTempURLogRepo(t)
	cacheDB := openMemoryDB(t)

	if _, err := NewURLogRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, &sync.RWMutex{}, "URLOG_CACHE_COLS"); err != nil {
		t.Fatalf("failed to create cached urlog repo: %v", err)
	}

	cols := map[string]bool{}
	rows, err := cacheDB.Query(`PRAGMA table_info("URLOG_CACHE_COLS")`)
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}

	if cols["THUMBNAIL_IMAGE"] {
		t.Error("キャッシュ表に THUMBNAIL_IMAGE 列がある。90MBがメモリに常駐してしまう")
	}
	// FAVICON_IMAGE は合計0.10MBしかないので持たせたまま
	if !cols["FAVICON_IMAGE"] {
		t.Error("FAVICON_IMAGE はキャッシュに残す方針だが列が無い")
	}
	if !cols["REP_NAME"] {
		t.Error("REP_NAME が無いとサムネイルの持ち主repを特定できない")
	}
}

// キャッシュに載せていなくても、表示経路ではサムネイルが返ること。
func TestURLogCache_GetURLogHistoriesReturnsThumbnail(t *testing.T) {
	ctx := context.Background()
	baseRepo := newTempURLogRepo(t)
	baseRepName, err := baseRepo.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName: %v", err)
	}

	urlog := makeURLog("urlog-thumb-001", "https://example.com", "example")
	urlog.RepName = baseRepName
	urlog.FaviconImage = "favicon-data"
	urlog.ThumbnailImage = "thumbnail-data"
	if err := baseRepo.AddURLogInfo(ctx, urlog); err != nil {
		t.Fatalf("AddURLogInfo: %v", err)
	}

	cacheDB := openMemoryDB(t)
	cached, err := NewURLogRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, &sync.RWMutex{}, "URLOG_CACHE_THUMB")
	if err != nil {
		t.Fatalf("failed to create cached urlog repo: %v", err)
	}
	t.Cleanup(func() { _ = cached.Close(ctx) })

	if err := cached.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	histories, err := cached.GetURLogHistories(ctx, "urlog-thumb-001")
	if err != nil {
		t.Fatalf("GetURLogHistories: %v", err)
	}
	if len(histories) == 0 {
		t.Fatal("GetURLogHistories が0件を返した")
	}
	if histories[0].ThumbnailImage != "thumbnail-data" {
		t.Errorf("サムネイルが取り直せていない: %q (キャッシュに無いぶんをrepから埋める必要がある)", histories[0].ThumbnailImage)
	}
	if histories[0].FaviconImage != "favicon-data" {
		t.Errorf("faviconはキャッシュから返るはず: %q", histories[0].FaviconImage)
	}
}

// サムネイルの取り直しが「その版を持つrepだけ」に限定されていること。
//
// 集約(u.urlogRep)へ丸ごと委譲する実装に戻すと、1件取るのに
// 配下rep数ぶんのクエリが飛ぶ（実データでは17rep = カード1枚あたり約13.8ms）。
// 結果は正しいままなので、回数を数えないと気づけない。
func TestURLogCache_ThumbnailRefetchHitsOnlyOwnerRep(t *testing.T) {
	ctx := context.Background()

	ownerCalls, otherCalls := 0, 0
	ownerBase := newTempURLogRepo(t)
	ownerRepName, err := ownerBase.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName: %v", err)
	}
	owner := &countingURLogRep{URLogRepository: ownerBase, repName: ownerRepName, getURLogCalls: &ownerCalls}
	other := &countingURLogRep{URLogRepository: newTempURLogRepo(t), repName: "OTHER_URLOG_REP", getURLogCalls: &otherCalls}

	urlog := makeURLog("urlog-owner-001", "https://example.com", "example")
	urlog.RepName = ownerRepName
	urlog.ThumbnailImage = "owner-thumbnail"
	if err := ownerBase.AddURLogInfo(ctx, urlog); err != nil {
		t.Fatalf("AddURLogInfo: %v", err)
	}

	reps := URLogRepositories{owner, other}
	cacheDB := openMemoryDB(t)
	cached, err := NewURLogRepositoryCachedSQLite3Impl(ctx, reps, cacheDB, &sync.RWMutex{}, "URLOG_CACHE_OWNER")
	if err != nil {
		t.Fatalf("failed to create cached urlog repo: %v", err)
	}
	t.Cleanup(func() { _ = cached.Close(ctx) })

	if err := cached.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	ownerCalls, otherCalls = 0, 0

	histories, err := cached.GetURLogHistories(ctx, "urlog-owner-001")
	if err != nil {
		t.Fatalf("GetURLogHistories: %v", err)
	}
	if len(histories) == 0 || histories[0].ThumbnailImage != "owner-thumbnail" {
		t.Fatalf("サムネイルが取り直せていない: %+v", histories)
	}
	if otherCalls != 0 {
		t.Errorf("その版を持たないrepにも問い合わせている (%d回)。rep数に比例して遅くなる", otherCalls)
	}
	if ownerCalls == 0 {
		t.Error("持ち主のrepに問い合わせていない")
	}
}
