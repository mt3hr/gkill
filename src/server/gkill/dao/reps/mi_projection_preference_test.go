package reps

// リポジトリ経由の射影正準化の回帰テスト（2巡目の指摘 事象10）。
//
// Mi の5射影（mi_create / mi_check / mi_limit / mi_start / mi_end）は同じ1行から
// SQL が合成するラベルで、MI テーブルに DATA_TYPE 列は無い。5つとも UPDATE_TIME が
// 同着なので、素の slices.MaxFunc は「SQLite の UNION が返した先頭」を返し、
// SELECT の列の並びが違うだけで勝つ射影が変わっていた（実際 GetMi は mi_check・
// GetKyou は mi_create を返し、1つの応答の中で種別名が食い違っていた）。
//
// 比較関数そのものの単体テストは gkill_repositories_test.go の
// TestCompareMiProjectionPreference。ここでは SQL 込みの GetKyou / GetMi 経由で
// 「同着のとき正準 mi_create が勝つ」を raw / cached の両方に固定する。

import (
	"context"
	"testing"
	"time"
)

func TestMiGetKyouAndGetMiReturnCanonicalProjection(t *testing.T) {
	ctx := context.Background()

	for name, repo := range map[string]MiRepository{"raw": newTempMiRepo(t), "cached": newCachedMiRepo(t)} {
		t.Run(name, func(t *testing.T) {
			// 期限・予定開始・予定終了も入れて5射影全部を同着で出す。
			// これらの時刻は作成時刻より後ろにして、related_time 順の偶然で
			// mi_create が先頭に来る並び（退行を隠す並び）を作らない。
			mi := makeMi("mi-proj-canonical", "射影の正準を確かめるタスク")
			limitTime := mi.CreateTime.Add(1 * time.Hour)
			estimateStartTime := mi.CreateTime.Add(2 * time.Hour)
			estimateEndTime := mi.CreateTime.Add(3 * time.Hour)
			mi.LimitTime = &limitTime
			mi.EstimateStartTime = &estimateStartTime
			mi.EstimateEndTime = &estimateEndTime
			if err := repo.AddMiInfo(ctx, mi); err != nil {
				t.Fatalf("AddMiInfo failed: %v", err)
			}

			kyou, err := repo.GetKyou(ctx, "mi-proj-canonical", nil)
			if err != nil {
				t.Fatalf("GetKyou failed: %v", err)
			}
			if kyou == nil {
				t.Fatal("GetKyou returned nil")
			}
			if kyou.DataType != "mi_create" {
				t.Errorf("GetKyou の DataType = %q, want %q（同着の射影は正準が勝つ）", kyou.DataType, "mi_create")
			}

			gotMi, err := repo.GetMi(ctx, "mi-proj-canonical", nil)
			if err != nil {
				t.Fatalf("GetMi failed: %v", err)
			}
			if gotMi == nil {
				t.Fatal("GetMi returned nil")
			}
			// ここが mi_check に戻ると、updated_mi と updated_kyou の種別名が
			// 1つの応答の中で食い違う（2巡目の指摘で実測した壊れ方）
			if gotMi.DataType != "mi_create" {
				t.Errorf("GetMi の DataType = %q, want %q（GetKyou と食い違うと応答の中で種別名がずれる）", gotMi.DataType, "mi_create")
			}
		})
	}
}
