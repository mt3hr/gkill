package reps

import "context"

type GkillRepositories struct {
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
}

func NewGkillRepositories(userID string) *GkillRepositories {
	return &GkillRepositories{userID: userID}
}

func (g *GkillRepositories) GetUserID(ctx context.Context) (string, error) {
	return g.userID, nil
}

func (g *GkillRepositories) Close(ctx context.Context) error {
	for _, rep := range g.TagReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.TextReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.KmemoReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.URLogReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}

	for _, rep := range g.NlogReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.TimeIsReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.MiReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.LantanaReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.IDFKyouReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.ReKyouReps.ReKyouRepositories {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	for _, rep := range g.GitCommitLogReps {
		err := rep.Close(ctx)
		if err != nil {
			return err
		}
	}
	/*
		for _, rep := range g.GPSLogReps {
			err := rep.Close(ctx)
			if err != nil {
				return err
			}
		}
	*/
	g.userID = ""

	return nil
}
