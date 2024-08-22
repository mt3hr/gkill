// ˅
package reps

import "context"

// ˄

type GkillRepositories struct {
	// ˅

	// ˄

	userID string

	Reps Repositories

	TagReps TagRepositories

	TextReps TextRepositories

	KmemoReps KmemoRepositories

	URLogReps URLogRepositories

	NlogReps NlogRepositories

	TimeIsReps TimeIsRepositories

	MiReps MiRepositories

	LantanaReps LantanaRepositories

	IDFKyouReps IDFKyouRepositories

	ReKyouReps *ReKyouRepositories

	GitCommitLogReps GitCommitLogRepositories

	GPSLogReps GPSLogRepositories

	WriteTagRep TagRepository

	WriteTextRep TextRepository

	WriteKmemoRep KmemoRepository

	WriteURLogRep URLogRepository

	WriteNlogRep NlogRepository

	WriteTimeIsRep TimeIsRepository

	WriteMiRep MiRepository

	WriteLantanaRep LantanaRepository

	WriteIDFKyouRep IDFKyouRepository

	WriteReKyouRep ReKyouRepository

	WriteGPSLogRep GPSLogRepository

	// ˅

	// ˄
}

func (g *GkillRepositories) GetUserID(ctx context.Context) (string, error) {
	// ˅
	return g.userID, nil
	// ˄
}

// ˅

// ˄
