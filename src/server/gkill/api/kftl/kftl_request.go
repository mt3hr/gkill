package kftl

import (
	"context"
	"fmt"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/app/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

// KFTLRequest is the interface implemented by all KFTL request types.
// Mirrors: src/classes/kftl/kftl-request.ts (abstract class)
type KFTLRequest interface {
	DoRequest(ctx context.Context) error
	GetRequestID() string
	GetTags() []string
	GetTextsMap() map[string]string
	GetRelatedTime() time.Time
	SetRelatedTime(t time.Time)
	SetRelatedTimePtr(t *time.Time)
	AddTag(tag string)
	AddTextLine(textID, line string)
	GetCurrentTextID() *string
	SetCurrentTextID(textID *string)
	GetContext() *KFTLStatementLineContext
}

// KFTLRequestBase is the base struct embedded by all concrete request types.
// Mirrors: src/classes/kftl/kftl-request.ts
type KFTLRequestBase struct {
	RequestID     string
	Tags          []string
	TextsMap      map[string]string // textID → accumulated text content
	CurrentTextID *string
	relatedTime   *time.Time // nil means use addSecond offset
	Ctx           *KFTLStatementLineContext
	CreateTime    time.Time
}

func (b *KFTLRequestBase) GetRequestID() string                  { return b.RequestID }
func (b *KFTLRequestBase) GetTags() []string                     { return b.Tags }
func (b *KFTLRequestBase) GetTextsMap() map[string]string        { return b.TextsMap }
func (b *KFTLRequestBase) GetCurrentTextID() *string             { return b.CurrentTextID }
func (b *KFTLRequestBase) SetCurrentTextID(textID *string)       { b.CurrentTextID = textID }
func (b *KFTLRequestBase) GetContext() *KFTLStatementLineContext { return b.Ctx }

// GetRelatedTime returns the effective related time.
// If not explicitly set, returns now + addSecond offset.
// Mirrors: KFTLRequest.get_related_time()
func (b *KFTLRequestBase) GetRelatedTime() time.Time {
	if b.relatedTime != nil {
		return *b.relatedTime
	}
	return time.Now().Add(time.Duration(b.Ctx.AddSecond) * time.Second)
}

func (b *KFTLRequestBase) SetRelatedTime(t time.Time)     { b.relatedTime = &t }
func (b *KFTLRequestBase) SetRelatedTimePtr(t *time.Time) { b.relatedTime = t }

func (b *KFTLRequestBase) AddTag(tag string) {
	b.Tags = append(b.Tags, tag)
}

func (b *KFTLRequestBase) AddTextLine(textID, line string) {
	if b.TextsMap == nil {
		b.TextsMap = make(map[string]string)
	}
	existing, ok := b.TextsMap[textID]
	if !ok || existing == "" {
		b.TextsMap[textID] = line
	} else {
		b.TextsMap[textID] = existing + "\n" + line
	}
}

// updateLatestDataRepositoryAddress updates the in-memory cache and DAO for one entity.
// Mirrors the pattern used in gkill_server_api.go L2070-2082.
func updateLatestDataRepositoryAddress(ctx context.Context, repos *reps.GkillRepositories,
	id string, targetIDInData *string, isDeleted bool, updateTime time.Time, repName string) {
	repos.LatestDataRepositoryAddresses[id] = gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              isDeleted,
		TargetID:                               id,
		TargetIDInData:                         targetIDInData,
		DataUpdateTime:                         updateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	_, _ = repos.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(
		ctx, repos.LatestDataRepositoryAddresses[id])
}

// doBaseRequest adds tags and texts for the given targetID.
// Mirrors: KFTLRequest.do_request() in TS (the tag/text portion).
func (b *KFTLRequestBase) doBaseRequest(ctx context.Context, targetID string) error {
	relatedTime := b.GetRelatedTime()
	now := b.CreateTime

	// Add tags
	for _, tag := range b.Tags {
		tagObj := reps.Tag{
			ID:           sqlite3impl.GenerateNewID(),
			TargetID:     targetID,
			Tag:          tag,
			RelatedTime:  relatedTime,
			CreateTime:   now,
			CreateApp:    b.Ctx.ApplicationName,
			CreateDevice: b.Ctx.Device,
			CreateUser:   b.Ctx.UserID,
			UpdateTime:   now,
			UpdateApp:    b.Ctx.ApplicationName,
			UpdateDevice: b.Ctx.Device,
			UpdateUser:   b.Ctx.UserID,
		}
		err := b.Ctx.Repositories.WriteTagRep.AddTagInfo(ctx, tagObj)
		if err != nil {
			return fmt.Errorf("error at add tag info target_id=%s tag=%s: %w", targetID, tag, err)
		}
		repName, _ := b.Ctx.Repositories.WriteTagRep.GetRepName(ctx)
		updateLatestDataRepositoryAddress(ctx, b.Ctx.Repositories, tagObj.ID, &targetID, false, now, repName)
		// キャッシュに書き込み
		if len(b.Ctx.Repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			_ = b.Ctx.Repositories.TagReps[0].AddTagInfo(ctx, tagObj)
		}
	}

	// Add texts
	for textID, textContent := range b.TextsMap {
		if textContent == "" {
			continue
		}
		textObj := reps.Text{
			ID:           textID,
			TargetID:     targetID,
			Text:         textContent,
			RelatedTime:  relatedTime,
			CreateTime:   now,
			CreateApp:    b.Ctx.ApplicationName,
			CreateDevice: b.Ctx.Device,
			CreateUser:   b.Ctx.UserID,
			UpdateTime:   now,
			UpdateApp:    b.Ctx.ApplicationName,
			UpdateDevice: b.Ctx.Device,
			UpdateUser:   b.Ctx.UserID,
		}
		err := b.Ctx.Repositories.WriteTextRep.AddTextInfo(ctx, textObj)
		if err != nil {
			return fmt.Errorf("error at add text info target_id=%s text_id=%s: %w", targetID, textID, err)
		}
		repName, _ := b.Ctx.Repositories.WriteTextRep.GetRepName(ctx)
		updateLatestDataRepositoryAddress(ctx, b.Ctx.Repositories, textObj.ID, &targetID, false, now, repName)
		// キャッシュに書き込み
		if len(b.Ctx.Repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			_ = b.Ctx.Repositories.TextReps[0].AddTextInfo(ctx, textObj)
		}
	}

	return nil
}
