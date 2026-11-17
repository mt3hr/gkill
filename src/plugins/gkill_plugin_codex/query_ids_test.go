package main

import "testing"

// gkill の FindQuery.IDs で SQL を絞る（ADR-0318）。nil は指定なし、[] は0件。
func TestQueryKyousByIDs(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageSources(t, "parent", "sub", "old", "index")
	mustBuild(t, c, pluginDir, sourceDir)

	all, err := c.QueryKyous(pluginDir, nil, nil, 0, false, nil)
	if err != nil || len(all) < 2 {
		t.Fatalf("指定なし = %d件, %v", len(all), err)
	}
	rows, err := c.QueryKyous(pluginDir, nil, nil, 0, false, []string{all[1].ID})
	if err != nil || len(rows) != 1 || rows[0].ID != all[1].ID {
		t.Errorf("ID 1つ = %+v, %v, want その1件", rows, err)
	}
	rows, err = c.QueryKyous(pluginDir, nil, nil, 0, false, []string{})
	if err != nil || len(rows) != 0 {
		t.Errorf("IDs が [] = %d件, %v, want 0件", len(rows), err)
	}
}
