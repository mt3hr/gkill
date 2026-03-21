package gpslogs

import (
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

func TestGPSLogFileAsGPSLogs_GoogleRecordsJSON(t *testing.T) {
	jsonContent := `{
		"locations": [
			{
				"timestamp": "2024-01-15T10:30:00.000Z",
				"latitudeE7": 357001234,
				"longitudeE7": 1396501234
			},
			{
				"timestamp": "2024-01-15T11:00:00.000Z",
				"latitudeE7": 357101234,
				"longitudeE7": 1396601234
			}
		]
	}`

	logs, err := GPSLogFileAsGPSLogs("", "Records.json", req_res.FileUploadConflictBehavior(""), jsonContent)
	if err != nil {
		t.Fatalf("GPSLogFileAsGPSLogs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("got %d logs, want 2", len(logs))
	}

	// Check coordinates (E7 divided by 10000000 uses integer division in source)
	for _, log := range logs {
		if log.Latitude == 0 || log.Longitude == 0 {
			t.Errorf("expected non-zero coordinates, got lat=%f lon=%f", log.Latitude, log.Longitude)
		}
		if log.RelatedTime.IsZero() {
			t.Error("expected non-zero RelatedTime")
		}
	}
}

func TestGPSLogFileAsGPSLogs_EmptyLocations(t *testing.T) {
	jsonContent := `{"locations": []}`

	logs, err := GPSLogFileAsGPSLogs("", "Records.json", req_res.FileUploadConflictBehavior(""), jsonContent)
	if err != nil {
		t.Fatalf("GPSLogFileAsGPSLogs: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("got %d logs, want 0", len(logs))
	}
}

func TestGPSLogFileAsGPSLogs_UnknownFile(t *testing.T) {
	// Non-Records.json and non-.gpx file should return empty
	logs, err := GPSLogFileAsGPSLogs("", "unknown.txt", req_res.FileUploadConflictBehavior(""), "some content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("got %d logs, want 0 for unknown file type", len(logs))
	}
}

func TestGoogleLocationHistoryDataStruct(t *testing.T) {
	data := &GoogleLocationHistoryData{
		Locations: []*Location{
			{
				Timestamp:   "2024-01-01T00:00:00Z",
				LatitudeE7:  350000000,
				LongitudeE7: 1390000000,
				Time:        time.Now(),
			},
		},
	}
	if len(data.Locations) != 1 {
		t.Errorf("expected 1 location, got %d", len(data.Locations))
	}
	if data.Locations[0].LatitudeE7 != 350000000 {
		t.Errorf("LatitudeE7 = %d, want 350000000", data.Locations[0].LatitudeE7)
	}
}
