package reps

// H-07: 型別の単体取得 GetXxx(id, nil) が「最新版」を返すことを固定する。
//
// GenerateFindSQLCommon は query.OnlyLatestData を読まず、引数の onlyLatestData しか見ない。
// 各実装はここを false で固定したまま `return &xxx[0], nil` していたため、updateTime 未指定の
// とき **そのIDの全バージョンを無順序に読んで格納順の先頭**（多くの場合いちばん古い版）を
// 返していた。GetKyou では既に修正済み（get_kyou_latest_version_test.go）で、本テストは
// 型別 GetXxx への水平展開を守る。
//
// カバー範囲: kmemo/kc/lantana/nlog/urlog/timeis/mi の raw・cached と idf の raw。
// 未カバー（コード修正は適用済み・build で検証済み）:
//   - re_kyou の GetReKyou（ターゲット解決を伴い、raw 側は *GkillRepositories を要する）
//   - idf の cached（cached IDF の共通ヘルパが無い）
//   - git_commit_log の cached（同一IDの2版を git ディレクトリで作るのが煩雑）

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestGetTypedGetterReturnsLatestVersion(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)
	newer := base.Add(time.Hour)

	// assertVersions は「未指定で最新版・指定でその版」を確認する共通アサート。
	assertVersions := func(t *testing.T, latestUpdate, oldUpdate time.Time) {
		t.Helper()
		if !latestUpdate.Equal(newer) {
			t.Errorf("GetXxx(id, nil) が最新版を返していない: got %v, want %v", latestUpdate, newer)
		}
		if !oldUpdate.Equal(base) {
			t.Errorf("GetXxx(id, &base) が指定版を返していない: got %v, want %v", oldUpdate, base)
		}
	}

	t.Run("kmemo", func(t *testing.T) {
		for name, repo := range map[string]KmemoRepository{"raw": newTempKmemoRepo(t), "cached": newCachedKmemoRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeKmemo("k1", "old")
				v1.CreateTime, v1.RelatedTime, v1.UpdateTime = base, base, base
				if err := repo.AddKmemoInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.Content, v2.UpdateTime = "new", newer
				if err := repo.AddKmemoInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetKmemo(ctx, "k1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetKmemo latest: %v", err)
				}
				old, err := repo.GetKmemo(ctx, "k1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetKmemo old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("kc", func(t *testing.T) {
		for name, repo := range map[string]KCRepository{"raw": newTempKCRepo(t), "cached": newCachedKCRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeKC("kc1", "title", 1)
				v1.CreateTime, v1.RelatedTime, v1.UpdateTime = base, base, base
				if err := repo.AddKCInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.NumValue, v2.UpdateTime = json.Number("2"), newer
				if err := repo.AddKCInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetKC(ctx, "kc1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetKC latest: %v", err)
				}
				old, err := repo.GetKC(ctx, "kc1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetKC old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("lantana", func(t *testing.T) {
		for name, repo := range map[string]LantanaRepository{"raw": newTempLantanaRepo(t), "cached": newCachedLantanaRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeLantana("l1", 3)
				v1.CreateTime, v1.RelatedTime, v1.UpdateTime = base, base, base
				if err := repo.AddLantanaInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.Mood, v2.UpdateTime = 7, newer
				if err := repo.AddLantanaInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetLantana(ctx, "l1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetLantana latest: %v", err)
				}
				old, err := repo.GetLantana(ctx, "l1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetLantana old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("nlog", func(t *testing.T) {
		for name, repo := range map[string]NlogRepository{"raw": newTempNlogRepo(t), "cached": newCachedNlogRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeNlog("n1", "title", "shop", 100)
				v1.CreateTime, v1.RelatedTime, v1.UpdateTime = base, base, base
				if err := repo.AddNlogInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.Amount, v2.UpdateTime = json.Number("200"), newer
				if err := repo.AddNlogInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetNlog(ctx, "n1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetNlog latest: %v", err)
				}
				old, err := repo.GetNlog(ctx, "n1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetNlog old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("urlog", func(t *testing.T) {
		for name, repo := range map[string]URLogRepository{"raw": newTempURLogRepo(t), "cached": newCachedURLogRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeURLog("u1", "https://example.com", "old")
				v1.CreateTime, v1.RelatedTime, v1.UpdateTime = base, base, base
				if err := repo.AddURLogInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.Title, v2.UpdateTime = "new", newer
				if err := repo.AddURLogInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetURLog(ctx, "u1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetURLog latest: %v", err)
				}
				old, err := repo.GetURLog(ctx, "u1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetURLog old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("timeis", func(t *testing.T) {
		for name, repo := range map[string]TimeIsRepository{"raw": newTempTimeIsRepo(t), "cached": newCachedTimeIsRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeTimeIs("t1", "old")
				v1.CreateTime, v1.StartTime, v1.UpdateTime = base, base, base
				if err := repo.AddTimeIsInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.Title, v2.UpdateTime = "new", newer
				if err := repo.AddTimeIsInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetTimeIs(ctx, "t1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetTimeIs latest: %v", err)
				}
				old, err := repo.GetTimeIs(ctx, "t1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetTimeIs old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("mi", func(t *testing.T) {
		for name, repo := range map[string]MiRepository{"raw": newTempMiRepo(t), "cached": newCachedMiRepo(t)} {
			t.Run(name, func(t *testing.T) {
				v1 := makeMi("m1", "old")
				v1.CreateTime, v1.UpdateTime = base, base
				if err := repo.AddMiInfo(ctx, v1); err != nil {
					t.Fatal(err)
				}
				v2 := v1
				v2.Title, v2.UpdateTime = "new", newer
				if err := repo.AddMiInfo(ctx, v2); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetMi(ctx, "m1", nil)
				if err != nil || latest == nil {
					t.Fatalf("GetMi latest: %v", err)
				}
				old, err := repo.GetMi(ctx, "m1", &base)
				if err != nil || old == nil {
					t.Fatalf("GetMi old: %v", err)
				}
				assertVersions(t, latest.UpdateTime, old.UpdateTime)
			})
		}
	})

	t.Run("idf raw", func(t *testing.T) {
		repo := newTempIDFKyouRepo(t)
		v1 := makeIDFKyou("i1", "a.txt")
		v1.CreateTime, v1.RelatedTime, v1.UpdateTime = base, base, base
		if err := repo.AddIDFKyouInfo(ctx, v1); err != nil {
			t.Fatal(err)
		}
		v2 := v1
		v2.UpdateTime = newer
		if err := repo.AddIDFKyouInfo(ctx, v2); err != nil {
			t.Fatal(err)
		}
		latest, err := repo.GetIDFKyou(ctx, "i1", nil)
		if err != nil || latest == nil {
			t.Fatalf("GetIDFKyou latest: %v", err)
		}
		old, err := repo.GetIDFKyou(ctx, "i1", &base)
		if err != nil || old == nil {
			t.Fatalf("GetIDFKyou old: %v", err)
		}
		assertVersions(t, latest.UpdateTime, old.UpdateTime)
	})
}
