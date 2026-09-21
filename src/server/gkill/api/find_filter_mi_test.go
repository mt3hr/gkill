package api

// mi板向けフィルタ（filterMiForMi / overrideKyous / sortResultKyous）の回帰テスト。
//
// 修正対象のバグ:
//   - MiCheckState未指定(空文字)だとswitchにdefaultが無く、mi検索が無条件0件になっていた
//   - overrideKyousのフォールバックが「_create」を名乗りながらUpdateTimeを表示時刻にしていた
//   - miソートのCreateTime同着にタイブレークが無く、不安定ソートで順序が実行毎に変わっていた
//
// 対になるクライアントのテスト:
// src/client/__tests__/unit/classes/kyou-local-insert-mi-parity.test.ts
// （追加した記録を再検索せず列へ差し込むため、クライアントが同じ規則を実装している）

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// MiCheckState未指定(空文字)はAll扱いでmiが返ること
func TestFilterMiForMi_EmptyCheckStateTreatsAsAll(t *testing.T) {
	ctx := context.Background()

	miRep, err := reps.NewMiRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "mi.db"), true)
	if err != nil {
		t.Fatalf("failed to create mi repository: %v", err)
	}
	t.Cleanup(func() { miRep.Close(ctx) })

	baseTime := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)
	mi := reps.Mi{
		ID: "mi-1", Title: "タスク", BoardName: "default",
		CreateTime: baseTime, UpdateTime: baseTime,
		CreateApp: "test", CreateDevice: "test", CreateUser: "test",
		UpdateApp: "test", UpdateDevice: "test", UpdateUser: "test",
	}
	if err := miRep.AddMiInfo(ctx, mi); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			ForMi:           true,
			MiCheckState:    "", // 未指定
			OnlyLatestData:  true,
			IncludeCreateMi: true,
			IncludeCheckMi:  true,
			IncludeLimitMi:  true,
			IncludeStartMi:  true,
			IncludeEndMi:    true,
		},
		Repositories: &reps.GkillRepositories{
			MiReps:       reps.MiRepositories{miRep},
			MiReKyouReps: reps.MiReKyouRepositories{},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"mi-1": {{ID: "mi-1", DataType: "mi_create", RelatedTime: baseTime, UpdateTime: baseTime}},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterMiForMi(ctx, findCtx); err != nil {
		t.Fatalf("filterMiForMi failed: %v", err)
	}

	if _, exist := findCtx.MatchKyousCurrent["mi-1"]; !exist {
		t.Errorf("mi_check_state未指定はAll扱いでmiが残るはず: got %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

// ソート基準の時刻が未設定のMiは、_createを名乗り作成日時を表示時刻にすること
// (以前はUpdateTimeが入っており、作成日時ではない時刻が作成として描画されていた)
func TestOverrideKyous_FallbackUsesCreateTime(t *testing.T) {
	createTime := time.Date(2026, 8, 1, 9, 0, 0, 0, time.Local)
	updateTime := time.Date(2026, 8, 5, 21, 0, 0, 0, time.Local)

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			ForMi:      true,
			MiSortType: find.EstimateStartTime,
		},
		MatchMisAtFilterMi: map[string]reps.Mi{
			"mi-1": {ID: "mi-1", CreateTime: createTime, UpdateTime: updateTime, EstimateStartTime: nil},
		},
		MiReKyouIDs: map[string]struct{}{},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"mi-1": {{ID: "mi-1", DataType: "mi", RelatedTime: updateTime, UpdateTime: updateTime}},
		},
	}

	f := &FindFilter{}
	if _, err := f.overrideKyous(context.Background(), findCtx); err != nil {
		t.Fatalf("overrideKyous failed: %v", err)
	}

	kyou := findCtx.MatchKyousCurrent["mi-1"][0]
	if kyou.DataType != "mi_create" {
		t.Errorf("DataType = %q, want mi_create", kyou.DataType)
	}
	if !kyou.RelatedTime.Equal(createTime) {
		t.Errorf("フォールバックの表示時刻は作成日時のはず: got %v, want %v", kyou.RelatedTime, createTime)
	}
}

// miソートのCreateTime同着はIDで決定化されること
func TestSortResultKyous_MiCreateTimeTieBreaksByID(t *testing.T) {
	sameTime := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			ForMi:      true,
			MiSortType: find.CreateTime,
		},
		MatchMisAtFilterMi: map[string]reps.Mi{
			"mi-c": {ID: "mi-c", CreateTime: sameTime},
			"mi-a": {ID: "mi-a", CreateTime: sameTime},
			"mi-b": {ID: "mi-b", CreateTime: sameTime},
		},
		ResultKyous: []reps.Kyou{
			{ID: "mi-c", DataType: "mi_create", RelatedTime: sameTime},
			{ID: "mi-a", DataType: "mi_create", RelatedTime: sameTime},
			{ID: "mi-b", DataType: "mi_create", RelatedTime: sameTime},
		},
	}

	f := &FindFilter{}
	if _, err := f.sortResultKyous(context.Background(), findCtx); err != nil {
		t.Fatalf("sortResultKyous failed: %v", err)
	}

	got := []string{}
	for _, kyou := range findCtx.ResultKyous {
		got = append(got, kyou.ID)
	}
	want := []string{"mi-a", "mi-b", "mi-c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("同着はID昇順で決定化されるはず: got %v, want %v", got, want)
		}
	}
}

// for_mi の期間・時間帯フィルタは mi_sort_type の射影時刻に対して掛かる。
//
// overrideKyous は sortAndTrimKyousMap(期間判定)の**後**に RelatedTime を
// 射影時刻(期限・見積開始など)へ上書きするため、refilterOverriddenKyousForMi で
// 上書き後の時刻をもう一度判定しないと「期限で並べているのに作成日時で絞られ、
// 窓の外の時刻を持つ行が返る」（指摘 S1'）。両側（残る/消える）と、
// 期間フィルタ無しのときは何もしないこと（対照）を固定する。
func TestForMi_PeriodFilterAppliesToProjectedTime(t *testing.T) {
	ctx := context.Background()

	createTime := time.Date(2026, 8, 1, 9, 0, 0, 0, time.Local) // 朝に作成
	limitTime := time.Date(2026, 8, 3, 22, 0, 0, 0, time.Local) // 期限は夜

	newFindCtx := func(query *find.FindQuery) *FindKyouContext {
		return &FindKyouContext{
			ParsedFindQuery: query,
			MatchMisAtFilterMi: map[string]reps.Mi{
				"mi-1": {ID: "mi-1", CreateTime: createTime, LimitTime: &limitTime},
			},
			MiReKyouIDs: map[string]struct{}{},
			MatchKyousCurrent: map[string][]reps.Kyou{
				// sortAndTrim を通過した時点の行（射影前の時刻を持つ）
				"mi-1": {{ID: "mi-1", DataType: "mi", RelatedTime: createTime, UpdateTime: createTime}},
			},
		}
	}
	runOverrideAndRefilter := func(t *testing.T, findCtx *FindKyouContext) {
		t.Helper()
		f := &FindFilter{}
		if _, err := f.overrideKyous(ctx, findCtx); err != nil {
			t.Fatalf("overrideKyous failed: %v", err)
		}
		if _, err := f.refilterOverriddenKyousForMi(ctx, findCtx); err != nil {
			t.Fatalf("refilterOverriddenKyousForMi failed: %v", err)
		}
	}

	t.Run("射影時刻(期限22時)が窓21-23時の内側なら残る", func(t *testing.T) {
		start := int64(21 * 3600)
		end := int64(23 * 3600)
		findCtx := newFindCtx(&find.FindQuery{
			ForMi: true, MiSortType: find.LimitTime,
			PeriodOfTimeStartTimeSecond: &start, PeriodOfTimeEndTimeSecond: &end,
		})
		runOverrideAndRefilter(t, findCtx)
		if _, exist := findCtx.MatchKyousCurrent["mi-1"]; !exist {
			t.Error("期限(22:00)が窓21-23時の内側なのに消えた")
		}
	})

	t.Run("射影時刻(期限22時)が窓8-10時の外側なら消える", func(t *testing.T) {
		start := int64(8 * 3600)
		end := int64(10 * 3600)
		findCtx := newFindCtx(&find.FindQuery{
			ForMi: true, MiSortType: find.LimitTime,
			PeriodOfTimeStartTimeSecond: &start, PeriodOfTimeEndTimeSecond: &end,
		})
		runOverrideAndRefilter(t, findCtx)
		if _, exist := findCtx.MatchKyousCurrent["mi-1"]; exist {
			// 旧実装では作成日時(09:00)で判定され、窓8-10時に「入っている」と誤判定していた
			t.Error("期限(22:00)が窓8-10時の外側なのに残った（作成日時で判定している）")
		}
	})

	t.Run("カレンダー窓も射影時刻で判定する", func(t *testing.T) {
		windowStart := time.Date(2026, 8, 3, 0, 0, 0, 0, time.Local)
		windowEnd := time.Date(2026, 8, 3, 23, 59, 59, 0, time.Local)
		findCtx := newFindCtx(&find.FindQuery{
			ForMi: true, MiSortType: find.LimitTime,
			CalendarStartDate: &windowStart, CalendarEndDate: &windowEnd,
		})
		runOverrideAndRefilter(t, findCtx)
		if _, exist := findCtx.MatchKyousCurrent["mi-1"]; !exist {
			t.Error("期限(8/3)がカレンダー窓(8/3)の内側なのに消えた")
		}

		// 窓を作成日(8/1)にすると、期限(8/3)は外なので消える
		windowStart2 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
		windowEnd2 := time.Date(2026, 8, 1, 23, 59, 59, 0, time.Local)
		findCtx2 := newFindCtx(&find.FindQuery{
			ForMi: true, MiSortType: find.LimitTime,
			CalendarStartDate: &windowStart2, CalendarEndDate: &windowEnd2,
		})
		runOverrideAndRefilter(t, findCtx2)
		if _, exist := findCtx2.MatchKyousCurrent["mi-1"]; exist {
			t.Error("期限(8/3)がカレンダー窓(8/1)の外側なのに残った")
		}
	})

	t.Run("期間フィルタ無しなら何も消えない(対照)", func(t *testing.T) {
		findCtx := newFindCtx(&find.FindQuery{ForMi: true, MiSortType: find.LimitTime})
		runOverrideAndRefilter(t, findCtx)
		if _, exist := findCtx.MatchKyousCurrent["mi-1"]; !exist {
			t.Error("期間フィルタ無しなのに消えた")
		}
	})

	t.Run("空スライスのバケツはキーごと消える(kyous[0] panic 防止)", func(t *testing.T) {
		start := int64(21 * 3600)
		end := int64(23 * 3600)
		findCtx := newFindCtx(&find.FindQuery{
			ForMi: true, MiSortType: find.LimitTime,
			PeriodOfTimeStartTimeSecond: &start, PeriodOfTimeEndTimeSecond: &end,
		})
		findCtx.MatchKyousCurrent["empty-bucket"] = []reps.Kyou{}
		runOverrideAndRefilter(t, findCtx)
		if _, exist := findCtx.MatchKyousCurrent["empty-bucket"]; exist {
			t.Error("空スライスのバケツが残っている（後段の kyous[0] 参照で panic する）")
		}
	})
}
