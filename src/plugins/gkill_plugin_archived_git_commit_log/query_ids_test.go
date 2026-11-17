package main

import "testing"

// gkill の FindQuery.IDs（コミットのハッシュ）と Reps（リポジトリ名）で SQL を絞る（ADR-0318）。
// nil は指定なし、[] は0件。
func TestQueryCommitsByIDsAndReps(t *testing.T) {
	c, pluginDir := newTestCache(t)
	zipPath, hashes := makeRepoZip(t, t.TempDir(), "repo-beta", twoCommits())
	mustBuild(t, c, pluginDir, zipPath)

	count := func(ids, reps []string) int {
		t.Helper()
		rows, err := c.QueryCommits(pluginDir, nil, nil, 0, false, ids, reps)
		if err != nil {
			t.Fatalf("QueryCommits: %v", err)
		}
		return len(rows)
	}
	if got := count(nil, nil); got != 2 {
		t.Errorf("指定なし = %d件, want 2件", got)
	}
	if got := count([]string{hashes[0]}, nil); got != 1 {
		t.Errorf("ハッシュ1つ = %d件, want 1件", got)
	}
	if got := count([]string{}, nil); got != 0 {
		t.Errorf("IDs が [] = %d件, want 0件", got)
	}
	if got := count(nil, []string{"repo-beta"}); got != 2 {
		t.Errorf("rep 名 repo-beta = %d件, want 2件", got)
	}
	if got := count(nil, []string{"repo-alpha"}); got != 0 {
		t.Errorf("rep 名 repo-alpha = %d件, want 0件", got)
	}
	if got := count(nil, []string{}); got != 0 {
		t.Errorf("Reps が [] = %d件, want 0件", got)
	}
}
