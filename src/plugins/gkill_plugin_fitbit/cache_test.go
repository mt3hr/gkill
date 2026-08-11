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

// テスト用のZIPはここで組み立てる。testdata には生のCSVを置いたままにして、
// gitで差分が読めるようにするため（バイナリはコミットしない）。
const (
	// testEntryDir はZIP内でCSVを置く場所。実データと同じ形にしてある。
	testEntryDir = "Takeout/Fit/Physical Activity_GoogleData/"

	// testZipName は既定のZIP名。Takeout の命名に合わせてあるので、
	// 世代の識別子に書き出し時刻が混ざる経路も通る。
	testZipName = "takeout-20240501T000000Z-1-001.zip"
)

// readTestData は testdata のCSVを読む。
func readTestData(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(testDataDir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return body
}

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

// writeTestZipFromTestData は testdata のCSVを詰めたZIPを作る。
func writeTestZipFromTestData(t *testing.T, dir string, zipName string, modified time.Time, names ...string) string {
	t.Helper()
	files := map[string][]byte{}
	for _, name := range names {
		files[name] = readTestData(t, name)
	}
	return writeTestZip(t, dir, zipName, files, modified)
}

// newTestCache は独立したキャッシュを作る。
// globalCache は共有なのでテストでは使わない。
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

// testExportTime は既定の書き出し時刻。
var testExportTime = time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

// copyTestData はテストデータを詰めたZIPを作り、それを置いたフォルダを返す。
func copyTestData(t *testing.T, names ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	writeTestZipFromTestData(t, dir, testZipName, testExportTime, names...)
	return dir
}

func testConfig(t *testing.T, sourceDir string) pluginConfig {
	t.Helper()
	patterns := []string{sourceDir}
	return pluginConfig{
		Patterns:    patterns,
		Source:      sdk.ExpandSourcePatterns(patterns),
		Timezone:    "Asia/Tokyo",
		ScanWorkers: 1,
	}
}

// buildAndRead はキャッシュを作って (指標キー|日付) → 値 の形で返す。
func buildAndRead(t *testing.T, c *cache, pluginDir string, config pluginConfig) map[string]dailyMetric {
	t.Helper()
	if err := c.build(pluginDir, config); err != nil {
		t.Fatalf("build: %v", err)
	}
	metrics, err := c.QueryDailyMetrics(pluginDir, config, nil, nil, 0)
	if err != nil {
		t.Fatalf("QueryDailyMetrics: %v", err)
	}
	byKey := map[string]dailyMetric{}
	for _, metric := range metrics {
		byKey[metric.MetricKey+"|"+metric.DateLocal] = metric
	}
	return byKey
}

// TestCache_FoldsAcrossFilesForOneLocalDay は差分キャッシュの中核を確認する。
//
// 心拍は1日1ファイルなので、JSTの1日はUTCの2ファイルにまたがる。
// 部分集計をファイル単位で持ち、日次の値は畳み直して出す仕組みが効いているか。
func TestCache_FoldsAcrossFilesForOneLocalDay(t *testing.T) {
	sourceDir := copyTestData(t, "heart_rate_2024-04-03.csv", "heart_rate_2024-04-04.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	metrics := buildAndRead(t, c, pluginDir, config)

	// JST 04-04 は 15:00Z(80) + 16:00Z(100) + 翌ファイルの 01:00Z(120) の3件
	avg, exist := metrics["heart_rate_avg|2024-04-04"]
	if !exist {
		t.Fatalf("04-04 の心拍平均が無い: %v", metrics)
	}
	if avg.SampleCount != 3 {
		t.Errorf("件数 = %d, want 3（2ファイルの寄与が畳めていない）", avg.SampleCount)
	}
	if avg.NumValue != "100.0" {
		t.Errorf("平均 = %s, want 100.0", avg.NumValue)
	}
	if avg.MaxValue != 120 {
		t.Errorf("最大 = %v, want 120", avg.MaxValue)
	}
}

// TestCache_IncrementalUpdate は、変化したファイルだけを読み直すことと、
// 読み直した結果が正しく畳み直されることを確認する。
func TestCache_IncrementalUpdate(t *testing.T) {
	sourceDir := copyTestData(t, "heart_rate_2024-04-03.csv", "heart_rate_2024-04-04.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	buildAndRead(t, c, pluginDir, config)

	// 2回目は何も変わっていないので取り込みが走らない
	before := c.meta("build_total_files")
	if err := c.build(pluginDir, config); err != nil {
		t.Fatalf("build(2回目): %v", err)
	}
	if c.meta("build_total_files") != "0" {
		t.Errorf("変更が無いのに %s ファイルを取り込もうとした（前回 %s）", c.meta("build_total_files"), before)
	}

	// 片方のファイルだけを書き換えたZIPに差し替える。
	//
	// エントリの更新時刻は据え置く。Takeout は全エントリに書き出し時刻を入れるので
	// 中身が変わっても更新時刻は動かない ―― 差分判定が CRC32 を見ていなければ
	// ここで変更を取りこぼす。
	writeTestZip(t, sourceDir, testZipName, map[string][]byte{
		"heart_rate_2024-04-03.csv": readTestData(t, "heart_rate_2024-04-03.csv"),
		"heart_rate_2024-04-04.csv": []byte(
			"timestamp,beats per minute,data source\n" +
				"2024-04-04T01:00:00Z,60.0,Fitbit App\n" +
				"2024-04-04T02:00:00Z,60.0,Fitbit App\n"),
	}, testExportTime)

	metrics := buildAndRead(t, c, pluginDir, config)
	if c.meta("build_total_files") != "1" {
		t.Errorf("変更したのは1ファイルなのに %s ファイル取り込んだ", c.meta("build_total_files"))
	}
	// JST 04-04 は 80 + 100 + 60 + 60 の4件で平均75
	avg := metrics["heart_rate_avg|2024-04-04"]
	if avg.SampleCount != 4 {
		t.Errorf("件数 = %d, want 4", avg.SampleCount)
	}
	if avg.NumValue != "75.0" {
		t.Errorf("平均 = %s, want 75.0", avg.NumValue)
	}
}

// TestCache_RemovedFileDropsItsContribution は、ファイルが消えたら
// その寄与だけが落ち、他のファイルの寄与が残ることを確認する。
func TestCache_RemovedFileDropsItsContribution(t *testing.T) {
	sourceDir := copyTestData(t, "heart_rate_2024-04-03.csv", "heart_rate_2024-04-04.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	buildAndRead(t, c, pluginDir, config)

	// 片方のエントリを落としたZIPに差し替える
	writeTestZipFromTestData(t, sourceDir, testZipName, testExportTime, "heart_rate_2024-04-03.csv")
	metrics := buildAndRead(t, c, pluginDir, config)

	// 04-04 は残るが、消したファイルの1件ぶん減る
	avg, exist := metrics["heart_rate_avg|2024-04-04"]
	if !exist {
		t.Fatal("消したのは片方だけなのに 04-04 が丸ごと消えた")
	}
	if avg.SampleCount != 2 {
		t.Errorf("件数 = %d, want 2", avg.SampleCount)
	}
	// 04-03 は別ファイル由来なので無傷
	if _, exist := metrics["heart_rate_avg|2024-04-03"]; !exist {
		t.Error("04-03 まで消えている")
	}
}

// TestCache_AllFilesRemovedDropsTheDay は、寄与ファイルが全部消えたら
// 集計結果ごと消えることを確認する。
func TestCache_AllFilesRemovedDropsTheDay(t *testing.T) {
	sourceDir := copyTestData(t, "steps_2024-04-01.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	metrics := buildAndRead(t, c, pluginDir, config)
	if _, exist := metrics["steps_daily|2024-04-03"]; !exist {
		t.Fatalf("歩数が取り込まれていない: %v", metrics)
	}

	if err := os.Remove(filepath.Join(sourceDir, testZipName)); err != nil {
		t.Fatalf("remove: %v", err)
	}
	metrics = buildAndRead(t, c, pluginDir, config)
	if _, exist := metrics["steps_daily|2024-04-03"]; exist {
		t.Error("寄与ファイルが全部消えたのに集計結果が残っている")
	}
}

// TestCache_SplitArchivesAreSummed は、1つの書き出しが複数のZIPに分かれていても
// 日次の値が合算されることを確認する。
//
// Google Takeout は大きい書き出しを -1-001.zip / -1-002.zip … に分ける。
// 心拍のように現地1日がUTC2ファイルにまたがる指標は、
// その2ファイルが別のパートに入りうるので、合算できないと値が半分になる。
func TestCache_SplitArchivesAreSummed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	// 同じ書き出し時刻＝同じ世代。パートごとに別のZIPへ入れる
	writeTestZipFromTestData(t, dir, "takeout-20240501T000000Z-1-001.zip", testExportTime, "heart_rate_2024-04-03.csv")
	writeTestZipFromTestData(t, dir, "takeout-20240501T000000Z-1-002.zip", testExportTime, "heart_rate_2024-04-04.csv")

	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	metrics := buildAndRead(t, c, pluginDir, testConfig(t, dir))

	// 分割していないときと同じ3件（15:00Z, 16:00Z, 翌ファイルの01:00Z）
	avg, exist := metrics["heart_rate_avg|2024-04-04"]
	if !exist {
		t.Fatalf("04-04 の心拍平均が無い: %v", metrics)
	}
	if avg.SampleCount != 3 {
		t.Errorf("件数 = %d, want 3（分割されたパートの寄与が合算されていない）", avg.SampleCount)
	}

	// 世代は1つにまとまっているはず
	exports := c.loadExports()
	if len(exports) != 1 {
		t.Fatalf("世代数 = %d, want 1: %+v", len(exports), exports)
	}
	if exports[0].ArchiveCount != 2 {
		t.Errorf("ZIP数 = %d, want 2", exports[0].ArchiveCount)
	}
}

// TestCache_NewerExportWinsInsteadOfSumming は、書き出しをまたぐ同じ日が
// 合算されず、新しい書き出しの値だけが使われることを確認する。
//
// ここが壊れると、古い書き出しを消さずに新しいのを置いたときに
// 歩数やカロリーが2倍になる。
func TestCache_NewerExportWinsInsteadOfSumming(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "GoogleTakeout_Test_20240501")
	newDir := filepath.Join(root, "GoogleTakeout_Test_20240601")
	oldTime := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	writeTestZipFromTestData(t, oldDir, "takeout-20240501T000000Z-1-001.zip", oldTime, "steps_2024-04-01.csv")
	writeTestZipFromTestData(t, newDir, "takeout-20240601T000000Z-1-001.zip", newTime, "steps_2024-04-01.csv")

	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, filepath.Join(root, "GoogleTakeout_Test_*"))

	// 片方だけを取り込んだときの値を基準にする
	single := newTestCache(t, t.TempDir())
	singlePluginDir := t.TempDir()
	expected := buildAndRead(t, single, singlePluginDir, testConfig(t, oldDir))["steps_daily|2024-04-03"]
	if expected.KyouID == "" {
		t.Fatal("基準にする歩数が取り込めていない")
	}

	metrics := buildAndRead(t, c, pluginDir, config)
	got, exist := metrics["steps_daily|2024-04-03"]
	if !exist {
		t.Fatalf("歩数が無い: %v", metrics)
	}
	if got.NumValue != expected.NumValue {
		t.Errorf("歩数 = %s, want %s（書き出しをまたいで合算している）", got.NumValue, expected.NumValue)
	}
	if got.SampleCount != expected.SampleCount {
		t.Errorf("件数 = %d, want %d（書き出しをまたいで合算している）", got.SampleCount, expected.SampleCount)
	}

	// 採用したのは新しいほうであること
	exports := c.loadExports()
	if len(exports) != 2 {
		t.Fatalf("世代数 = %d, want 2: %+v", len(exports), exports)
	}
	if exports[0].Dir != newDir {
		t.Errorf("採用した世代 = %s, want %s", exports[0].Dir, newDir)
	}

	// 新しいほうを消したら古いほうが復活する
	if err := os.RemoveAll(newDir); err != nil {
		t.Fatalf("remove: %v", err)
	}
	metrics = buildAndRead(t, c, pluginDir, config)
	revived, exist := metrics["steps_daily|2024-04-03"]
	if !exist {
		t.Fatal("新しい書き出しを消したら歩数ごと消えた（古いほうが復活していない）")
	}
	if revived.NumValue != expected.NumValue {
		t.Errorf("復活後の歩数 = %s, want %s", revived.NumValue, expected.NumValue)
	}
}

// TestCache_SameFolderExportsAreNotSummed は、1つのフォルダに時期の違う
// 書き出しを置いても合算されないことを確認する。
//
// 世代の単位をフォルダだけにすると、この置き方で値が2倍になる。
func TestCache_SameFolderExportsAreNotSummed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test")
	writeTestZipFromTestData(t, dir, "takeout-20240501T000000Z-1-001.zip",
		time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), "steps_2024-04-01.csv")
	writeTestZipFromTestData(t, dir, "takeout-20240601T000000Z-1-001.zip",
		time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), "steps_2024-04-01.csv")

	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	metrics := buildAndRead(t, c, pluginDir, testConfig(t, dir))

	single := newTestCache(t, t.TempDir())
	singleDir := filepath.Join(t.TempDir(), "GoogleTakeout_Only")
	writeTestZipFromTestData(t, singleDir, "takeout-20240501T000000Z-1-001.zip",
		time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), "steps_2024-04-01.csv")
	expected := buildAndRead(t, single, t.TempDir(), testConfig(t, singleDir))["steps_daily|2024-04-03"]

	got := metrics["steps_daily|2024-04-03"]
	if got.NumValue != expected.NumValue {
		t.Errorf("歩数 = %s, want %s（同じフォルダの別の書き出しを合算している）", got.NumValue, expected.NumValue)
	}
	if len(c.loadExports()) != 2 {
		t.Errorf("世代数 = %d, want 2（書き出し時刻で分かれていない）", len(c.loadExports()))
	}
}

// TestCache_KyouIDIsStable は、作り直してもKyou IDが変わらないことを確認する。
// IDが揺れると、ユーザが付けたタグやテキストの紐付けが切れる。
func TestCache_KyouIDIsStable(t *testing.T) {
	sourceDir := copyTestData(t, "steps_2024-04-01.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	first := buildAndRead(t, c, pluginDir, config)
	// 世代を変えて全部作り直させる
	c.setMeta("registry_version", "force-rebuild")
	second := buildAndRead(t, c, pluginDir, config)

	for key, metric := range first {
		if second[key].KyouID != metric.KyouID {
			t.Errorf("%s のIDが %q → %q に変わった", key, metric.KyouID, second[key].KyouID)
		}
	}
}

// TestCache_UnitConversion は単位換算を確認する。
func TestCache_UnitConversion(t *testing.T) {
	sourceDir := copyTestData(t, "weight.csv", "height.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	metrics := buildAndRead(t, c, pluginDir, config)

	// 90000g → 90.0kg
	weight, exist := metrics["weight|2024-04-03"]
	if !exist {
		t.Fatalf("体重が無い: %v", metrics)
	}
	if weight.NumValue != "90.0" {
		t.Errorf("体重 = %s, want 90.0", weight.NumValue)
	}
	// 1650mm → 165.0cm
	height, exist := metrics["height|2024-04-02"]
	if !exist {
		t.Fatalf("身長が無い: %v", metrics)
	}
	if height.NumValue != "165.0" {
		t.Errorf("身長 = %s, want 165.0", height.NumValue)
	}
}

// TestCache_TimezoneChangeRebuilds は、タイムゾーンを変えると
// 日付が振り直されることを確認する。
func TestCache_TimezoneChangeRebuilds(t *testing.T) {
	sourceDir := copyTestData(t, "heart_rate_2024-04-03.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)

	jst := testConfig(t, sourceDir)
	jstMetrics := buildAndRead(t, c, pluginDir, jst)
	if _, exist := jstMetrics["heart_rate_avg|2024-04-04"]; !exist {
		t.Fatalf("JSTで04-04が無い: %v", jstMetrics)
	}

	utc := testConfig(t, sourceDir)
	utc.Timezone = "UTC"
	utcMetrics := buildAndRead(t, c, pluginDir, utc)
	// UTCなら3件とも04-03に入る
	if _, exist := utcMetrics["heart_rate_avg|2024-04-04"]; exist {
		t.Error("UTCに変えたのに04-04が残っている（作り直されていない）")
	}
	day3, exist := utcMetrics["heart_rate_avg|2024-04-03"]
	if !exist || day3.SampleCount != 3 {
		t.Errorf("UTCの04-03 = %+v, want 3件", day3)
	}
}

// TestCache_QueryFiltersByPeriod は、期間の絞り込みがSQL側で効くことを確認する。
func TestCache_QueryFiltersByPeriod(t *testing.T) {
	sourceDir := copyTestData(t, "heart_rate_2024-04-03.csv", "heart_rate_2024-04-04.csv")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(t, sourceDir)

	if err := c.build(pluginDir, config); err != nil {
		t.Fatalf("build: %v", err)
	}

	loc := testLocation(t)
	start := noonUnixOf("2024-04-04", loc) - 3600
	end := noonUnixOf("2024-04-04", loc) + 3600
	metrics, err := c.QueryDailyMetrics(pluginDir, config, &start, &end, 0)
	if err != nil {
		t.Fatalf("QueryDailyMetrics: %v", err)
	}
	for _, metric := range metrics {
		if metric.DateLocal != "2024-04-04" {
			t.Errorf("期間外の日が返った: %s", metric.DateLocal)
		}
	}
	if len(metrics) == 0 {
		t.Error("期間内の記録が0件")
	}
}
