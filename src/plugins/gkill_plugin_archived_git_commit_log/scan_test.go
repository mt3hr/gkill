package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// zip 内のパスから .git の場所と rep 名を決める規則。
// 実データの形: name/.git/…（大半）、ルート直下 .git/…、入れ子 kokko/wiki/.git/…、1 zip に複数。
func TestGitDirOfAndRepName(t *testing.T) {
	for _, tc := range []struct {
		entry   string
		gitDir  string
		ok      bool
		repName string
	}{
		{entry: "racoonboard/.git/HEAD", gitDir: "racoonboard/.git", ok: true, repName: "racoonboard"},
		{entry: "kokko/wiki/.git/objects/ab/cdef", gitDir: "kokko/wiki/.git", ok: true, repName: "wiki"},
		{entry: ".git/refs/heads/master", gitDir: ".git", ok: true, repName: "weplog"},
		{entry: "a/.git/modules/b/.git/HEAD", gitDir: "a/.git", ok: true, repName: "a"},
		{entry: "racoonboard/README.md", ok: false},
		{entry: "racoonboard/.gitignore", ok: false},
		{entry: "racoonboard/.git", ok: false}, // gitdir: ポインタのファイル
	} {
		gitDir, ok := gitDirOf(tc.entry)
		if ok != tc.ok || gitDir != tc.gitDir {
			t.Errorf("gitDirOf(%q) = %q, %v; want %q, %v", tc.entry, gitDir, ok, tc.gitDir, tc.ok)
			continue
		}
		if !ok {
			continue
		}
		if got := repNameOf("D:/Kyou/Box_A/weplog.zip", gitDir); got != tc.repName {
			t.Errorf("repNameOf(%q) = %q, want %q", gitDir, got, tc.repName)
		}
	}
}

// go-git が読むエントリだけを受け取り、hooks / logs / index は読まないこと。
func TestAcceptGitEntry(t *testing.T) {
	for _, tc := range []struct {
		entry string
		want  bool
	}{
		{"x/.git/HEAD", true},
		{"x/.git/config", true},
		{"x/.git/packed-refs", true},
		{"x/.git/shallow", true},
		{"x/.git/refs/heads/master", true},
		{"x/.git/refs/remotes/origin/HEAD", true},
		{"x/.git/objects/ab/cdef0123", true},
		{"x/.git/objects/pack/pack-1.pack", true},
		{"x/.git/objects/pack/pack-1.idx", true},
		{"x/.git/hooks/pre-commit.sample", false},
		{"x/.git/logs/HEAD", false},
		{"x/.git/index", false},
		{"x/.git/info/exclude", false},
		{"x/.git/COMMIT_EDITMSG", false},
		{"x/.git/description", false},
		{"x/README.md", false},
		{"x/.git", false},
	} {
		if got := acceptGitEntry(tc.entry); got != tc.want {
			t.Errorf("acceptGitEntry(%q) = %v, want %v", tc.entry, got, tc.want)
		}
	}
}

// 束の指紋はエントリの (名前, CRC32, サイズ) だけで決まり、並び順に依らないこと。
func TestGroupRepoBundlesFingerprint(t *testing.T) {
	entries := []sdk.SourceEntry{
		{ArchivePath: "D:/a.zip", EntryName: "x/.git/objects/ab/cd", CRC32: 1, Size: 10},
		{ArchivePath: "D:/a.zip", EntryName: "x/.git/HEAD", CRC32: 2, Size: 20},
		{ArchivePath: "D:/a.zip", EntryName: "y/.git/objects/ef/01", CRC32: 3, Size: 30}, // HEAD が無いので捨てる
		{ArchivePath: "D:/b.zip", EntryName: ".git/HEAD", CRC32: 2, Size: 20},
	}
	bundles := groupRepoBundles(entries)
	if len(bundles) != 2 {
		t.Fatalf("束の数 = %d, want 2（HEAD の無い y は捨てる）: %+v", len(bundles), bundles)
	}
	if bundles[0].ArchivePath != "D:/a.zip" || bundles[0].GitDir != "x/.git" || bundles[0].RepName != "x" || bundles[0].TotalSize != 30 {
		t.Errorf("束[0] = %+v", bundles[0])
	}
	if bundles[1].ArchivePath != "D:/b.zip" || bundles[1].GitDir != ".git" || bundles[1].RepName != "b" {
		t.Errorf("束[1] = %+v", bundles[1])
	}

	reversed := []sdk.SourceEntry{entries[1], entries[0]}
	if got := groupRepoBundles(reversed)[0].Fingerprint; got != bundles[0].Fingerprint {
		t.Error("並び順で指紋が変わる")
	}
	changed := []sdk.SourceEntry{entries[0], {ArchivePath: "D:/a.zip", EntryName: "x/.git/HEAD", CRC32: 99, Size: 20}}
	if got := groupRepoBundles(changed)[0].Fingerprint; got == bundles[0].Fingerprint {
		t.Error("CRC32 が変わっても指紋が同じ")
	}
}

// Takeout 向けの問題（zip の中の zip・展開済みフォルダ・時期の混在）は出さないこと。
func TestRelevantProblems(t *testing.T) {
	problems := relevantProblems([]sdk.SourceProblem{
		{Kind: sdk.ProblemNestedZip}, {Kind: sdk.ProblemExtractedFolder}, {Kind: sdk.ProblemMixedExports},
		{Kind: sdk.ProblemMissingPattern}, {Kind: sdk.ProblemSpannedArchive}, {Kind: sdk.ProblemBrokenArchive},
	})
	if len(problems) != 3 {
		t.Errorf("残った問題 = %+v, want missing_pattern / spanned_archive / broken_archive", problems)
	}
}
