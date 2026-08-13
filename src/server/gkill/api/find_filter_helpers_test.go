package api

// find_filter_helpers.go のヘルパの回帰テスト。
//
// collectFromRepos は複数リポジトリへ並列にfnを投げて結果を集約する。
// 失敗したリポジトリのエラーは errors.Join でまとめて返す契約で、
// 「最初の1件だけ返す」形にすると、どのリポジトリが落ちたのか分からなくなる
// （同じ検索で複数repが落ちるのは珍しくない）。
//
// inclusiveTimeIntervalIndex は両端を含む区間の集合。区間をマージしてから
// 二分探索で判定するので、「境界ちょうど」「接する区間の結合」「終端なし」
// 「逆転区間の除外」の4点が崩れていないことを固定する。

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// 複数のリポジトリが失敗したとき、全部のメッセージが結果エラーに含まれること
func TestCollectFromRepos_JoinsAllRepositoryErrors(t *testing.T) {
	repNames := []string{"rep-a", "rep-b"}

	items, err := collectFromRepos(repNames, func(repName string) ([]string, error) {
		return nil, fmt.Errorf("取得失敗 %s", repName)
	})

	if err == nil {
		t.Fatal("リポジトリが全て失敗したのにエラーが返っていない")
	}
	for _, repName := range repNames {
		if want := "取得失敗 " + repName; !strings.Contains(err.Error(), want) {
			t.Errorf("エラー %q が結果に含まれていない: %v", want, err)
		}
	}
	if items != nil {
		t.Errorf("エラー時は結果を返さないはず: %v", items)
	}
}

func intervalTestTime(hour int) time.Time {
	return time.Date(2026, 8, 13, hour, 0, 0, 0, time.UTC)
}

// 重なる区間・接する区間が畳まれ、両端と終端なしが正しく判定されること
func TestInclusiveTimeIntervalIndex_MergesOverlapsAndIncludesBothEnds(t *testing.T) {
	end11 := intervalTestTime(11)
	end12 := intervalTestTime(12)
	end14 := intervalTestTime(14)
	invalidEnd := intervalTestTime(1)
	index := newInclusiveTimeIntervalIndex([]inclusiveTimeInterval{
		{start: intervalTestTime(10), end: &end11},
		{start: intervalTestTime(11), end: &end12}, // 境界で接する区間は結合
		{start: intervalTestTime(13), end: &end14},
		{start: intervalTestTime(15), end: nil},
		{start: intervalTestTime(2), end: &invalidEnd}, // 逆転区間は無効
	})

	for _, c := range []struct {
		target time.Time
		want   bool
	}{
		{intervalTestTime(9), false},
		{intervalTestTime(10), true},
		{intervalTestTime(11), true},
		{intervalTestTime(12), true},
		{intervalTestTime(12).Add(time.Minute), false},
		{intervalTestTime(13), true},
		{intervalTestTime(14), true},
		{intervalTestTime(15), true},
		{intervalTestTime(23), true},
	} {
		if got := index.contains(c.target); got != c.want {
			t.Errorf("contains(%s) = %v, want %v", c.target, got, c.want)
		}
	}
	if len(index.intervals) != 3 {
		t.Errorf("merged intervals = %d, want 3", len(index.intervals))
	}
}

// 総当たりに対する二分探索の効果を測る。区間数・Kyou数がともに増えると差が開く
func BenchmarkInclusiveTimeIntervalIndex(b *testing.B) {
	const intervalCount = 2_000
	const targetCount = 2_000
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	intervals := make([]inclusiveTimeInterval, 0, intervalCount)
	for i := range intervalCount {
		start := base.Add(time.Duration(i*2) * time.Minute)
		end := start.Add(time.Minute)
		intervals = append(intervals, inclusiveTimeInterval{start: start, end: &end})
	}
	index := newInclusiveTimeIntervalIndex(intervals)
	targets := make([]time.Time, 0, targetCount)
	for i := range targetCount {
		targets = append(targets, base.Add(time.Duration(i)*time.Minute))
	}

	b.Run("linear", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			matched := 0
			for _, target := range targets {
				for _, interval := range intervals {
					if !target.Before(interval.start) && !target.After(*interval.end) {
						matched++
						break
					}
				}
			}
			if matched == 0 {
				b.Fatal("no matches")
			}
		}
	})

	b.Run("interval_index", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			matched := 0
			for _, target := range targets {
				if index.contains(target) {
					matched++
				}
			}
			if matched == 0 {
				b.Fatal("no matches")
			}
		}
	})
}
