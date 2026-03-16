package api

import (
	"math"
	"sync"
	"time"
)

const (
	R_DEGREE = math.Pi / 180
)

// containsString はスライス内に対象文字列が含まれるかを返す
func containsString(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

// calcDistanceKm は2点間の距離(km)を返す (Haversine)
func calcDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	lat1 *= R_DEGREE
	lng1 *= R_DEGREE
	lat2 *= R_DEGREE
	lng2 *= R_DEGREE
	return 6371.0 * math.Acos(math.Cos(lat1)*math.Cos(lat2)*math.Cos(lng2-lng1)+math.Sin(lat1)*math.Sin(lat2))
}

// upsertIfNewer はmapにkeyが存在しないか、既存より新しい場合に上書きする
func upsertIfNewer[V any](m map[string]V, key string, val V, getTime func(V) time.Time) {
	if existing, exist := m[key]; exist {
		if getTime(val).After(getTime(existing)) {
			m[key] = val
		}
	} else {
		m[key] = val
	}
}

// collectFromRepos は複数リポジトリに対して並列にfnを実行し、結果を集約する
func collectFromRepos[R any, T any](repos []R, fn func(R) ([]T, error)) ([]T, error) {
	lenOfRepos := len(repos)
	if lenOfRepos == 0 {
		return nil, nil
	}

	wg := &sync.WaitGroup{}
	resultsCh := make(chan []T, lenOfRepos)
	errch := make(chan error, lenOfRepos)
	defer close(resultsCh)
	defer close(errch)

	for _, repo := range repos {
		wg.Add(1)
		go func(r R) {
			defer wg.Done()
			items, err := fn(r)
			if err != nil {
				errch <- err
				return
			}
			resultsCh <- items
			errch <- nil
		}(repo)
	}
	wg.Wait()

	var combined []T
	var firstErr error
	for range lenOfRepos {
		e := <-errch
		if e != nil {
			if firstErr == nil {
				firstErr = e
			}
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}

	for range lenOfRepos {
		combined = append(combined, (<-resultsCh)...)
	}
	return combined, nil
}
