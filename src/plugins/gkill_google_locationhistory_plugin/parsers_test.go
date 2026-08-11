package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testDataDir = "testdata"

func testPath(name string) string { return filepath.Join(testDataDir, name) }

// TestDetectFormat は、形式の判定が中身で行われることを確認する。
// ファイル名ではなく中身で決めるので、任意のフォルダを指定しても動く。
func TestDetectFormat(t *testing.T) {
	cases := map[string]string{
		"timeline_edits_small.json":     formatTimelineEdits,
		"android_location_history.json": formatAndroidTimeline,
		"records.json":                  formatRecordsJSON,
		"gps_location_2024-04-18.csv":   formatFitbitGPSCSV,
		"semantic_2024_JANUARY.json":    formatSemanticLocationHistory,
		"not_location.json":             "",
		"not_location.csv":              "",
	}
	for name, want := range cases {
		got := detectFormat(testEntry(t, name))
		if got != want {
			t.Errorf("detectFormat(%s) = %q, want %q", name, got, want)
		}
	}
}

// TestDetectFormat_SemanticIsRecognizedButUnsupported は、
// セマンティックロケーション履歴を「認識するが読めない」扱いにしていることを確認する。
//
// placeVisit.location だけを読むと1日2点程度しか出ず、
// 「読めているのに中身が薄い」状態になって気づけない。
func TestDetectFormat_SemanticIsRecognizedButUnsupported(t *testing.T) {
	if isSupportedFormat(formatSemanticLocationHistory) {
		t.Error("セマンティックロケーション履歴が対応済みになっている")
	}
	points, err := parseByFormat(formatSemanticLocationHistory, testEntry(t, "semantic_2024_JANUARY.json"))
	if err != nil {
		t.Fatalf("parseByFormat: %v", err)
	}
	if len(points) != 0 {
		t.Errorf("未対応なのに %d 点返した", len(points))
	}
}

// TestParseTimelineEdits は、座標を持つのが position だけであることを確認する。
func TestParseTimelineEdits(t *testing.T) {
	points, err := parseTimelineEdits(testEntry(t, "timeline_edits_small.json"))
	if err != nil {
		t.Fatalf("parseTimelineEdits: %v", err)
	}

	positions := 0
	visits := 0
	for _, point := range points {
		switch point.Source {
		case sourceVisit:
			visits++
		case sourceActivity:
		default:
			positions++
		}
	}
	if positions != 2 {
		t.Errorf("測位の点 = %d, want 2（wifiScan と activityRecord を拾っている）", positions)
	}
	if visits != 1 {
		t.Errorf("滞在地の点 = %d, want 1", visits)
	}

	// E7 が度に直せること
	first := points[0]
	if e7ToDegree(first.LatE7) < 35.3 || e7ToDegree(first.LatE7) > 35.4 {
		t.Errorf("緯度 = %v, want 35.35 前後", e7ToDegree(first.LatE7))
	}
	if first.AccuracyMm != 100000 {
		t.Errorf("精度 = %d, want 100000", first.AccuracyMm)
	}
	if first.Source != "WIFI" {
		t.Errorf("出所 = %q, want WIFI", first.Source)
	}
	if first.DeviceID != "-1467294991" {
		t.Errorf("端末 = %q", first.DeviceID)
	}
	want := time.Date(2026, 6, 20, 15, 23, 10, 829000000, time.UTC).UnixMilli()
	if first.UnixMilli != want {
		t.Errorf("時刻 = %d, want %d", first.UnixMilli, want)
	}
}

// TestParseAndroidTimeline は、"35.1234°, 139.1234°" の形と
// 分オフセットの時刻計算を確認する。
func TestParseAndroidTimeline(t *testing.T) {
	points, err := parseAndroidTimeline(testEntry(t, "android_location_history.json"))
	if err != nil {
		t.Fatalf("parseAndroidTimeline: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("点数 = %d, want 2", len(points))
	}
	if e7ToDegree(points[0].LatE7) != 35.1234 || e7ToDegree(points[0].LngE7) != 139.1234 {
		t.Errorf("1点目 = %v, %v, want 35.1234, 139.1234", e7ToDegree(points[0].LatE7), e7ToDegree(points[0].LngE7))
	}
	// 30分オフセット
	if points[1].UnixMilli-points[0].UnixMilli != 30*60*1000 {
		t.Errorf("時刻の差 = %dms, want 1800000", points[1].UnixMilli-points[0].UnixMilli)
	}
}

func TestParseDegreePoint(t *testing.T) {
	cases := []struct {
		value   string
		wantLat float64
		wantLng float64
		wantOK  bool
	}{
		{"35.1234000°, 139.1234000°", 35.1234, 139.1234, true},
		{"35.1234000°,139.1234000°", 35.1234, 139.1234, true},
		{"-35.1234000°, -139.1234000°", -35.1234, -139.1234, true},
		{"壊れている", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, c := range cases {
		latE7, lngE7, ok := parseDegreePoint(c.value)
		if ok != c.wantOK {
			t.Errorf("parseDegreePoint(%q) ok = %v, want %v", c.value, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if e7ToDegree(latE7) != c.wantLat || e7ToDegree(lngE7) != c.wantLng {
			t.Errorf("parseDegreePoint(%q) = %v, %v; want %v, %v",
				c.value, e7ToDegree(latE7), e7ToDegree(lngE7), c.wantLat, c.wantLng)
		}
	}
}

// TestParseRecordsJSON は、timestampMs と timestamp の両方を読めることと、
// accuracy がメートルからミリメートルに直ることを確認する。
func TestParseRecordsJSON(t *testing.T) {
	points, err := parseRecordsJSON(testEntry(t, "records.json"))
	if err != nil {
		t.Fatalf("parseRecordsJSON: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("点数 = %d, want 2", len(points))
	}
	if points[0].UnixMilli != 1700000000000 {
		t.Errorf("timestampMs = %d, want 1700000000000", points[0].UnixMilli)
	}
	// accuracy 25m → 25000mm
	if points[0].AccuracyMm != 25000 {
		t.Errorf("精度 = %d, want 25000（メートルからミリメートルに直っていない）", points[0].AccuracyMm)
	}
	want := time.Date(2023, 11, 15, 0, 0, 0, 0, time.UTC).UnixMilli()
	if points[1].UnixMilli != want {
		t.Errorf("timestamp = %d, want %d", points[1].UnixMilli, want)
	}
	if points[1].AccuracyMm != 3000000 {
		t.Errorf("精度 = %d, want 3000000", points[1].AccuracyMm)
	}
}

// TestParseFitbitGPSCSV は、2重に書き出された行が両方 rawPoint になることを確認する。
// 重複除去は読み出し時に行うので、ここでは落とさない。
func TestParseFitbitGPSCSV(t *testing.T) {
	points, err := parseFitbitGPSCSV(testEntry(t, "gps_location_2024-04-18.csv"))
	if err != nil {
		t.Fatalf("parseFitbitGPSCSV: %v", err)
	}
	if len(points) != 4 {
		t.Fatalf("点数 = %d, want 4（重複除去はここでは行わない）", len(points))
	}
	for _, point := range points {
		if point.Source != sourceFitbit {
			t.Errorf("出所 = %q, want %q", point.Source, sourceFitbit)
		}
		if point.AccuracyMm != accuracyUnknown {
			t.Errorf("精度 = %d, want %d（不明）", point.AccuracyMm, accuracyUnknown)
		}
	}
}

func TestDegreeToE7RoundTrip(t *testing.T) {
	for _, degree := range []float64{35.348935, -35.348935, 139.515455, 0} {
		if got := e7ToDegree(degreeToE7(degree)); got != degree {
			t.Errorf("往復 %v → %v", degree, got)
		}
	}
}

// TestOpenSources_DetectsByContent は、走査が中身で形式を判定することを確認する。
//
// 位置情報ではないファイルも Format="" として扱う。
// 呼び出し側がこれも覚えることで、次の走査で先頭を読み直さずに済む。
func TestOpenSources_DetectsByContent(t *testing.T) {
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

	dir := filepath.Join(t.TempDir(), "GoogleTakeout_Test_20240501")
	writeTestZipFromTestData(t, dir, testZipName, testExportTime, names...)
	sources, err := openSources([]string{dir})
	if err != nil {
		t.Fatalf("openSources: %v", err)
	}
	defer func() { _ = sources.Close() }()

	byFormat := map[string]int{}
	for _, entry := range sources.Entries() {
		formatID := detectFormat(entry)
		byFormat[formatID]++
		if (entry.Name == "not_location.json" || entry.Name == "not_location.csv") && formatID != "" {
			t.Errorf("関係ないファイルに形式が付いている: %s → %q", entry.Name, formatID)
		}
	}
	for _, formatID := range []string{formatTimelineEdits, formatAndroidTimeline, formatRecordsJSON, formatFitbitGPSCSV, formatSemanticLocationHistory} {
		if byFormat[formatID] != 1 {
			t.Errorf("%s = %d件, want 1", formatID, byFormat[formatID])
		}
	}
	// not_location.json だけ。not_location.csv は gps_location で始まらないので走査が拾わない
	if byFormat[""] != 1 {
		t.Errorf("位置情報ではないファイル = %d件, want 1", byFormat[""])
	}
}

// TestRefresh_ReusesKnownClassification は、変化していないエントリの
// 形式判定をやり直さないことを確認する。
//
// これが効かないと、走査のたびに全エントリの先頭64KBを伸長することになり、
// Takeout 全体では毎回1分以上かかる（実測）。
func TestRefresh_ReusesKnownClassification(t *testing.T) {
	sourceDir := copyTestData(t, "gps_location_2024-04-18.csv", "not_location.json")
	pluginDir := t.TempDir()
	c := newTestCache(t, pluginDir)
	config := testConfig(sourceDir)

	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	// 覚えている形式をわざと嘘にしておく。
	// 読み直していれば正しい形式に戻るし、再利用していれば嘘のまま残る。
	if _, err := c.db.Exec(`UPDATE file_cache SET format = 'sentinel'`); err != nil {
		t.Fatalf("poison file_cache: %v", err)
	}
	if err := c.refresh(pluginDir, config); err != nil {
		t.Fatalf("refresh(2回目): %v", err)
	}

	known, err := c.loadFileCache()
	if err != nil {
		t.Fatalf("loadFileCache: %v", err)
	}
	if len(known) == 0 {
		t.Fatal("file_cache が空")
	}
	for path, file := range known {
		if file.Format != "sentinel" {
			t.Errorf("%s の形式を判定し直している（%q）", path, file.Format)
		}
	}
}
