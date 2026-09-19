package main

import (
	"reflect"
	"testing"
)

func namedSources(names ...string) []foldedSource {
	sources := make([]foldedSource, 0, len(names))
	for _, name := range names {
		sources = append(sources, foldedSource{metricKey: "steps_daily", dateLocal: "2025-12-15", dataSource: name, sum: 1})
	}
	return sources
}

func sourceNames(sources []foldedSource) []string {
	names := make([]string, 0, len(sources))
	for _, source := range sources {
		names = append(names, source.dataSource)
	}
	return names
}

// TestChooseDataSources は「時計が1つでもあれば時計だけ、無ければ補助を書いた順に1つ」の規則を確認する。
func TestChooseDataSources(t *testing.T) {
	secondary := []string{"Phone Health Connect", "Google Health App"}
	cases := []struct {
		name      string
		input     []string
		secondary []string
		want      []string
	}{
		{"補助が空なら全部", []string{"Pixel Watch 2", "Phone Health Connect"}, nil, []string{"Pixel Watch 2", "Phone Health Connect"}},
		{"時計があれば時計だけ", []string{"Phone Health Connect", "Pixel Watch 2"}, secondary, []string{"Pixel Watch 2"}},
		{"時計が複数なら全部足す", []string{"Pixel Watch 2", "Phone Health Connect", "Pixel Watch 3"}, secondary, []string{"Pixel Watch 2", "Pixel Watch 3"}},
		{"時計が無ければ補助の先頭", []string{"Google Health App", "Phone Health Connect"}, secondary, []string{"Phone Health Connect"}},
		{"補助の先頭が無ければ次", []string{"Google Health App"}, secondary, []string{"Google Health App"}},
		{"大小と前後の空白は無視", []string{"  phone health connect ", "Pixel Watch 2"}, []string{"PHONE HEALTH CONNECT"}, []string{"Pixel Watch 2"}},
		{"データソース列が無い（空）は時計扱い", []string{"", "Phone Health Connect"}, secondary, []string{""}},
		{"1つしか無ければそのまま", []string{"Phone Health Connect"}, secondary, []string{"Phone Health Connect"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sourceNames(chooseDataSources(namedSources(tc.input...), tc.secondary))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("chooseDataSources(%v, %v) = %v, want %v", tc.input, tc.secondary, got, tc.want)
			}
		})
	}
}

// TestMergeFoldedSources は採用したデータソースの束ね方を確認する。
func TestMergeFoldedSources(t *testing.T) {
	if _, ok := mergeFoldedSources(nil); ok {
		t.Error("0件で ok=true")
	}
	merged, ok := mergeFoldedSources([]foldedSource{
		{dataSource: "Pixel Watch 3", sum: 20, count: 2, minValue: 5, maxValue: 15, maxMtime: 10,
			lastValue: 15, lastUnix: 200, hourSums: "a", hourCounts: "b", sourcePaths: "p1"},
		{dataSource: "Pixel Watch 2", sum: 10, count: 1, minValue: 10, maxValue: 10, maxMtime: 20,
			lastValue: 10, lastUnix: 100, hourSums: "c", hourCounts: "d", sourcePaths: "p2"},
	})
	if !ok {
		t.Fatal("ok=false")
	}
	if merged.sum != 30 || merged.count != 3 || merged.minValue != 5 || merged.maxValue != 15 || merged.maxMtime != 20 {
		t.Errorf("sum/count/min/max/mtime = %v/%v/%v/%v/%v", merged.sum, merged.count, merged.minValue, merged.maxValue, merged.maxMtime)
	}
	// その日の最後のサンプルは lastUnix が最大のもの
	if merged.lastValue != 15 || merged.lastUnix != 200 {
		t.Errorf("last = %v@%d, want 15@200", merged.lastValue, merged.lastUnix)
	}
	if merged.dataSource != "Pixel Watch 2\nPixel Watch 3" {
		t.Errorf("devices = %q（名前順に並ぶこと）", merged.dataSource)
	}
	if merged.hourSums != "a\nc" || merged.hourCounts != "b\nd" || merged.sourcePaths != "p1\np2" {
		t.Errorf("連結 = %q / %q / %q", merged.hourSums, merged.hourCounts, merged.sourcePaths)
	}
}
