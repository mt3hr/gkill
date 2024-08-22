// ˅
package reps

import (
	"context"
	"time"
)

// ˄

type GPSLogRepository interface {
	GetAllGPSLogs(ctx context.Context) ([]*GPSLog, error)

	GetGPSLogs(ctx context.Context, startTime time.Time, endTime time.Time) ([]*GPSLog, error)

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	UpdateCache(ctx context.Context) error

	// ˅

	// ˄
}

// ˅

// ˄
