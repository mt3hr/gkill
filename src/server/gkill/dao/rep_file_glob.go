package dao

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// repFilePatternIgnoreCase はパターン照合で大文字小文字を無視するかどうか。
//
// go-zglob は windows / darwin のとき正規表現へ (?i:) を付けて大小を無視する。
// 実データでは大小が食い違っている設定は1件も無いが、綴りを間違えた設定を
// 救っているのはこの無視なので、置き換えでも同じ条件を維持する。
//
// テストから切り替えられるよう定数ではなく変数にしている。
var repFilePatternIgnoreCase = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

// expandRepFilePattern は REPOSITORY.FILE のパターンを実在するパスの一覧へ展開する。
//
// go-zglob の Glob は ** を含まないパターンでも対象ツリーを丸ごと再帰 walk する。
// マッチしたディレクトリで SkipDir を返さないうえ、枝刈り条件が root 直下では
// 成立しないため。数万ディレクトリ規模の置き場では、数十件を返すために
// 木の全部を歩いて秒単位かかる。
//
// だが * は go-zglob 内部で [^/]* になり / をまたげない。つまり
// 「静的なディレクトリ + 最終セグメントに *」という形のパターンでは、
// root 直下より深い階層に新しいマッチは原理的に存在しない。
// そこでその形のときだけ親ディレクトリを1回列挙して済ませる。集合は完全に等価。
//
// ** や {a,b} のように本当に木を歩く必要があるパターンは go-zglob へ渡す
// （実データには1件も無いが、documents/reverse/dvnf-rep-type-spec.md が
// directory 型の例として ** を挙げているため、経路は残す）。
//
// 呼び出し元は展開できなかったパターンを「該当なし」として扱えばよいので、
// go-zglob の Glob と同じくエラーは返さずログへ落とす。
func expandRepFilePattern(ctx context.Context, pattern string) []string {
	normalized := normalizeRepFilePattern(pattern)
	if normalized == "" {
		return nil
	}

	if repFilePatternNeedsWalk(normalized) {
		matchFiles, err := zglob.Glob(normalized)
		if err != nil {
			slog.Log(ctx, gkill_log.Warn, "error at glob repository file pattern", "pattern", fmt.Sprintf("%q", normalized), "error", fmt.Sprintf("%q", err))
		}
		return matchFiles
	}

	dir, base := splitRepFilePattern(normalized)

	// ワイルドカードが無いならリテラル。実在するかだけ見る。
	if !strings.Contains(base, "*") {
		if _, err := os.Stat(filepath.FromSlash(normalized)); err != nil {
			return nil
		}
		return []string{normalized}
	}

	// 環境変数が未設定だと $UNSET/Archive_* が /Archive_* へ展開され、
	// go-zglob ならドライブ全体を walk していた。ここで塞ぐ。
	if isFilesystemRootDir(dir) {
		slog.Log(ctx, gkill_log.Warn, "skip repository file pattern that expands to a filesystem root", "pattern", fmt.Sprintf("%q", normalized))
		return nil
	}

	listDir := dir
	if listDir == "" {
		listDir = "."
	}
	entries, err := os.ReadDir(filepath.FromSlash(listDir))
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at read directory for repository file pattern", "pattern", fmt.Sprintf("%q", normalized), "error", fmt.Sprintf("%q", err))
		return nil
	}

	matchFiles := []string{}
	for _, entry := range entries {
		if !matchRepFileSegment(base, entry.Name()) {
			continue
		}
		if dir == "" {
			matchFiles = append(matchFiles, entry.Name())
			continue
		}
		matchFiles = append(matchFiles, dir+"/"+entry.Name())
	}

	// 綴り違いと環境変数の未設定に気づけるようにしておく。
	// go-zglob 時代は木を歩いた末に黙って0件を返していた。
	if len(matchFiles) == 0 {
		slog.Log(ctx, gkill_log.Warn, "repository file pattern matched nothing", "pattern", fmt.Sprintf("%q", normalized))
	}
	return matchFiles
}

// normalizeRepFilePattern は展開前のパターンを / 区切りへ揃え、区切りの重複と . / .. を畳む。
//
// go-zglob も内部で同じこと（toSlash + path.Clean）をしている。
// 設定に使われている環境変数のうち1つは末尾に区切りを持つ値なので、
// 展開すると区切りが二連になる。畳まないと親ディレクトリの特定を誤る。
func normalizeRepFilePattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	normalized := filepath.ToSlash(pattern)
	normalized = path.Clean(normalized)
	if normalized == "." {
		return ""
	}
	return normalized
}

// repFilePatternNeedsWalk は go-zglob へ渡さないと展開できないパターンかを返す。
//
// ? と [ は go-zglob がメタ文字と見なさない（glob 判定が "*{" だけを見る）ので、
// ここでもリテラルとして扱う。filepath.Match へ渡すと意味論が変わるため使わない。
func repFilePatternNeedsWalk(pattern string) bool {
	if strings.Contains(pattern, "**") {
		return true
	}
	if strings.Contains(pattern, "{") {
		return true
	}
	if strings.Contains(pattern, "!(") {
		return true
	}
	// * が最終セグメント以外にあると、1回の列挙では展開できない
	if i := strings.LastIndex(pattern, "/"); i >= 0 && strings.Contains(pattern[:i], "*") {
		return true
	}
	return false
}

// splitRepFilePattern は正規化済みパターンを親ディレクトリと最終セグメントへ分ける。
// 親が無いとき（相対パスの1セグメントのみ）は dir が空になる。
func splitRepFilePattern(pattern string) (dir string, base string) {
	i := strings.LastIndex(pattern, "/")
	if i < 0 {
		return "", pattern
	}
	if i == 0 {
		return "/", pattern[1:]
	}
	return pattern[:i], pattern[i+1:]
}

// isFilesystemRootDir は親ディレクトリがファイルシステムのルートかを返す。
// "/" のほか、Windows のドライブ指定（"E:"）も対象にする。
func isFilesystemRootDir(dir string) bool {
	if dir == "/" {
		return true
	}
	if dir == "" {
		return false
	}
	return filepath.VolumeName(filepath.FromSlash(dir)) == filepath.FromSlash(dir)
}

// matchRepFileSegment は1セグメントぶんのパターン照合を行う。
// メタ文字は * だけで、* は / をまたがない（そもそも1セグメントなので / を含まない）。
func matchRepFileSegment(pattern string, name string) bool {
	if repFilePatternIgnoreCase {
		pattern = strings.ToLower(pattern)
		name = strings.ToLower(name)
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == name
	}

	head := parts[0]
	if !strings.HasPrefix(name, head) {
		return false
	}
	rest := name[len(head):]

	tail := parts[len(parts)-1]
	for _, middle := range parts[1 : len(parts)-1] {
		i := strings.Index(rest, middle)
		if i < 0 {
			return false
		}
		rest = rest[i+len(middle):]
	}

	if len(tail) > len(rest) {
		return false
	}
	return strings.HasSuffix(rest, tail)
}
