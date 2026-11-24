package gkill_server_api

import (
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// TestExportMerger_DeduplicatesAcrossColumnsAndKeepsVersions は、複数列をまとめたときに
// 同一 (ID, UpdateTime) は1件へ畳み、同じ ID でも更新時刻が違う版は落とさずに残すこと、
// GPSLog は (RelatedTime, Latitude, Longitude) の組で重複除去されることを確認する。
func TestExportMerger_DeduplicatesAcrossColumnsAndKeepsVersions(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	bundleA := &kyouBundle{
		Kyous: []reps.Kyou{
			{ID: "k1", UpdateTime: t1, DataType: "kmemo"},
			{ID: "k1", UpdateTime: t2, DataType: "kmemo"},
		},
		Kmemos: []reps.Kmemo{
			{ID: "k1", UpdateTime: t1, Content: "v1"},
			{ID: "k1", UpdateTime: t2, Content: "v2"},
		},
		GPSLogs: []reps.GPSLog{
			{RelatedTime: t1, Latitude: 35.1234, Longitude: 139.5678},
			{RelatedTime: t2, Latitude: 35.1234, Longitude: 139.5678},
		},
		AttachedTags: []reps.Tag{
			{ID: "tag1", TargetID: "k1", Tag: "TagA", UpdateTime: t1},
		},
	}
	bundleB := &kyouBundle{
		Kyous: []reps.Kyou{
			{ID: "k1", UpdateTime: t1, DataType: "kmemo"},
			{ID: "k2", UpdateTime: t1, DataType: "kmemo"},
		},
		Kmemos: []reps.Kmemo{
			{ID: "k1", UpdateTime: t1, Content: "v1"},
			{ID: "k2", UpdateTime: t1, Content: "second"},
		},
		GPSLogs: []reps.GPSLog{
			// 同じ時刻・同じ座標は重複除去される
			{RelatedTime: t1, Latitude: 35.1234, Longitude: 139.5678},
			// 座標が違えば別地点として残る
			{RelatedTime: t1, Latitude: 35.1235, Longitude: 139.5678},
		},
		AttachedTags: []reps.Tag{
			{ID: "tag1", TargetID: "k1", Tag: "TagA", UpdateTime: t1},
			{ID: "tag2", TargetID: "k2", Tag: "TagB", UpdateTime: t1},
		},
	}

	merger := newExportMerger()
	merger.add(bundleA)
	merger.add(bundleB)

	payload := minimalExportPayload("マージ確認")
	payload.KCs = []reps.KC{}
	payload.GPSLogs = []reps.GPSLog{}
	payload.AttachedTags = []reps.Tag{}
	merger.writeTo(payload)

	if len(payload.Kyous) != 3 {
		t.Errorf("Kyous 件数 = %d, want 3 (k1@t1, k1@t2, k2@t1)", len(payload.Kyous))
	}
	if len(payload.Kmemos) != 3 {
		t.Errorf("Kmemos 件数 = %d, want 3", len(payload.Kmemos))
	}
	if len(payload.GPSLogs) != 3 {
		t.Errorf("GPSLogs 件数 = %d, want 3", len(payload.GPSLogs))
	}
	if len(payload.AttachedTags) != 2 {
		t.Errorf("AttachedTags 件数 = %d, want 2", len(payload.AttachedTags))
	}
	// 何も足されていない種別は空配列のまま残る（nil にしない）
	if payload.KCs == nil || len(payload.KCs) != 0 {
		t.Errorf("KCs = %#v, want empty non-nil slice", payload.KCs)
	}
}
