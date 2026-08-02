package api

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

const (
	R_DEGREE = math.Pi / 180
)

// drainFindErrors は FindKyous が並行実行したタグ/テキスト取得のエラーを回収する。
//
// wg.Wait() をこの関数の中で行うのが要点。
// 以前は呼び出し側で「完了通知チャネルをノンブロッキングに吸い出す」形をとっており、
// しかもそれを wg.Wait() より前に呼んでいたため、goroutine起動直後の空チャネルを見て
// 即座に抜けていた。結果、タグ取得・非表示タグ・タグ検索・テキスト検索・
// TimeIsテキスト・TimeIsタグの6経路のエラーが常に捨てられ、検索は成功扱いのまま
// 不完全な結果を返していた。待ち合わせを関数の内側に閉じ込めることで、
// 呼び出し順を間違えてもこの不具合が再発しないようにしている。
//
// wgの完了後は各goroutineが本体を実行し終えているので、errch / gkillErrch には
// 起動数ぶんの値が入り切っている。よってノンブロッキングに吸い出すだけでよい。
func drainFindErrors(wg *sync.WaitGroup, errch chan error, gkillErrch chan []*message.GkillError) ([]*message.GkillError, error) {
	wg.Wait()

	gkillErrors := []*message.GkillError{}
	errs := []error{}

errLoop:
	for {
		select {
		case err := <-errch:
			if err != nil {
				errs = append(errs, err)
			}
		default:
			break errLoop
		}
	}

gkillErrLoop:
	for {
		select {
		case gkillErr := <-gkillErrch:
			if len(gkillErr) != 0 {
				gkillErrors = append(gkillErrors, gkillErr...)
			}
		default:
			break gkillErrLoop
		}
	}

	return gkillErrors, errors.Join(errs...)
}

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
		}(repo)
	}
	wg.Wait()
	close(errch)
	close(resultsCh)

	var errs []error
	for e := range errch {
		errs = append(errs, e)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	var combined []T
	for items := range resultsCh {
		combined = append(combined, items...)
	}
	return combined, nil
}
