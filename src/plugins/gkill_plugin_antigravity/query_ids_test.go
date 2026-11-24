package main

import "testing"

// gkill の FindQuery.IDs で SQL を絞る（ADR-0318）。nil は指定なし、[] は0件。
// gkill 本体も ID で絞り直すので結果は変わらず、読む量だけが減る。
func TestQueryKyousByIDs(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA, conversationB)
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

// 期間は両端を含む。
func TestQueryKyousByCalendar(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA, conversationB)
	mustBuild(t, c, pluginDir, sourceDir)

	all, err := c.QueryKyous(pluginDir, nil, nil, 0, false, nil)
	if err != nil {
		t.Fatalf("QueryKyous: %v", err)
	}
	newest := unixToTime(all[0].RelatedUnix)
	rows, err := c.QueryKyous(pluginDir, &newest, &newest, 0, false, nil)
	if err != nil || len(rows) != 1 || rows[0].ID != all[0].ID {
		t.Errorf("両端を含む期間 = %d件, %v", len(rows), err)
	}
}

// 本体の詳細表示は body_json から組み直す。会話タイトルは別カラムから載せる。
func TestQueryBodyCarriesTitleAndProject(t *testing.T) {
	c, pluginDir := newTestCache(t)
	sourceDir := stageConversations(t, conversationA)
	mustBuild(t, c, pluginDir, sourceDir)

	rows, err := c.QueryKyous(pluginDir, nil, nil, 0, false, nil)
	if err != nil || len(rows) == 0 {
		t.Fatalf("QueryKyous: %d件 %v", len(rows), err)
	}
	built, err := c.QueryBody(pluginDir, rows[0].ID)
	if err != nil {
		t.Fatalf("QueryBody: %v", err)
	}
	if built.ID != rows[0].ID {
		t.Errorf("ID = %q, want %q", built.ID, rows[0].ID)
	}
	if built.Title != rows[0].Title {
		t.Errorf("タイトルが載っていない: %q, want %q", built.Title, rows[0].Title)
	}
}
