package reps

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// matchFindWords は検索対象テキストがキーワード条件を満たすか判定します。
//
// text・words・notWords はいずれも呼び出し元で小文字化済みであること。
// 肯定語が空のときは肯定条件なしとして扱い、AND・OR どちらでも通します。
// 除外語はいずれか1語でも含まれていれば不一致です。
//
// SQLite で検索するリポジトリは sqlite3impl.GenerateFindSQLCommon が生成する
// WHERE 句で同じ判定をしています。片方だけを変えるとリポジトリ種別によって
// 検索結果が食い違うので、意味を変えるときは必ず両方を揃えること。
//
// 以前はこの判定が各リポジトリに直接書かれており、除外語のループが
// match = strings.Contains(...) と代入になっていたため、
// 「除外語を含まない」行まで不一致として落としていた。
// 除外語を1語でも指定すると全件0件になるのがその症状。
// 同様に OR 検索の分岐が match = false から始まっていたため、
// 除外語だけを指定した検索（肯定語が空）も必ず0件になっていた。
func matchFindWords(text string, words []string, notWords []string, wordsAnd bool) bool {
	if wordsAnd {
		for _, word := range words {
			if !strings.Contains(text, word) {
				return false
			}
		}
	} else if len(words) != 0 {
		matched := false
		for _, word := range words {
			if strings.Contains(text, word) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	for _, notWord := range notWords {
		if strings.Contains(text, notWord) {
			return false
		}
	}
	return true
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
		slog.Log(ctx, gkill_log.Debug, "error at open file for find word", "file", absolutePath, "error", err)
		return text
	}
	b, err := io.ReadAll(file)
	file.Close()
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at read all file content for find word", "file", absolutePath, "error", err)
		return text
	}
	return text + strings.ToLower(string(b))
}

// findWordTextOfGitCommit はGitCommitLogのキーワード検索対象テキストを小文字で組み立てます。
//
// 対象はコミットメッセージとコミットIDです。
// NUL区切りで連結しているのは、境界をまたいだ検索語が誤ってヒットしないようにするため。
// 連結した1本のテキストとして扱うことで、肯定語は「どちらかに含む」、
// 除外語は「どちらにも含まない」となり、キャッシュ側のSQL（COMMIT_MESSAGE と ID を見る）と
// 同じ意味になります。
func findWordTextOfGitCommit(message string, commitID string) string {
	return strings.ToLower(message) + "\x00" + strings.ToLower(commitID)
}

// lowerFindWords は検索語を小文字化した新しいスライスを返します。
//
// query は全リポジトリで共有されているので、スライスの中身を直接書き換えると
// 並列に走っている他リポジトリの検索語まで小文字化して壊してしまう。必ず複製すること。
func lowerFindWords(words []string) []string {
	if len(words) == 0 {
		return nil
	}
	lowered := make([]string, len(words))
	for i, word := range words {
		lowered[i] = strings.ToLower(word)
	}
	return lowered
}
