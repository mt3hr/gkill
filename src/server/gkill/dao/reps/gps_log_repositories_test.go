package reps

import (
	"context"
	"testing"
	"time"
)

// stubGPSLogRepository はテスト用の固定点列を返す GPSLogRepository。
type stubGPSLogRepository struct {
	points []GPSLog
}

func (s *stubGPSLogRepository) GetAllGPSLogs(ctx context.Context) ([]GPSLog, error) {
	return append([]GPSLog{}, s.points...), nil
}

func (s *stubGPSLogRepository) GetGPSLogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]GPSLog, error) {
	return append([]GPSLog{}, s.points...), nil
}

func (s *stubGPSLogRepository) GetPath(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (s *stubGPSLogRepository) GetRepName(ctx context.Context) (string, error) {
	return "stub-gps", nil
}

func (s *stubGPSLogRepository) UpdateCache(ctx context.Context) error {
	return nil
}

func (s *stubGPSLogRepository) UnWrapTyped() ([]GPSLogRepository, error) {
	return []GPSLogRepository{s}, nil
}

// GPS集約の重複排除: 複数repが同一点を返しても1点に畳まれ、
// 同時刻・異座標の点は両方残る（外部監査 C3。GPXの±24hマージン読みで
// 同一点が最大3重に返り、素直に数えると期間比較が壊れていた）。
func TestGPSLogRepositoriesDedupAcrossReps(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 8, 19, 12, 0, 0, 0, time.Local)

	samePoint := GPSLog{RelatedTime: at, Latitude: 35.35, Longitude: 139.52}
	otherPlace := GPSLog{RelatedTime: at, Latitude: 35.36, Longitude: 139.52}
	otherTime := GPSLog{RelatedTime: at.Add(time.Second), Latitude: 35.35, Longitude: 139.52}

	reps := GPSLogRepositories{
		&stubGPSLogRepository{points: []GPSLog{samePoint, otherPlace}},
		&stubGPSLogRepository{points: []GPSLog{samePoint, otherTime}},
		&stubGPSLogRepository{points: []GPSLog{samePoint}},
	}

	start := at.Add(-time.Hour)
	end := at.Add(time.Hour)
	got, err := reps.GetGPSLogs(ctx, &start, &end)
	if err != nil {
		t.Fatalf("GetGPSLogs failed: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("同一点3重は1点へ、異座標・異時刻は残って計3点のはず: got %d点 %v", len(got), got)
	}

	counts := map[GPSLog]int{}
	for _, p := range got {
		counts[GPSLog{RelatedTime: p.RelatedTime, Latitude: p.Latitude, Longitude: p.Longitude}]++
	}
	for point, count := range counts {
		if count != 1 {
			t.Errorf("点 %v が %d 回返っている（重複排除漏れ）", point, count)
		}
	}

	// GetAllGPSLogs も同じ集約を通ること
	gotAll, err := reps.GetAllGPSLogs(ctx)
	if err != nil {
		t.Fatalf("GetAllGPSLogs failed: %v", err)
	}
	if len(gotAll) != 3 {
		t.Errorf("GetAllGPSLogs も重複排除されるはず: got %d点", len(gotAll))
	}

	// 並びは RelatedTime 降順
	for i := 1; i < len(got); i++ {
		if got[i].RelatedTime.After(got[i-1].RelatedTime) {
			t.Errorf("RelatedTime降順になっていない: %v の後に %v", got[i-1].RelatedTime, got[i].RelatedTime)
		}
	}
}
