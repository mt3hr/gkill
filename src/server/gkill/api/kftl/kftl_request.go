package kftl

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
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
	// GetCreatedRecords は DoRequest が実際に書いたものを返す。
	// 実装は KFTLRequestBase に1本だけあり、全具象がそれを埋め込んでいる。
	GetCreatedRecords() []KFTLCreatedRecord
}

// KFTLRequestBase is the base struct embedded by all concrete request types.
// KFTLCreatedRecord は KFTL が実際に書いた1件。
//
// KFTL は1つのテキストから複数のKyouを作るのに、応答は「記録しました」の1文だけで、
// 件数も種別もIDも返していなかった（2026-08-24 の再監査）。IDそのものは
// リクエストIDと同じ値で最初から手元にあったが、**本文が空の kmemo / Mi / Nlog は
// 何も書かずに成功し、打刻の終了は既存レコードの更新**なので、リクエストを
// 事前に並べるだけでは「作られたもの」にならない。書いた側が控える。
type KFTLCreatedRecord struct {
	ID       string
	DataType string
	// Updated は新規作成ではなく既存レコードの更新であることを表す（打刻の終了）。
	Updated bool
}

// Mirrors: src/classes/kftl/kftl-request.ts
type KFTLRequestBase struct {
	RequestID     string
	Tags          []string
	TextsMap      map[string]string // textID → accumulated text content
	CurrentTextID *string
	relatedTime   *time.Time // nil means use addSecond offset
	Ctx           *KFTLStatementLineContext
	CreateTime    time.Time

	// created は DoRequest が実際に書いたもの。recordCreated / recordUpdated だけが積む。
	created []KFTLCreatedRecord
}

// recordCreated は新規作成した1件を控える。**書き込みが成功した直後にだけ呼ぶこと。**
func (b *KFTLRequestBase) recordCreated(dataType, id string) {
	b.created = append(b.created, KFTLCreatedRecord{ID: id, DataType: dataType})
}

// recordUpdated は既存レコードを更新した1件を控える（打刻の終了）。
func (b *KFTLRequestBase) recordUpdated(dataType, id string) {
	b.created = append(b.created, KFTLCreatedRecord{ID: id, DataType: dataType, Updated: true})
}

// GetCreatedRecords は DoRequest が実際に書いたものを返す。
func (b *KFTLRequestBase) GetCreatedRecords() []KFTLCreatedRecord { return b.created }

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

// logWriteThroughCacheFailure logs a failed write-through to the cached rep.
//
// A missed write-through is repaired by the next UpdateCache (1m by default),
// so the save itself is not failed here. Discarding it with `_ =` however leaves
// no trace at all, which makes "the tag I just added is invisible for a minute"
// impossible to diagnose afterwards.
// The 25 write-through calls in usecase/*.go wrap + slog the same way.
func logWriteThroughCacheFailure(ctx context.Context, dataType string, id string, err error) {
	if err == nil {
		return
	}
	err = fmt.Errorf("error at write through %s cache id = %s: %w", dataType, id, err)
	// The record is saved but the cache is not, so it stays invisible for up to a minute.
	// Error, because nothing else reports it: the response is a success.
	slog.Log(ctx, gkill_log.Error, "error at write through cache", "data_type", fmt.Sprintf("%q", dataType), "error", fmt.Sprintf("%q", err))
}

// logGetRepNameFailure logs a failed rep name lookup.
//
// Bailing out here would stop the KFTL submission midway even though the record
// itself is already saved (commit_tx is not a DB transaction, so nothing rolls back).
// Keep going and just leave a trace.
func logGetRepNameFailure(ctx context.Context, dataType string, id string, err error) {
	if err == nil {
		return
	}
	err = fmt.Errorf("error at get rep name for %s id = %s: %w", dataType, id, err)
	slog.Log(ctx, gkill_log.Warn, "error at get rep name", "data_type", fmt.Sprintf("%q", dataType), "error", fmt.Sprintf("%q", err))
}

// updateLatestDataRepositoryAddress updates the in-memory cache and DAO for one entity.
// Mirrors the pattern used in gkill_server_api.go L2070-2082.
func updateLatestDataRepositoryAddress(ctx context.Context, repos *reps.GkillRepositories,
	id string, targetIDInData *string, isDeleted bool, updateTime time.Time, repName string) {
	latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              isDeleted,
		TargetID:                               id,
		TargetIDInData:                         targetIDInData,
		DataUpdateTime:                         updateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	repos.SetLatestDataRepositoryAddress(id, latestDataRepositoryAddress)
	if _, err := repos.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(
		ctx, latestDataRepositoryAddress); err != nil {
		err = fmt.Errorf("error at add or update latest data repository address id = %s: %w", id, err)
		slog.Log(ctx, gkill_log.Error, "error at add or update latest data repository address", "error", fmt.Sprintf("%q", err))
	}
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
		repName, repNameErr := b.Ctx.Repositories.WriteTagRep.GetRepName(ctx)
		logGetRepNameFailure(ctx, "tag", tagObj.ID, repNameErr)
		updateLatestDataRepositoryAddress(ctx, b.Ctx.Repositories, tagObj.ID, &targetID, false, now, repName)
		// キャッシュに書き込み
		logWriteThroughCacheFailure(ctx, "tag", tagObj.ID, b.Ctx.Repositories.WriteThroughTagCache(ctx, tagObj))
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
		repName, repNameErr := b.Ctx.Repositories.WriteTextRep.GetRepName(ctx)
		logGetRepNameFailure(ctx, "text", textObj.ID, repNameErr)
		updateLatestDataRepositoryAddress(ctx, b.Ctx.Repositories, textObj.ID, &targetID, false, now, repName)
		// キャッシュに書き込み
		logWriteThroughCacheFailure(ctx, "text", textObj.ID, b.Ctx.Repositories.WriteThroughTextCache(ctx, textObj))
	}

	return nil
}
