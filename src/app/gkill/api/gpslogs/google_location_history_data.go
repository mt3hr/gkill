package gpslogs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/twpayne/go-gpx"
)

type GoogleLocationHistoryData struct {
	Locations []*Location `json:"locations"`
}

type Location struct {
	Timestamp   string    `json:"timestamp"`
	LatitudeE7  int       `json:"latitudeE7"`
	LongitudeE7 int       `json:"longitudeE7"`
	Time        time.Time `json:"-"`
}

func GPSLogFileAsGPSLogs(repDir string, sourceFileName string, behavior req_res.FileUploadConflictBehavior, content string) ([]*reps.GPSLog, error) {
	gpsLogs := []*reps.GPSLog{}

	reader := bufio.NewReader(strings.NewReader(content))
	notDuplicationPointMap := map[time.Time]*reps.GPSLog{}
	if sourceFileName == "Records.json" {
		// googleロケーション履歴データの場合
		googleLocationHistoryData := &GoogleLocationHistoryData{}
		err := json.NewDecoder(reader).Decode(googleLocationHistoryData)
		if err != nil {
			err = fmt.Errorf("error at parse google location history data: %w", err)
			return nil, err
		}
		for _, location := range googleLocationHistoryData.Locations {
			location.Time, err = time.Parse(time.RFC3339Nano, location.Timestamp)
			location.Time = location.Time.In(time.Local)
			if err != nil {
				err = fmt.Errorf("error at parse time %s: %w", location.Timestamp, err)
				return nil, err
			}
			notDuplicationPointMap[location.Time] = &reps.GPSLog{
				RelatedTime: location.Time,
				Longitude:   float64(location.LongitudeE7 / 10000000),
				Latitude:    float64(location.LatitudeE7 / 10000000),
			}
		}
	} else if strings.HasSuffix(sourceFileName, ".gpx") {
		gpxData, err := gpx.Read(reader)
		if err != nil {
			err = fmt.Errorf("error in reading file %s. %w", sourceFileName, err)
			return nil, err
		}

		for _, trk := range gpxData.Trk {
			for _, trkseg := range trk.TrkSeg {
				for _, pt := range trkseg.TrkPt {
					gpsLog := &reps.GPSLog{}
					gpsLog.RelatedTime = pt.Time
					gpsLog.Longitude = pt.Lon
					gpsLog.Latitude = pt.Lat
					gpsLogs = append(gpsLogs, gpsLog)
				}
			}
		}
	}
	for _, gpsLog := range notDuplicationPointMap {
		gpsLogs = append(gpsLogs, gpsLog)
	}
	return gpsLogs, nil
}
