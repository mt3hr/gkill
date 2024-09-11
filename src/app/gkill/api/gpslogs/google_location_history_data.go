package gpslogs

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
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

func GPSLogFileAsGPSLogs(repDir string, sourceFileName string, behavior req_res.FileUploadConflictBehavior, base64Data string) ([]*reps.GPSLog, error) {
	gpsLogs := []*reps.GPSLog{}

	base64Reader := bufio.NewReader(strings.NewReader(base64Data))
	decoder := base64.NewDecoder(base64.RawStdEncoding.Strict(), base64Reader)

	notDuplicationPointMap := map[time.Time]*reps.GPSLog{}
	if sourceFileName == "Records.json" {
		// googleロケーション履歴データの場合
		googleLocationHistoryData := &GoogleLocationHistoryData{}
		err := json.NewDecoder(decoder).Decode(googleLocationHistoryData)
		if err != nil {
			err = fmt.Errorf("error at parse google location history data: %w", err)
			return nil, err
		}
		for _, location := range googleLocationHistoryData.Locations {
			location.Time, err = time.Parse(time.RFC3339Nano, location.Timestamp)
			location.Time = location.Time.In(time.Local)
			if err != nil {
				err = fmt.Errorf("error at parse time %s: %w")
				return nil, err
			}
			notDuplicationPointMap[location.Time] = &reps.GPSLog{
				RelatedTime: location.Time,
				Longitude:   float64(location.LongitudeE7 / 10000000),
				Latitude:    float64(location.LatitudeE7 / 10000000),
			}
		}
	}
	for _, gpsLog := range notDuplicationPointMap {
		gpsLogs = append(gpsLogs, gpsLog)
	}
	return gpsLogs, nil
}
