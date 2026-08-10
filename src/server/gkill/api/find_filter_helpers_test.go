package api

// find_filter_helpers.go のヘルパの回帰テスト。
//
// collectFromRepos は複数リポジトリへ並列にfnを投げて結果を集約する。
// 失敗したリポジトリのエラーは errors.Join でまとめて返す契約で、
// 「最初の1件だけ返す」形にすると、どのリポジトリが落ちたのか分からなくなる
// （同じ検索で複数repが落ちるのは珍しくない）。

import (
	"fmt"
	"strings"
	"testing"
)

// 複数のリポジトリが失敗したとき、全部のメッセージが結果エラーに含まれること
func TestCollectFromRepos_JoinsAllRepositoryErrors(t *testing.T) {
	repNames := []string{"rep-a", "rep-b"}

	items, err := collectFromRepos(repNames, func(repName string) ([]string, error) {
		return nil, fmt.Errorf("取得失敗 %s", repName)
	})

	if err == nil {
		t.Fatal("リポジトリが全て失敗したのにエラーが返っていない")
	}
	for _, repName := range repNames {
		if want := "取得失敗 " + repName; !strings.Contains(err.Error(), want) {
			t.Errorf("エラー %q が結果に含まれていない: %v", want, err)
		}
	}
	if items != nil {
		t.Errorf("エラー時は結果を返さないはず: %v", items)
	}
}
