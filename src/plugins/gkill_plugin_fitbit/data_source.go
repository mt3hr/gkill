package main

import (
	"sort"
	"strings"
)

// foldedSource は畳み直しの入力。採用した世代の中で、1つのデータソースが1日に寄与した集計。
// 同じ (指標, 日) に複数ある（時計とスマホなど）ので、chooseDataSources で採るものを決める。
type foldedSource struct {
	metricKey   string
	dateLocal   string
	dataSource  string
	sum         float64
	count       int64
	minValue    float64
	maxValue    float64
	maxMtime    int64
	exportID    string
	hourSums    string // 改行連結（GROUP_CONCAT のまま）
	hourCounts  string
	sourcePaths string
	lastValue   float64
	lastUnix    int64
}

// normalizeDataSource はデータソース名を比較用に正規化する（前後の空白を落として小文字）。
func normalizeDataSource(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// chooseDataSources は1つの (指標, 日) に寄与したデータソースのうち、日次の値に採るものを返す。
//
// 規則:
//   - secondary に無いソース（時計・トラッカー）が1つでもあれば、それら**全部**を採る。
//     複数あるのは日の途中で時計を替えたときで、時間帯は重ならないので足してよい。
//   - 無ければ secondary を書いた順に見て、行がある最初の1つ**だけ**を採る。
//   - secondary が空なら全部を採る（2026-09-20 以前と同じ「全部合算」）。
//
// 「値が大きいほうを採る」にはしない。スマホの歩数が時計より多い日（2026-09-13:
// スマホ 28,991 / 時計 17,340）でも Fitbit アプリの日計は時計の 17,340 で、
// 大きいほうを採ると時計を着けていた日にスマホの値が混ざる。
//
// 返り値の並びは入力の並びを保つ（GROUP_CONCAT の連結順がそのまま source_paths の順になる）。
func chooseDataSources(sources []foldedSource, secondary []string) []foldedSource {
	if len(secondary) == 0 || len(sources) <= 1 {
		return sources
	}
	rankOf := map[string]int{}
	for i, name := range secondary {
		normalized := normalizeDataSource(name)
		if _, exist := rankOf[normalized]; !exist {
			rankOf[normalized] = i
		}
	}

	primary := make([]foldedSource, 0, len(sources))
	bestRank := len(secondary)
	for _, source := range sources {
		rank, isSecondary := rankOf[normalizeDataSource(source.dataSource)]
		if !isSecondary {
			primary = append(primary, source)
			continue
		}
		if rank < bestRank {
			bestRank = rank
		}
	}
	if len(primary) != 0 {
		return primary
	}
	// 時計の行が無い日。secondary の先頭に近いものを1つだけ
	chosen := make([]foldedSource, 0, 1)
	for _, source := range sources {
		if rank, ok := rankOf[normalizeDataSource(source.dataSource)]; ok && rank == bestRank {
			chosen = append(chosen, source)
		}
	}
	return chosen
}

// mergeFoldedSources は採用したデータソースの集計を1日ぶんに束ねる。
// 1つも無ければ ok=false（呼び出し側で「寄与が消えた日」として扱う）。
func mergeFoldedSources(sources []foldedSource) (foldedSource, bool) {
	if len(sources) == 0 {
		return foldedSource{}, false
	}
	merged := sources[0]
	devices := []string{sources[0].dataSource}
	hourSums := []string{sources[0].hourSums}
	hourCounts := []string{sources[0].hourCounts}
	sourcePaths := []string{sources[0].sourcePaths}
	for _, source := range sources[1:] {
		merged.sum += source.sum
		merged.count += source.count
		if source.minValue < merged.minValue {
			merged.minValue = source.minValue
		}
		if source.maxValue > merged.maxValue {
			merged.maxValue = source.maxValue
		}
		if source.maxMtime > merged.maxMtime {
			merged.maxMtime = source.maxMtime
		}
		if source.lastUnix >= merged.lastUnix {
			merged.lastUnix = source.lastUnix
			merged.lastValue = source.lastValue
		}
		devices = append(devices, source.dataSource)
		hourSums = append(hourSums, source.hourSums)
		hourCounts = append(hourCounts, source.hourCounts)
		sourcePaths = append(sourcePaths, source.sourcePaths)
	}
	sort.Strings(devices)
	merged.dataSource = strings.Join(devices, "\n")
	merged.hourSums = strings.Join(hourSums, "\n")
	merged.hourCounts = strings.Join(hourCounts, "\n")
	merged.sourcePaths = strings.Join(sourcePaths, "\n")
	return merged, true
}
