package reps

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

type GPSLogRepositories []GPSLogRepository

// sortAndDedupGPSLogs は GPS 点列を RelatedTime 降順に並べ、
// (RelatedTime, Latitude, Longitude) が完全一致する重複点を隣接圧縮で畳む。
//
// 重複の混入経路は3つ: (1) rep 横断の単純連結（同名ディレクトリの二重登録等）、
// (2) GPXの±24hマージン読み（1日の照会で前後日の3ファイルを読むため、エクスポータが
// 隣接日ファイルへ重複区間を書くと同一点が最大3重になる）、(3) 1ファイル内の trk/trkseg 重複。
// GPSLog は ID を持たないため値の完全一致でしか同定できない。
// タイブレークに Latitude/Longitude を足すのは、隣接圧縮が成立するための全順序化で、
// 同時刻・異座標の点はもちろん両方残る。
// 経緯: 1年分の外部監査 C3「古い期間のGPSログは同一点が最大3重に返る」
// （素直に数えると2025-09月実測で40.0%が重複だった）。
func sortAndDedupGPSLogs(gpsLogs []GPSLog) []GPSLog {
	slices.SortFunc(gpsLogs, func(a, b GPSLog) int {
		if c := b.RelatedTime.Compare(a.RelatedTime); c != 0 {
			return c
		}
		if a.Latitude != b.Latitude {
			if a.Latitude < b.Latitude {
				return -1
			}
			return 1
		}
		if a.Longitude != b.Longitude {
			if a.Longitude < b.Longitude {
				return -1
			}
			return 1
		}
		return 0
	})
	if len(gpsLogs) < 2 {
		return gpsLogs
	}
	out := gpsLogs[:1]
	for i := 1; i < len(gpsLogs); i++ {
		prev := out[len(out)-1]
		if gpsLogs[i].RelatedTime.Equal(prev.RelatedTime) &&
			gpsLogs[i].Latitude == prev.Latitude &&
			gpsLogs[i].Longitude == prev.Longitude {
			continue
		}
		out = append(out, gpsLogs[i])
	}
	return out
}

func (g GPSLogRepositories) GetAllGPSLogs(ctx context.Context) ([]GPSLog, error) {
	gpsLogs := []GPSLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []GPSLog, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		err := threads.Go(ctx, wg, func() {
			matchGPSLogsInRep, err := rep.GetAllGPSLogs(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchGPSLogsInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at find gpsLogs: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// GPSLogs集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchGPSLogsInRep := <-ch:
			if matchGPSLogsInRep == nil {
				continue loop
			}
			gpsLogs = append(gpsLogs, matchGPSLogsInRep...)
		default:
			break loop
		}
	}

	// 重複点の排除は sortAndDedupGPSLogs のコメントを参照
	return sortAndDedupGPSLogs(gpsLogs), nil
}

func (g GPSLogRepositories) GetGPSLogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]GPSLog, error) {
	gpsLogs := []GPSLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []GPSLog, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		err := threads.Go(ctx, wg, func() {
			matchGPSLogsInRep, err := rep.GetGPSLogs(ctx, startTime, endTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchGPSLogsInRep
		})
		if err != nil {
			errch <- err
		}
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = errors.Join(err, fmt.Errorf("error at find gpsLogs: %w", e))
			existErr = true
		default:
			break errloop
		}
	}
	if existErr {
		return nil, err
	}

	// GPSLogs集約。UpdateTimeが最新のものを収める
loop:
	for {
		select {
		case matchGPSLogsInRep := <-ch:
			if matchGPSLogsInRep == nil {
				continue loop
			}
			gpsLogs = append(gpsLogs, matchGPSLogsInRep...)
		default:
			break loop
		}
	}

	// 重複点の排除は sortAndDedupGPSLogs のコメントを参照
	return sortAndDedupGPSLogs(gpsLogs), nil
}

func (g GPSLogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// 並列処理
	matchPaths := []string{}
	for _, rep := range g {
		kyous, err := rep.GetPath(ctx, id)
		if len(kyous) == 0 || err != nil {
			continue
		}
		matchPathInRep, err := rep.GetPath(ctx, id)
		if err != nil {
			continue
		}
		matchPaths = append(matchPaths, matchPathInRep)
	}
	if len(matchPaths) == 0 {
		return "", fmt.Errorf("not found path for id: %s", id)
	}
	return matchPaths[0], nil
}

func (g GPSLogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "GPSLogReps", nil
}

func (g GPSLogRepositories) UpdateCache(ctx context.Context) error {
	return nil
}
