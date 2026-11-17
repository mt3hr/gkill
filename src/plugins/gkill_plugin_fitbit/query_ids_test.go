package main

import "testing"

// gkill の FindQuery.IDs で SQL を絞る（ADR-0318）。nil は指定なし、[] は0件。
func TestCache_QueryFiltersByIDs(t *testing.T) {
	sourceDir := copyTestData(t, "heart_rate_2024-04-03.csv", "heart_rate_2024-04-04.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)
	if err := c.build(pluginDir, config); err != nil {
		t.Fatalf("build: %v", err)
	}

	all, err := c.QueryDailyMetrics(pluginDir, config, nil, nil, 0, nil)
	if err != nil || len(all) < 2 {
		t.Fatalf("指定なし = %d件, %v", len(all), err)
	}
	metrics, err := c.QueryDailyMetrics(pluginDir, config, nil, nil, 0, []string{all[1].KyouID})
	if err != nil || len(metrics) != 1 || metrics[0].KyouID != all[1].KyouID {
		t.Errorf("ID 1つ = %+v, %v, want その1件", metrics, err)
	}
	metrics, err = c.QueryDailyMetrics(pluginDir, config, nil, nil, 0, []string{})
	if err != nil || len(metrics) != 0 {
		t.Errorf("IDs が [] = %d件, %v, want 0件", len(metrics), err)
	}
}
