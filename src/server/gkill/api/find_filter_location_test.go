package api

// 地図検索（filterLocationKyous / calcDistanceKm）の回帰テスト。
//
// 修正対象のバグ:
//   - 区間構築が「圏内点→次点」しか追加せず、圏外→圏内の入りの区間が落ち、
//     最後の点だけが圏内の場合は区間が1つも作られず全Kyouが消えていた
//   - 区間判定が両端排他で、GPSログ時刻ちょうどのKyouが落ちていた
//   - MapRadius=0のままradius 0で距離判定され、全Kyouが黙って消えていた
//   - calcDistanceKmが同一座標で浮動小数誤差によりAcosの定義域を超えNaNを返していた

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// testPtr はポインタ型フィールドへリテラルを渡すためのテストヘルパー。
func testPtr[T any](v T) *T { return &v }

type stubGPSLogRepository struct {
	gpsLogs []reps.GPSLog
}

func (s *stubGPSLogRepository) GetAllGPSLogs(_ context.Context) ([]reps.GPSLog, error) {
	return s.gpsLogs, nil
}

func (s *stubGPSLogRepository) GetGPSLogs(_ context.Context, _ *time.Time, _ *time.Time) ([]reps.GPSLog, error) {
	return s.gpsLogs, nil
}

func (s *stubGPSLogRepository) GetPath(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (s *stubGPSLogRepository) GetRepName(_ context.Context) (string, error) {
	return "stub-gps", nil
}

func (s *stubGPSLogRepository) UpdateCache(_ context.Context) error {
	return nil
}

func (s *stubGPSLogRepository) UnWrapTyped() ([]reps.GPSLogRepository, error) {
	return []reps.GPSLogRepository{s}, nil
}

// 同一座標の距離は0(NaNではない)であること
func TestCalcDistanceKm_SamePointIsZeroNotNaN(t *testing.T) {
	got := calcDistanceKm(35.681236, 139.767125, 35.681236, 139.767125)
	if math.IsNaN(got) {
		t.Fatal("同一座標の距離がNaNになっている(Acosの定義域超過)")
	}
	if got != 0 {
		t.Errorf("同一座標の距離は0のはず: got %v", got)
	}
}

// 最後のGPSログだけが圏内でも、その手前の区間でKyouがマッチすること。
// 境界(ログ時刻ちょうど)のKyouも含まれること。
func TestFilterLocationKyous_LastPointInRadiusAndBoundary(t *testing.T) {
	ctx := context.Background()

	centerLat, centerLng := 35.0, 135.0
	farTime := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)
	arriveTime := time.Date(2026, 8, 1, 11, 0, 0, 0, time.Local)

	gpsRep := &stubGPSLogRepository{
		gpsLogs: []reps.GPSLog{
			{RelatedTime: farTime, Latitude: 36.0, Longitude: 136.0},          // 圏外
			{RelatedTime: arriveTime, Latitude: centerLat, Longitude: centerLng}, // 圏内(最後の点)
		},
	}

	betweenTime := time.Date(2026, 8, 1, 10, 30, 0, 0, time.Local)
	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			MapRadius:    testPtr(100.0), // meters
			MapLatitude:  testPtr(centerLat),
			MapLongitude: testPtr(centerLng),
		},
		Repositories: &reps.GkillRepositories{
			GPSLogReps: reps.GPSLogRepositories{gpsRep},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-between":  {{ID: "kyou-between", RelatedTime: betweenTime}},
			"kyou-boundary": {{ID: "kyou-boundary", RelatedTime: arriveTime}},
			"kyou-outside":  {{ID: "kyou-outside", RelatedTime: farTime.Add(-time.Hour)}},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterLocationKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterLocationKyous failed: %v", err)
	}

	if _, exist := findCtx.MatchKyousCurrent["kyou-between"]; !exist {
		t.Errorf("圏内点へ向かう区間内のKyouはマッチするはず(以前は最後の点だけ圏内だと全滅)")
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-boundary"]; !exist {
		t.Errorf("ログ時刻ちょうどのKyouはマッチするはず(以前は両端排他で落ちた)")
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-outside"]; exist {
		t.Errorf("区間外のKyouはマッチしないはず")
	}
}

// MapRadius=0は絞り込まずに素通しすること(以前は全Kyouが黙って消えていた)
func TestFilterLocationKyous_ZeroRadiusPassesThrough(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			MapRadius:    testPtr(0.0),
			MapLatitude:  testPtr(35.0),
			MapLongitude: testPtr(135.0),
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-1": {{ID: "kyou-1", RelatedTime: time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)}},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterLocationKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterLocationKyous failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent) != 1 {
		t.Errorf("半径未指定は素通しのはず: got %d件", len(findCtx.MatchKyousCurrent))
	}
}
