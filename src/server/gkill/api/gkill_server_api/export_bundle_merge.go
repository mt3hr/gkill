package gkill_server_api

// 列ごとの収集結果を和集合にまとめる。
//
// 列をまたいで同じ記録が出るのは普通で、列ごとに本体を持つと素直に膨らむので、
// 本体は1回だけ持ち、列は ID の参照で持つ。
//
// 型別データ・付随データは版を絞らずに引くので、**ID だけで潰さない**
// （同じ ID の複数版が正しく来る）。鍵は ID と更新時刻の対にする。

import (
	"strconv"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// exportMerger は種別ごとに「もう入れたもの」を覚えながら足していく。
type exportMerger struct {
	seen map[string]map[string]struct{}

	kyous         []reps.Kyou
	kmemos        []reps.Kmemo
	kcs           []reps.KC
	timeiss       []reps.TimeIs
	mis           []reps.Mi
	nlogs         []reps.Nlog
	lantanas      []reps.Lantana
	urlogs        []reps.URLog
	idfKyous      []reps.IDFKyou
	reKyous       []reps.ReKyou
	gitCommitLogs []reps.GitCommitLog
	gpsLogs       []reps.GPSLog

	attachedTags          []reps.Tag
	attachedTexts         []reps.Text
	attachedNotifications []reps.Notification
	attachedTimeIss       []reps.TimeIs
	attachedTimeIsKyous   []reps.Kyou
}

func newExportMerger() *exportMerger {
	return &exportMerger{seen: map[string]map[string]struct{}{}}
}

// isNew は kind の中で key が初めてなら真を返し、覚える。
func (m *exportMerger) isNew(kind string, key string) bool {
	set, exist := m.seen[kind]
	if !exist {
		set = map[string]struct{}{}
		m.seen[kind] = set
	}
	if _, exist := set[key]; exist {
		return false
	}
	set[key] = struct{}{}
	return true
}

// formatCoordinate は緯度経度を鍵に使える形にする。
// 丸めると別の地点が同じ鍵になるので、桁を落とさない。
func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// versionKey は ID と更新時刻の対。版を落とさないための鍵。
func versionKey(id string, updateTime time.Time) string {
	return id + "\x00" + updateTime.Format(time.RFC3339Nano)
}

func mergeInto[T any](m *exportMerger, kind string, dst []T, src []T, key func(T) string) []T {
	for _, item := range src {
		if m.isNew(kind, key(item)) {
			dst = append(dst, item)
		}
	}
	return dst
}

// add は1列ぶんの収集結果を足す。
func (m *exportMerger) add(bundle *kyouBundle) {
	m.kyous = mergeInto(m, "kyou", m.kyous, bundle.Kyous,
		func(v reps.Kyou) string { return versionKey(v.ID, v.UpdateTime) })
	m.kmemos = mergeInto(m, "kmemo", m.kmemos, bundle.Kmemos,
		func(v reps.Kmemo) string { return versionKey(v.ID, v.UpdateTime) })
	m.kcs = mergeInto(m, "kc", m.kcs, bundle.KCs,
		func(v reps.KC) string { return versionKey(v.ID, v.UpdateTime) })
	m.timeiss = mergeInto(m, "timeis", m.timeiss, bundle.TimeIss,
		func(v reps.TimeIs) string { return versionKey(v.ID, v.UpdateTime) })
	m.mis = mergeInto(m, "mi", m.mis, bundle.Mis,
		func(v reps.Mi) string { return versionKey(v.ID, v.UpdateTime) })
	m.nlogs = mergeInto(m, "nlog", m.nlogs, bundle.Nlogs,
		func(v reps.Nlog) string { return versionKey(v.ID, v.UpdateTime) })
	m.lantanas = mergeInto(m, "lantana", m.lantanas, bundle.Lantanas,
		func(v reps.Lantana) string { return versionKey(v.ID, v.UpdateTime) })
	m.urlogs = mergeInto(m, "urlog", m.urlogs, bundle.URLogs,
		func(v reps.URLog) string { return versionKey(v.ID, v.UpdateTime) })
	m.idfKyous = mergeInto(m, "idf_kyou", m.idfKyous, bundle.IDFKyous,
		func(v reps.IDFKyou) string { return versionKey(v.ID, v.UpdateTime) })
	m.reKyous = mergeInto(m, "rekyou", m.reKyous, bundle.ReKyous,
		func(v reps.ReKyou) string { return versionKey(v.ID, v.UpdateTime) })
	m.gitCommitLogs = mergeInto(m, "git_commit_log", m.gitCommitLogs, bundle.GitCommitLogs,
		func(v reps.GitCommitLog) string { return versionKey(v.ID, v.UpdateTime) })

	// GPSLog には ID が無い。時刻と座標の対で潰す
	m.gpsLogs = mergeInto(m, "gps_log", m.gpsLogs, bundle.GPSLogs, func(v reps.GPSLog) string {
		return v.RelatedTime.Format(time.RFC3339Nano) + "\x00" +
			formatCoordinate(v.Latitude) + "\x00" + formatCoordinate(v.Longitude)
	})

	m.attachedTags = mergeInto(m, "tag", m.attachedTags, bundle.AttachedTags,
		func(v reps.Tag) string { return versionKey(v.ID, v.UpdateTime) })
	m.attachedTexts = mergeInto(m, "text", m.attachedTexts, bundle.AttachedTexts,
		func(v reps.Text) string { return versionKey(v.ID, v.UpdateTime) })
	m.attachedNotifications = mergeInto(m, "notification", m.attachedNotifications, bundle.AttachedNotifications,
		func(v reps.Notification) string { return versionKey(v.ID, v.UpdateTime) })
	m.attachedTimeIss = mergeInto(m, "attached_timeis", m.attachedTimeIss, bundle.AttachedTimeIss,
		func(v reps.TimeIs) string { return versionKey(v.ID, v.UpdateTime) })
	m.attachedTimeIsKyous = mergeInto(m, "attached_timeis_kyou", m.attachedTimeIsKyous, bundle.AttachedTimeIsKyous,
		func(v reps.Kyou) string { return versionKey(v.ID, v.UpdateTime) })
}

// writeTo は集めたものを焼き込みへ移す。空のままの種別は空配列を保つ。
func (m *exportMerger) writeTo(payload *exportedKyouPayload) {
	if len(m.kyous) != 0 {
		payload.Kyous = m.kyous
	}
	if len(m.kmemos) != 0 {
		payload.Kmemos = m.kmemos
	}
	if len(m.kcs) != 0 {
		payload.KCs = m.kcs
	}
	if len(m.timeiss) != 0 {
		payload.TimeIss = m.timeiss
	}
	if len(m.mis) != 0 {
		payload.Mis = m.mis
	}
	if len(m.nlogs) != 0 {
		payload.Nlogs = m.nlogs
	}
	if len(m.lantanas) != 0 {
		payload.Lantanas = m.lantanas
	}
	if len(m.urlogs) != 0 {
		payload.URLogs = m.urlogs
	}
	if len(m.idfKyous) != 0 {
		payload.IDFKyous = m.idfKyous
	}
	if len(m.reKyous) != 0 {
		payload.ReKyous = m.reKyous
	}
	if len(m.gitCommitLogs) != 0 {
		payload.GitCommitLogs = m.gitCommitLogs
	}
	if len(m.gpsLogs) != 0 {
		payload.GPSLogs = m.gpsLogs
	}
	if len(m.attachedTags) != 0 {
		payload.AttachedTags = m.attachedTags
	}
	if len(m.attachedTexts) != 0 {
		payload.AttachedTexts = m.attachedTexts
	}
	if len(m.attachedNotifications) != 0 {
		payload.AttachedNotifications = m.attachedNotifications
	}
	if len(m.attachedTimeIss) != 0 {
		payload.AttachedTimeIss = m.attachedTimeIss
	}
	if len(m.attachedTimeIsKyous) != 0 {
		payload.AttachedTimeIsKyous = m.attachedTimeIsKyous
	}
}
