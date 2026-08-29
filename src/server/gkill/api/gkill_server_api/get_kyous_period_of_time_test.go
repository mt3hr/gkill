package gkill_server_api

import (
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// 時間帯フィルタの狭い窓（09:00〜10:00）を、HTTPハンドラ経由の end-to-end で固定する。
//
// 検査の狙いは2つ:
//  1. 秒オブデイ表現(MCP契約)とepoch表現(Web契約)が**同じ結果**になること。
//     解釈の正本は find.NormalizeSecondOfDay
//     （documents/adr/0108-period-of-time-second-of-day.md）。
//  2. SQL経路（キャッシュrepの GenerateFindSQLCommon）とGo経路（find_filter の
//     sortAndTrimKyousMap）のどちらを通っても同じ結果になること。
//     cache_in_memory の ON/OFF で通る経路が変わるので両方で回す。
//
// 既存の時間帯テストは1日全体を覆う窓しか使っておらず、秒の解釈が変わっても
// 赤くならなかった。この窓の狭さが本質。
func TestHandleGetKyous_PeriodOfTimeNarrowWindowBothCaches(t *testing.T) {
	for _, cacheInMemory := range []bool{false, true} {
		name := "cache_in_memory=false"
		if cacheInMemory {
			name = "cache_in_memory=true"
		}
		t.Run(name, func(t *testing.T) {
			if cacheInMemory {
				useCacheInMemory(t)
			}
			tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
			defer cleanup()

			sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

			day := time.Date(2026, 8, 19, 0, 0, 0, 0, time.Local)
			beforeID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "窓の前", day.Add(8*time.Hour+59*time.Minute+59*time.Second))
			atStartID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "窓の開始ちょうど", day.Add(9*time.Hour))
			insideID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "窓の内側", day.Add(9*time.Hour+30*time.Minute))
			atEndID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "窓の終了ちょうど", day.Add(10*time.Hour))
			afterID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "窓の後", day.Add(10*time.Hour+1*time.Second))

			secOfDayStart := int64(9 * 3600)
			secOfDayEnd := int64(10 * 3600)
			epochStart := time.Date(2026, 1, 1, 9, 0, 0, 0, time.Local).Unix()
			epochEnd := time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local).Unix()

			for _, c := range []struct {
				name       string
				start, end int64
			}{
				{name: "秒オブデイ表現(MCP契約)", start: secOfDayStart, end: secOfDayEnd},
				{name: "epoch表現(Web契約)", start: epochStart, end: epochEnd},
			} {
				t.Run(c.name, func(t *testing.T) {
					query := &find.FindQuery{
						PeriodOfTimeStartTimeSecond: &c.start,
						PeriodOfTimeEndTimeSecond:   &c.end,
					}
					res := getKyousWithQuery(t, tsURL, sessionID, query)
					if len(res.Errors) > 0 {
						t.Fatalf("get_kyous errors: %+v", res.Errors)
					}
					ids := kyouIDSet(res.Kyous)

					for id, label := range map[string]string{atStartID: "開始ちょうど", insideID: "内側", atEndID: "終了ちょうど"} {
						if !ids[id] {
							t.Errorf("%s(09:00〜10:00の内側)が返っていない", label)
						}
					}
					for id, label := range map[string]string{beforeID: "08:59:59", afterID: "10:00:01"} {
						if ids[id] {
							t.Errorf("%s(窓の外)が返っている", label)
						}
					}
				})
			}
		})
	}
}
