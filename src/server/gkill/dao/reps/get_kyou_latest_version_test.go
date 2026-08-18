package reps

// GetKyou(id, nil) が「最新版」を返すことを固定する回帰テスト。
//
// GenerateFindSQLCommon は query.OnlyLatestData を読まず、引数の onlyLatestData しか見ない。
// 各実装はここを false で固定したまま `return &kyous[0], nil` していたため、
// updateTime 未指定のとき **そのIDの全バージョンを無順序に読んで格納順の先頭**
// (多くの場合いちばん古い版)を返していた。
// handle_add_* / handle_update_* はこの戻り値で added_kyou / updated_kyou を組み立てる。

import (
	"context"
	"testing"
	"time"
)

func TestGetKyouReturnsLatestVersion(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name string
		repo func(t *testing.T) KmemoRepository
	}{
		{"実データrep", func(t *testing.T) KmemoRepository { return newTempKmemoRepo(t) }},
		{"キャッシュrep", func(t *testing.T) KmemoRepository { return newCachedKmemoRepo(t) }},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			repo := testCase.repo(t)

			base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)
			oldVersion := makeKmemo("kmemo-latest-1", "ふるい")
			oldVersion.RelatedTime = base
			oldVersion.CreateTime = base
			oldVersion.UpdateTime = base
			if err := repo.AddKmemoInfo(ctx, oldVersion); err != nil {
				t.Fatalf("failed to add old version: %v", err)
			}

			newVersion := oldVersion
			newVersion.Content = "あたらしい"
			newVersion.UpdateTime = base.Add(time.Hour)
			if err := repo.AddKmemoInfo(ctx, newVersion); err != nil {
				t.Fatalf("failed to add new version: %v", err)
			}

			kyou, err := repo.GetKyou(ctx, "kmemo-latest-1", nil)
			if err != nil {
				t.Fatalf("failed to get kyou: %v", err)
			}
			if kyou == nil {
				t.Fatalf("見つからない")
			}
			if !kyou.UpdateTime.Equal(newVersion.UpdateTime) {
				t.Errorf("最新版が返っていない: got UpdateTime=%v, want %v", kyou.UpdateTime, newVersion.UpdateTime)
			}

			// updateTime を指定したときは、その版が返る(こちらは従来どおり)
			specified, err := repo.GetKyou(ctx, "kmemo-latest-1", &oldVersion.UpdateTime)
			if err != nil {
				t.Fatalf("failed to get kyou by update time: %v", err)
			}
			if specified == nil {
				t.Fatalf("版を指定したのに見つからない")
			}
			if !specified.UpdateTime.Equal(oldVersion.UpdateTime) {
				t.Errorf("指定した版が返っていない: got UpdateTime=%v, want %v", specified.UpdateTime, oldVersion.UpdateTime)
			}
		})
	}
}
