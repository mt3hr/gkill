package reps

// 委譲クエリの正規化を却下した理由（絞り込みが効かなくなり結果が変わる）:
// documents/adr/0007-memoize-rekyou-target-resolution.md

// ReKyou/MiReKyou のターゲット解決をリクエスト単位でメモする。
//
// ReKyou も MiReKyou もタイトルを持たないので、ワード検索を
// ターゲットKyouを持つ実データrepへ委譲する。この委譲は入れ子になっていて、
// 既定構成の1検索では
//
//	ReKyou   → 実データrep + MiReKyou   (MiReKyouはさらに実データrepへ)
//	MiReKyou → 実データrep
//
// と、まったく同じ query で実データrepを3回舐めていた。
// 委譲先には query がそのまま渡っているので3回とも結果は同一で、1回引けば足りる。
//
// 委譲クエリを1本に正規化してしまう案は採らない。
// 委譲先の絞り込み(カレンダー・ID・時間帯 など)が効かなくなり、
// 例えば「今日作った、2020年のKmemoを指すReKyou」がカレンダー範囲=今日の検索で
// 落ちるようになる。ここで潰すのは「同じ query での重複解決」だけなので、
// 結果は1ビットも変わらない。
//
// メモを GkillRepositories に持たせるとリクエストをまたいで古い結果を返すので、
// contextに載せてリクエストごとに捨てる。
// query に載せないのは、repがqueryを書き換えてはいけないという不変条件が
// shared_find_query_mutation_test.go で固定されているため。

import (
	"context"
	"fmt"
	"sync"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// targetResolutionScope はワード委譲検索の対象rep集合を表す。
//
// ReKyou の委譲先は「data + miRekyou」で
// (collectNonReKyouRepositories がその形になっている)、
// MiReKyou の委譲先は「data」のみ。
// ReKyou は data ぶんをMiReKyouと共有し、miRekyou ぶんだけ追加で引く。
type targetResolutionScope string

const (
	// targetResolutionScopeData は実データrepのみ。ReKyouもMiReKyouも含まない。
	targetResolutionScopeData targetResolutionScope = "data"
	// targetResolutionScopeMiReKyou はMiReKyou repのみ。
	targetResolutionScopeMiReKyou targetResolutionScope = "mirekyou"
)

type targetResolutionMemoKeyType struct{}

var targetResolutionMemoContextKey = targetResolutionMemoKeyType{}

// wordMatchMemoKey は同じ解決とみなす条件。
// query はポインタで比較する。委譲先には呼び出し元の query が
// そのまま渡るので、同一検索の中では同じポインタになる。
type wordMatchMemoKey struct {
	scope targetResolutionScope
	query *find.FindQuery
}

type wordMatchMemoEntry struct {
	once sync.Once
	ids  map[string]bool
	err  error
}

type targetResolutionMemo struct {
	m           sync.Mutex
	wordMatches map[wordMatchMemoKey]*wordMatchMemoEntry
}

// WithTargetResolutionMemo は1検索ぶんのターゲット解決メモをcontextに載せる。
// すでに載っている場合はそのまま返す。
func WithTargetResolutionMemo(ctx context.Context) context.Context {
	if ctx.Value(targetResolutionMemoContextKey) != nil {
		return ctx
	}
	return context.WithValue(ctx, targetResolutionMemoContextKey, &targetResolutionMemo{
		wordMatches: map[wordMatchMemoKey]*wordMatchMemoEntry{},
	})
}

func targetResolutionMemoFrom(ctx context.Context) *targetResolutionMemo {
	memo, _ := ctx.Value(targetResolutionMemoContextKey).(*targetResolutionMemo)
	return memo
}

// resolveWordMatchTargetIDs はワード委譲検索にヒットしたKyouのID集合を返す。
// contextにメモがあれば、同じ scope と query の組はリクエスト内で1回しか検索しない。
// メモが無ければ今までどおりその場で検索する(repを直接叩く経路や単体テスト)。
//
// 返すmapはメモと共有しているので、呼び出し側は書き換えてはいけない。
func resolveWordMatchTargetIDs(ctx context.Context, scope targetResolutionScope, reps Repositories, query *find.FindQuery) (map[string]bool, error) {
	memo := targetResolutionMemoFrom(ctx)
	if memo == nil {
		return findWordMatchTargetIDs(ctx, reps, query)
	}

	key := wordMatchMemoKey{scope: scope, query: query}
	memo.m.Lock()
	entry, exist := memo.wordMatches[key]
	if !exist {
		entry = &wordMatchMemoEntry{}
		memo.wordMatches[key] = entry
	}
	memo.m.Unlock()

	// 解決中はメモのロックを持たない。
	// data の解決は実データrepしか見ないのでMiReKyouにもReKyouにも戻らず、
	// miRekyou の解決から呼ばれる data の解決は別エントリなので、
	// 同じ once を再入することはない
	// (ReKyou→MiReKyou→ReKyou→… の無限再帰を防いでいるrep集合の非対称と同じ理由)。
	entry.once.Do(func() {
		entry.ids, entry.err = findWordMatchTargetIDs(ctx, reps, query)
	})
	return entry.ids, entry.err
}

func findWordMatchTargetIDs(ctx context.Context, reps Repositories, query *find.FindQuery) (map[string]bool, error) {
	matchKyousMap, err := reps.FindKyousSequential(ctx, query)
	if err != nil {
		err = fmt.Errorf("error at find kyous for word filter: %w", err)
		return nil, err
	}
	targetIDs := map[string]bool{}
	for _, kyous := range matchKyousMap {
		for _, kyou := range kyous {
			targetIDs[kyou.ID] = true
		}
	}
	return targetIDs, nil
}

// resolveReKyouWordMatchTargetIDs はReKyouのワード委譲検索の結果を返す。
//
// ReKyouの委譲先は「実データrep + MiReKyou」なので、
// 実データrepぶんはMiReKyou側の解決とメモを共有し、MiReKyouぶんだけ別に引く。
// FindKyousSequential はrepごとの結果を1つのmapにまとめるだけなので、
// 分けて引いて和をとった結果はまとめて引いた結果と同じになる。
func resolveReKyouWordMatchTargetIDs(ctx context.Context, dataReps Repositories, miReKyouReps Repositories, query *find.FindQuery) (map[string]bool, error) {
	dataIDs, err := resolveWordMatchTargetIDs(ctx, targetResolutionScopeData, dataReps, query)
	if err != nil {
		return nil, err
	}
	miReKyouIDs, err := resolveWordMatchTargetIDs(ctx, targetResolutionScopeMiReKyou, miReKyouReps, query)
	if err != nil {
		return nil, err
	}

	// メモが持っているmapを書き換えないよう、和は新しいmapに作る
	targetIDs := make(map[string]bool, len(dataIDs)+len(miReKyouIDs))
	for id := range dataIDs {
		targetIDs[id] = true
	}
	for id := range miReKyouIDs {
		targetIDs[id] = true
	}
	return targetIDs, nil
}

// resolveMiReKyouWordMatchTargetIDs はMiReKyouのワード委譲検索の結果を返す。
// 委譲先は実データrepのみで、ReKyou側と同じメモを使う。
func resolveMiReKyouWordMatchTargetIDs(ctx context.Context, dataReps Repositories, query *find.FindQuery) (map[string]bool, error) {
	return resolveWordMatchTargetIDs(ctx, targetResolutionScopeData, dataReps, query)
}

// isWordFilterEnabled はワード委譲検索が要るかを返す。
// ReKyou/MiReKyouの3実装で同じ判定をするのでここに置く。
// （nilは未使用、非nil空は語なし=委譲不要なので、実質len判定だけで足りる）
func isWordFilterEnabled(query *find.FindQuery) bool {
	return len(query.Words) > 0 || len(query.NotWords) > 0
}
