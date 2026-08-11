package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

const testDataDir = "testdata/Physical Activity_GoogleData"

func testLocation(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("Asia/Tokyo を読めない: %v", err)
	}
	return loc
}

// openTestSources は testdata のCSVを詰めたZIPを開く。
// 返した SourceSet はテストの終わりに閉じる。
func openTestSources(t *testing.T, names ...string) *sdk.SourceSet {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	writeTestZipFromTestData(t, dir, testZipName, testExportTime, names...)
	sources, err := openSources([]string{dir})
	if err != nil {
		t.Fatalf("openSources: %v", err)
	}
	t.Cleanup(func() { _ = sources.Close() })
	return sources
}

// testEntry は testdata の1ファイルを取り込み対象のエントリとして返す。
func testEntry(t *testing.T, fileName string) sdk.SourceEntry {
	t.Helper()
	// 走査は指標に一致する名前しか拾わないので、対象外を試すときは自分で作る
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	writeTestZip(t, dir, testZipName, map[string][]byte{fileName: readTestData(t, fileName)}, testExportTime)
	sources, err := sdk.OpenSources([]string{dir}, nil)
	if err != nil {
		t.Fatalf("OpenSources: %v", err)
	}
	t.Cleanup(func() { _ = sources.Close() })
	entries := sources.Entries()
	if len(entries) != 1 {
		t.Fatalf("エントリ数 = %d, want 1", len(entries))
	}
	return entries[0]
}

// ingestForTest はテストデータの1ファイルを取り込んで
// (指標キー, 現地日付) → 部分集計 の形で返す。
func ingestForTest(t *testing.T, fileName string, prefix string) map[string]partialDaily {
	t.Helper()
	partials, err := ingestEntry(testEntry(t, fileName), metricsByPrefix[prefix], testLocation(t))
	if err != nil {
		t.Fatalf("ingestEntry(%s): %v", fileName, err)
	}
	byKey := map[string]partialDaily{}
	for _, partial := range partials {
		byKey[partial.MetricKey+"|"+partial.DateLocal] = partial
	}
	return byKey
}

// TestIngestFile_StepsSumsPerLocalDay は、歩数がJSTの日ごとに合計されることを確認する。
func TestIngestFile_StepsSumsPerLocalDay(t *testing.T) {
	partials := ingestForTest(t, "steps_2024-04-01.csv", "steps")

	// 02:05Z / 02:28Z / 10:00Z はすべて JST の 2024-04-03
	partial, exist := partials["steps_daily|2024-04-03"]
	if !exist {
		t.Fatalf("2024-04-03 の歩数が無い: %v", partials)
	}
	if partial.SumValue != 100 {
		t.Errorf("歩数の合計 = %v, want 100", partial.SumValue)
	}
	if partial.CountValue != 3 {
		t.Errorf("件数 = %d, want 3", partial.CountValue)
	}
	if len(partial.Devices) != 2 {
		t.Errorf("デバイス = %v, want 2種", partial.Devices)
	}
}

// TestIngestFile_HeartRateCrossesLocalDayBoundary は、
// UTCの15:00でJSTの日が変わることを確認する。
// 1ファイルが2つの現地日に寄与するので、部分集計もその形で出る。
func TestIngestFile_HeartRateCrossesLocalDayBoundary(t *testing.T) {
	partials := ingestForTest(t, "heart_rate_2024-04-03.csv", "heart_rate")

	// 14:59Z → JST 04-03、15:00Z と 16:00Z → JST 04-04
	day3, exist := partials["heart_rate_avg|2024-04-03"]
	if !exist {
		t.Fatalf("04-03 の心拍が無い: %v", partials)
	}
	if day3.CountValue != 1 || day3.SumValue != 60 {
		t.Errorf("04-03 = 件数%d 合計%v, want 1件 60", day3.CountValue, day3.SumValue)
	}

	day4, exist := partials["heart_rate_avg|2024-04-04"]
	if !exist {
		t.Fatalf("04-04 の心拍が無い: %v", partials)
	}
	if day4.CountValue != 2 || day4.SumValue != 180 {
		t.Errorf("04-04 = 件数%d 合計%v, want 2件 180", day4.CountValue, day4.SumValue)
	}
	if day4.MinValue != 80 || day4.MaxValue != 100 {
		t.Errorf("04-04 の最小/最大 = %v/%v, want 80/100", day4.MinValue, day4.MaxValue)
	}
}

// TestIngestFile_DailyFileKeepsLiteralDate は、既に日次のファイルが
// タイムゾーン変換されないことを確認する。
//
// daily_readiness.csv の 2024-11-13 は現地の暦日そのもの。
// UTCとして解釈してJSTに直すと 2024-11-13 09:00 になり、
// 日付は偶然合うが、Z付きで書かれた daily_resting_heart_rate.csv の
// 2024-04-04T00:00:00Z は 2024-04-04 09:00 JST → 2024-04-04 のまま……ではなく
// 変換すると1日ずれる場合があるので、literal を使うのが正しい。
func TestIngestFile_DailyFileKeepsLiteralDate(t *testing.T) {
	readiness := ingestForTest(t, "daily_readiness.csv", "daily_readiness")
	if _, exist := readiness["readiness_score|2024-11-13"]; !exist {
		t.Errorf("2024-11-13 がそのまま使われていない: %v", readiness)
	}
	if _, exist := readiness["readiness_score|2024-11-14"]; !exist {
		t.Errorf("2024-11-14 がそのまま使われていない: %v", readiness)
	}

	// Z付きでも日付部分をそのまま使う
	resting := ingestForTest(t, "daily_resting_heart_rate.csv", "daily_resting_heart_rate")
	partial, exist := resting["resting_heart_rate|2024-04-04"]
	if !exist {
		t.Fatalf("2024-04-04 の安静時心拍が無い: %v", resting)
	}
	if partial.LastValue != 68.982 {
		t.Errorf("安静時心拍 = %v, want 68.982", partial.LastValue)
	}
}

// TestIngestFile_SkipsNaN は NaN の行が捨てられることを確認する。
// daily_sleep_temperature_derivations.csv には実在する。
func TestIngestFile_SkipsNaN(t *testing.T) {
	partials := ingestForTest(t, "daily_sleep_temperature_derivations.csv", "daily_sleep_temperature_derivations")

	if _, exist := partials["sleep_skin_temperature|2024-04-05"]; exist {
		t.Error("値が NaN の日が取り込まれている")
	}
	partial, exist := partials["sleep_skin_temperature|2024-04-04"]
	if !exist {
		t.Fatalf("2024-04-04 が無い: %v", partials)
	}
	if partial.LastValue != 32.16 {
		t.Errorf("= %v, want 32.16", partial.LastValue)
	}
}

// TestIngestFile_MatchColFilters は、絞り込み列が効くことを確認する。
func TestIngestFile_MatchColFilters(t *testing.T) {
	partials := ingestForTest(t, "active_zone_minutes_2024-04-01.csv", "active_zone_minutes")

	total, exist := partials["azm_total|2024-04-03"]
	if !exist || total.SumValue != 6 {
		t.Errorf("合計 = %+v, want 6", total)
	}
	fatBurn, exist := partials["azm_fat_burn|2024-04-03"]
	if !exist || fatBurn.SumValue != 3 {
		t.Errorf("脂肪燃焼 = %+v, want 3", fatBurn)
	}
	cardio, exist := partials["azm_cardio|2024-04-03"]
	if !exist || cardio.SumValue != 3 {
		t.Errorf("有酸素 = %+v, want 3", cardio)
	}
	if _, exist := partials["azm_peak|2024-04-03"]; exist {
		t.Error("ピークの行は無いのに集計されている")
	}
}

// TestIngestFile_CountAggregation は、件数を数える指標を確認する。
func TestIngestFile_CountAggregation(t *testing.T) {
	partials := ingestForTest(t, "activity_level_2024-04-03.csv", "activity_level")

	sedentary, exist := partials["activity_level_sedentary|2024-04-03"]
	if !exist || sedentary.CountValue != 2 {
		t.Errorf("座位 = %+v, want 2件", sedentary)
	}
	lightly, exist := partials["activity_level_lightly|2024-04-03"]
	if !exist || lightly.CountValue != 1 {
		t.Errorf("低活動 = %+v, want 1件", lightly)
	}
}

// TestIngestFile_UnknownHeaderIsIgnored は、名前は一致するが
// 中身が違うファイルが対象外になることを確認する。
func TestIngestFile_UnknownHeaderIsIgnored(t *testing.T) {
	partials, err := ingestEntry(testEntry(t, "not_a_metric.csv"), metricsByPrefix["steps"], testLocation(t))
	if err != nil {
		t.Fatalf("ingestEntry: %v", err)
	}
	if len(partials) != 0 {
		t.Errorf("対象外のファイルから %d 件取り込まれた", len(partials))
	}
}

// TestOpenSources_PicksOnlyMetricFiles は、走査が対象エントリだけを拾うことを確認する。
func TestOpenSources_PicksOnlyMetricFiles(t *testing.T) {
	names := []string{}
	dirEntries, err := os.ReadDir(testDataDir)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			names = append(names, dirEntry.Name())
		}
	}
	sources := openTestSources(t, names...)

	prefixes := map[string]int{}
	for _, entry := range sources.Entries() {
		prefix, ok := metricPrefixOf(entry.Name)
		if !ok {
			t.Errorf("指標に一致しないエントリを拾っている: %s", entry.Name)
			continue
		}
		prefixes[prefix]++
		if entry.Name == "steps_readme.txt" || entry.Name == "not_a_metric.csv" || entry.Name == "heart_rate_variability_2024-04-01.csv" {
			t.Errorf("対象外のファイルを拾っている: %s", entry.Name)
		}
	}
	if prefixes["heart_rate"] != 2 {
		t.Errorf("heart_rate = %d件, want 2", prefixes["heart_rate"])
	}
	if prefixes["steps"] != 1 {
		t.Errorf("steps = %d件, want 1", prefixes["steps"])
	}
	if prefixes["weight"] != 1 || prefixes["height"] != 1 {
		t.Errorf("weight/height = %d/%d件, want 1/1", prefixes["weight"], prefixes["height"])
	}
}

// TestParseSourcePatterns は、配列でも改行区切りでも読めることを確認する。
func TestParseSourcePatterns(t *testing.T) {
	if got := parseSourcePatterns([]any{"/a", "/b"}); len(got) != 2 {
		t.Errorf("配列 = %v, want 2件", got)
	}
	if got := parseSourcePatterns("/a\n/b\n"); len(got) != 2 {
		t.Errorf("改行区切り = %v, want 2件", got)
	}
	// 空なら既定にフォールバックする
	if got := parseSourcePatterns(nil); len(got) == 0 {
		t.Error("空のとき既定にフォールバックしていない")
	}
}
