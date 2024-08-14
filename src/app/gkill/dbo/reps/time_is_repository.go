// ˅
package reps

import "context"

// ˄

type TimeIsRepository interface {
	FindKyous(ctx context.Context, queryJSON string) ([]*Kyou, error)

	GetKyou(ctx context.Context, id string) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]*Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindTimeIs(ctx context.Context, queryJSON string) ([]*TimeIs, error)

	GetTimeIs(ctx context.Context, id string) (*TimeIs, error)

	GetTimeIsHistories(ctx context.Context, id string) ([]*TimeIs, error)

	AddTimeIsInfo(ctx context.Context, timeis *TimeIs) error

	// ˅

	// ˄
}

// ˅

// ˄
