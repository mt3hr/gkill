package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// テスト用のZIPはここで組み立てる。testdata には生のJSON/CSVを置いたままにして、
// gitで差分が読めるようにするため（バイナリはコミットしない）。
const (
	// testEntryDir はZIP内でデータを置く場所。実データと同じ形にしてある。
	testEntryDir = "Takeout/タイムライン/"

	// testZipName は既定のZIP名。
	testZipName = "takeout-20240501T000000Z-1-001.zip"
)

// testExportTime は既定の書き出し時刻。
var testExportTime = time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

// writeTestZip は取り込み元のZIPを作る（既にあれば作り直す）。
//
// modified は全エントリに同じ値を入れる。Takeout の実物がそうなっているので、
// 「更新時刻では中身の変化を判定できない」という前提もそのまま再現される。
func writeTestZip(t *testing.T, dir string, zipName string, files map[string][]byte, modified time.Time) string {
	t.Helper()
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for name, body := range files {
		header := &zip.FileHeader{Name: testEntryDir + name, Method: zip.Deflate}
		header.Modified = modified
		// 日本語のエントリ名を含むので UTF-8 フラグを立てる（実データもそうなっている）
		header.Flags |= 0x800
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entry.Write(body); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	zipPath := filepath.Join(dir, zipName)
	if err := os.WriteFile(zipPath, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("write %s: %v", zipPath, err)
	}
	return zipPath
}

// writeTestZipFromTestData は testdata のファイルを詰めたZIPを作る。
func writeTestZipFromTestData(t *testing.T, dir string, zipName string, modified time.Time, names ...string) string {
	t.Helper()
	files := map[string][]byte{}
	for _, name := range names {
		body, err := os.ReadFile(testPath(name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		files[name] = body
	}
	return writeTestZip(t, dir, zipName, files, modified)
}

// testEntry は testdata の1ファイルを取り込み対象のエントリとして返す。
func testEntry(t *testing.T, name string) sdk.SourceEntry {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	writeTestZipFromTestData(t, dir, testZipName, testExportTime, name)
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

// newTestCache は独立したキャッシュを作る。globalCache は共有なので使わない。
func newTestCache(t *testing.T, pluginDir string) *cache {
	t.Helper()
	c := &cache{}
	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	t.Cleanup(func() {
		if c.db != nil {
			_ = c.db.Close()
		}
	})
	return c
}

// copyTestData はテストデータを詰めたZIPを作り、それを置いたフォルダを返す。
func copyTestData(t *testing.T, names ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	writeTestZipFromTestData(t, dir, testZipName, testExportTime, names...)
	return dir
}

// readGPSLogs は同期で走査してから点を読む。
// GetGPSLogs はバックグラウンドで走査するので、テストでは先に refresh を回す。
func readGPSLogs(t *testing.T, c *cache, pluginDir string, config pluginConfig, q sdk.GPSLogQuery) sdk.GPSLogPage {
	t.Helper()
	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	page, err := c.GetGPSLogs(pluginDir, config, q)
	if err != nil {
		t.Fatalf("GetGPSLogs: %v", err)
	}
	return page
}

func testConfig(sourceDir string) pluginConfig {
	patterns := []string{sourceDir}
	return pluginConfig{
		Patterns:          patterns,
		Source:            sdk.ExpandSourcePatterns(patterns),
		AccuracyMaxMeters: defaultAccuracyMaxMeters,
		IncludeFitbitGPS:  true,
		VisitPoints:       false,
		MaxPoints:         defaultMaxPoints,
	}
}

// TestCache_DedupesAcrossDuplicatedRows は重複除去を確認する。
//
// ワークアウトのトラックは全行が Fitbit App と Pixel Watch 2 の2重に書き出される。
// 実データでは 12,748行 が 6,483点 になる。
func TestCache_DedupesAcrossDuplicatedRows(t *testing.T) {
	sourceDir := copyTestData(t, "gps_location_2024-04-18.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)

	page := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(page.GPSLogs) != 2 {
		t.Errorf("点数 = %d, want 2（4行が2点に重複除去されていない）", len(page.GPSLogs))
	}
}

// TestCache_AccuracyFilter は精度フィルタを確認する。
// 精度が分からない点は残す（測れないものをフィルタで消さない）。
func TestCache_AccuracyFilter(t *testing.T) {
	sourceDir := copyTestData(t, "timeline_edits_small.json", "gps_location_2024-04-18.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)

	// 既定 100m: 100,000mm(=100m) は残り、2,599,999mm(=2.6km) は落ちる
	config := testConfig(sourceDir)
	page := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	// WIFI(100m) 1点 + Fitbit(精度不明) 2点 = 3点。CELL(2.6km) は落ちる
	if len(page.GPSLogs) != 3 {
		t.Errorf("既定の精度フィルタ = %d点, want 3", len(page.GPSLogs))
	}

	// フィルタ無効なら CELL も残る
	config.AccuracyMaxMeters = 0
	page = readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(page.GPSLogs) != 4 {
		t.Errorf("フィルタ無効 = %d点, want 4", len(page.GPSLogs))
	}

	// 50m にすると WIFI(100m) も落ちる
	config.AccuracyMaxMeters = 50
	page = readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(page.GPSLogs) != 2 {
		t.Errorf("50m = %d点, want 2（精度不明のFitbitだけ残る）", len(page.GPSLogs))
	}
}

// TestCache_VisitPointsAreOptOut は、滞在地の点が既定で出ないことを確認する。
func TestCache_VisitPointsAreOptOut(t *testing.T) {
	sourceDir := copyTestData(t, "timeline_edits_small.json")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)

	config := testConfig(sourceDir)
	page := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	withoutVisits := len(page.GPSLogs)

	config.VisitPoints = true
	page = readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(page.GPSLogs) <= withoutVisits {
		t.Errorf("visit_points を有効にしても点が増えない: %d → %d", withoutVisits, len(page.GPSLogs))
	}
}

// TestCache_IncludeFitbitGPS は、ワークアウトのトラックを外せることを確認する。
func TestCache_IncludeFitbitGPS(t *testing.T) {
	sourceDir := copyTestData(t, "gps_location_2024-04-18.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)

	config := testConfig(sourceDir)
	config.IncludeFitbitGPS = false
	page := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(page.GPSLogs) != 0 {
		t.Errorf("include_fitbit_gps=false なのに %d点 返った", len(page.GPSLogs))
	}
}

// TestCache_SortedAscendingAndPaged は、昇順で返ることと
// ページングが行を飛ばしたり重ねたりしないことを確認する。
func TestCache_SortedAscendingAndPaged(t *testing.T) {
	sourceDir := copyTestData(t, "timeline_edits_small.json", "gps_location_2024-04-18.csv", "records.json")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)

	all := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if all.HasMore {
		t.Error("全件取ったのに続きがあることになっている")
	}
	for i := 1; i < len(all.GPSLogs); i++ {
		if all.GPSLogs[i].RelatedTime.Before(all.GPSLogs[i-1].RelatedTime) {
			t.Fatalf("%d番目で昇順が崩れている", i)
		}
	}

	// 1件ずつページングして全部つながること
	paged := []sdk.GPSLog{}
	offset := 0
	for {
		page, err := c.GetGPSLogs(pluginDir, config, sdk.GPSLogQuery{Offset: offset, Limit: 1})
		if err != nil {
			t.Fatalf("GetGPSLogs(paged): %v", err)
		}
		paged = append(paged, page.GPSLogs...)
		if !page.HasMore || len(page.GPSLogs) == 0 {
			break
		}
		offset += len(page.GPSLogs)
	}
	if len(paged) != len(all.GPSLogs) {
		t.Fatalf("ページングの合計 = %d点, 全件 = %d点", len(paged), len(all.GPSLogs))
	}
	for i := range paged {
		if !paged[i].RelatedTime.Equal(all.GPSLogs[i].RelatedTime) ||
			paged[i].Latitude != all.GPSLogs[i].Latitude {
			t.Fatalf("%d番目がページングでずれた", i)
		}
	}
}

// TestCache_PeriodFilterIncludesBothEnds は、期間の両端が含まれることを確認する。
func TestCache_PeriodFilterIncludesBothEnds(t *testing.T) {
	sourceDir := copyTestData(t, "gps_location_2024-04-18.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)

	all := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(all.GPSLogs) < 2 {
		t.Fatalf("点が足りない: %d", len(all.GPSLogs))
	}

	// 同じ時刻を2つ渡すと、ちょうどその時刻の点だけが返る
	target := all.GPSLogs[0].RelatedTime
	page := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{StartTime: &target, EndTime: &target})
	if len(page.GPSLogs) != 1 || !page.GPSLogs[0].RelatedTime.Equal(target) {
		t.Errorf("同一時刻 = %d点, want 1点", len(page.GPSLogs))
	}
}

// TestCache_IncrementalRefresh は、変化が無ければ読み直さないことと、
// ファイルが消えたらその点が落ちることを確認する。
func TestCache_IncrementalRefresh(t *testing.T) {
	sourceDir := copyTestData(t, "timeline_edits_small.json", "gps_location_2024-04-18.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)

	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	before := c.Stats(pluginDir, config).TotalPoints
	if before == 0 {
		t.Fatal("1点も取り込まれていない")
	}

	// 2回目は何も変わらない
	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh(2回目): %v", err)
	}
	if after := c.Stats(pluginDir, config).TotalPoints; after != before {
		t.Errorf("変化が無いのに点数が %d → %d に変わった", before, after)
	}

	// 片方のエントリを落としたZIPに差し替える
	writeTestZipFromTestData(t, sourceDir, testZipName, testExportTime, "timeline_edits_small.json")
	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh(削除後): %v", err)
	}
	stats := c.Stats(pluginDir, config)
	if stats.TotalPoints >= before {
		t.Errorf("ファイルを消したのに点が減っていない: %d → %d", before, stats.TotalPoints)
	}
	if stats.TotalPoints == 0 {
		t.Error("消していないファイルの点まで消えている")
	}
}

// TestCache_ReportsUnsupportedFormat は、未対応形式が統計に出ることを確認する。
// 黙って無視すると「入れたのに出てこない」理由が分からなくなる。
func TestCache_ReportsUnsupportedFormat(t *testing.T) {
	sourceDir := copyTestData(t, "semantic_2024_JANUARY.json")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)

	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	stats := c.Stats(pluginDir, config)
	if len(stats.UnsupportedFiles) != 1 {
		t.Errorf("未対応ファイル = %v, want 1件", stats.UnsupportedFiles)
	}
}

// TestCache_MaxPointsCaps は、上限が効くことを確認する。
func TestCache_MaxPointsCaps(t *testing.T) {
	sourceDir := copyTestData(t, "timeline_edits_small.json", "gps_location_2024-04-18.csv", "records.json")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)
	config.MaxPoints = 2

	page := readGPSLogs(t, c, pluginDir, config, sdk.GPSLogQuery{})
	if len(page.GPSLogs) != 2 {
		t.Errorf("上限2 のとき %d点 返った", len(page.GPSLogs))
	}
}

// TestCache_EmptySourceIsNotAnError は、取り込み元が空でもエラーにならないことを確認する。
func TestCache_EmptySourceIsNotAnError(t *testing.T) {
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t.TempDir())

	page, err := c.GetGPSLogs(pluginDir, config, sdk.GPSLogQuery{})
	if err != nil {
		t.Fatalf("空の取り込み元でエラーになった: %v", err)
	}
	if len(page.GPSLogs) != 0 {
		t.Errorf("= %d点, want 0", len(page.GPSLogs))
	}
	_ = time.Now()
}
