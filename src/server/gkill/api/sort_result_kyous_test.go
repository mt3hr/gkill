package api

// sortResultKyous の並び順が、素直に本体を並べ替えたときと同一であることを固定する。
//
// 本体(232バイト)を直接 slices.SortFunc すると、比較のたびの Unix() 計算と
// 232バイトの memmove で 56万件では効く。キー配列を並べてから巡回置換で
// 本体を動かす形にしてあるが、**並び順は1件も変わってはいけない**。

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// referenceSortResultKyous は変更前の実装(本体を直接並べ替える)。
func referenceSortResultKyous(kyous []reps.Kyou) {
	slices.SortFunc(kyous, func(a, b reps.Kyou) int {
		aUnix := a.RelatedTime.Unix()
		bUnix := b.RelatedTime.Unix()
		if aUnix != bUnix {
			if aUnix > bUnix {
				return -1
			}
			return 1
		}
		if a.ID < b.ID {
			return -1
		} else if a.ID > b.ID {
			return 1
		}
		return 0
	})
}

func TestSortResultKyousMatchesReferenceOrder(t *testing.T) {
	// 同着(同一秒)が必ず出るように、秒の種類を件数より十分少なくする
	const kyouCount = 10000
	const distinctSeconds = 300

	random := rand.New(rand.NewSource(20260818))
	base := time.Date(2026, 8, 18, 0, 0, 0, 0, time.Local)
	source := make([]reps.Kyou, 0, kyouCount)
	for range kyouCount {
		relatedTime := base.Add(time.Duration(random.Intn(distinctSeconds)) * time.Second)
		source = append(source, reps.Kyou{
			ID:          fmt.Sprintf("kyou-%08d", random.Intn(kyouCount)),
			DataType:    "kmemo",
			RelatedTime: relatedTime,
			UpdateTime:  relatedTime,
		})
	}

	expected := slices.Clone(source)
	referenceSortResultKyous(expected)

	actual := slices.Clone(source)
	findCtx := &FindKyouContext{ParsedFindQuery: &find.FindQuery{}, ResultKyous: actual}
	if _, err := (&FindFilter{}).sortResultKyous(context.Background(), findCtx); err != nil {
		t.Fatalf("sortResultKyous failed: %v", err)
	}
	actual = findCtx.ResultKyous

	if len(actual) != len(expected) {
		t.Fatalf("件数が変わっている: got %d, want %d", len(actual), len(expected))
	}
	for i := range expected {
		// 同着(同一秒・同一ID)は入れ替わってよいので、比較キーだけを見る
		if actual[i].RelatedTime.Unix() != expected[i].RelatedTime.Unix() || actual[i].ID != expected[i].ID {
			t.Fatalf("並び順が変わっている: index=%d got (%d, %s), want (%d, %s)",
				i, actual[i].RelatedTime.Unix(), actual[i].ID,
				expected[i].RelatedTime.Unix(), expected[i].ID)
		}
	}
}

// 巡回置換は「各要素をちょうど1回だけ動かす」ので、要素の集合が保たれることも確かめる。
// 添字の付け替えを間違えると、静かに同じ要素が2つになる。
func TestSortResultKyousKeepsEveryElement(t *testing.T) {
	const kyouCount = 5000
	random := rand.New(rand.NewSource(1))
	base := time.Date(2026, 8, 18, 0, 0, 0, 0, time.Local)
	kyous := make([]reps.Kyou, 0, kyouCount)
	for i := range kyouCount {
		kyous = append(kyous, reps.Kyou{
			ID:          fmt.Sprintf("kyou-%08d", i),
			RelatedTime: base.Add(time.Duration(random.Intn(kyouCount)) * time.Second),
		})
	}

	before := map[string]int{}
	for _, kyou := range kyous {
		before[kyou.ID]++
	}

	findCtx := &FindKyouContext{ParsedFindQuery: &find.FindQuery{}, ResultKyous: kyous}
	if _, err := (&FindFilter{}).sortResultKyous(context.Background(), findCtx); err != nil {
		t.Fatalf("sortResultKyous failed: %v", err)
	}

	after := map[string]int{}
	for _, kyou := range findCtx.ResultKyous {
		after[kyou.ID]++
	}
	if len(after) != len(before) {
		t.Fatalf("要素の集合が変わっている: got %d種, want %d種", len(after), len(before))
	}
	for id, count := range before {
		if after[id] != count {
			t.Fatalf("%s の件数が変わっている: got %d, want %d", id, after[id], count)
		}
	}
	if !slices.IsSortedFunc(findCtx.ResultKyous, func(a, b reps.Kyou) int {
		if a.RelatedTime.Unix() != b.RelatedTime.Unix() {
			if a.RelatedTime.Unix() > b.RelatedTime.Unix() {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	}) {
		t.Errorf("並んでいない")
	}
}
