package reps

import (
	"context"
	"database/sql"
	"sync"
)

type TempReps struct {
	IDFKyouTempRep      IDFKyouTempRepository
	KCTempRep           KCTempRepository
	KmemoTempRep        KmemoTempRepository
	LantanaTempRep      LantanaTempRepository
	MiTempRep           MiTempRepository
	NlogTempRep         NlogTempRepository
	NotificationTempRep NotificationTempRepository
	ReKyouTempRep       ReKyouTempRepository
	TagTempRep          TagTempRepository
	TextTempRep         TextTempRepository
	TimeIsTempRep       TimeIsTempRepository
	URLogTempRep        URLogTempRepository
}

func NewTempReps(db *sql.DB, m *sync.RWMutex) (*TempReps, error) {
	ctx := context.Background()

	idfKyouTempRep, err := NewIDFKyouTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	kcTempRep, err := NewKCTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	kmemoTempRep, err := NewKmemoTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	lantanaTempRep, err := NewLantanaTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	miTempRep, err := NewMiTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	nlogTempRep, err := NewNlogTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	notificationTempRep, err := NewNotificationTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	rekyouTempRep, err := NewReKyouTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	tagTempRep, err := NewTagTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	textTempRep, err := NewTextTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	timeisTempRep, err := NewTimeIsTempRepositorySQLite3Impl(ctx, db, m)
	if err != nil {
		return nil, err
	}
	urlogTempRep, err := NewURLogTempRepositorySQLite3Impl(ctx, db, m)
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
