package reps

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// commit_tx.go の CommitTx / DiscardTx の挙動を固定する。
//
// 2026-09-15 まで commit_tx は種別ごとの逐次追記で、途中の種別で失敗すると書けた種別だけが残っていた
// （部分確定。外部レビュー #4「トランザクションがユーザの期待するトランザクションではない」）。
// 今は書き込み rep のファイルを1接続に ATTACH した1つの SQLite トランザクションで確定するので、
// 失敗したら**何も残らない**。それをここで固定する。
// documents/adr/0219-commit-tx-is-one-sqlite-transaction.md

var commitTxTestMemoryDBSeq atomic.Int64

// openSharedMemoryDB は接続プールの全接続で同じ中身が見える名前付きインメモリDBを開く。
// 素の ":memory:" は接続ごとに別のDBになるので、temp rep のように INSERT と SELECT が
// 別の接続に乗ると行が消えたように見える。
func openSharedMemoryDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("file:commit_tx_test_%d?mode=memory&cache=shared", commitTxTestMemoryDBSeq.Add(1))
	db, err := sql.Open("sqlite", name)
	if err != nil {
		t.Fatalf("failed to open shared in-memory sqlite: %v", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(0)
	t.Cleanup(func() { db.Close() })
	return db
}

// newCommitTxTestRepositories は temp rep と kc / kmemo / tag の書き込み rep を持つ GkillRepositories を作る。
// キャッシュ rep は無し（WriteThroughXxxCache は何もしない経路）。
func newCommitTxTestRepositories(t *testing.T) *GkillRepositories {
	t.Helper()
	ctx := context.Background()
	tempReps, err := NewTempReps(openSharedMemoryDB(t), &sync.RWMutex{})
	if err != nil {
		t.Fatalf("failed to create temp reps: %v", err)
	}
	addressDAO, err := gkill_cache.NewLatestDataRepositoryAddressSQLite3Impl("testuser", openSharedMemoryDB(t), &sync.RWMutex{})
	if err != nil {
		t.Fatalf("failed to create latest data repository address dao: %v", err)
	}
	repositories := &GkillRepositories{
		TempReps:                       tempReps,
		LatestDataRepositoryAddressDAO: addressDAO,
		WriteKCRep:                     newTempKCRepo(t),
		WriteKmemoRep:                  newTempKmemoRepo(t),
		WriteTagRep:                    newTempTagRepo(t),
	}
	_ = ctx
	return repositories
}

func commitTxTestKC(id string, now time.Time) KC {
	return KC{
		ID: id, Title: "kc " + id, NumValue: json.Number("1"), RelatedTime: now,
		CreateTime: now, CreateApp: "test", CreateDevice: "device", CreateUser: "testuser",
		UpdateTime: now, UpdateApp: "test", UpdateDevice: "device", UpdateUser: "testuser",
	}
}

func commitTxTestKmemo(id string, content string, now time.Time) Kmemo {
	return Kmemo{
		ID: id, Content: content, RelatedTime: now,
		CreateTime: now, CreateApp: "test", CreateDevice: "device", CreateUser: "testuser",
		UpdateTime: now, UpdateApp: "test", UpdateDevice: "device", UpdateUser: "testuser",
	}
}

func commitTxTestTag(id string, targetID string, now time.Time) Tag {
	return Tag{
		ID: id, TargetID: targetID, Tag: "tag " + id, RelatedTime: now,
		CreateTime: now, CreateApp: "test", CreateDevice: "device", CreateUser: "testuser",
		UpdateTime: now, UpdateApp: "test", UpdateDevice: "device", UpdateUser: "testuser",
	}
}

// 途中の種別で失敗したら、それより前に書いた種別も含めて何も残らない。
// kc は書けるが kmemo は本文が空で insertKmemoRow が弾く（INSERT 前の検査）。
// 以前の逐次追記なら kc だけが残っていた。
func TestCommitTx_RollsBackEverythingWhenOneRowFails(t *testing.T) {
	ctx := context.Background()
	repositories := newCommitTxTestRepositories(t)
	const txID, userID, device = "tx-rollback", "testuser", "device"
	now := time.Now()

	if err := repositories.TempReps.KCTempRep.AddKCInfo(ctx, commitTxTestKC("kc-1", now), txID, userID, device); err != nil {
		t.Fatalf("stage kc: %v", err)
	}
	if err := repositories.TempReps.KmemoTempRep.AddKmemoInfo(ctx, commitTxTestKmemo("kmemo-empty", "   ", now), txID, userID, device); err != nil {
		t.Fatalf("stage kmemo: %v", err)
	}

	committed, err := repositories.CommitTx(ctx, txID, userID, device)
	if err == nil {
		t.Fatal("本文が空の kmemo を含む tx の commit が成功してしまった")
	}
	if committed != nil {
		t.Errorf("失敗した commit が committed を返した: %+v", committed)
	}

	kc, err := repositories.WriteKCRep.GetKC(ctx, "kc-1", nil)
	if err != nil {
		t.Fatalf("get kc: %v", err)
	}
	if kc != nil {
		t.Errorf("失敗した tx の kc が実 rep に残っている（部分確定）: %+v", kc)
	}
	staged, err := repositories.TempReps.KCTempRep.GetKCsByTXID(ctx, txID, userID, device)
	if err != nil {
		t.Fatalf("get staged kc: %v", err)
	}
	if len(staged) != 1 {
		t.Errorf("失敗した tx の temp rep の行が消えている（再 commit / discard ができない）: %d 件", len(staged))
	}
}

// 全部通れば全種別が実 rep にあり、temp rep は空になり、最新版アドレス表は書き込み rep 名で埋まる。
func TestCommitTx_CommitsAllTypesAndConsumesTemp(t *testing.T) {
	ctx := context.Background()
	repositories := newCommitTxTestRepositories(t)
	const txID, userID, device = "tx-commit", "testuser", "device"
	now := time.Now()

	if err := repositories.TempReps.KCTempRep.AddKCInfo(ctx, commitTxTestKC("kc-2", now), txID, userID, device); err != nil {
		t.Fatalf("stage kc: %v", err)
	}
	if err := repositories.TempReps.KmemoTempRep.AddKmemoInfo(ctx, commitTxTestKmemo("kmemo-2", "memo", now), txID, userID, device); err != nil {
		t.Fatalf("stage kmemo: %v", err)
	}
	if err := repositories.TempReps.TagTempRep.AddTagInfo(ctx, commitTxTestTag("tag-2", "kmemo-2", now), txID, userID, device); err != nil {
		t.Fatalf("stage tag: %v", err)
	}

	committed, err := repositories.CommitTx(ctx, txID, userID, device)
	if err != nil {
		t.Fatalf("commit tx: %v", err)
	}
	got := map[string]CommittedRecord{}
	for _, record := range committed {
		got[record.DataType+":"+record.ID] = record
	}
	for _, key := range []string{"kc:kc-2", "kmemo:kmemo-2", "tag:tag-2"} {
		record, ok := got[key]
		if !ok {
			t.Errorf("committed に %s が無い: %+v", key, committed)
			continue
		}
		if record.Updated {
			t.Errorf("%s は新規作成なのに updated=true", key)
		}
	}

	if kc, err := repositories.WriteKCRep.GetKC(ctx, "kc-2", nil); err != nil || kc == nil {
		t.Errorf("kc が実 rep に無い: kc=%v err=%v", kc, err)
	}
	if kmemo, err := repositories.WriteKmemoRep.GetKmemo(ctx, "kmemo-2", nil); err != nil || kmemo == nil {
		t.Errorf("kmemo が実 rep に無い: kmemo=%v err=%v", kmemo, err)
	}
	tag, err := repositories.WriteTagRep.GetTag(ctx, "tag-2", nil)
	if err != nil || tag == nil {
		t.Fatalf("tag が実 rep に無い: tag=%v err=%v", tag, err)
	}
	if tag.TargetID != "kmemo-2" {
		t.Errorf("tag の対象が違う: %q", tag.TargetID)
	}

	for _, check := range []struct {
		name  string
		count func() (int, error)
	}{
		{"kc", func() (int, error) {
			rows, err := repositories.TempReps.KCTempRep.GetKCsByTXID(ctx, txID, userID, device)
			return len(rows), err
		}},
		{"kmemo", func() (int, error) {
			rows, err := repositories.TempReps.KmemoTempRep.GetKmemosByTXID(ctx, txID, userID, device)
			return len(rows), err
		}},
		{"tag", func() (int, error) {
			rows, err := repositories.TempReps.TagTempRep.GetTagsByTXID(ctx, txID, userID, device)
			return len(rows), err
		}},
	} {
		n, err := check.count()
		if err != nil {
			t.Fatalf("get staged %s: %v", check.name, err)
		}
		if n != 0 {
			t.Errorf("commit 後も temp rep に %s が %d 件残っている（同じ tx の再 commit で二重登録になる）", check.name, n)
		}
	}

	writeRepName, err := repositories.WriteKmemoRep.GetRepName(ctx)
	if err != nil {
		t.Fatalf("get write rep name: %v", err)
	}
	addr, ok := repositories.GetLatestDataRepositoryAddress("kmemo-2")
	if !ok {
		t.Fatal("commit した kmemo が最新版アドレス表に無い")
	}
	if addr.LatestDataRepositoryName != writeRepName {
		t.Errorf("アドレス表の rep 名が書き込み rep 名でない: got %q want %q（temp rep の合成名を持ち込むと一覧から消える）", addr.LatestDataRepositoryName, writeRepName)
	}
	tagAddr, ok := repositories.GetLatestDataRepositoryAddress("tag-2")
	if !ok || tagAddr.TargetIDInData == nil || *tagAddr.TargetIDInData != "kmemo-2" {
		t.Errorf("tag のアドレス表に TargetIDInData が無い: %+v", tagAddr)
	}
}

// 既存の記録の新しい版（打刻の終了など）は Updated=true で返る。
func TestCommitTx_MarksExistingIDAsUpdated(t *testing.T) {
	ctx := context.Background()
	repositories := newCommitTxTestRepositories(t)
	const txID, userID, device = "tx-update", "testuser", "device"
	first := time.Now().Add(-2 * time.Second)

	if err := repositories.WriteKmemoRep.AddKmemoInfo(ctx, commitTxTestKmemo("kmemo-3", "v1", first)); err != nil {
		t.Fatalf("add first version: %v", err)
	}
	repositories.SetLatestDataRepositoryAddress("kmemo-3", gkill_cache.LatestDataRepositoryAddress{TargetID: "kmemo-3", DataUpdateTime: first})

	second := commitTxTestKmemo("kmemo-3", "v2", first.Add(2*time.Second))
	if err := repositories.TempReps.KmemoTempRep.AddKmemoInfo(ctx, second, txID, userID, device); err != nil {
		t.Fatalf("stage kmemo: %v", err)
	}
	committed, err := repositories.CommitTx(ctx, txID, userID, device)
	if err != nil {
		t.Fatalf("commit tx: %v", err)
	}
	if len(committed) != 1 || !committed[0].Updated {
		t.Errorf("既存 ID の新しい版が updated=true で返らない: %+v", committed)
	}
	latest, err := repositories.WriteKmemoRep.GetKmemo(ctx, "kmemo-3", nil)
	if err != nil || latest == nil || latest.Content != "v2" {
		t.Errorf("最新版が commit した版でない: %+v err=%v", latest, err)
	}
}

// 書き込み rep が未設定の種別に未確定データがあれば、**書く前に**エラーで返す。
func TestCommitTx_MissingWriteRepFailsBeforeWriting(t *testing.T) {
	ctx := context.Background()
	repositories := newCommitTxTestRepositories(t)
	const txID, userID, device = "tx-missing", "testuser", "device"
	now := time.Now()

	if err := repositories.TempReps.KCTempRep.AddKCInfo(ctx, commitTxTestKC("kc-4", now), txID, userID, device); err != nil {
		t.Fatalf("stage kc: %v", err)
	}
	text := Text{
		ID: "text-4", TargetID: "kc-4", Text: "text", RelatedTime: now,
		CreateTime: now, CreateApp: "test", CreateDevice: "device", CreateUser: "testuser",
		UpdateTime: now, UpdateApp: "test", UpdateDevice: "device", UpdateUser: "testuser",
	}
	// WriteTextRep はこのフィクスチャでは nil
	if err := repositories.TempReps.TextTempRep.AddTextInfo(ctx, text, txID, userID, device); err != nil {
		t.Fatalf("stage text: %v", err)
	}

	_, err := repositories.CommitTx(ctx, txID, userID, device)
	var missing *CommitTxWriteRepMissingError
	if !errors.As(err, &missing) || missing.DataType != "text" {
		t.Fatalf("書き込み rep 未設定のエラーでない: %v", err)
	}
	kc, err := repositories.WriteKCRep.GetKC(ctx, "kc-4", nil)
	if err != nil {
		t.Fatalf("get kc: %v", err)
	}
	if kc != nil {
		t.Errorf("書き込み rep 未設定の tx で kc だけ書かれている: %+v", kc)
	}
}

// 空の tx の commit は何もせず成功する。
func TestCommitTx_EmptyTxIsNoop(t *testing.T) {
	ctx := context.Background()
	repositories := newCommitTxTestRepositories(t)
	committed, err := repositories.CommitTx(ctx, "tx-empty", "testuser", "device")
	if err != nil {
		t.Fatalf("commit empty tx: %v", err)
	}
	if len(committed) != 0 {
		t.Errorf("空の tx が committed を返した: %+v", committed)
	}
}

// ATTACH した1接続で非修飾のテーブル名（INSERT INTO KMEMO ...）がそのまま目当てのファイルに落ちる前提は、
// 13 型のテーブル名が重複しないことに依る。leaf 実装の CREATE TABLE をソース走査で固定する。
func TestCommitTx_LeafTableNamesAreUnique(t *testing.T) {
	// MiReKyou だけは定数（miReKyouTableName）を連結して CREATE TABLE を組むので、定数の値も拾う
	createTable := regexp.MustCompile(`CREATE TABLE IF NOT EXISTS "?([A-Z_]+)"?|TableName = "([A-Z_]+)"`)
	files, err := filepath.Glob("*_repository_sqlite3_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string][]string{}
	for _, file := range files {
		if strings.Contains(file, "_temp_") {
			// temp rep は別の DB（ATTACH しない）。IDF の temp だけ命名が leaf と同じ末尾になる
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range createTable.FindAllStringSubmatch(string(content), -1) {
			name := m[1]
			if name == "" {
				name = m[2]
			}
			if name == "GKILL_META_INFO" {
				// スキーマ版の表。全ファイルにあるが commit_tx は触らない
				continue
			}
			seen[name] = append(seen[name], file)
		}
	}
	if len(seen) < 13 {
		t.Fatalf("leaf のテーブルが %d 種しか見つからない（13 のはず）。正規表現がずれている可能性がある", len(seen))
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		files := seen[name]
		unique := map[string]struct{}{}
		for _, f := range files {
			unique[f] = struct{}{}
		}
		if len(unique) > 1 {
			t.Errorf("テーブル名 %s が複数の leaf にある（ATTACH した接続で非修飾名が別のファイルに落ちる）: %s",
				name, strings.Join(files, ", "))
		}
	}
}
