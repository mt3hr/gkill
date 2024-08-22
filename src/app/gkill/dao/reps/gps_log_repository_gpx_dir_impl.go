// ˅
package reps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/twpayne/go-gpx"
)

// ˄

type gpsLogRepositoryDirectoryImpl struct {
	// ˅
	dirname string
	// ˄
}

// ˅
func NewGPXDirRep(dirname string) GPSLogRepository {
	return &gpsLogRepositoryDirectoryImpl{dirname: dirname}
}

func (g *gpsLogRepositoryDirectoryImpl) GetAllGPSLogs(ctx context.Context) ([]*GPSLog, error) {
	gpsLogs := []*GPSLog{}
	dirEntries, err := os.ReadDir(g.dirname)
	if err != nil {
		err = fmt.Errorf("error at get all gps logs:%w", err)
		return nil, err
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.Type().IsDir() {
			continue
		}
		fileInfo, err := dirEntry.Info()
		if err != nil {
			err = fmt.Errorf("error at get file info: %w", err)
			return nil, err
		}
		if strings.Contains(fileInfo.Name(), ".gpx") {
			gpxFileName := filepath.Join(g.dirname, fileInfo.Name())
			gpsLogsFromFile, err := g.gpxFileToGPSLogs(gpxFileName)
			if err != nil {
				err = fmt.Errorf("error at gpx file to gpsLogs %s: %w", gpxFileName, err)
				return nil, err
			}
			gpsLogs = append(gpsLogs, gpsLogsFromFile...)
		}
	}
	sort.Slice(gpsLogs, func(i, j int) bool {
		return gpsLogs[i].RelatedTime.After(gpsLogs[j].RelatedTime)
	})
	return gpsLogs, nil
}

func (g *gpsLogRepositoryDirectoryImpl) GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) ([]*GPSLog, error) {
	// 順番がおかしかったら入れ替える
	if startTime.After(endTime) {
		startTime, endTime = endTime, startTime
	}

	// ファイル名をリストアップ
	dates := []string{}
	timeLayout := "20060102"
	startDate := startTime.Format(timeLayout)
	endDate := endTime.Format(timeLayout)
	currentDate, err := time.Parse(timeLayout, startDate)
	if err != nil {
		err = fmt.Errorf("error at parse date at get gps logs %s: %w", startDate, err)
		return nil, err
	}
	for {
		currentDateStr := currentDate.Format(timeLayout)
		dates = append(dates, currentDateStr)
		if currentDateStr == endDate {
			break
		}
	}

	// gpsLogs集約
	gpsLogs := []*GPSLog{}
	for _, date := range dates {
		gpxFileName, err := g.findGPXFileByDate(ctx, date)
		if err != nil {
			err = fmt.Errorf("fialed to find gpx file by date %s. %w", date, err)
			return nil, err
		}

		if _, err := os.Stat(gpxFileName); err != nil {
			return nil, nil
		}
		matchGPSLogs, err := g.gpxFileToPoints(gpxFileName)
		if err != nil {
			err = fmt.Errorf("error at gpx file to points %s: %w", gpxFileName, err)
			return nil, err
		}

		gpsLogs = append(gpsLogs, matchGPSLogs...)
	}
	return gpsLogs, nil
}

func (g *gpsLogRepositoryDirectoryImpl) GetPath(ctx context.Context, id string) (string, error) {
	return filepath.Abs(g.dirname)
}

func (g *gpsLogRepositoryDirectoryImpl) GetRepName(ctx context.Context) (string, error) {
	return filepath.Base(g.dirname), nil
}

func (g *gpsLogRepositoryDirectoryImpl) UpdateCache(ctx context.Context) error {
	return nil
}

func (g *gpsLogRepositoryDirectoryImpl) gpxFileToGPSLogs(gpxfilename string) (gpsLogs []*GPSLog, err error) {
	gpxFile, err := os.OpenFile(gpxfilename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("failed to open file %s. %w", gpxfilename, err)
		return nil, err
	}
	defer gpxFile.Close()

	gpxData, err := gpx.Read(gpxFile)
	if err != nil {
		err = fmt.Errorf("error in reading file %s. %w", gpxfilename, err)
		return nil, err
	}

	for _, trk := range gpxData.Trk {
		for _, trkseg := range trk.TrkSeg {
			for _, pt := range trkseg.TrkPt {
				gpsLog := &GPSLog{}
				gpsLog.RelatedTime = pt.Time
				gpsLog.Longitude = pt.Lon
				gpsLog.Latitude = pt.Lat
				gpsLogs = append(gpsLogs, gpsLog)
			}
		}
	}
	return gpsLogs, nil
}

func (g *gpsLogRepositoryDirectoryImpl) findGPXFileByDate(ctx context.Context, date string) (filename string, err error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		err = fmt.Errorf("failed to parse time %s. %w", date, err)
		return "", err
	}
	filenameBase := parsedDate.Format("20060102.gpx")
	filename = filepath.Join(g.dirname, filenameBase)
	return filename, nil
}

func (g *gpsLogRepositoryDirectoryImpl) gpxFileToPoints(gpxfilename string) (gpsLogs []*GPSLog, err error) {
	gpxFile, err := os.OpenFile(gpxfilename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		err = fmt.Errorf("failed to open file %s. %w", gpxfilename, err)
		return nil, err
	}
	defer gpxFile.Close()

	gpxData, err := gpx.Read(gpxFile)
	if err != nil {
		err = fmt.Errorf("error in reading file %s. %w", gpxfilename, err)
		return nil, err
	}

	for _, trk := range gpxData.Trk {
		for _, trkseg := range trk.TrkSeg {
			for _, pt := range trkseg.TrkPt {
				gpsLog := &GPSLog{}
				gpsLog.RelatedTime = pt.Time
				gpsLog.Longitude = pt.Lon
				gpsLog.Latitude = pt.Lat
				gpsLogs = append(gpsLogs, gpsLog)
			}
		}
	}
	return gpsLogs, nil
}

// ˄
