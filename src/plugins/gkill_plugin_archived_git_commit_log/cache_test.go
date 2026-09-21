package main

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

// 初回構築で native の git rep と同じ列が入ること:
// ID=ハッシュ・rep 名=ディレクトリ名・時刻=コミッタ日時（コミット固有のゾーン）・
// author 名・メッセージは trim しない・行数は StatsContext の合計。
func TestBuildFromScratch(t *testing.T) {
	c, pluginDir := newTestCache(t)
	zipPath, hashes := makeRepoZip(t, t.TempDir(), "racoonboard", twoCommits())

	mustBuild(t, c, pluginDir, zipPath)

	byHash := commitsByHash(t, c, pluginDir)
	if len(byHash) != 2 {
		t.Fatalf("コミット数 = %d, want 2", len(byHash))
	}

	first := byHash[hashes[0]]
	if first.RepName != "racoonboard" {
		t.Errorf("RepName = %q, want racoonboard（.git を含むディレクトリ名）", first.RepName)
	}
	if first.Message != "Initial commit" || first.AuthorName != "Test Author" || first.AuthorEmail != "author@example.com" {
		t.Errorf("列が写っていない: %+v", first)
	}
	if !first.committedAt().Equal(testTimeUTC) {
		t.Errorf("committedAt = %v, want %v", first.committedAt(), testTimeUTC)
	}
	if _, offset := first.committedAt().Zone(); offset != 0 {
		t.Errorf("UTC のコミットのゾーンが %d 秒になっている", offset)
	}
	if first.Addition != 2 || first.Deletion != 0 {
		t.Errorf("root commit の行数 = +%d -%d, want +2 -0（空ツリーとの差）", first.Addition, first.Deletion)
	}

	second := byHash[hashes[1]]
	if second.Message != "to github\n" {
		t.Errorf("メッセージが trim されている: %q", second.Message)
	}
	if !second.committedAt().Equal(testTimeJST) {
		t.Errorf("committedAt = %v, want %v", second.committedAt(), testTimeJST)
	}
	if _, offset := second.committedAt().Zone(); offset != 9*3600 {
		t.Errorf("JST のコミットのゾーンが %d 秒になっている（native はコミット固有のゾーンを保つ）", offset)
	}
	if second.Addition != 3 || second.Deletion != 1 {
		t.Errorf("行数 = +%d -%d, want +3 -1", second.Addition, second.Deletion)
	}

	body, err := c.QueryBody(pluginDir, hashes[1])
	if err != nil {
		t.Fatalf("QueryBody: %v", err)
	}
	if len(body.Files) != 2 || body.Files[0].Path != "README.md" || body.Files[1].Path != "main.go" {
		t.Errorf("ファイル別の行数 = %+v", body.Files)
	}
	if len(body.Sources) != 1 || body.Sources[0].ArchivePath != zipPath || body.Sources[0].GitDir != "racoonboard/.git" {
		t.Errorf("出どころ = %+v, want %s!/racoonboard/.git", body.Sources, zipPath)
	}

	names, err := c.RepNames(pluginDir)
	if err != nil || !slices.Equal(names, []string{"racoonboard"}) {
		t.Errorf("RepNames = %v, %v", names, err)
	}
}

// Kyou への変換が native と同じ形になること（型別データ付き・アプリ名 git・author 名）。
func TestKyouOfMatchesNativeShape(t *testing.T) {
	row := commitRow{Hash: "abc", RepName: "ocha", CommitterUnix: testTimeJST.Unix(), TZOffsetSec: 9 * 3600,
		AuthorName: "Test Author", Message: "msg\n", Addition: 1, Deletion: 2}
	kyou := kyouOf(row)
	if kyou.ID != "abc" || kyou.RepName != "ocha" || kyou.DataType != "git_commit_log" {
		t.Errorf("ID/RepName/DataType = %q/%q/%q", kyou.ID, kyou.RepName, kyou.DataType)
	}
	if kyou.CreateApp != "git" || kyou.UpdateApp != "git" || kyou.CreateUser != "Test Author" || kyou.UpdateUser != "Test Author" {
		t.Errorf("アプリ名・利用者名が native と違う: %+v", kyou)
	}
	if !kyou.RelatedTime.Equal(testTimeJST) || !kyou.CreateTime.Equal(testTimeJST) || !kyou.UpdateTime.Equal(testTimeJST) {
		t.Errorf("時刻がコミッタ日時になっていない: %+v", kyou)
	}
	if kyou.Typed == nil || kyou.Typed.GitCommitLog == nil ||
		kyou.Typed.GitCommitLog.CommitMessage != "msg\n" || kyou.Typed.GitCommitLog.Addition != 1 || kyou.Typed.GitCommitLog.Deletion != 2 {
		t.Errorf("型別データが載っていない: %+v", kyou.Typed)
	}
}

// 同じリポジトリを別の時期に固めた zip が2つあっても、同じハッシュは1件。
// 片方だけにあるコミットは残り、出どころは両方が記録される。
func TestSameRepositoryInTwoArchivesIsDeduplicated(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()

	work := filepath.Join(t.TempDir(), "idfsaa")
	hashes := initTestRepo(t, work, twoCommits())
	olderZip := filepath.Join(dir, "old", "idfsaa.zip")
	zipDir(t, olderZip, work, "idfsaa")

	// もう1件積んでから固め直す
	extra := initTestRepoAppend(t, work, testCommit{Message: "remove go.mod", When: testTimeJST.Add(time.Hour), Files: map[string]*string{"main.go": nil}})
	newerZip := filepath.Join(dir, "new", "idfsaa.zip")
	zipDir(t, newerZip, work, "idfsaa")

	mustBuild(t, c, pluginDir, olderZip, newerZip)

	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 3 {
		t.Errorf("コミット数 = %d, want 3（2件は両方の zip にあるが1行ずつ）", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_repo WHERE hash = ?`, hashes[0]); got != 2 {
		t.Errorf("共通コミットの出どころ = %d, want 2", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_repo WHERE hash = ?`, extra); got != 1 {
		t.Errorf("新しい zip だけのコミットの出どころ = %d, want 1", got)
	}
	names, _ := c.RepNames(pluginDir)
	if !slices.Equal(names, []string{"idfsaa"}) {
		t.Errorf("RepNames = %v, want [idfsaa]（同名は1つ）", names)
	}

	// 古い zip を外しても、新しい zip に入っているコミットは全部残る
	mustBuild(t, c, pluginDir, newerZip)
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 3 {
		t.Errorf("古い zip を外した後のコミット数 = %d, want 3", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_repo WHERE hash = ?`, hashes[0]); got != 1 {
		t.Errorf("古い zip を外した後の出どころ = %d, want 1", got)
	}

	// 新しい zip も外すと全部消える
	mustBuild(t, c, pluginDir, filepath.Join(dir, "nothing_here"))
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 0 {
		t.Errorf("全部外した後のコミット数 = %d, want 0", got)
	}
	names, _ = c.RepNames(pluginDir)
	if names == nil || len(names) != 0 {
		t.Errorf("RepNames = %#v, want 空スライス（nil ではない）", names)
	}
}

// initTestRepoAppend は既存のリポジトリにコミットを1件積む。
func initTestRepoAppend(t *testing.T, dir string, commit testCommit) string {
	t.Helper()
	return commitTestRepo(t, dir, []testCommit{commit})[0]
}

// git init 直後（コミット0件）の .git は 0 件で正常終了し、エラーも問題も出ないこと。
// 実データでは十数個ある。
func TestEmptyRepositoryYieldsNothing(t *testing.T) {
	c, pluginDir := newTestCache(t)
	work := filepath.Join(t.TempDir(), "ndsg")
	initTestRepo(t, work, nil)
	zipPath := filepath.Join(t.TempDir(), "ndsg.zip")
	zipDir(t, zipPath, work, "ndsg")

	mustBuild(t, c, pluginDir, zipPath)

	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 0 {
		t.Errorf("コミット数 = %d, want 0", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo WHERE note != ''`); got != 0 {
		t.Errorf("空のリポジトリが「読めなかった」扱いになっている")
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo`); got != 1 {
		t.Errorf("repo 行 = %d, want 1（走査済みとして記録する）", got)
	}
	if state := c.getMeta("build_state"); state != "idle" {
		t.Errorf("build_state = %q, want idle", state)
	}
}

// .git の無い zip はリポジトリとして数えず、問題としても報告しない（ソース snapshot は普通にある）。
func TestArchiveWithoutGitIsSkippedSilently(t *testing.T) {
	c, pluginDir := newTestCache(t)
	work := filepath.Join(t.TempDir(), "hbg_src")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "storage.go"), []byte("package hbg"), 0o644); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "hbg.zip")
	zipDir(t, zipPath, work, "")

	mustBuild(t, c, pluginDir, zipPath)

	if got := countRows(t, c, `SELECT COUNT(*) FROM repo`); got != 0 {
		t.Errorf("repo 行 = %d, want 0", got)
	}
	if problems := c.loadSourceProblems(); len(problems) != 0 {
		t.Errorf("問題として報告された: %+v", problems)
	}
	if got := c.getMeta("target_repo_count"); got != "0" {
		t.Errorf("target_repo_count = %q, want 0", got)
	}
}

// 1つの zip に複数のリポジトリ（入れ子を含む）があれば、それぞれ別の rep 名になること。
// ルート直下の .git は zip 名を rep 名にする。
func TestMultipleRepositoriesInOneArchive(t *testing.T) {
	c, pluginDir := newTestCache(t)
	root := t.TempDir()
	initTestRepo(t, filepath.Join(root, "kokko"), twoCommits()[:1])
	initTestRepo(t, filepath.Join(root, "kokko", "wiki"), []testCommit{
		{Message: "wiki", When: testTimeJST, Files: map[string]*string{"Home.md": str("# wiki\n")}},
	})
	initTestRepo(t, filepath.Join(root, "dllog"), []testCommit{
		{Message: "dllog", When: testTimeJST, Files: map[string]*string{"a.txt": str("a\n")}},
	})
	multiZip := filepath.Join(t.TempDir(), "kokko.zip")
	zipDir(t, multiZip, root, "")

	rootWork := filepath.Join(t.TempDir(), "weplog_work")
	initTestRepo(t, rootWork, []testCommit{
		{Message: "weplog", When: testTimeUTC, Files: map[string]*string{"w.txt": str("w\n")}},
	})
	rootZip := filepath.Join(t.TempDir(), "weplog.zip")
	zipDir(t, rootZip, rootWork, "")

	mustBuild(t, c, pluginDir, multiZip, rootZip)

	names, _ := c.RepNames(pluginDir)
	if !slices.Equal(names, []string{"dllog", "kokko", "weplog", "wiki"}) {
		t.Errorf("RepNames = %v, want [dllog kokko weplog wiki]", names)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo`); got != 4 {
		t.Errorf("repo 行 = %d, want 4", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo WHERE git_dir = '.git' AND rep_name = 'weplog'`); got != 1 {
		t.Errorf("ルート直下の .git の rep 名が zip 名になっていない")
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo WHERE git_dir = 'kokko/wiki/.git' AND rep_name = 'wiki'`); got != 1 {
		t.Errorf("入れ子の .git が wiki として記録されていない")
	}
}

// packfile にまとめたリポジトリ（実データの半数）も読めること。
func TestPackedRepositoryIsReadable(t *testing.T) {
	c, pluginDir := newTestCache(t)
	work := filepath.Join(t.TempDir(), "etrobo2022")
	hashes := initTestRepo(t, work, twoCommits())
	repackTestRepo(t, work)
	if packs, _ := filepath.Glob(filepath.Join(work, ".git", "objects", "pack", "*.pack")); len(packs) == 0 {
		t.Fatal("repack しても packfile ができていない（テストの前提が崩れている）")
	}
	zipPath := filepath.Join(t.TempDir(), "etrobo2022.zip")
	zipDir(t, zipPath, work, "etrobo2022")

	mustBuild(t, c, pluginDir, zipPath)

	byHash := commitsByHash(t, c, pluginDir)
	if len(byHash) != 2 {
		t.Fatalf("コミット数 = %d, want 2", len(byHash))
	}
	if byHash[hashes[1]].Addition != 3 || byHash[hashes[1]].Deletion != 1 {
		t.Errorf("pack からの行数 = +%d -%d, want +3 -1", byHash[hashes[1]].Addition, byHash[hashes[1]].Deletion)
	}
}

// 2回目の構築は指紋が同じなので読み直さない（repo.scanned_unix が動かない）。
func TestBuildIsIncrementalByFingerprint(t *testing.T) {
	c, pluginDir := newTestCache(t)
	zipPath, _ := makeRepoZip(t, t.TempDir(), "racoonboard", twoCommits())

	mustBuild(t, c, pluginDir, zipPath)
	if _, err := c.conn().Exec(`UPDATE repo SET scanned_unix = 0`); err != nil {
		t.Fatal(err)
	}
	mustBuild(t, c, pluginDir, zipPath)
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo WHERE scanned_unix = 0`); got != 1 {
		t.Errorf("指紋が同じなのに読み直した")
	}
	if got := c.getMeta("build_total_repos"); got != "0" {
		t.Errorf("build_total_repos = %q, want 0", got)
	}
}

// .git の上限を超えるリポジトリは読まずに理由を残し、他のリポジトリは読むこと。
func TestOversizedGitDirIsSkippedWithNote(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	makeRepoZip(t, dir, "big", twoCommits())
	makeRepoZip(t, dir, "small", twoCommits()[:1])

	config := testConfig(dir)
	config.MaxGitDirBytes = 1 // 何でも超える
	if err := c.build(t.Context(), pluginDir, config); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 0 {
		t.Errorf("上限を超えたのに読んでいる: %d", got)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM repo WHERE note LIKE '%too large%'`); got != 2 {
		t.Errorf("理由が残っていない")
	}
	stats := c.Stats(pluginDir)
	if len(stats.Repos) != 2 || stats.Repos[0].Note == "" {
		t.Errorf("設定画面に理由が出ない: %+v", stats.Repos)
	}

	// 上限を戻すと指紋は同じでも note 付きの repo は読み直される？ —— 指紋が同じなら読み直さない仕様なので、
	// 上限を変えたときは schema_version と同じく利用者がキャッシュを消す。ここでは仕様どおり動くことだけ見る。
	mustBuild(t, c, pluginDir, dir)
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 0 {
		t.Errorf("指紋が同じなのに読み直した: %d", got)
	}
}

// schema_version を上げると次の起動で作り直すこと。
func TestSchemaVersionBumpRebuilds(t *testing.T) {
	c, pluginDir := newTestCache(t)
	zipPath, _ := makeRepoZip(t, t.TempDir(), "racoonboard", twoCommits())
	mustBuild(t, c, pluginDir, zipPath)
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 2 {
		t.Fatalf("コミット数 = %d, want 2", got)
	}

	if _, err := c.conn().Exec(`UPDATE cache_meta SET value = 'stale' WHERE key = 'schema_version'`); err != nil {
		t.Fatal(err)
	}
	if err := initSchema(c.conn()); err != nil {
		t.Fatalf("initSchema: %v", err)
	}
	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 0 {
		t.Errorf("世代が違うのに残っている: %d", got)
	}
	if got := c.getMeta("schema_version"); got != cacheSchemaVersion {
		t.Errorf("schema_version = %q, want %q", got, cacheSchemaVersion)
	}
}

// 構築中でも読み取りがブロックしないこと（構築と読み取りが別ロック）。
func TestConcurrentReadDuringBuild(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c", "d"} {
		// 中身が同じだとハッシュも同じになり1件に畳まれるので、リポジトリごとに内容を変える
		commits := twoCommits()
		commits[0].Files["README.md"] = str("hello " + name + "\n")
		makeRepoZip(t, dir, name, commits)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mustBuild(t, c, pluginDir, dir)
	}()
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if _, err := c.QueryCommits(pluginDir, nil, nil, 10, false); err != nil {
					t.Errorf("QueryCommits during build: %v", err)
					return
				}
				if _, err := c.RepNames(pluginDir); err != nil {
					t.Errorf("RepNames during build: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	if got := countRows(t, c, `SELECT COUNT(*) FROM commit_log`); got != 8 {
		t.Errorf("コミット数 = %d, want 8", got)
	}
}

// 何にもマッチしない指定は問題として設定画面に出ること。
func TestMissingPatternIsReported(t *testing.T) {
	c, pluginDir := newTestCache(t)
	mustBuild(t, c, pluginDir, filepath.Join(t.TempDir(), "no_such_dir_*"))
	problems := c.loadSourceProblems()
	if len(problems) != 1 || problems[0].Kind != "missing_pattern" {
		t.Errorf("problems = %+v, want missing_pattern が1件", problems)
	}
}

// 複数の zip に入っているコミットの rep 名は「最新のコミットを持つリポジトリ」のものになること。
// 改名したプロジェクト（古い名前の zip と、続きを含む新しい名前の zip）の共通の履歴は新しい名前に付き、
// 新しい zip を外せば古い名前に戻る。zip のパス順（取り込み順）には依らない。
func TestSharedCommitsTakeTheNameOfTheNewestRepository(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()

	// 古い名前 "weplog"（パス順では zzz_ で後ろ、取り込み順では最後になるようにする）
	work := filepath.Join(t.TempDir(), "weplog")
	hashes := initTestRepo(t, work, twoCommits())
	oldZip := filepath.Join(dir, "zzz_old", "weplog.zip")
	zipDir(t, oldZip, work, "weplog")

	// 続きを積んで新しい名前 "urlog" で固める（パス順では前）
	extra := initTestRepoAppend(t, work, testCommit{Message: "renamed", When: testTimeJST.Add(24 * time.Hour), Files: map[string]*string{"x.txt": str("x")}})
	newZip := filepath.Join(dir, "aaa_new", "urlog.zip")
	zipDir(t, newZip, work, "urlog")

	mustBuild(t, c, pluginDir, oldZip, newZip)

	byHash := commitsByHash(t, c, pluginDir)
	if len(byHash) != 3 {
		t.Fatalf("コミット数 = %d, want 3", len(byHash))
	}
	for _, hash := range append(hashes, extra) {
		if byHash[hash].RepName != "urlog" {
			t.Errorf("%s の rep 名 = %q, want urlog（最新のコミットを持つ側）", hash[:8], byHash[hash].RepName)
		}
	}
	names, _ := c.RepNames(pluginDir)
	if !slices.Equal(names, []string{"urlog"}) {
		t.Errorf("RepNames = %v, want [urlog]（古い名前は現れない）", names)
	}

	// 新しい zip を外すと古い名前に戻る
	mustBuild(t, c, pluginDir, oldZip)
	byHash = commitsByHash(t, c, pluginDir)
	if len(byHash) != 2 {
		t.Fatalf("新しい zip を外した後のコミット数 = %d, want 2", len(byHash))
	}
	for _, hash := range hashes {
		if byHash[hash].RepName != "weplog" {
			t.Errorf("%s の rep 名 = %q, want weplog（残った zip の名前に戻る）", hash[:8], byHash[hash].RepName)
		}
	}
}
