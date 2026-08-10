package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

func makeTimeIs(id, title string) TimeIs {
	now := testTime()
	return TimeIs{
		IsDeleted:    false,
		ID:           id,
		Title:        title,
		DataType:     "timeis",
		StartTime:    now,
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "test_user",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateUser:   "test_user",
		UpdateDevice: "test_device",
	}
}

func TestTimeIsAddAndGet(t *testing.T) {
	repo := newTempTimeIsRepo(t)
	ctx := context.Background()

	ti := makeTimeIs("timeis-001", "作業A")
	if err := repo.AddTimeIsInfo(ctx, ti); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	got, err := repo.GetTimeIs(ctx, "timeis-001", nil)
	if err != nil {
		t.Fatalf("GetTimeIs failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTimeIs returned nil")
	}
	if got.Title != "作業A" {
		t.Errorf("Title = %q, want %q", got.Title, "作業A")
	}
	if !got.StartTime.Equal(ti.StartTime) {
		t.Errorf("StartTime = %v, want %v", got.StartTime, ti.StartTime)
	}
}

func TestTimeIsFindTimeIs(t *testing.T) {
	repo := newTempTimeIsRepo(t)
	ctx := context.Background()

	ti1 := makeTimeIs("timeis-f1", "作業1")
	ti1.UpdateTime = ti1.UpdateTime.Add(1 * time.Second)
	if err := repo.AddTimeIsInfo(ctx, ti1); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	ti2 := makeTimeIs("timeis-f2", "作業2")
	ti2.UpdateTime = ti2.UpdateTime.Add(2 * time.Second)
	if err := repo.AddTimeIsInfo(ctx, ti2); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	timeIss, err := repo.FindTimeIs(ctx, query)
	if err != nil {
		t.Fatalf("FindTimeIs failed: %v", err)
	}
	if len(timeIss) != 2 {
		t.Errorf("expected 2 TimeIs entries, got %d", len(timeIss))
	}
}

func TestTimeIsGetHistories(t *testing.T) {
	repo := newTempTimeIsRepo(t)
	ctx := context.Background()

	// Add first version
	ti1 := makeTimeIs("timeis-hist", "初版作業")
	if err := repo.AddTimeIsInfo(ctx, ti1); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	// Add second version with different UpdateTime
	ti2 := makeTimeIs("timeis-hist", "改訂版作業")
	ti2.UpdateTime = ti2.UpdateTime.Add(time.Hour)
	if err := repo.AddTimeIsInfo(ctx, ti2); err != nil {
		t.Fatalf("AddTimeIsInfo failed: %v", err)
	}

	histories, err := repo.GetTimeIsHistories(ctx, "timeis-hist")
	if err != nil {
		t.Fatalf("GetTimeIsHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}

// TimeIsの検索は「開始時刻の行」と「終了時刻の行」をUNIONで作る。
// このend分岐は以前 onlyLatestData を true に固定しており、
// start分岐だけが query.OnlyLatestData を見るという非対称になっていた。
// そのため履歴表示(OnlyLatestData=false)で終了時刻だけ旧版が消えていた。
// 「Plaing検索(PlaingTime非nil)のときだけ最新版に固定する」のは仕様なのでそのまま。
//
// endTimesOfTimeIsDataType は指定DataTypeの行の終了時刻を集める。
func endTimesOfTimeIsDataType(timeiss []TimeIs, dataType string) []time.Time {
	endTimes := []time.Time{}
	for _, timeis := range timeiss {
		if timeis.DataType != dataType || timeis.EndTime == nil {
			continue
		}
		endTimes = append(endTimes, *timeis.EndTime)
	}
	return endTimes
}

// containsTimeIsTime は時刻集合に指定時刻が含まれるかを返す。
func containsTimeIsTime(times []time.Time, want time.Time) bool {
	for _, t := range times {
		if t.Equal(want) {
			return true
		}
	}
	return false
}

// addTwoVersionTimeIs は終了時刻だけ違う2版のTimeIsを追加し、(旧版, 新版)を返す。
func addTwoVersionTimeIs(t *testing.T, repo TimeIsRepository, id string) (TimeIs, TimeIs) {
	t.Helper()
	ctx := context.Background()

	oldEndTime := testTime().Add(1 * time.Hour)
	v1 := makeTimeIs(id, "作業")
	v1.EndTime = &oldEndTime
	if err := repo.AddTimeIsInfo(ctx, v1); err != nil {
		t.Fatalf("AddTimeIsInfo(v1) failed: %v", err)
	}

	newEndTime := testTime().Add(2 * time.Hour)
	v2 := makeTimeIs(id, "作業")
	v2.UpdateTime = v1.UpdateTime.Add(time.Hour)
	v2.EndTime = &newEndTime
	if err := repo.AddTimeIsInfo(ctx, v2); err != nil {
		t.Fatalf("AddTimeIsInfo(v2) failed: %v", err)
	}
	return v1, v2
}

// assertTimeIsEndBranchHonorsOnlyLatestData は FindTimeIs / FindKyous の
// end分岐が query.OnlyLatestData に従うことを確かめる。
// 非cached実装とcached実装で同じ性質を見るので共通化する。
func assertTimeIsEndBranchHonorsOnlyLatestData(t *testing.T, repo TimeIsRepository, id string) {
	t.Helper()
	ctx := context.Background()
	v1, v2 := addTwoVersionTimeIs(t, repo, id)

	// OnlyLatestData=true: 開始・終了どちらも最新版だけ
	latestTimeIss, err := repo.FindTimeIs(ctx, &find.FindQuery{OnlyLatestData: true, IncludeEndTimeIs: true})
	if err != nil {
		t.Fatalf("FindTimeIs(OnlyLatestData=true) failed: %v", err)
	}
	if len(latestTimeIss) != 2 {
		t.Fatalf("最新版の開始行・終了行の2件だけが返るはず: got %d件", len(latestTimeIss))
	}
	for _, timeis := range latestTimeIss {
		if !timeis.UpdateTime.Equal(v2.UpdateTime) {
			t.Errorf("OnlyLatestData=true に旧版(%s)が混ざっている: DataType=%s", timeis.UpdateTime, timeis.DataType)
		}
	}

	// OnlyLatestData=false: 旧版の終了時刻の射影も返る
	allTimeIss, err := repo.FindTimeIs(ctx, &find.FindQuery{OnlyLatestData: false, IncludeEndTimeIs: true})
	if err != nil {
		t.Fatalf("FindTimeIs(OnlyLatestData=false) failed: %v", err)
	}
	endTimes := endTimesOfTimeIsDataType(allTimeIss, "timeis_end")
	if len(endTimes) != 2 {
		t.Errorf("OnlyLatestData=false では終了行も全版返るはず: got %d件 (%v)", len(endTimes), endTimes)
	}
	if !containsTimeIsTime(endTimes, *v1.EndTime) {
		t.Errorf("旧版の終了時刻 %s が返っていない: got %v", v1.EndTime, endTimes)
	}
	if !containsTimeIsTime(endTimes, *v2.EndTime) {
		t.Errorf("新版の終了時刻 %s が返っていない: got %v", v2.EndTime, endTimes)
	}

	// Kyou射影でも同じ。end分岐のKyouはRELATED_TIMEが終了時刻になる
	allKyous, err := repo.FindKyous(ctx, &find.FindQuery{OnlyLatestData: false, IncludeEndTimeIs: true})
	if err != nil {
		t.Fatalf("FindKyous(OnlyLatestData=false) failed: %v", err)
	}
	endRelatedTimes := []time.Time{}
	for _, kyou := range allKyous[id] {
		if kyou.DataType == "timeis_end" {
			endRelatedTimes = append(endRelatedTimes, kyou.RelatedTime)
		}
	}
	if !containsTimeIsTime(endRelatedTimes, *v1.EndTime) {
		t.Errorf("旧版の終了Kyou %s が返っていない: got %v", v1.EndTime, endRelatedTimes)
	}

	latestKyous, err := repo.FindKyous(ctx, &find.FindQuery{OnlyLatestData: true, IncludeEndTimeIs: true})
	if err != nil {
		t.Fatalf("FindKyous(OnlyLatestData=true) failed: %v", err)
	}
	if len(latestKyous[id]) != 2 {
		t.Fatalf("最新版の開始Kyou・終了Kyouの2件だけが返るはず: got %d件", len(latestKyous[id]))
	}
	for _, kyou := range latestKyous[id] {
		if !kyou.UpdateTime.Equal(v2.UpdateTime) {
			t.Errorf("OnlyLatestData=true に旧版のKyou(%s)が混ざっている: DataType=%s", kyou.UpdateTime, kyou.DataType)
		}
	}
}

func TestTimeIsFindEndBranchHonorsOnlyLatestData(t *testing.T) {
	assertTimeIsEndBranchHonorsOnlyLatestData(t, newTempTimeIsRepo(t), "timeis-endlatest-001")
}
