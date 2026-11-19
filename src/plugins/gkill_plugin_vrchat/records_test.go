package main

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func mustRecords(t *testing.T, path string) []record {
	t.Helper()
	parsed, err := parseLogFile(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return buildRecords(parsed)
}

func recordsOfKind(records []record, kind string) []record {
	kept := []record{}
	for _, r := range records {
		if r.Kind == kind {
			kept = append(kept, r)
		}
	}
	return kept
}

// 正常終了の起動: ワールドの滞在・URLog・同じ部屋の人・出来事がそろい、終了の行で全部閉じる。
func TestBuildRecordsNormalSession(t *testing.T) {
	records := mustRecords(t, normalFixture())

	worlds := recordsOfKind(records, kindWorldTimeIs)
	if len(worlds) != 2 {
		t.Fatalf("ワールドの TimeIs = %d件, want 2件: %+v", len(worlds), worlds)
	}
	wantWorlds := []struct{ title, start, end string }{
		{"WorldAlpha", "2024.01.01 10:02:00", "2024.01.01 10:30:00"}, // OnLeftRoom で閉じる
		{"WorldBeta", "2024.01.01 10:32:00", "2024.01.01 11:00:00"},  // 終了の行で閉じる
	}
	for i, want := range wantWorlds {
		got := worlds[i]
		if got.Title != want.title || !got.Start.Equal(testTime(t, want.start)) || !got.End.Equal(testTime(t, want.end)) {
			t.Errorf("ワールド%d = %s %v〜%v, want %s %s〜%s", i, got.Title, got.Start, got.End, want.title, want.start, want.end)
		}
		if !slices.Equal(got.Tags, []string{tagWorld}) {
			t.Errorf("ワールド%d のタグ = %v", i, got.Tags)
		}
	}

	urlogs := recordsOfKind(records, kindWorldURLog)
	if len(urlogs) != 2 {
		t.Fatalf("URLog = %d件, want 2件", len(urlogs))
	}
	if urlogs[0].URL != worldLaunchURLBase+"?instanceId=10000~region%28jp%29&worldId=wrld_00000000-0000-4000-8000-000000000001" {
		t.Errorf("URL = %q", urlogs[0].URL)
	}
	if urlogs[0].Title != "WorldAlpha" || !urlogs[0].Start.Equal(testTime(t, "2024.01.01 10:02:00")) {
		t.Errorf("URLog = %+v", urlogs[0])
	}
	if !strings.Contains(urlogs[0].Description, "10000~region(jp)") || !slices.Equal(urlogs[0].Tags, []string{tagWorld}) {
		t.Errorf("URLog の説明・タグ = %q %v", urlogs[0].Description, urlogs[0].Tags)
	}

	players := recordsOfKind(records, kindPlayer)
	type span struct{ title, start, end string }
	got := []span{}
	for _, p := range players {
		got = append(got, span{p.Title, p.Start.Format(logTimeLayout), p.End.Format(logTimeLayout)})
		if !slices.Equal(p.Tags, []string{tagRoom}) {
			t.Errorf("人のタグ = %v", p.Tags)
		}
	}
	userB := "UserB（usr_00000000-0000-4000-8000-00000000000b）"
	userC := "UserC（usr_00000000-0000-4000-8000-00000000000c）"
	want := []span{
		{userB, "2024.01.01 10:03:00", "2024.01.01 10:10:00"}, // OnPlayerLeft で閉じる
		{userB, "2024.01.01 10:11:00", "2024.01.01 10:30:00"}, // 入り直し。自分の退室で閉じる
		{userC, "2024.01.01 10:20:00", "2024.01.01 10:30:00"},
		{userC, "2024.01.01 10:33:00", "2024.01.01 11:00:00"}, // 終了の行で閉じる
	}
	if !slices.Equal(got, want) {
		t.Errorf("同じ部屋の人 =\n%v\nwant\n%v（自分 testuser と終了後の UserD は入らない）", got, want)
	}

	events := recordsOfKind(records, kindEvent)
	contents := []string{}
	for _, e := range events {
		contents = append(contents, e.Content)
		if !slices.Equal(e.Tags, []string{tagEvent}) {
			t.Errorf("出来事のタグ = %v", e.Tags)
		}
	}
	wantContents := []string{"アバター変更: AvatarOne"}
	if !slices.Equal(contents, wantContents) {
		t.Errorf("アバターの切り替え = %v, want %v（他人のアバターは入らない）", contents, wantContents)
	}
	if !strings.Contains(events[0].SearchText, "WorldAlpha") {
		t.Errorf("出来事の検索用テキストにワールド名が無い: %q", events[0].SearchText)
	}

	// 動画は URLog。続けて同じ URL は1件、タイトルと説明に再生したワールド。
	videos := recordsOfKind(records, kindVideo)
	if len(videos) != 1 {
		t.Fatalf("動画 = %d件, want 1（続けて同じ動画は1件）", len(videos))
	}
	video := videos[0]
	if kindDataType(video.Kind) != dataTypeURLog || video.URL != "https://example.com/video1" {
		t.Errorf("動画の型・URL = %s %q", kindDataType(video.Kind), video.URL)
	}
	if !strings.HasPrefix(video.Title, videoTitle+"（") || !strings.Contains(video.Description, "再生したワールド: ") {
		t.Errorf("動画のタイトル・説明 = %q %q", video.Title, video.Description)
	}
	if !slices.Equal(video.Tags, []string{tagEvent}) || !strings.Contains(video.SearchText, video.URL) {
		t.Errorf("動画のタグ・検索用テキスト = %v %q", video.Tags, video.SearchText)
	}
	if video.ID == events[0].ID {
		t.Error("動画とアバターの切り替えの ID が重なった")
	}
}

// 落ちた起動（終了の行が無い）: 最後の時刻付きの行で、開いている TimeIs を全部閉じる。
func TestBuildRecordsCrashClosesAtLastLine(t *testing.T) {
	records := mustRecords(t, crashFixture())
	last := testTime(t, "2024.01.02 10:45:00")
	timeis := append(recordsOfKind(records, kindWorldTimeIs), recordsOfKind(records, kindPlayer)...)
	if len(timeis) != 2 {
		t.Fatalf("TimeIs = %d件, want ワールド1件と人1件", len(timeis))
	}
	for _, r := range timeis {
		if !r.End.Equal(last) {
			t.Errorf("%s の終了 = %v, want 最後の行 %v", r.Title, r.End, last)
		}
	}
}

// プレイ中に運んだ短い版と、あとの完全な版とで、両方にある記録の ID が変わらない。
// 変わると、利用者が付けたタグやテキストが外れる。
func TestBuildRecordsIDsAreStableWhenLogGrows(t *testing.T) {
	data, err := os.ReadFile(normalFixture())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.SplitAfter(string(data), "\n")
	// UserB が出て入り直した直後、UserC が来る前で切る
	prefix := strings.Join(lines[:17], "")
	if !strings.Contains(lines[16], "OnPlayerJoined UserB") {
		t.Fatalf("フィクスチャの行がずれている: %q", lines[16])
	}
	short, err := parseLog(strings.NewReader(prefix))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	full := mustRecords(t, normalFixture())
	fullIDs := map[string]record{}
	for _, r := range full {
		fullIDs[r.ID] = r
	}
	shortRecords := buildRecords(short)
	if len(shortRecords) == 0 {
		t.Fatal("短い版から記録が作られていない")
	}
	for _, r := range shortRecords {
		other, ok := fullIDs[r.ID]
		if !ok {
			t.Errorf("短い版の %s %q の ID が完全な版に無い", r.Kind, r.Title+r.Content)
			continue
		}
		if other.Kind != r.Kind || other.Title != r.Title || other.Content != r.Content || !other.Start.Equal(r.Start) {
			t.Errorf("同じ ID が別の記録を指している: %+v / %+v", r, other)
		}
	}
}

// ID は種類をまたいでも重ならない（索引が同じ ID を1レコードにまとめてしまうため。ADR-0317）。
func TestBuildRecordsIDsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, path := range []string{normalFixture(), crashFixture()} {
		for _, r := range mustRecords(t, path) {
			if previous, dup := seen[r.ID]; dup {
				t.Errorf("ID %s が %s と %s で重なっている", r.ID, previous, r.Kind)
			}
			seen[r.ID] = r.Kind
		}
	}
}

// ID はファイル名や置き場所に依らない（同期で運んだコピーと元のファイルが同じ ID になる）。
func TestBuildRecordsIDsDoNotDependOnPath(t *testing.T) {
	data, err := os.ReadFile(normalFixture())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	parsed, err := parseLog(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fromReader := buildRecords(parsed)
	fromFile := mustRecords(t, normalFixture())
	if len(fromReader) != len(fromFile) {
		t.Fatalf("件数が違う: %d / %d", len(fromReader), len(fromFile))
	}
	for i := range fromFile {
		if fromReader[i].ID != fromFile[i].ID {
			t.Errorf("ID が違う: %s / %s", fromReader[i].ID, fromFile[i].ID)
		}
	}
}
