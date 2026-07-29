package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mattn/go-zglob"
)

// データソースの指定を実在するパスへ展開する。
// gkill_plugin_claudecode と同じ仕様。プラグインは独立モジュールで共有できないためコピーしている。

// parseSourcePatterns は設定値をパターンのリストにする。
// 設定は文字列(改行区切り)でも配列でも書ける。
//
//	"source_dirs": "C:\\a\nC:\\b"
//	"source_dirs": ["C:\\a", "C:\\b"]
//
// 空なら defaultDir にフォールバックする。
func parseSourcePatterns(value any, defaultDir string) []string {
	var patterns []string
	add := func(s string) {
		s = strings.TrimSpace(strings.TrimSuffix(s, "\r"))
		if s == "" {
			return
		}
		patterns = append(patterns, expandHome(os.ExpandEnv(s)))
	}

	switch v := value.(type) {
	case nil:
	case string:
		for line := range strings.SplitSeq(v, "\n") {
			add(line)
		}
	case []string:
		for _, s := range v {
			for line := range strings.SplitSeq(s, "\n") {
				add(line)
			}
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				for line := range strings.SplitSeq(s, "\n") {
					add(line)
				}
			}
		}
	}

	if len(patterns) == 0 && defaultDir != "" {
		patterns = append(patterns, defaultDir)
	}
	return patterns
}

// expandHome は先頭の ~ をホームディレクトリに展開する。
// Windowsサービスとして動くgkillでは実行アカウントのホームになる点に注意
// (LocalSystemなら systemprofile)。確実に指定したいときは絶対パスを使う。
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// hasGlobMeta はパターンにワイルドカードが含まれるかを判定する。
func hasGlobMeta(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// expandedSource はパターンを展開した結果。
type expandedSource struct {
	Dirs    []string // 再帰的に走査するディレクトリ
	Files   []string // 直接対象にするファイル
	Missing []string // 実在しなかった、あるいは何にもマッチしなかったパターン
}

// expandSourcePatterns はパターンを実在するパスへ展開する。
// ワイルドカード(*, **, ?, [])を含むパターンはグロブ展開し、
// マッチしたものがディレクトリなら再帰走査の対象、ファイルならそのまま対象にする。
func expandSourcePatterns(patterns []string) expandedSource {
	var result expandedSource
	seen := map[string]bool{}

	classify := func(path string) bool {
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		if seen[abs] {
			return true
		}
		info, err := os.Stat(abs)
		if err != nil {
			return false
		}
		seen[abs] = true
		if info.IsDir() {
			result.Dirs = append(result.Dirs, abs)
		} else {
			result.Files = append(result.Files, abs)
		}
		return true
	}

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if !hasGlobMeta(pattern) {
			if !classify(pattern) {
				result.Missing = append(result.Missing, pattern)
			}
			continue
		}
		matches, err := zglob.Glob(pattern)
		if err != nil || len(matches) == 0 {
			result.Missing = append(result.Missing, pattern)
			continue
		}
		matched := false
		for _, m := range matches {
			if classify(m) {
				matched = true
			}
		}
		if !matched {
			result.Missing = append(result.Missing, pattern)
		}
	}
	return result
}

// collectSourceFiles は展開結果から、matches が真を返すファイルを集める。
// ディレクトリは再帰的に走査し、直接指定されたファイルは名前で判定せずそのまま採る。
func collectSourceFiles(src expandedSource, matches func(name string) bool) []string {
	var files []string
	seen := map[string]bool{}

	add := func(path string) {
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		if seen[abs] {
			return
		}
		seen[abs] = true
		files = append(files, abs)
	}

	for _, dir := range src.Dirs {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// 読めないディレクトリはスキップして続行する
				return nil //nolint:nilerr
			}
			if d.IsDir() {
				return nil
			}
			if matches(d.Name()) {
				add(path)
			}
			return nil
		})
	}
	// 直接指定されたファイルは、名前が規則に合わなくても使う
	for _, f := range src.Files {
		add(f)
	}

	slices.Sort(files)
	return files
}

// sourceSignature は全ソースファイルの path:mtime:size を連結した署名を返す。
// どれか1つでも変わればキャッシュを作り直す。
func sourceSignature(paths []string) string {
	var sb strings.Builder
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		sb.WriteString(p)
		sb.WriteString(":")
		sb.WriteString(strconv.FormatInt(info.ModTime().Unix(), 10))
		sb.WriteString(":")
		sb.WriteString(strconv.FormatInt(info.Size(), 10))
		sb.WriteString("\n")
	}
	return sb.String()
}
