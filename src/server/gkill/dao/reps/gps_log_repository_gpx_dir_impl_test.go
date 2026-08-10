package reps

// GPXファイルのディレクトリを直接読むrep(gpsLogRepositoryDirectoryImpl)のテスト。
//
// このrepは日付から yyyyMMdd.gpx を引くので、
// 「対象日のファイルが無い」ことが正常系として頻繁に起きる（記録していない日）。
// 期間の判定も他の時刻フィルタと同じく両端を含む。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// gpsLogTestBaseTime はGPXに書き込む基準時刻。ファイル名(yyyyMMdd)もこの日付から決まる。
func gpsLogTestBaseTime() time.Time {
	return time.Date(2025, 1, 15, 1, 0, 0, 0, time.UTC)
}

// writeTestGPXFile は指定時刻のトラックポイントだけを持つ最小のGPXを
// yyyyMMdd.gpx として書き出す。日付は先頭の時刻から決める。
func writeTestGPXFile(t *testing.T, dir string, times ...time.Time) {
	t.Helper()
	if len(times) == 0 {
		t.Fatal("トラックポイントの時刻が空")
	}

	trkpts := &strings.Builder{}
	for i, pointTime := range times {
		fmt.Fprintf(trkpts, "      <trkpt lat=\"35.%d\" lon=\"139.%d\"><time>%s</time></trkpt>\n",
			i, i, pointTime.UTC().Format(time.RFC3339))
	}
	gpxContent := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="gkill_test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
` + trkpts.String() + `    </trkseg>
  </trk>
</gpx>
`

	fileName := times[0].Format("20060102") + ".gpx"
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(gpxContent), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

// newTempGPSLogRepo は3点だけ入ったGPXファイルを1つ置いたrepと、その3点の時刻を返す。
func newTempGPSLogRepo(t *testing.T) (GPSLogRepository, time.Time, time.Time, time.Time) {
	t.Helper()
	dir := t.TempDir()

	firstTime := gpsLogTestBaseTime()
	middleTime := firstTime.Add(30 * time.Minute)
	lastTime := firstTime.Add(1 * time.Hour)
	writeTestGPXFile(t, dir, firstTime, middleTime, lastTime)

	return NewGPXDirRep(dir), firstTime, middleTime, lastTime
}

// 期間の両端を含むこと。同一時刻を2つ渡せばその点だけが返る。
func TestGPSLogGPXDirGetGPSLogsIncludesBothEnds(t *testing.T) {
	repo, firstTime, _, lastTime := newTempGPSLogRepo(t)
	ctx := context.Background()

	allInRange, err := repo.GetGPSLogs(ctx, &firstTime, &lastTime)
	if err != nil {
		t.Fatalf("GetGPSLogs failed: %v", err)
	}
	if len(allInRange) != 3 {
		t.Errorf("範囲の両端は含むはず: got %d件", len(allInRange))
	}

	onlyFirst, err := repo.GetGPSLogs(ctx, &firstTime, &firstTime)
	if err != nil {
		t.Fatalf("GetGPSLogs failed: %v", err)
	}
	if len(onlyFirst) != 1 {
		t.Fatalf("開始=終了ならその時刻の点だけが返るはず: got %d件", len(onlyFirst))
	}
	if !onlyFirst[0].RelatedTime.Equal(firstTime) {
		t.Errorf("RelatedTime = %v, want %v", onlyFirst[0].RelatedTime, firstTime)
	}
}

// 開始と終了を逆に渡されたら入れ替えて扱うこと（0件にはしない）。
func TestGPSLogGPXDirGetGPSLogsSwapsReversedRange(t *testing.T) {
	repo, firstTime, _, lastTime := newTempGPSLogRepo(t)
	ctx := context.Background()

	reversed, err := repo.GetGPSLogs(ctx, &lastTime, &firstTime)
	if err != nil {
		t.Fatalf("GetGPSLogs failed: %v", err)
	}
	if len(reversed) != 3 {
		t.Errorf("逆順で渡されたら入れ替えて同じ範囲として扱うはず: got %d件", len(reversed))
	}
}

// start/endともnilなら全件（GetAllGPSLogsと同じ）。
func TestGPSLogGPXDirGetGPSLogsWithoutRangeReturnsAll(t *testing.T) {
	repo, _, _, _ := newTempGPSLogRepo(t)
	ctx := context.Background()

	all, err := repo.GetGPSLogs(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetGPSLogs failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("start/endともnilなら全件返るはず: got %d件", len(all))
	}
}

// 対象日のGPXファイルが無ければ読み飛ばすだけで、エラーにはしないこと。
// 記録していない日を含む期間検索は日常的に起きる。
func TestGPSLogGPXDirGetGPSLogsSkipsMissingDateFile(t *testing.T) {
	repo, _, _, _ := newTempGPSLogRepo(t)
	ctx := context.Background()

	// GPXを置いた日から十分離れた期間。前後1日の余白を足しても該当ファイルが無い
	otherDayStart := gpsLogTestBaseTime().AddDate(0, 1, 0)
	otherDayEnd := otherDayStart.Add(1 * time.Hour)

	logs, err := repo.GetGPSLogs(ctx, &otherDayStart, &otherDayEnd)
	if err != nil {
		t.Fatalf("ファイルが無い日はエラーにせず読み飛ばすはず: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("該当日のファイルが無いので0件のはず: got %d件", len(logs))
	}
}
