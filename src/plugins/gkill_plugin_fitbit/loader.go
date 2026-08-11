package main

import (
	"encoding/csv"
	"errors"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// defaultSourcePattern は設定が空のときに使う既定のデータソース。
const defaultSourcePattern = "~/Kyou/GoogleTakeout_*"

// parseSourcePatterns は設定値をパターンのリストにする。
// 展開の実体はSDKにある（位置情報プラグインと共通）。
func parseSourcePatterns(value any) []string {
	return sdk.ParseSourcePatterns(value, defaultSourcePattern)
}

// openSources は取り込み元のZIPを開き、指標に対応するエントリだけを列挙する。
//
// ZIPしか読まない。展開済みのフォルダは対象外
// （何が入っているか分からないうえ、どの書き出しのものか判別できないため）。
//
// 返した *sdk.SourceSet は必ず Close すること。
func openSources(patterns []string) (*sdk.SourceSet, error) {
	return sdk.OpenSources(patterns, func(entryName string) bool {
		_, ok := metricPrefixOf(pathBase(entryName))
		return ok
	})
}

// pathBase はZIP内のパス（区切りは常に "/"）からベース名を取り出す。
func pathBase(entryName string) string {
	if index := strings.LastIndexByte(entryName, '/'); index >= 0 {
		return entryName[index+1:]
	}
	return entryName
}

// scannedFileOf は取り込み元のエントリを差分判定用の形に写す。
func scannedFileOf(entry sdk.SourceEntry) scannedFile {
	prefix, _ := metricPrefixOf(entry.Name)
	return scannedFile{
		Path:      entry.Path,
		MtimeUnix: entry.MtimeUnix,
		Size:      entry.Size,
		CRC32:     entry.CRC32,
		ExportID:  entry.ExportID,
		Prefix:    prefix,
	}
}

// columnPlan は1つのメトリクス定義に対する列の解決結果。
type columnPlan struct {
	def        metricDef
	valueIndex int
	matchIndex int
}

// ingestEntry はCSVを1回読んで、そのエントリが各日に寄与する部分集計を返す。
//
// 同じ接頭辞を持つ定義（心拍の平均/最大/最小など）をまとめて処理するので、
// 1エントリにつき1回しか読まない。
//
// ZIPのエントリを丸ごとメモリに載せないこと。csv.Reader を伸長ストリームに
// 直接かぶせて流し読みする（心拍だけで展開後853MBある）。
func ingestEntry(entry sdk.SourceEntry, defs []metricDef, loc *time.Location) ([]partialDaily, error) {
	source, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = source.Close() }()

	reader := csv.NewReader(source)
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	reader.LazyQuotes = true

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	// ReuseRecordなので、ヘッダは自分のスライスに写してから使う
	headerCopy := append([]string{}, headers...)

	timeIndex := -1
	plans := []columnPlan{}
	for _, def := range defs {
		if timeIndex < 0 {
			timeIndex = findCol(headerCopy, def.TimeCol)
		}
		valueIndex := -1
		if def.Agg != aggCount {
			valueIndex = findCol(headerCopy, def.ValueCol)
			if valueIndex < 0 {
				// この定義が求める列が無い＝別のファイル。この定義だけ諦める
				continue
			}
		}
		matchIndex := -1
		if def.MatchCol != "" {
			matchIndex = findCol(headerCopy, def.MatchCol)
			if matchIndex < 0 {
				continue
			}
		}
		plans = append(plans, columnPlan{def: def, valueIndex: valueIndex, matchIndex: matchIndex})
	}
	if timeIndex < 0 || len(plans) == 0 {
		// 名前は一致したが中身が違うファイル。対象外として扱う
		return nil, nil
	}

	deviceIndex := findCol(headerCopy, "datasource")
	bucket := newLocalDayBucket(loc)
	// (metricKey, dateLocal) → 部分集計
	partials := map[string]*partialDaily{}

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			// 壊れた行は飛ばして続ける
			var parseErr *csv.ParseError
			if errors.As(err, &parseErr) {
				continue
			}
			return nil, err
		}
		if timeIndex >= len(record) {
			continue
		}
		timeValue := record[timeIndex]
		if timeValue == "" {
			continue
		}
		unixSec, dateOnly, ok := parseTimestamp(timeValue)
		if !ok {
			continue
		}

		device := ""
		if deviceIndex >= 0 && deviceIndex < len(record) {
			device = record[deviceIndex]
		}

		for _, plan := range plans {
			if plan.matchIndex >= 0 {
				if plan.matchIndex >= len(record) || record[plan.matchIndex] != plan.def.MatchValue {
					continue
				}
			}

			value := 1.0
			if plan.def.Agg != aggCount {
				if plan.valueIndex >= len(record) {
					continue
				}
				raw := strings.TrimSpace(strings.ReplaceAll(record[plan.valueIndex], ",", ""))
				if raw == "" {
					continue
				}
				parsed, err := strconv.ParseFloat(raw, 64)
				if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
					// daily_sleep_temperature_derivations.csv には NaN が実在する
					continue
				}
				value = parsed
			}

			// 日付の決め方。
			// 既に日次のファイルはタイムスタンプの日付部分をそのまま使う。
			// タイムゾーン変換をすると全部1日ずれる。
			dateLocal := ""
			hourOfDay := 0
			if plan.def.Daily || dateOnly {
				dateLocal = dateOnlyPrefix(timeValue)
				if dateLocal == "" {
					continue
				}
			} else {
				date, secondsOfDay := bucket.dateOf(unixSec)
				dateLocal = date
				hourOfDay = secondsOfDay / 3600
				if hourOfDay > 23 {
					hourOfDay = 23
				}
			}

			key := plan.def.Key + "\x00" + dateLocal
			partial, exist := partials[key]
			if !exist {
				partial = &partialDaily{
					MetricKey: plan.def.Key,
					DateLocal: dateLocal,
					MinValue:  math.Inf(1),
					MaxValue:  math.Inf(-1),
					Devices:   map[string]struct{}{},
				}
				partials[key] = partial
			}
			partial.SumValue += value
			partial.CountValue++
			partial.MinValue = math.Min(partial.MinValue, value)
			partial.MaxValue = math.Max(partial.MaxValue, value)
			if unixSec >= partial.LastUnix {
				partial.LastUnix = unixSec
				partial.LastValue = value
			}
			if device != "" {
				partial.Devices[device] = struct{}{}
			}
			partial.HourSums[hourOfDay] += value
			partial.HourCounts[hourOfDay]++
		}
	}

	results := make([]partialDaily, 0, len(partials))
	for _, partial := range partials {
		if math.IsInf(partial.MinValue, 1) {
			partial.MinValue = 0
		}
		if math.IsInf(partial.MaxValue, -1) {
			partial.MaxValue = 0
		}
		results = append(results, *partial)
	}
	return results, nil
}
