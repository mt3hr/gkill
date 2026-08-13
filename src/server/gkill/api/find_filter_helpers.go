package api

import (
	"errors"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
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

// calcDistanceKm は2点間の距離(km)を返す (球面余弦定理)
func calcDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	lat1 *= R_DEGREE
	lng1 *= R_DEGREE
	lat2 *= R_DEGREE
	lng2 *= R_DEGREE
	cosValue := math.Cos(lat1)*math.Cos(lat2)*math.Cos(lng2-lng1) + math.Sin(lat1)*math.Sin(lat2)
	// 同一座標だと浮動小数誤差で1をわずかに超え、Acosの定義域外(NaN)になり
	// 「中心点ちょうどのGPSログが圏外扱い」になるため[-1,1]に丸める
	cosValue = math.Min(1, math.Max(-1, cosValue))
	return 6371.0 * math.Acos(cosValue)
}

// kyouEntryKey はsortAndTrimKyousMapの重複排除キー。
// 「版(UpdateTime) × 射影(DataType) × 表示時刻(RelatedTime)」で同一entryを識別し、
// 同一版の同一射影がrep間で重複したものだけを1件に潰す
type kyouEntryKey struct {
	updateTimeUnix  int64
	dataType        string
	relatedTimeUnix int64
}

// uniqueStrings は順序を保って重複を除いたスライスを返す
func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exist := seen[value]; exist {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// compareKyouEntryPriority は同一Kyouの複数entry(版・射影)の代表選出の優先度を比較する。
// 新しい版が先、同じ版なら開始射影(_start)優先、以降はDataType辞書順→RelatedTime降順で決定化する。
// 負の値ならaが優先。
func compareKyouEntryPriority(a, b reps.Kyou) int {
	if c := b.UpdateTime.Compare(a.UpdateTime); c != 0 {
		return c
	}
	aStart := strings.HasSuffix(a.DataType, "_start")
	bStart := strings.HasSuffix(b.DataType, "_start")
	if aStart != bStart {
		if aStart {
			return -1
		}
		return 1
	}
	if c := strings.Compare(a.DataType, b.DataType); c != 0 {
		return c
	}
	return b.RelatedTime.Compare(a.RelatedTime)
}

// newestKyouEntry は同一Kyouの複数entryから表示用の1件を決定的に選ぶ。
// 以前は不安定ソートの先頭を採っており、同一UpdateTimeの射影(TimeIsのstart/end等)の
// どちらが残るかが実行毎に変わっていた
func newestKyouEntry(kyous []reps.Kyou) reps.Kyou {
	best := kyous[0]
	for _, kyou := range kyous[1:] {
		if compareKyouEntryPriority(kyou, best) < 0 {
			best = kyou
		}
	}
	return best
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

// inclusiveTimeInterval は両端を含む時間区間。
// end=nil は終端なしを表す。
type inclusiveTimeInterval struct {
	start time.Time
	end   *time.Time
}

// inclusiveTimeIntervalIndex は重複・連続する時間区間をマージした検索用索引。
//
// TimeIs検索と位置情報検索は、以前は「区間数 × Kyou数」の総当たりだった。
// 区間を開始時刻順にマージしておけば、各Kyouは二分探索1回で判定できる。
type inclusiveTimeIntervalIndex struct {
	intervals []inclusiveTimeInterval
}

// newInclusiveTimeIntervalIndex は区間の集まりから索引を作る。
//
// 開始時刻で並べたうえで、重なる区間・隣接する区間を1つに畳む。
// 畳んでおくことで contains が「直前の区間1つ」だけを見れば済むようになる。
// 終端が開始より前の壊れた区間は、どの時刻にも一致しないので先に捨てる。
func newInclusiveTimeIntervalIndex(intervals []inclusiveTimeInterval) inclusiveTimeIntervalIndex {
	validIntervals := make([]inclusiveTimeInterval, 0, len(intervals))
	for _, interval := range intervals {
		if interval.end != nil && interval.end.Before(interval.start) {
			continue
		}
		validIntervals = append(validIntervals, interval)
	}

	slices.SortFunc(validIntervals, func(a, b inclusiveTimeInterval) int {
		if c := a.start.Compare(b.start); c != 0 {
			return c
		}
		if a.end == nil {
			if b.end == nil {
				return 0
			}
			return 1
		}
		if b.end == nil {
			return -1
		}
		return a.end.Compare(*b.end)
	})

	merged := make([]inclusiveTimeInterval, 0, len(validIntervals))
	for _, interval := range validIntervals {
		if len(merged) == 0 {
			merged = append(merged, interval)
			continue
		}

		last := &merged[len(merged)-1]
		if last.end != nil && interval.start.After(*last.end) {
			merged = append(merged, interval)
			continue
		}

		// 重複区間を拡張する。どちらかが終端なしなら和集合も終端なし。
		if last.end == nil || interval.end == nil {
			last.end = nil
			continue
		}
		if interval.end.After(*last.end) {
			last.end = interval.end
		}
	}

	return inclusiveTimeIntervalIndex{intervals: merged}
}

// contains は target がいずれかの区間に含まれるかを返す。両端を含む。
func (i inclusiveTimeIntervalIndex) contains(target time.Time) bool {
	// targetより後に始まる最初の区間を探し、その直前だけを確認する。
	// 区間はマージ済みなので、さらに前の区間を見る必要はない。
	index, _ := slices.BinarySearchFunc(i.intervals, target, func(interval inclusiveTimeInterval, target time.Time) int {
		if interval.start.After(target) {
			return 1
		}
		return -1
	})
	if index == 0 {
		return false
	}

	interval := i.intervals[index-1]
	return interval.end == nil || !target.After(*interval.end)
}
