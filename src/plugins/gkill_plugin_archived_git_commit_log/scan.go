package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// zip は展開しない。sdk.OpenSources で中央ディレクトリだけを列挙し、
// .git 配下のうち go-git が読むエントリだけを受け取る。
// hooks / logs / index / info は読まない（コミットログの復元に要らず、hooks の sample だけで
// 1リポジトリ十数ファイル増える）。

// acceptGitEntry は zip 内のパスが「読むべき .git 配下のファイル」かを返す。
//
// .git がファイル（worktree / submodule の gitdir: ポインタ）の場合は、
// 中身がそのファイル1つなので "/.git/" を含まず、ここで落ちる。
func acceptGitEntry(entryName string) bool {
	rel, ok := gitRelativePath(entryName)
	if !ok {
		return false
	}
	switch rel {
	case "HEAD", "config", "packed-refs", "shallow":
		return true
	}
	return strings.HasPrefix(rel, "refs/") || strings.HasPrefix(rel, "objects/")
}

// gitDirOf は zip 内のパスから、それが属する .git ディレクトリのパス（zip 内、末尾は ".git"）を返す。
// ".git/" が最初に現れる位置で切るので、入れ子の .git/modules/x/.git/... は外側のリポジトリに属する。
func gitDirOf(entryName string) (string, bool) {
	if strings.HasPrefix(entryName, gitDirName+"/") {
		return gitDirName, true
	}
	index := strings.Index(entryName, "/"+gitDirName+"/")
	if index < 0 {
		return "", false
	}
	return entryName[:index+1+len(gitDirName)], true
}

// gitRelativePath は zip 内のパスから .git 配下の相対パス（"objects/ab/cdef…" など）を返す。
func gitRelativePath(entryName string) (string, bool) {
	gitDir, ok := gitDirOf(entryName)
	if !ok {
		return "", false
	}
	return strings.TrimPrefix(entryName, gitDir+"/"), true
}

// repNameOf はリポジトリの rep 名を返す。native の git rep（filepath.Base(ディレクトリ)）と同じ規則。
//
//	name/.git/…        → "name"
//	kokko/wiki/.git/…  → "wiki"
//	.git/…（ルート直下） → zip のファイル名から拡張子を除いたもの
func repNameOf(archivePath string, gitDir string) string {
	if gitDir == gitDirName {
		base := filepath.Base(archivePath)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	return path.Base(path.Dir(gitDir))
}

// repoBundle は zip の中の1リポジトリぶんのエントリの束。
type repoBundle struct {
	ArchivePath string
	GitDir      string
	RepName     string
	// Fingerprint は全エントリの (名前, CRC32, サイズ) から作る指紋。
	// これが前回と同じなら中身も同じなので読み直さない（ADR-0303: mtime は使わない）。
	Fingerprint string
	TotalSize   int64
	Entries     []sdk.SourceEntry
}

// key はキャッシュの repo 表の主キーに対応する文字列。
func (b repoBundle) key() string {
	return bundleKey(b.ArchivePath, b.GitDir)
}

func bundleKey(archivePath string, gitDir string) string {
	return archivePath + "\x00" + gitDir
}

// groupRepoBundles はエントリを (zip, .git の場所) ごとに束ねる。並びは決定的（zip のパス → .git の場所）。
// HEAD の無い束は git リポジトリとして開けないので捨てる（objects だけが残っている壊れた zip など）。
func groupRepoBundles(entries []sdk.SourceEntry) []repoBundle {
	byKey := map[string]*repoBundle{}
	for _, entry := range entries {
		gitDir, ok := gitDirOf(entry.EntryName)
		if !ok {
			continue
		}
		key := bundleKey(entry.ArchivePath, gitDir)
		bundle, exist := byKey[key]
		if !exist {
			bundle = &repoBundle{
				ArchivePath: entry.ArchivePath,
				GitDir:      gitDir,
				RepName:     repNameOf(entry.ArchivePath, gitDir),
			}
			byKey[key] = bundle
		}
		bundle.Entries = append(bundle.Entries, entry)
		bundle.TotalSize += entry.Size
	}

	bundles := make([]repoBundle, 0, len(byKey))
	for _, bundle := range byKey {
		slices.SortFunc(bundle.Entries, func(a, b sdk.SourceEntry) int {
			return strings.Compare(a.EntryName, b.EntryName)
		})
		if !slices.ContainsFunc(bundle.Entries, func(entry sdk.SourceEntry) bool {
			return entry.EntryName == bundle.GitDir+"/HEAD"
		}) {
			continue
		}
		bundle.Fingerprint = fingerprintOf(bundle.Entries)
		bundles = append(bundles, *bundle)
	}
	slices.SortFunc(bundles, func(a, b repoBundle) int {
		if c := strings.Compare(a.ArchivePath, b.ArchivePath); c != 0 {
			return c
		}
		return strings.Compare(a.GitDir, b.GitDir)
	})
	return bundles
}

// fingerprintOf は並べ替え済みのエントリから指紋を作る。
func fingerprintOf(entries []sdk.SourceEntry) string {
	hasher := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hasher, "%s\x00%s\x00%s\n", entry.EntryName, strconv.FormatUint(uint64(entry.CRC32), 16), strconv.FormatInt(entry.Size, 10))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

// openSources は設定の指定から zip を開き、.git 配下のエントリだけを列挙する。
func openSources(patterns []string) (*sdk.SourceSet, error) {
	return sdk.OpenSources(patterns, acceptGitEntry)
}

// relevantProblems は設定画面に出す問題だけを残す。
//
// SDK の走査は Google Takeout 向けに作られており、「zip の中の zip」や
// 「展開済みの Takeout フォルダ」も報告する。前者は成果物入りのリポジトリで普通に起きるうえ
// 中に入らなくてよく、後者は文言が Takeout の話なので、このプラグインでは出さない。
func relevantProblems(problems []sdk.SourceProblem) []sdk.SourceProblem {
	kept := make([]sdk.SourceProblem, 0, len(problems))
	for _, problem := range problems {
		switch problem.Kind {
		case sdk.ProblemNestedZip, sdk.ProblemExtractedFolder, sdk.ProblemMixedExports:
			continue
		}
		kept = append(kept, problem)
	}
	return kept
}
