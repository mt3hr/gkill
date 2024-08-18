// ˅
package req_res

// ˄

type GetGPSLogResponse struct {
	// ˅

	// ˄

	Messages []*GkillMessage `json:"messages"`

	Errors []*GkillError `json:"errors"`

	GPSLogs []*GPSLog `json:"gps_logs"`

	// ˅

	// ˄
}

// ˅

// ˄
