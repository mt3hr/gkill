package reps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// キャッシュrepの GetLatestDataRepositoryAddress は、行ごとの REP_NAME 列を
// LATEST_DATA_REPOSITORY_NAME に射影しなければならない。
// GetRepName() をバインドしてはいけない ―― キャッシュrepが包んでいるのは集約
// （KmemoRepositories など）で、その GetRepName は "KmemoReps" のような
// 実在しない名前を返すためである。
//
// GkillRepositories.GetKyou はアドレス表のこの名前を Reps.UnWrap() が返す
// leaf rep の実名と突き合わせて問い合わせ先を1repに絞る。集約名を焼くと
// 比較が永遠に外れ、全repが continue されて「エラーも立たず nil」で返る。
// 呼び出し元の usecase/tag.go・usecase/text.go の実在検査はその nil を
// 「対象が存在しない」と読むので、実在する記録へのタグ/テキスト追加が
// ERR000092 で全滅する（2026-08-24 の事故。ADR-0019）。
//
// 型を1つ足すたびに同じ穴が開きうるので、ソース走査で落とす。
//
// MiReKyou がこの走査に出てこないのは意図どおり。あちらの SQL は leaf 実装と
// 共有の queryMiReKyouLatestDataRepositoryAddress にあり、かつキャッシュrepの
// UnWrap() が自分自身を返すので、アドレス表の名前と突き合わせ相手の名前が
// 同じ GetRepName() で一貫している。REP_NAME 列へ変えるとむしろ壊れる。
func TestCachedRepLatestDataAddressUsesRowRepName(t *testing.T) {
	const forbidden = "? AS LATEST_DATA_REPOSITORY_NAME"

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}

	scanned := 0
	violations := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_cached_sqlite3_impl.go") {
			continue
		}
		content, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		scanned++
		for i, line := range strings.Split(string(content), "\n") {
			if strings.Contains(line, forbidden) {
				violations = append(violations, fmt.Sprintf("%s:%d", filepath.ToSlash(name), i+1))
			}
		}
	}

	if scanned == 0 {
		t.Fatal("*_cached_sqlite3_impl.go を1つも走査できていない。走査条件が壊れている")
	}
	if len(violations) != 0 {
		t.Errorf("キャッシュrepが集約名を最新版アドレス表へ焼いている（REP_NAME 列を射影すること）: %v", violations)
	}
}
