package main

import "testing"

// TestMetricPrefixOf_HeartRateVariabilityIsNotHeartRate は最重要の回帰テスト。
//
// 接頭辞を strings.HasPrefix で判定すると
// heart_rate_variability_2024-04-01.csv が "heart_rate" に食われ、
// 心拍数の日平均に心拍変動の値（数十ms）が混ざる。
// 旧バッチはこれを「HRVを含むパスなら除外」という場当たりの正規表現で避けていた。
func TestMetricPrefixOf_HeartRateVariabilityIsNotHeartRate(t *testing.T) {
	if _, ok := metricPrefixOf("heart_rate_variability_2024-04-01.csv"); ok {
		t.Error("heart_rate_variability が対象になっている。心拍数の集計に混ざる")
	}
	prefix, ok := metricPrefixOf("heart_rate_2024-04-03.csv")
	if !ok || prefix != "heart_rate" {
		t.Errorf("heart_rate_2024-04-03.csv → %q, %v, want heart_rate, true", prefix, ok)
	}
}

func TestMetricPrefixOf(t *testing.T) {
	cases := []struct {
		name       string
		wantPrefix string
		wantOK     bool
	}{
		{"steps_2024-04-01.csv", "steps", true},
		{"weight.csv", "weight", true},
		{"height.csv", "height", true},
		{"daily_readiness.csv", "daily_readiness", true},
		{"daily_resting_heart_rate.csv", "daily_resting_heart_rate", true},
		{"active_zone_minutes_2024-04-01.csv", "active_zone_minutes", true},
		// readme は拾わない
		{"steps_readme.txt", "", false},
		// レジストリに無い
		{"not_a_metric.csv", "", false},
		{"gps_location_2024-04-18.csv", "", false},
		// 日付が日付の形をしていない
		{"steps_20240401.csv", "", false},
	}
	for _, c := range cases {
		prefix, ok := metricPrefixOf(c.name)
		if ok != c.wantOK || prefix != c.wantPrefix {
			t.Errorf("metricPrefixOf(%q) = %q, %v; want %q, %v", c.name, prefix, ok, c.wantPrefix, c.wantOK)
		}
	}
}

// TestMetricRegistry_KeysAndTitlesAreUnique は、キーと表示名が重複しないことを確認する。
//
// キーが重複するとKyou IDが衝突して片方が消える。
// 表示名が重複すると推移グラフのグループが混ざる。
func TestMetricRegistry_KeysAndTitlesAreUnique(t *testing.T) {
	keys := map[string]struct{}{}
	titles := map[string]struct{}{}
	for _, def := range metricRegistry {
		if def.Key == "" || def.Title == "" {
			t.Errorf("キーか表示名が空: %+v", def)
		}
		if _, duplicated := keys[def.Key]; duplicated {
			t.Errorf("キーが重複している: %q", def.Key)
		}
		keys[def.Key] = struct{}{}
		if _, duplicated := titles[def.Title]; duplicated {
			t.Errorf("表示名が重複している: %q", def.Title)
		}
		titles[def.Title] = struct{}{}
	}
}

// TestMetricRegistry_HasValueColumnUnlessCount は、
// 件数以外の指標に値列が指定してあることを確認する。
func TestMetricRegistry_HasValueColumnUnlessCount(t *testing.T) {
	for _, def := range metricRegistry {
		if def.Agg == aggCount {
			if def.MatchCol == "" {
				t.Errorf("%q: 件数の指標なのに絞り込み列が無い", def.Key)
			}
			continue
		}
		if def.ValueCol == "" {
			t.Errorf("%q: 値列が指定されていない", def.Key)
		}
		if def.TimeCol == "" {
			t.Errorf("%q: 時刻列が指定されていない", def.Key)
		}
	}
}

func TestNormHeader(t *testing.T) {
	cases := map[string]string{
		"Beats per minute":    "beatsperminute",
		" Heart Rate ":        "heartrate",
		"time_stamp":          "timestamp",
		"weight grams":        "weightgrams",
		"heart rate zone":     "heartratezone",
		"data source":         "datasource",
		"Kilocalories":        "kilocalories",
		"temperature celsius": "temperaturecelsius",
	}
	for input, want := range cases {
		if got := normHeader(input); got != want {
			t.Errorf("normHeader(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFindCol(t *testing.T) {
	headers := []string{"timestamp", "Beats per minute", "data source"}
	if got := findCol(headers, "beatsperminute"); got != 1 {
		t.Errorf("findCol(beatsperminute) = %d, want 1", got)
	}
	if got := findCol(headers, "datasource"); got != 2 {
		t.Errorf("findCol(datasource) = %d, want 2", got)
	}
	if got := findCol(headers, "unknown"); got != -1 {
		t.Errorf("findCol(unknown) = %d, want -1", got)
	}
	if got := findCol(headers, ""); got != -1 {
		t.Errorf("findCol(空) = %d, want -1", got)
	}
}
