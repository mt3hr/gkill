package sdk

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// buildTestZip は指定した中身でZIPを1つ作る。
//
// テストごとにZIPを組み立てるので、バイナリをリポジトリに置かずに済む。
// modified を明示するのは、世代の新旧をテストが決められるようにするため
// （os.Chtimes に頼るとファイルシステムの解像度でぶれる）。
func buildTestZip(t *testing.T, zipPath string, modified time.Time, entries map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(zipPath), os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create %s: %v", zipPath, err)
	}
	defer func() { _ = file.Close() }()

	writer := zip.NewWriter(file)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: modified}
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create header %s: %v", name, err)
		}
		if _, err := entryWriter.Write([]byte(entries[name])); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
}

func mustOpenSources(t *testing.T, patterns []string, accept func(string) bool) *SourceSet {
	t.Helper()
	set, err := OpenSources(patterns, accept)
	if err != nil {
		t.Fatalf("OpenSources: %v", err)
	}
	t.Cleanup(func() { _ = set.Close() })
	return set
}

func problemKinds(set *SourceSet) []SourceProblemKind {
	kinds := []SourceProblemKind{}
	for _, problem := range set.Problems() {
		kinds = append(kinds, problem.Kind)
	}
	return kinds
}

// TestOpenSources_ReadsZipEntries は基本の読み取りを確認する。
func TestOpenSources_ReadsZipEntries(t *testing.T) {
	dir := t.TempDir()
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{
			"Takeout/Google Health/Physical Activity_GoogleData/steps_2024-04-01.csv": "timestamp,steps\n",
			"Takeout/タイムライン/Timeline Edits.json":                                      `{"timelineEdits":[]}`,
		})

	set := mustOpenSources(t, []string{dir}, nil)
	if len(set.Entries()) != 2 {
		t.Fatalf("エントリ = %d件, want 2: %+v", len(set.Entries()), set.Entries())
	}

	byName := map[string]SourceEntry{}
	for _, entry := range set.Entries() {
		byName[entry.Name] = entry
	}

	// 日本語のエントリ名が解決できること
	timeline, exist := byName["Timeline Edits.json"]
	if !exist {
		t.Fatalf("日本語パスのエントリが取れていない: %+v", byName)
	}
	if !strings.Contains(timeline.EntryName, "タイムライン") {
		t.Errorf("EntryName = %q, want タイムライン を含む", timeline.EntryName)
	}

	steps := byName["steps_2024-04-01.csv"]
	if steps.Size != int64(len("timestamp,steps\n")) {
		t.Errorf("Size = %d, want %d", steps.Size, len("timestamp,steps\n"))
	}
	if steps.CRC32 == 0 {
		t.Error("CRC32 が入っていない。差分判定に使えない")
	}
	if !strings.Contains(steps.Path, "!/") {
		t.Errorf("Path = %q, want ZIPのパスとエントリ名が繋がった形", steps.Path)
	}

	reader, err := steps.Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(body) != "timestamp,steps\n" {
		t.Errorf("中身 = %q", string(body))
	}
}

// TestOpenSources_IgnoresLooseFiles は、展開済みのファイルを読まないことを確認する。
func TestOpenSources_IgnoresLooseFiles(t *testing.T) {
	dir := t.TempDir()
	looseDir := filepath.Join(dir, "Takeout", "Google Health")
	if err := os.MkdirAll(looseDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(looseDir, "steps_2024-04-01.csv"), []byte("timestamp,steps\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	set := mustOpenSources(t, []string{dir}, nil)
	if len(set.Entries()) != 0 {
		t.Errorf("ZIPが無いのに %d 件読んだ", len(set.Entries()))
	}
	if !slices.Contains(problemKinds(set), ProblemExtractedFolder) {
		t.Errorf("展開済みフォルダとして報告されていない: %+v", set.Problems())
	}
}

// TestExportIDOf_SplitPartsShareOneExport は分割パートが同じ世代になることを確認する。
func TestExportIDOf_SplitPartsShareOneExport(t *testing.T) {
	dir := filepath.FromSlash("/tmp/GoogleTakeout_X1Yoga_20260811")
	first := ExportIDOf(filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"))
	second := ExportIDOf(filepath.Join(dir, "takeout-20260808T230152Z-1-002.zip"))
	if first != second {
		t.Errorf("分割パートが別の世代になっている: %q vs %q", first, second)
	}
}

// TestExportIDOf_DifferentExportsInOneDirAreDistinct は、
// 同じフォルダに違う時期の書き出しを置いても別の世代になることを確認する。
//
// ここが同じ世代になると、重なる日の歩数などが合算されて2倍になる。
func TestExportIDOf_DifferentExportsInOneDirAreDistinct(t *testing.T) {
	dir := filepath.FromSlash("/tmp/GoogleTakeout_X1Yoga_20260811")
	august := ExportIDOf(filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"))
	november := ExportIDOf(filepath.Join(dir, "takeout-20261108T104412Z-1-001.zip"))
	if august == november {
		t.Errorf("違う書き出しが同じ世代になっている: %q（歩数などが二重計上される）", august)
	}
}

func TestTakeoutStamp(t *testing.T) {
	cases := map[string]string{
		"takeout-20260808T230152Z-1-001.zip": "20260808T230152Z",
		"takeout-20260808T230152Z-001.zip":   "20260808T230152Z",
		"takeout-20260808t230152z.zip":       "20260808T230152Z",
		"not-a-takeout.zip":                  "",
		"takeout-short.zip":                  "",
		"takeout-2026080XT230152Z-1-001.zip": "",
		"":                                   "",
	}
	for input, want := range cases {
		if got := TakeoutStamp(input); got != want {
			t.Errorf("TakeoutStamp(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestOpenSources_ExportsAreSortedNewestFirst は世代の並び順を確認する。
func TestOpenSources_ExportsAreSortedNewestFirst(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "GoogleTakeout_X1Yoga_20260101")
	newDir := filepath.Join(root, "GoogleTakeout_X1Yoga_20260811")
	buildTestZip(t, filepath.Join(oldDir, "takeout-20260101T000000Z-1-001.zip"),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "old"})
	buildTestZip(t, filepath.Join(newDir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "new"})

	set := mustOpenSources(t, []string{filepath.Join(root, "GoogleTakeout_*")}, nil)
	exports := set.Exports()
	if len(exports) != 2 {
		t.Fatalf("世代 = %d件, want 2: %+v", len(exports), exports)
	}
	if !strings.Contains(exports[0].ExportID, "20260811") {
		t.Errorf("先頭が新しい世代になっていない: %q", exports[0].ExportID)
	}
	if exports[0].NewestMtimeUnix <= exports[1].NewestMtimeUnix {
		t.Errorf("更新時刻の順序が逆: %d <= %d", exports[0].NewestMtimeUnix, exports[1].NewestMtimeUnix)
	}
}

// TestOpenSources_SplitPartsAreOneExport は分割パートが1つの世代にまとまることを確認する。
func TestOpenSources_SplitPartsAreOneExport(t *testing.T) {
	dir := t.TempDir()
	modified := time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC)
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"), modified,
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "part1"})
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-002.zip"), modified,
		map[string]string{"Takeout/a/steps_2024-05-01.csv": "part2"})

	set := mustOpenSources(t, []string{dir}, nil)
	if len(set.Exports()) != 1 {
		t.Fatalf("世代 = %d件, want 1（分割パートは1つの世代）: %+v", len(set.Exports()), set.Exports())
	}
	if len(set.Exports()[0].ArchivePaths) != 2 {
		t.Errorf("ZIP = %d本, want 2", len(set.Exports()[0].ArchivePaths))
	}
	if len(set.Entries()) != 2 {
		t.Errorf("エントリ = %d件, want 2", len(set.Entries()))
	}
}

// TestOpenSources_MixedExportsInOneDirIsReported は、
// 1フォルダに時期の違う書き出しがあるときに知らせることを確認する。
func TestOpenSources_MixedExportsInOneDirIsReported(t *testing.T) {
	dir := t.TempDir()
	buildTestZip(t, filepath.Join(dir, "takeout-20260101T000000Z-1-001.zip"),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "old"})
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "new"})

	set := mustOpenSources(t, []string{dir}, nil)
	if len(set.Exports()) != 2 {
		t.Fatalf("世代 = %d件, want 2: %+v", len(set.Exports()), set.Exports())
	}
	if !slices.Contains(problemKinds(set), ProblemMixedExports) {
		t.Errorf("混在が報告されていない: %+v", set.Problems())
	}
}

// TestOpenSources_SpannedArchiveIsReportedNotFatal は、
// 分割アーカイブ(.z01)を検出しても他のZIPは読めることを確認する。
//
// Goの archive/zip は分割アーカイブを読めない。最後の .zip だけ開けてしまうことがあり、
// そのまま読むと壊れた値が返るので、開く前に見分けて拒否する。
func TestOpenSources_SpannedArchiveIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	// 分割アーカイブの体裁を作る（.z01 の中身は問わない）
	buildTestZip(t, filepath.Join(dir, "spanned.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "spanned"})
	if err := os.WriteFile(filepath.Join(dir, "spanned.z01"), []byte("volume"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// 正常なZIPも置く
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-05-01.csv": "ok"})

	set := mustOpenSources(t, []string{dir}, nil)
	if !slices.Contains(problemKinds(set), ProblemSpannedArchive) {
		t.Errorf("分割アーカイブが報告されていない: %+v", set.Problems())
	}
	// 正常なZIPのぶんは読めていること
	if len(set.Entries()) != 1 {
		t.Errorf("エントリ = %d件, want 1（正常なZIPのぶん）: %+v", len(set.Entries()), set.Entries())
	}
}

// TestOpenSources_NestedZipIsReported はZIPの中のZIPに潜らないことを確認する。
func TestOpenSources_NestedZipIsReported(t *testing.T) {
	dir := t.TempDir()
	buildTestZip(t, filepath.Join(dir, "outer.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{
			"Takeout/a/steps_2024-04-01.csv": "ok",
			"Takeout/a/inner.zip":            "PK\x03\x04dummy",
		})

	set := mustOpenSources(t, []string{dir}, nil)
	if len(set.Entries()) != 1 {
		t.Errorf("エントリ = %d件, want 1（内側のZIPは対象外）", len(set.Entries()))
	}
	if !slices.Contains(problemKinds(set), ProblemNestedZip) {
		t.Errorf("入れ子のZIPが報告されていない: %+v", set.Problems())
	}
}

// TestOpenSources_BrokenArchiveIsReportedNotFatal は、
// 壊れたZIPが1本あっても他は読めることを確認する。
func TestOpenSources_BrokenArchiveIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.zip"), []byte("not a zip at all"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{"Takeout/a/steps_2024-04-01.csv": "ok"})

	set := mustOpenSources(t, []string{dir}, nil)
	if !slices.Contains(problemKinds(set), ProblemBrokenArchive) {
		t.Errorf("壊れたZIPが報告されていない: %+v", set.Problems())
	}
	if len(set.Entries()) != 1 {
		t.Errorf("エントリ = %d件, want 1（正常なZIPのぶん）", len(set.Entries()))
	}
}

// TestOpenSources_AcceptFiltersEntries は絞り込みが効くことを確認する。
func TestOpenSources_AcceptFiltersEntries(t *testing.T) {
	dir := t.TempDir()
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{
			"Takeout/a/steps_2024-04-01.csv": "keep",
			"Takeout/a/steps_readme.txt":     "drop",
			"Takeout/a/other.json":           "drop",
		})

	set := mustOpenSources(t, []string{dir}, func(entryName string) bool {
		return strings.HasSuffix(entryName, ".csv")
	})
	if len(set.Entries()) != 1 || set.Entries()[0].Name != "steps_2024-04-01.csv" {
		t.Errorf("絞り込みが効いていない: %+v", set.Entries())
	}
}

// TestSourceEntry_ReadHead は先頭だけ読めることを確認する。
// 形式判定は先頭64KBしか読まないので、途中でやめてもエラーにならないこと。
func TestSourceEntry_ReadHead(t *testing.T) {
	dir := t.TempDir()
	body := strings.Repeat("abcdefgh", 20000) // 160,000バイト
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC),
		map[string]string{"Takeout/a/big.json": body})

	set := mustOpenSources(t, []string{dir}, nil)
	entry := set.Entries()[0]

	head, err := entry.ReadHead(64 * 1024)
	if err != nil {
		t.Fatalf("ReadHead: %v", err)
	}
	if len(head) != 64*1024 {
		t.Errorf("先頭 = %dバイト, want 65536", len(head))
	}
	if !strings.HasPrefix(string(head), "abcdefgh") {
		t.Errorf("中身が違う: %q", string(head[:16]))
	}

	// エントリより大きい要求はエントリの長さに収まる
	small, err := set.Entries()[0].ReadHead(1 << 30)
	if err != nil {
		t.Fatalf("ReadHead(大): %v", err)
	}
	if len(small) != len(body) {
		t.Errorf("= %dバイト, want %d", len(small), len(body))
	}
}

// TestSourceEntry_ConcurrentOpen は同じZIPの別エントリを並行に開けることを確認する。
// 取り込みを並列で回すのでここが壊れると読み違える。
func TestSourceEntry_ConcurrentOpen(t *testing.T) {
	dir := t.TempDir()
	entries := map[string]string{}
	for i := range 32 {
		entries[filepath.ToSlash(filepath.Join("Takeout/a", string(rune('a'+i))+".csv"))] =
			strings.Repeat(string(rune('a'+i)), 5000)
	}
	buildTestZip(t, filepath.Join(dir, "takeout-20260808T230152Z-1-001.zip"),
		time.Date(2026, 8, 8, 16, 1, 54, 0, time.UTC), entries)

	set := mustOpenSources(t, []string{dir}, nil)
	results := make([]string, len(set.Entries()))
	done := make(chan int, len(set.Entries()))
	for i, entry := range set.Entries() {
		go func(i int, entry SourceEntry) {
			reader, err := entry.Open()
			if err != nil {
				done <- i
				return
			}
			defer func() { _ = reader.Close() }()
			body, _ := io.ReadAll(reader)
			results[i] = string(body)
			done <- i
		}(i, entry)
	}
	for range set.Entries() {
		<-done
	}
	for i, entry := range set.Entries() {
		want := entries[entry.EntryName]
		if results[i] != want {
			t.Fatalf("%s の中身が並行読みで壊れた（先頭 %q）", entry.EntryName, truncate(results[i], 8))
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// TestReadAllLimited は上限を超えたらエラーになることを確認する。
func TestReadAllLimited(t *testing.T) {
	if _, err := ReadAllLimited(strings.NewReader("12345"), 10); err != nil {
		t.Errorf("上限内でエラー: %v", err)
	}
	if _, err := ReadAllLimited(strings.NewReader("12345678901"), 10); err == nil {
		t.Error("上限を超えてもエラーにならない")
	}
}

// TestSplitZipEntryPath は表示用の分解を確認する。
func TestSplitZipEntryPath(t *testing.T) {
	archive, entry, ok := SplitZipEntryPath(`C:\a\takeout.zip!/Takeout/b/steps.csv`)
	if !ok || archive != `C:\a\takeout.zip` || entry != "Takeout/b/steps.csv" {
		t.Errorf("= %q, %q, %v", archive, entry, ok)
	}
	if _, _, ok := SplitZipEntryPath(`C:\a\steps.csv`); ok {
		t.Error("ZIP内でないパスで ok になった")
	}
}

func TestExpandSourcePatterns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.zip"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	expanded := ExpandSourcePatterns([]string{dir, filepath.Join(dir, "*.zip"), filepath.Join(dir, "no-such-*")})
	if len(expanded.Dirs) != 1 {
		t.Errorf("Dirs = %v, want 1件", expanded.Dirs)
	}
	if len(expanded.Files) != 1 {
		t.Errorf("Files = %v, want 1件", expanded.Files)
	}
	if len(expanded.Missing) != 1 {
		t.Errorf("Missing = %v, want 1件", expanded.Missing)
	}
}

func TestParseSourcePatterns(t *testing.T) {
	if got := ParseSourcePatterns([]any{"/a", "/b"}, ""); len(got) != 2 {
		t.Errorf("配列 = %v, want 2件", got)
	}
	if got := ParseSourcePatterns("/a\n/b\n", ""); len(got) != 2 {
		t.Errorf("改行区切り = %v, want 2件", got)
	}
	if got := ParseSourcePatterns(nil, "/fallback"); len(got) != 1 || got[0] != "/fallback" {
		t.Errorf("既定値へのフォールバック = %v", got)
	}
	if got := ParseSourcePatterns(nil, ""); len(got) != 0 {
		t.Errorf("既定値が空 = %v, want 0件", got)
	}
}
