package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// ---- Timeline Edits.json ----

// timelineEditsFile は タイムライン/Timeline Edits.json の構造。
// 座標を持つのは rawSignal.signal.position だけで、
// wifiScan と activityRecord は座標を持たない。
type timelineEditsFile struct {
	TimelineEdits []struct {
		DeviceID  string `json:"deviceId"`
		RawSignal *struct {
			Signal struct {
				Position *struct {
					Point                *latLngE7 `json:"point"`
					AccuracyMm           *int32    `json:"accuracyMm"`
					AltitudeMeters       *float64  `json:"altitudeMeters"`
					Source               string    `json:"source"`
					Timestamp            string    `json:"timestamp"`
					SpeedMetersPerSecond *float64  `json:"speedMetersPerSecond"`
				} `json:"position"`
			} `json:"signal"`
		} `json:"rawSignal"`
		InferredSemanticSegment   *semanticSegmentEntry `json:"inferredSemanticSegment"`
		UserEditedSemanticSegment *semanticSegmentEntry `json:"userEditedSemanticSegment"`
	} `json:"timelineEdits"`
}

type latLngE7 struct {
	LatE7 int32 `json:"latE7"`
	LngE7 int32 `json:"lngE7"`
}

type semanticSegmentEntry struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Segment   struct {
		Visit *struct {
			TopCandidate struct {
				PlaceID       string    `json:"placeId"`
				SemanticType  string    `json:"semanticType"`
				PlaceLocation *latLngE7 `json:"placeLocation"`
			} `json:"topCandidate"`
		} `json:"visit"`
		Activity *struct {
			Start *struct {
				LatLng *latLngE7 `json:"latLng"`
			} `json:"start"`
			End *struct {
				LatLng *latLngE7 `json:"latLng"`
			} `json:"end"`
		} `json:"activity"`
	} `json:"segment"`
}

// parseTimelineEdits は Timeline Edits.json を読む。
//
// 滞在地・移動区間の端点は既定では出さない（visit_points で有効にできる）。
// 生の測位より桁違いに粗く、地図に混ぜると軌跡が読めなくなるため。
func parseTimelineEdits(entry sdk.SourceEntry) ([]rawPoint, error) {
	body, err := readEntryAll(entry)
	if err != nil {
		return nil, err
	}
	var file timelineEditsFile
	if err := json.Unmarshal(body, &file); err != nil {
		return nil, err
	}

	points := []rawPoint{}
	for _, edit := range file.TimelineEdits {
		if edit.RawSignal != nil {
			position := edit.RawSignal.Signal.Position
			if position != nil && position.Point != nil {
				unixMilli, ok := parseRFC3339Milli(position.Timestamp)
				if ok {
					accuracy := int32(accuracyUnknown)
					if position.AccuracyMm != nil {
						accuracy = *position.AccuracyMm
					}
					source := position.Source
					if source == "" {
						source = sourceUnknown
					}
					points = append(points, rawPoint{
						UnixMilli:  unixMilli,
						LatE7:      position.Point.LatE7,
						LngE7:      position.Point.LngE7,
						AccuracyMm: accuracy,
						Source:     source,
						DeviceID:   edit.DeviceID,
					})
				}
			}
		}

		for _, segment := range []*semanticSegmentEntry{edit.InferredSemanticSegment, edit.UserEditedSemanticSegment} {
			if segment == nil {
				continue
			}
			startMilli, startOK := parseRFC3339Milli(segment.StartTime)
			endMilli, endOK := parseRFC3339Milli(segment.EndTime)

			if visit := segment.Segment.Visit; visit != nil && visit.TopCandidate.PlaceLocation != nil && startOK {
				points = append(points, rawPoint{
					UnixMilli:  startMilli,
					LatE7:      visit.TopCandidate.PlaceLocation.LatE7,
					LngE7:      visit.TopCandidate.PlaceLocation.LngE7,
					AccuracyMm: accuracyUnknown,
					Source:     sourceVisit,
					DeviceID:   edit.DeviceID,
				})
			}
			if activity := segment.Segment.Activity; activity != nil {
				if activity.Start != nil && activity.Start.LatLng != nil && startOK {
					points = append(points, rawPoint{
						UnixMilli: startMilli, LatE7: activity.Start.LatLng.LatE7, LngE7: activity.Start.LatLng.LngE7,
						AccuracyMm: accuracyUnknown, Source: sourceActivity, DeviceID: edit.DeviceID,
					})
				}
				if activity.End != nil && activity.End.LatLng != nil && endOK {
					points = append(points, rawPoint{
						UnixMilli: endMilli, LatE7: activity.End.LatLng.LatE7, LngE7: activity.End.LatLng.LngE7,
						AccuracyMm: accuracyUnknown, Source: sourceActivity, DeviceID: edit.DeviceID,
					})
				}
			}
		}
	}
	return points, nil
}

// ---- 端末からの書き出し (location-history.json) ----

// androidTimelineEntry は端末が書き出す location-history.json の1要素。
// 座標は "35.1234°, 139.1234°" の文字列で入っている。
type androidTimelineEntry struct {
	StartTime    string `json:"startTime"`
	EndTime      string `json:"endTime"`
	TimelinePath []struct {
		Point                              string `json:"point"`
		DurationMinutesOffsetFromStartTime string `json:"durationMinutesOffsetFromStartTime"`
	} `json:"timelinePath"`
}

// parseAndroidTimeline は端末からの書き出しを読む。
func parseAndroidTimeline(entry sdk.SourceEntry) ([]rawPoint, error) {
	body, err := readEntryAll(entry)
	if err != nil {
		return nil, err
	}
	// トップレベルが配列の版と、オブジェクトで包んである版の両方がある
	var entries []androidTimelineEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		var wrapper struct {
			SemanticSegments []androidTimelineEntry `json:"semanticSegments"`
		}
		if wrapErr := json.Unmarshal(body, &wrapper); wrapErr != nil {
			return nil, err
		}
		entries = wrapper.SemanticSegments
	}

	points := []rawPoint{}
	for _, entry := range entries {
		startMilli, ok := parseRFC3339Milli(entry.StartTime)
		if !ok {
			continue
		}
		for _, step := range entry.TimelinePath {
			latE7, lngE7, ok := parseDegreePoint(step.Point)
			if !ok {
				continue
			}
			offsetMinutes, _ := strconv.ParseInt(strings.TrimSpace(step.DurationMinutesOffsetFromStartTime), 10, 64)
			points = append(points, rawPoint{
				UnixMilli:  startMilli + offsetMinutes*60*1000,
				LatE7:      latE7,
				LngE7:      lngE7,
				AccuracyMm: accuracyUnknown,
				Source:     sourceUnknown,
			})
		}
	}
	return points, nil
}

// parseDegreePoint は "35.1234°, 139.1234°" を E7 の緯度経度にする。
func parseDegreePoint(value string) (int32, int32, bool) {
	cleaned := strings.ReplaceAll(value, "°", "")
	latText, lngText, found := strings.Cut(cleaned, ",")
	if !found {
		return 0, 0, false
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(latText), 64)
	if err != nil {
		return 0, 0, false
	}
	lng, err := strconv.ParseFloat(strings.TrimSpace(lngText), 64)
	if err != nil {
		return 0, 0, false
	}
	return degreeToE7(lat), degreeToE7(lng), true
}

// ---- 旧ロケーション履歴 (Records.json) ----

// parseRecordsJSON は Records.json を読む。
//
// 数百MB〜1GBになることがあるので、全体を Unmarshal せず
// locations 配列をトークンとして流し読みする。
func parseRecordsJSON(entry sdk.SourceEntry) ([]rawPoint, error) {
	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	decoder := json.NewDecoder(file)
	// locations 配列の開始まで進める
	if err := seekToArray(decoder, "locations"); err != nil {
		return nil, err
	}

	type recordsLocation struct {
		LatitudeE7  int32  `json:"latitudeE7"`
		LongitudeE7 int32  `json:"longitudeE7"`
		Accuracy    *int32 `json:"accuracy"` // メートル単位
		TimestampMs string `json:"timestampMs"`
		Timestamp   string `json:"timestamp"`
		Source      string `json:"source"`
		DeviceTag   int64  `json:"deviceTag"`
	}

	points := []rawPoint{}
	for decoder.More() {
		var location recordsLocation
		if err := decoder.Decode(&location); err != nil {
			// 壊れた要素は飛ばす
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		unixMilli := int64(0)
		if location.TimestampMs != "" {
			parsed, err := strconv.ParseInt(location.TimestampMs, 10, 64)
			if err != nil {
				continue
			}
			unixMilli = parsed
		} else {
			parsed, ok := parseRFC3339Milli(location.Timestamp)
			if !ok {
				continue
			}
			unixMilli = parsed
		}
		accuracy := int32(accuracyUnknown)
		if location.Accuracy != nil {
			// Records.json の accuracy はメートル。ミリメートルに直す
			accuracy = *location.Accuracy * 1000
		}
		source := location.Source
		if source == "" {
			source = sourceUnknown
		}
		deviceID := ""
		if location.DeviceTag != 0 {
			deviceID = strconv.FormatInt(location.DeviceTag, 10)
		}
		points = append(points, rawPoint{
			UnixMilli:  unixMilli,
			LatE7:      location.LatitudeE7,
			LngE7:      location.LongitudeE7,
			AccuracyMm: accuracy,
			Source:     source,
			DeviceID:   deviceID,
		})
	}
	return points, nil
}

// seekToArray は指定キーの配列の中身の直前までデコーダを進める。
func seekToArray(decoder *json.Decoder, key string) error {
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := token.(string)
		if !ok || name != key {
			continue
		}
		// 配列の開始 [ を読み捨てる
		if _, err := decoder.Token(); err != nil {
			return err
		}
		return nil
	}
}

// ---- ワークアウトのトラック (gps_location_*.csv) ----

// parseFitbitGPSCSV は timestamp,latitude,longitude,altitude,data source を読む。
func parseFitbitGPSCSV(entry sdk.SourceEntry) ([]rawPoint, error) {
	file, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	timeIndex, latIndex, lngIndex := -1, -1, -1
	for i, header := range headers {
		switch strings.ToLower(strings.TrimSpace(header)) {
		case "timestamp":
			timeIndex = i
		case "latitude":
			latIndex = i
		case "longitude":
			lngIndex = i
		}
	}
	if timeIndex < 0 || latIndex < 0 || lngIndex < 0 {
		return nil, nil
	}

	points := []rawPoint{}
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		if timeIndex >= len(record) || latIndex >= len(record) || lngIndex >= len(record) {
			continue
		}
		unixMilli, ok := parseRFC3339Milli(record[timeIndex])
		if !ok {
			continue
		}
		lat, err := strconv.ParseFloat(strings.TrimSpace(record[latIndex]), 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(strings.TrimSpace(record[lngIndex]), 64)
		if err != nil {
			continue
		}
		points = append(points, rawPoint{
			UnixMilli:  unixMilli,
			LatE7:      degreeToE7(lat),
			LngE7:      degreeToE7(lng),
			AccuracyMm: accuracyUnknown,
			Source:     sourceFitbit,
		})
	}
	return points, nil
}

// ---- 共通 ----

// parseRFC3339Milli は RFC3339 の時刻をミリ秒に変換する。
func parseRFC3339Milli(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UnixMilli(), true
		}
	}
	return 0, false
}

// degreeToE7 は度をE7固定小数にする。
func degreeToE7(degree float64) int32 {
	scaled := degree * 1e7
	if scaled < 0 {
		return int32(scaled - 0.5)
	}
	return int32(scaled + 0.5)
}

// e7ToDegree はE7固定小数を度にする。
func e7ToDegree(e7 int32) float64 {
	return float64(e7) / 1e7
}
