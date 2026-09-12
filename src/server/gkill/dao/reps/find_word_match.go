package reps

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// キーワード判定の本体は api/find_word.MatchLoweredWords にある。
//
// SQLite で検索するリポジトリは sqlite3impl.GenerateFindSQLCommon が生成する
// WHERE 句で同じ判定をしていて、プラグイン SDK（plugin/sdk の Query.MatchText）も同じ関数を使う。
// Go 側で判定する IDF / git（local_dir）/ プラグイン型別アダプタは、このファイルのヘルパで
// 検索対象テキストと ID を組み立てて find_word へ渡す。規則を変えるときは SQL 側も揃えること。

// findWordIDOf はキーワード判定に渡す ID を返します。
//
// 肯定語は「対象テキストに含む OR ID が語で始まる」だが、query.WordsSkipIDMatch が真のときは
// ID を見ない（除外語を肯定語として再検索する内部クエリ用。find.FindQuery の doc を参照）。
// find_word.MatchLoweredWords は空の id を「ID 照合なし」と解釈する。
func findWordIDOf(query *find.FindQuery, id string) string {
	if query.WordsSkipIDMatch {
		return ""
	}
	return strings.ToLower(id)
}

// findWordTextOfIDFKyou はIDFKyouのキーワード検索対象テキストを小文字で組み立てます。
//
// 対象はrep内の相対パス（targetFile）と、拡張子が .md / .txt のときはその本文です。
// absolutePath は本文を読むためだけに使い、検索対象には含めません。
// 絶対パスを含めると、repの置かれたフォルダ名やドライブ文字が検索語・除外語に
// 引っかかってしまい、たとえば除外語 -downloads で Downloads rep が丸ごと消える。
//
// 本文が読めないファイルは本文なしとして扱い、検索自体は続行します。
// 1ファイルの読み取り失敗でrep全体の検索が落ちるのを避けるため。
func findWordTextOfIDFKyou(ctx context.Context, targetFile string, absolutePath string) string {
	text := strings.ToLower(targetFile)

	switch strings.ToLower(filepath.Ext(targetFile)) {
	case ".md", ".txt":
	default:
		return text
	}
	if absolutePath == "" {
		return text
	}

	file, err := os.OpenFile(absolutePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at open file for find word", "file", absolutePath, "error", fmt.Sprintf("%q", err))
		return text
	}
	b, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at read all file content for find word", "file", absolutePath, "error", fmt.Sprintf("%q", err))
		return text
	}
	return text + strings.ToLower(string(b))
}

// findWordTextOfGitCommit はGitCommitLogのキーワード検索対象テキストを小文字で組み立てます。
//
// 対象はコミットメッセージだけです。コミットIDは検索対象テキストに連結せず、
// find_word.MatchLoweredWords の id 引数として渡します（肯定語は前方一致、除外語は見ない）。
// キャッシュ側のSQL（COMMIT_MESSAGE 列 + ID の前方一致）と同じ意味になります。
// 以前は NUL 区切りで ID を連結して部分一致させており、短い hex 語で無関係なコミットが当たっていました。
func findWordTextOfGitCommit(message string) string {
	return strings.ToLower(message)
}
