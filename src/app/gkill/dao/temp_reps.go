package dao

import (
	"context"
	"database/sql"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
)

type TempReps struct {
	IDFKyouTempRep      reps.IDFKyouTempRepository
	KCTempRep           reps.KCTempRepository
	KmemoTempRep        reps.KmemoTempRepository
	LantanaTempRep      reps.LantanaTempRepository
	MiTempRep           reps.MiTempRepository
	NlogTempRep         reps.NlogTempRepository
	NotificationTempRep reps.NotificationTempRepository
	ReKyouTempRep       reps.ReKyouTempRepository
	TagTempRep          reps.TagTempRepository
	TextTempRep         reps.TextTempRepository
	TimeIsTempRep       reps.TimeIsTempRepository
	URLogTempRep        reps.URLogTempRepository
}

func NewTempReps(db *sql.DB, m *sync.Mutex) (*TempReps, error) {
	ctx := context.Background()

	idfKyouTempRep, err := reps.NewIDFKyouTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	kcTempRep, err := reps.NewKCTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	kmemoTempRep, err := reps.NewKmemoTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	lantanaTempRep, err := reps.NewLantanaTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	miTempRep, err := reps.NewMiTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	nlogTempRep, err := reps.NewNlogTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	notificationTempRep, err := reps.NewNotificationTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	rekyouTempRep, err := reps.NewReKyouTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	tagTempRep, err := reps.NewTagTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	textTempRep, err := reps.NewTextTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	timeisTempRep, err := reps.NewTimeIsTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	urlogTempRep, err := reps.NewURLogTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}

	return &TempReps{
		IDFKyouTempRep:      idfKyouTempRep,
		KCTempRep:           kcTempRep,
		KmemoTempRep:        kmemoTempRep,
		LantanaTempRep:      lantanaTempRep,
		MiTempRep:           miTempRep,
		NlogTempRep:         nlogTempRep,
		NotificationTempRep: notificationTempRep,
		ReKyouTempRep:       rekyouTempRep,
		TagTempRep:          tagTempRep,
		TextTempRep:         textTempRep,
		TimeIsTempRep:       timeisTempRep,
		URLogTempRep:        urlogTempRep,
	}, nil
}
