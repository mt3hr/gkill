package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// AddTag はタグを追加するユースケース
func (uc *UsecaseContext) AddTag(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, tag reps.Tag, txID *string) (*reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在する場合はエラー
	existTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get tag user id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			Cause:        err,
		})
		return nil, gkillErrors, nil
	}
	if existTag != nil {
		err = fmt.Errorf("exist tag id = %s", tag.ID)
		slog.Log(ctx, gkill_log.Debug, "exist tag id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.AlreadyExistTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	// 対象のKyouが無い場合はエラー
	//
	// **txIDがあるときは検査しない。** KFTL は1つのトランザクションで
	// 「対象のKyouを作る」→「それにタグを付ける」を挿入順に流すので、
	// この時点で対象はまだ一時リポジトリの中にいて確定していない。
	// 無条件に検査すると、メモ帳からのタグ付き投入が全滅する。
	//
	// 検査しないと、存在しないIDへタグを付けても成功が返り、
	// どこにも付いていない宙吊りのタグができる。しかもその名前は
	// get_all_tag_names の語彙に載るので、0件しかヒットしない候補が検索に混ざる。
	// IDの取り違えはAIが最も起こしやすい事故で、そのとき静かに失敗していた。
	if txID == nil {
		targetKyou, err := repositories.GetKyou(ctx, tag.TargetID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kyou user id = %s device = %s target id = %s: %w", userID, device, tag.TargetID, err)
			slog.Log(ctx, gkill_log.Debug, "error at get kyou user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}
		if targetKyou == nil {
			err = fmt.Errorf("not found target kyou id = %s", tag.TargetID)
			slog.Log(ctx, gkill_log.Debug, "not found target kyou id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.NotFoundKyouInfoError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			})
			return nil, gkillErrors, nil
		}
	}

	// 書き込み先 rep が未設定なら nil ポインタ参照で落ちる前に、設定不備として返す（reason: write_rep_missing）。
	if txID == nil && repositories.WriteTagRep == nil {
		gkillErrors = append(gkillErrors, writeRepMissingError(localeName, "FAILED_ADD_TAG_MESSAGE", "tag"))
		return nil, gkillErrors, nil
	}
	if txID == nil {
		err = repositories.WriteTagRep.AddTagInfo(ctx, tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error at add tag user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}
		// キャッシュに書き込み
		err = repositories.WriteThroughTagCache(ctx, tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Error, "error at write through cache", "error", fmt.Sprintf("%q", err))
		}
	} else {
		err = repositories.TempReps.TagTempRep.AddTagInfo(ctx, tag, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error at add tag user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}
	}

	// **tx 中は最新版アドレス表を進めない。** 実体は temp rep にしか無く、表は commit_tx が確定時に書く。
	// ここで進めると、失敗 → discard_tx のあとに表だけが新しい時刻で残り、find_filter.go の
	// 「表より古い版は除外」で**既存の記録が検索から消える**（2026-09-15 まで実際にそうなっていた）。
	if txID == nil {
		repName, err := repositories.WriteTagRep.GetRepName(ctx)
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(ctx, gkill_log.Debug, "error at get rep name user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}
		latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &tag.TargetID,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		repositories.SetLatestDataRepositoryAddress(tag.ID, latestDataRepositoryAddress)

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(ctx, gkill_log.Error, "error at add or update latest data repository address", "error", fmt.Sprintf("%q", err))
		}
	}

	addedTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get tag user id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
			Cause:        err,
		})
		return nil, gkillErrors, nil
	}

	return addedTag, nil, nil
}

// UpdateTag はタグを更新するユースケース
func (uc *UsecaseContext) UpdateTag(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, tag reps.Tag, txID *string) (*reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// 対象が存在しない場合はエラー
	existTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get tag user id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
			Cause:        err,
		})
		return nil, gkillErrors, nil
	}
	if existTag == nil {
		err = fmt.Errorf("not exist tag id = %s", tag.ID)
		slog.Log(ctx, gkill_log.Debug, "not exist tag id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.NotFoundTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		})
		return nil, gkillErrors, nil
	}

	// 書き込み先 rep が未設定なら nil ポインタ参照で落ちる前に、設定不備として返す（reason: write_rep_missing）。
	if txID == nil && repositories.WriteTagRep == nil {
		gkillErrors = append(gkillErrors, writeRepMissingError(localeName, "FAILED_UPDATE_TAG_MESSAGE", "tag"))
		return nil, gkillErrors, nil
	}
	if txID == nil {
		err = repositories.WriteTagRep.AddTagInfo(ctx, tag)
		if err != nil {
			err = fmt.Errorf("error at update tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error at update tag user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}

		// キャッシュに書き込み
		// **クライアントがエコーした rep_name をそのままキャッシュへ入れない。**
		// 実体は WriteTagRep（書き込み先rep）へ足したので、キャッシュ表の REP_NAME もそこに合わせる。
		// 取得元repの名前のまま入れると、端末別にrepを分けている環境で他端末由来の記録を編集したとき、
		// find_filter.go の filterKyousByRepName が「非空で、指定repに無い名前」として落とし、
		// **更新直後だけ一覧から消えて次の UpdateCache（最大1分）で戻る**。
		// 取れなければ空にする ―― 空は filterKyousByRepName が残すので安全側。
		if writeRepName, repNameErr := repositories.WriteTagRep.GetRepName(ctx); repNameErr == nil {
			tag.RepName = writeRepName
		} else {
			tag.RepName = ""
		}
		err = repositories.WriteThroughTagCache(ctx, tag)
		if err != nil {
			err = fmt.Errorf("error at update tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Error, "error at write through cache", "error", fmt.Sprintf("%q", err))
		}
	} else {
		err = repositories.TempReps.TagTempRep.AddTagInfo(ctx, tag, *txID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at update tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(ctx, gkill_log.Debug, "error at update tag user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.UpdateTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}
	}

	// **tx 中は最新版アドレス表を進めない。** 実体は temp rep にしか無く、表は commit_tx が確定時に書く。
	// ここで進めると、失敗 → discard_tx のあとに表だけが新しい時刻で残り、find_filter.go の
	// 「表より古い版は除外」で**既存の記録が検索から消える**（2026-09-15 まで実際にそうなっていた）。
	if txID == nil {
		repName, err := repositories.WriteTagRep.GetRepName(ctx)
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(ctx, gkill_log.Debug, "error at get rep name user id", "error", fmt.Sprintf("%q", err))
			gkillErrors = append(gkillErrors, &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
				Cause:        err,
			})
			return nil, gkillErrors, nil
		}
		latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &tag.TargetID,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		repositories.SetLatestDataRepositoryAddress(tag.ID, latestDataRepositoryAddress)

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress)
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(ctx, gkill_log.Error, "error at add or update latest data repository address", "error", fmt.Sprintf("%q", err))
		}
	}

	updatedTag, err := repositories.GetTag(ctx, tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get tag user id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
			Cause:        err,
		})
		return nil, gkillErrors, nil
	}

	return updatedTag, nil, nil
}

// GetTagsByTargetID はターゲットIDに紐づくタグを取得するユースケース
func (uc *UsecaseContext) GetTagsByTargetID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, targetID string) ([]reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	tags, err := repositories.GetTagsByTargetID(ctx, targetID)
	if err != nil {
		err = fmt.Errorf("error at get tags by target id user id = %s device = %s target id = %s: %w", userID, device, targetID, err)
		slog.Log(ctx, gkill_log.Debug, "error at get tags by target id user id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagsByTargetIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
			Cause:        err,
		})
		return nil, gkillErrors, nil
	}

	return tags, nil, nil
}

// GetTagHistoriesByTagID はタグIDに紐づくタグ履歴を取得するユースケース
func (uc *UsecaseContext) GetTagHistoriesByTagID(ctx context.Context, repositories *reps.GkillRepositories, userID, device, localeName string, id string, updateTime *time.Time, repName *string) ([]reps.Tag, []*message.GkillError, error) {
	var gkillErrors []*message.GkillError

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var tags []reps.Tag
	var err error
	if updateTime != nil {
		var tag *reps.Tag
		tag, err = repositories.GetTag(ctx, id, updateTime)
		if err == nil && tag != nil {
			tags = []reps.Tag{*tag}
		}
	} else {
		tags, err = repositories.TagReps.GetTagHistoriesByRepName(ctx, id, repName)
	}

	if err != nil {
		err = fmt.Errorf("error at get tag histories by tag id user id = %s device = %s target id = %s: %w", userID, device, id, err)
		slog.Log(ctx, gkill_log.Debug, "error at get tag histories by tag id user id", "error", fmt.Sprintf("%q", err))
		gkillErrors = append(gkillErrors, &message.GkillError{
			ErrorCode:    message.GetTagHistoriesByTagIDError,
			ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAG_HISTORIES_MESSAGE"}),
			Cause:        err,
		})
		return nil, gkillErrors, nil
	}

	return tags, nil, nil
}
