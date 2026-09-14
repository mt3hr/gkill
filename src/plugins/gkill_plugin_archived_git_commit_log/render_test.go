package main

import (
	"strings"
	"testing"
)

// 本文 HTML はメッセージ・パス・author をエスケープし、native と同じ +/- の並びで始まること。
func TestRenderCommitHTMLEscapes(t *testing.T) {
	body := commitBody{
		commitRow: commitRow{
			Hash: "2aa9a829e303ad135ca6c32575a8ab678c9eb268", RepName: "raco<on>board",
			CommitterUnix: testTimeJST.Unix(), TZOffsetSec: 9 * 3600,
			AuthorName: "Test <Author>", AuthorEmail: "author@example.com",
			Message: "fix <script>alert(1)</script>\n", Addition: 12, Deletion: 3,
		},
		Files:   []fileStat{{Path: "a<b>.go", Addition: 12, Deletion: 3}},
		Sources: []commitSource{{ArchivePath: "D:/repos/x<y>.zip", GitDir: "x/.git"}},
	}
	html := renderCommitHTML(body)

	for _, raw := range []string{"<script>alert(1)</script>", "raco<on>board", "Test <Author>", "a<b>.go", "x<y>.zip"} {
		if strings.Contains(html, raw) {
			t.Errorf("エスケープされていない: %q", raw)
		}
	}
	if !strings.Contains(html, `<span class="git_commit_addition"> + 12 </span><span class="git_commit_deletion"> - 3 </span>`) {
		t.Error("native の GitCommitLogView と同じ +/- の並びになっていない")
	}
	if !strings.Contains(html, `class="git_commit_log_message">fix &lt;script&gt;alert(1)&lt;/script&gt;`) {
		t.Error("メッセージが pre-line の要素に入っていない")
	}
	if !strings.Contains(html, "2021-03-08 01:58:55 +09:00") {
		t.Error("コミッタ日時がコミット固有のゾーンで出ていない")
	}
	for _, marker := range []string{"gkill_iframe_size", "gkill_theme", "data-theme", "ResizeObserver"} {
		if !strings.Contains(html, marker) {
			t.Errorf("iframe 用のスクリプトが無い: %q", marker)
		}
	}
}

func TestRenderCommitHTMLShowsStatsError(t *testing.T) {
	html := renderCommitHTML(commitBody{commitRow: commitRow{Hash: "x"}, StatsError: "object not found"})
	if !strings.Contains(html, "object not found") {
		t.Error("行数集計の失敗理由が出ていない")
	}
}

func TestRenderNotFoundHTML(t *testing.T) {
	html := renderNotFoundHTML()
	if !strings.Contains(html, "キャッシュにありません") || !strings.Contains(html, "gkill_iframe_size") {
		t.Errorf("not found の表示が崩れている: %s", html)
	}
}

// 設定画面は zip を開かず、キャッシュの状態と指定だけを描くこと。
func TestRenderConfigHTML(t *testing.T) {
	stats := cacheStats{
		CommitCount: 3, RepNameCount: 2, RepoCount: 2, TargetRepoCount: 2, BuildState: "idle",
		Repos: []repoRow{
			{ArchivePath: "D:/repos/a.zip", GitDir: "a/.git", RepName: "a", CommitCount: 2},
			{ArchivePath: "D:/repos/b.zip", GitDir: "b/.git", RepName: "b", CommitCount: 1, Note: "git directory is too large"},
		},
		SourceProblems: []sourceProblemRow{{Kind: "missing_pattern", Path: "D:/nowhere/*.zip", Message: "この指定は何にもマッチしませんでした。"}},
	}
	html := renderConfigHTML("C:/plugins/x", stats, []string{"D:/repos/*.zip", "D:/nowhere/*.zip"})
	for _, want := range []string{"3 コミット", "too large", "何にもマッチしませんでした", "gkill_source_dirs", "gkill_plugin_config", "D:/repos/*.zip", "max_git_dir_mb"} {
		if !strings.Contains(html, want) {
			t.Errorf("設定画面に %q が無い", want)
		}
	}
}
