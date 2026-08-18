package reps

// タグ絞り込みを「SQLで名前を絞る」から「全部取ってGoで照合する」へ寄せてよいかの検証。
//
// tag rep のワード検索は `findWordUseLike=false` / `ignoreCase=true` なので
// `LOWER(TAG) = LOWER(?) OR LOWER(ID) = LOWER(?)` を出す。列に関数がかかるため索引が効かず、
// **全行に LOWER() を適用**したうえで、クエリのタグ名の数だけそれを繰り返す。
// 一方「全部取ってGoで照合」は行数に比例した実体化(reps.Tag は240バイト+文字列10本)を払う。
//
// どちらが安いかはクエリのタグ名の個数で決まる。実測(2万タグ, 2026-08-19):
//
//	タグ名   SQLで絞る                          全部取ってGoで照合
//	  1      12,519,500 ns   4,645B      91確保   141,752,067 ns  38.5MB  580,074確保
//	  3      24,996,633 ns   8,170B     166確保   159,487,967 ns  38.5MB  580,074確保
//	 10      69,170,933 ns  22,341B     425確保   153,152,533 ns  38.5MB  580,074確保
//	 30     197,641,867 ns  51,333B   1,148確保   180,454,467 ns  38.5MB  580,074確保  ← ここで交差
//	100     724,706,233 ns 177,277B   3,684確保   257,011,367 ns  38.5MB  580,076確保
//
// SQL側は名前の数に比例、Go側は名前の数によらずほぼ一定(行数ぶんの実体化が支配的)。
// どちらも行数には比例するので、**交差する「名前の個数」は行数によらずほぼ一定**。
// api の maxTagNamesForSQLFilter はこれを根拠に決めてある。
// なお「タグ無し」仮想タグを使う検索は RelatedTagIDs のために結局全タグを取るので、
// そのときは名前の数によらずGo側で照合したほうが安い(スキャンが1回で済む)。
//
// **ns/op はこのマシンで倍近くぶれるので、判断は allocs/op と B/op で行うこと。**

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

const benchTagCount = 20_000

// seed は1件ずつのINSERTで数分かかるので、ベンチの間で使い回す
var (
	benchTagRepo     TagRepository
	benchTagRepoOnce sync.Once
)

func getBenchCachedTagRepo(b *testing.B) TagRepository {
	b.Helper()
	benchTagRepoOnce.Do(func() {
		ctx := context.Background()
		dir, err := os.MkdirTemp("", "gkill_tag_bench")
		if err != nil {
			b.Fatalf("temp dir: %v", err)
		}
		baseRepo, err := NewTagRepositorySQLite3Impl(ctx, filepath.Join(dir, "tag.db"), true)
		if err != nil {
			b.Fatalf("failed to create tag repo: %v", err)
		}
		for i := range benchTagCount {
			tag := makeTag(fmt.Sprintf("tag-%06d", i), fmt.Sprintf("target-%06d", i), fmt.Sprintf("タグ%06d", i))
			if err := baseRepo.AddTagInfo(ctx, tag); err != nil {
				b.Fatalf("AddTagInfo failed: %v", err)
			}
		}
		cacheDB, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			b.Fatalf("failed to open in-memory sqlite: %v", err)
		}
		repo, err := NewTagRepositoryCachedSQLite3Impl(ctx, baseRepo, cacheDB, &sync.RWMutex{}, "TAG_CACHE")
		if err != nil {
			b.Fatalf("failed to create cached tag repo: %v", err)
		}
		if err := repo.UpdateCache(ctx); err != nil {
			b.Fatalf("UpdateCache failed: %v", err)
		}
		benchTagRepo = repo
	})
	return benchTagRepo
}

func benchTagNames(count int) []string {
	names := make([]string, 0, count)
	for i := range count {
		names = append(names, fmt.Sprintf("タグ%06d", i))
	}
	return names
}

// SQLで名前を絞る(現行 findTags の形)。全行に LOWER() が名前の数だけかかる
func BenchmarkTagFindByNameInSQL(b *testing.B) {
	ctx := context.Background()
	repo := getBenchCachedTagRepo(b)
	for _, nameCount := range []int{1, 3, 10, 30, 100} {
		b.Run(fmt.Sprintf("names=%d", nameCount), func(b *testing.B) {
			query := &find.FindQuery{Words: benchTagNames(nameCount), WordsAnd: false, OnlyLatestData: true}
			b.ReportAllocs()
			for b.Loop() {
				tags, err := repo.FindTags(ctx, query)
				if err != nil {
					b.Fatal(err)
				}
				if len(tags) != nameCount {
					b.Fatalf("got %d tags, want %d", len(tags), nameCount)
				}
			}
		})
	}
}

// 全部取ってGoで照合する。名前の数によらず行数ぶんの実体化を払う
func BenchmarkTagFindByNameInGo(b *testing.B) {
	ctx := context.Background()
	repo := getBenchCachedTagRepo(b)
	for _, nameCount := range []int{1, 3, 10, 30, 100} {
		b.Run(fmt.Sprintf("names=%d", nameCount), func(b *testing.B) {
			query := &find.FindQuery{IsDeleted: false, OnlyLatestData: true}
			names := benchTagNames(nameCount)
			b.ReportAllocs()
			for b.Loop() {
				allTags, err := repo.FindTags(ctx, query)
				if err != nil {
					b.Fatal(err)
				}
				matched := 0
				for _, tag := range allTags {
					for _, name := range names {
						if strings.EqualFold(name, tag.Tag) || strings.EqualFold(name, tag.ID) {
							matched++
							break
						}
					}
				}
				if matched != nameCount {
					b.Fatalf("got %d matches, want %d", matched, nameCount)
				}
			}
		})
	}
}
