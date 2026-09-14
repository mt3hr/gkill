package main

import (
	"archive/zip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// testdata に .git は置けない（git が入れ子の .git を追跡しない）ので、
// テストのたびに go-git でリポジトリを作り、archive/zip で固める。

// テストで使う固定の時刻。コミットごとにタイムゾーンを変え、native と同じく
// コミット固有のゾーンで戻ることを確かめる。
var (
	testTimeJST = time.Date(2021, 3, 8, 1, 58, 55, 0, time.FixedZone("JST", 9*3600))
	testTimeUTC = time.Date(2020, 10, 12, 9, 5, 23, 0, time.UTC)
)

// testCommit はテストリポジトリに積むコミット1件。
type testCommit struct {
	Message string
	When    time.Time
	// Files はこのコミットで書くファイル（パス → 内容）。nil なら削除。
	Files map[string]*string
}

func str(s string) *string { return &s }

// initTestRepo は dir に git リポジトリを作り、commits を順に積む。返り値はハッシュ（積んだ順）。
func initTestRepo(t *testing.T, dir string, commits []testCommit) []string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return commitTestRepo(t, dir, commits)
}

// commitTestRepo は既存のリポジトリ dir に commits を順に積む。返り値はハッシュ（積んだ順）。
func commitTestRepo(t *testing.T, dir string, commits []testCommit) []string {
	t.Helper()
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("git open: %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	hashes := []string{}
	for _, commit := range commits {
		for path, content := range commit.Files {
			full := filepath.Join(dir, filepath.FromSlash(path))
			if content == nil {
				if _, err := worktree.Remove(path); err != nil {
					t.Fatalf("git rm %s: %v", path, err)
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(*content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := worktree.Add(path); err != nil {
				t.Fatalf("git add %s: %v", path, err)
			}
		}
		signature := &object.Signature{Name: "Test Author", Email: "author@example.com", When: commit.When}
		hash, err := worktree.Commit(commit.Message, &git.CommitOptions{Author: signature, Committer: signature})
		if err != nil {
			t.Fatalf("git commit: %v", err)
		}
		hashes = append(hashes, hash.String())
	}
	return hashes
}

// repackTestRepo は loose オブジェクトを packfile にまとめる。pack 形式の zip を作るため。
func repackTestRepo(t *testing.T, dir string) {
	t.Helper()
	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RepackObjects(&git.RepackConfig{}); err != nil {
		t.Fatalf("repack: %v", err)
	}
}

// zipDir は root 配下を zip に固める。prefix が空ならルート直下に（.git/… 形）、
// 非空なら prefix/… の下に入れる（name/.git/… 形）。
func zipDir(t *testing.T, zipPath string, root string, prefix string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	writer := zip.NewWriter(file)
	err = filepath.WalkDir(root, func(walkPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, walkPath)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(rel)
		if prefix != "" {
			name = prefix + "/" + name
		}
		if entry.IsDir() {
			_, err := writer.Create(name + "/")
			return err
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.Modified = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		w, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(walkPath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		t.Fatalf("zip %s: %v", root, err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

// makeRepoZip はリポジトリを作って name/.git/… 形の zip にし、zip のパスとハッシュを返す。
func makeRepoZip(t *testing.T, zipDirPath string, name string, commits []testCommit) (string, []string) {
	t.Helper()
	work := filepath.Join(t.TempDir(), name)
	hashes := initTestRepo(t, work, commits)
	zipPath := filepath.Join(zipDirPath, name+".zip")
	zipDir(t, zipPath, work, name)
	return zipPath, hashes
}

// twoCommits は2件のコミット（JST と UTC）。
func twoCommits() []testCommit {
	return []testCommit{
		{Message: "Initial commit", When: testTimeUTC, Files: map[string]*string{"README.md": str("hello\nworld\n")}},
		{Message: "to github\n", When: testTimeJST, Files: map[string]*string{"README.md": str("hello\n"), "main.go": str("package main\n\nfunc main() {}\n")}},
	}
}

func newTestCache(t *testing.T) (*cache, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	pluginDir := filepath.Join(home, "plugins", "testuser", appName)
	if err := os.MkdirAll(pluginDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	c := &cache{}
	t.Cleanup(func() {
		if c.db != nil {
			_ = c.db.Close()
		}
	})
	return c, pluginDir
}

func testConfig(patterns ...string) pluginConfig {
	return pluginConfig{Patterns: patterns, MaxGitDirBytes: defaultMaxGitDirMB << 20}
}

func mustBuild(t *testing.T, c *cache, pluginDir string, patterns ...string) {
	t.Helper()
	if err := c.build(t.Context(), pluginDir, testConfig(patterns...)); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func countRows(t *testing.T, c *cache, query string, args ...any) int {
	t.Helper()
	count := 0
	if err := c.conn().QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return count
}

// commitsByHash は QueryCommits の結果をハッシュで引けるようにする。
func commitsByHash(t *testing.T, c *cache, pluginDir string) map[string]commitRow {
	t.Helper()
	rows, err := c.QueryCommits(pluginDir, nil, nil, 0, false)
	if err != nil {
		t.Fatalf("QueryCommits: %v", err)
	}
	byHash := map[string]commitRow{}
	for _, row := range rows {
		byHash[row.Hash] = row
	}
	return byHash
}
