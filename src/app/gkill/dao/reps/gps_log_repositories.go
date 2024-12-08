package reps

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type GPSLogRepositories []GPSLogRepository

func (g GPSLogRepositories) GetAllGPSLogs(ctx context.Context) ([]*GPSLog, error) {
	gpsLogs := []*GPSLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*GPSLog, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GPSLogRepository) {
			defer wg.Done()
			matchGPSLogsInRep, err := rep.GetAllGPSLogs(ctx)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchGPSLogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find gpsLogs: %w", e)
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

	sort.Slice(gpsLogs, func(i, j int) bool {
		return gpsLogs[i].RelatedTime.After(gpsLogs[j].RelatedTime)
	})
	return gpsLogs, nil
}

func (g GPSLogRepositories) GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) ([]*GPSLog, error) {
	gpsLogs := []*GPSLog{}
	existErr := false
	var err error
	wg := &sync.WaitGroup{}
	ch := make(chan []*GPSLog, len(g))
	errch := make(chan error, len(g))
	defer close(ch)
	defer close(errch)

	// 並列処理
	for _, rep := range g {
		wg.Add(1)

		go func(rep GPSLogRepository) {
			defer wg.Done()
			matchGPSLogsInRep, err := rep.GetGPSLogs(ctx, startTime, endTime)
			if err != nil {
				errch <- err
				return
			}
			ch <- matchGPSLogsInRep
		}(rep)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case e := <-errch:
			err = fmt.Errorf("error at find gpsLogs: %w", e)
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

	sort.Slice(gpsLogs, func(i, j int) bool {
		return gpsLogs[i].RelatedTime.After(gpsLogs[j].RelatedTime)
	})
	return gpsLogs, nil
}

func (g GPSLogRepositories) GetPath(ctx context.Context, id string) (string, error) {
	err := fmt.Errorf("not implements GPSLogReps.GetPath")
	return "", err
}

func (g GPSLogRepositories) GetRepName(ctx context.Context) (string, error) {
	return "GPSLogReps", nil
}

func (g GPSLogRepositories) UpdateCache(ctx context.Context) error {
	return nil
}
