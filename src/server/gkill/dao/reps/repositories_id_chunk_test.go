package reps

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

// TestRepositoriesFindKyousSplitsHugeIDList は、IDを大量に渡された検索が
// 分割されて成立することを固定します。
//
// Miの検索は5射影のUNIONで、5本それぞれに ID IN (?, ?, ...) を丸ごと展開するため、
// バインド変数は 5N+5 になり、SQLiteの上限(32766)をN=6553で超えてPrepareが落ちます。
// 分割していないと、この検索は「too many SQL variables」で丸ごと失敗します。
//
// 実データでは、確認待ちの記録7,122件のIDを一度に渡して踏みました。
// そのときは失敗がGkillErrorにならず「成功・0件」に見えるため、
// 原因の分からない「検索結果が空」として表面化します。
func TestRepositoriesFindKyousSplitsHugeIDList(t *testing.T) {
	ctx := context.Background()

	repository, err := NewMiRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "mi.db"), true)
	if err != nil {
		t.Fatalf("failed to create mi repository: %v", err)
	}
	t.Cleanup(func() { repository.Close(ctx) })

	mi := makeMi("mi-huge-id-list-001", "たくさんのIDに紛れて探されるタスク")
	if err := repository.AddMiInfo(ctx, mi); err != nil {
		t.Fatalf("AddMiInfo failed: %v", err)
	}

	// 上限(6552件)を確実に超える数を渡す。
	ids := make([]string, 0, maxIDsPerFindQuery*2)
	ids = append(ids, mi.ID)
	for i := range maxIDsPerFindQuery * 2 {
		ids = append(ids, fmt.Sprintf("dummy-target-%05d", i))
	}

	query := makeDefaultFindQuery()
	query.IDs = ids

	matchKyous, err := Repositories{repository}.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	if len(matchKyous) == 0 {
		t.Fatalf("IDを%d件渡すと検索結果が空になる（分割されていない）", len(ids))
	}

	found := false
	for _, kyous := range matchKyous {
		for _, kyou := range kyous {
			if kyou.ID == mi.ID {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("ID %q が結果に無い", mi.ID)
	}
}

// TestRepositoriesFindKyousChunkedIDsMatchesUnchunked は、分割しても分割しなくても
// 同じ結果になることを固定します。IDリストはORの羅列なので、分割は結果を変えません。
func TestRepositoriesFindKyousChunkedIDsMatchesUnchunked(t *testing.T) {
	ctx := context.Background()

	repository, err := NewMiRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "mi.db"), true)
	if err != nil {
		t.Fatalf("failed to create mi repository: %v", err)
	}
	t.Cleanup(func() { repository.Close(ctx) })

	ids := []string{}
	for i := range 10 {
		mi := makeMi(fmt.Sprintf("mi-chunk-parity-%03d", i), fmt.Sprintf("タスク%d", i))
		if err := repository.AddMiInfo(ctx, mi); err != nil {
			t.Fatalf("AddMiInfo failed: %v", err)
		}
		ids = append(ids, mi.ID)
	}

	repositories := Repositories{repository}

	unchunkedQuery := makeDefaultFindQuery()
	unchunkedQuery.IDs = ids
	unchunked, err := repositories.FindKyous(ctx, unchunkedQuery)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}

	chunkedQuery := makeDefaultFindQuery()
	chunkedQuery.IDs = ids
	chunked, err := repositories.findKyousByChunkedIDs(ctx, chunkedQuery, true)
	if err != nil {
		t.Fatalf("findKyousByChunkedIDs failed: %v", err)
	}

	if len(chunked) != len(unchunked) {
		t.Fatalf("分割ありは%d件、分割なしは%d件", len(chunked), len(unchunked))
	}
	for key, kyous := range unchunked {
		if len(chunked[key]) != len(kyous) {
			t.Errorf("key %q: 分割ありは%d件、分割なしは%d件", key, len(chunked[key]), len(kyous))
		}
	}
}
