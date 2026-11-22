package main

import (
	"testing"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// endTimeIsRows は終了の Kyou の検査に使う行（合成値）。
//
//	timeis-1: 1000〜2000 の TimeIs（ワールドの滞在）
//	player-1: 1500〜3000 の TimeIs（同じ部屋の人）
//	event-1:  2500 の Kmemo（終了の Kyou は作らない）
func endTimeIsRows() []recordRow {
	return []recordRow{
		{ID: "event-1", Kind: kindEvent, Content: "動画: x", StartUnix: 2500, RelatedUnix: 2500, UpdateUnix: 2500, Tags: tagEvent},
		{ID: "player-1", Kind: kindPlayer, Title: "UserB", StartUnix: 1500, EndUnix: 3000, RelatedUnix: 1500, UpdateUnix: 3000, Tags: tagRoom},
		{ID: "timeis-1", Kind: kindWorldTimeIs, Title: "WorldAlpha", StartUnix: 1000, EndUnix: 2000, RelatedUnix: 1000, UpdateUnix: 2000, Tags: tagWorld},
	}
}

func dataTypesByID(kyous []sdk.Kyou) map[string][]string {
	got := map[string][]string{}
	for _, kyou := range kyous {
		got[kyou.ID] = append(got[kyou.ID], kyou.DataType)
	}
	return got
}

// IncludeEndTimeIs のときだけ、TimeIs の行から終了の Kyou（timeis_end）も返す（native の TimeIs と同じ）。
// 終了の Kyou は時刻が終了時刻で、ID・型別データ・タグ・版は開始の Kyou と同じ。
func TestKyousOfRows_EndTimeIsOnlyWhenAsked(t *testing.T) {
	rows := endTimeIsRows()

	if got := dataTypesByID(kyousOfRows(rows, sdk.Query{})); len(got["timeis-1"]) != 1 || len(got["player-1"]) != 1 || len(got["event-1"]) != 1 {
		t.Errorf("IncludeEndTimeIs=false: %v, want 1行1件", got)
	}

	kyous := kyousOfRows(rows, sdk.Query{IncludeEndTimeIs: true})
	got := dataTypesByID(kyous)
	if len(got["timeis-1"]) != 2 || len(got["player-1"]) != 2 || len(got["event-1"]) != 1 {
		t.Fatalf("IncludeEndTimeIs=true: %v, want TimeIs は開始と終了の2件・Kmemo は1件", got)
	}
	for _, kyou := range kyous {
		if kyou.DataType != dataTypeTimeIsEnd {
			continue
		}
		start := kyouOf(rowByID(t, rows, kyou.ID))
		if kyou.RelatedTime.Unix() != kyou.Typed.TimeIs.EndTime.Unix() {
			t.Errorf("%s: 終了の Kyou の時刻 = %v, want 終了時刻", kyou.ID, kyou.RelatedTime)
		}
		if !kyou.UpdateTime.Equal(start.UpdateTime) || kyou.Typed.TimeIs.Title != start.Typed.TimeIs.Title || kyou.Tags[0] != start.Tags[0] {
			t.Errorf("%s: 終了の Kyou = %+v, want 版・型別データ・タグは開始と同じ", kyou.ID, kyou)
		}
	}
	// 時刻の新しい順に並ぶ（終了の Kyou も自分の時刻の位置に入る）。
	for i := 1; i < len(kyous); i++ {
		if kyous[i-1].RelatedTime.Before(kyous[i].RelatedTime) {
			t.Fatalf("並びが新しい順でない: %v が %v より前", kyous[i-1].RelatedTime, kyous[i].RelatedTime)
		}
	}
}

// 開始の Kyou は開始時刻、終了の Kyou は終了時刻で期間を判定する。
// 終了時刻だけが期間に入る TimeIs は、終了の Kyou だけが返る。
func TestKyousOfRows_EndTimeIsUsesEndTimeForCalendar(t *testing.T) {
	start, end := time.Unix(1900, 0).UTC(), time.Unix(2100, 0).UTC()
	got := dataTypesByID(kyousOfRows(endTimeIsRows(), sdk.Query{IncludeEndTimeIs: true, CalendarStartTime: &start, CalendarEndTime: &end}))
	if len(got) != 1 || len(got["timeis-1"]) != 1 || got["timeis-1"][0] != dataTypeTimeIsEnd {
		t.Errorf("期間 1900〜2100: %v, want timeis-1 の終了の Kyou だけ", got)
	}
}

// LIMIT は開始と終了を合わせた数で、新しい順に並べてから効く。
func TestKyousOfRows_EndTimeIsLimit(t *testing.T) {
	kyous := kyousOfRows(endTimeIsRows(), sdk.Query{IncludeEndTimeIs: true, Limit: 2})
	if len(kyous) != 2 {
		t.Fatalf("LIMIT 2 = %d件", len(kyous))
	}
	if kyous[0].ID != "player-1" || kyous[0].DataType != dataTypeTimeIsEnd || kyous[1].ID != "event-1" {
		t.Errorf("LIMIT 2 = %s/%s, %s/%s, want 新しい2件（player-1 の終了 3000、event-1 2500）",
			kyous[0].ID, kyous[0].DataType, kyous[1].ID, kyous[1].DataType)
	}
}

func rowByID(t *testing.T, rows []recordRow, id string) recordRow {
	t.Helper()
	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}
	t.Fatalf("行 %s が無い", id)
	return recordRow{}
}

// キャッシュの問い合わせ: IncludeEndTimeIs のときは「終了時刻だけが期間に入る TimeIs」の行も拾い、
// IDs を指定したらその行だけを読む（nil は指定なし、[] は0件）。
func TestQueryKyousEndTimeAndIDs(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dir := t.TempDir()
	stageLog(t, dir, normalFixture(), "")
	mustBuild(t, c, pluginDir, dir)

	all, err := c.QueryKyous(pluginDir, kyouQuery{})
	if err != nil || len(all) == 0 {
		t.Fatalf("全件 = %d件, %v", len(all), err)
	}
	var timeis recordRow
	for _, row := range all {
		if row.Kind == kindWorldTimeIs {
			timeis = row
			break
		}
	}
	if timeis.ID == "" {
		t.Fatal("ワールドの滞在の行が無い")
	}

	// 期間を終了時刻のちょうど1点にすると、開始時刻では拾われず、終了時刻でだけ拾われる。
	at := time.Unix(timeis.EndUnix, 0).UTC()
	hasRow := func(rows []recordRow) bool {
		for _, row := range rows {
			if row.ID == timeis.ID {
				return true
			}
		}
		return false
	}
	if rows, _ := c.QueryKyous(pluginDir, kyouQuery{start: &at, end: &at}); hasRow(rows) && timeis.StartUnix != timeis.EndUnix {
		t.Errorf("IncludeEndTimeIs=false で終了時刻だけの行を拾った")
	}
	if rows, _ := c.QueryKyous(pluginDir, kyouQuery{start: &at, end: &at, includeEndTimeIs: true}); !hasRow(rows) {
		t.Errorf("IncludeEndTimeIs=true で終了時刻が期間に入る行を拾わない")
	}

	if rows, err := c.QueryKyous(pluginDir, kyouQuery{ids: []string{timeis.ID}}); err != nil || len(rows) != 1 || rows[0].ID != timeis.ID {
		t.Errorf("IDs 指定 = %+v, %v, want その1行", rows, err)
	}
	if rows, err := c.QueryKyous(pluginDir, kyouQuery{ids: []string{}}); err != nil || len(rows) != 0 {
		t.Errorf("IDs が [] = %d件, %v, want 0件", len(rows), err)
	}
}

// 同じ部屋の人のタグは VRChat_room。
func TestRoomTagName(t *testing.T) {
	if tagRoom != "VRChat_room" {
		t.Errorf("tagRoom = %q", tagRoom)
	}
}
