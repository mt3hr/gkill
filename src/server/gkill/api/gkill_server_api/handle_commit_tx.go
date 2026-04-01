package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleCommitTx(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.CommitTxRequest{}
	response := &req_res.CommitTxResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse commit tx response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidCommitTxResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse commit tx request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidCommitTxRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	txID := request.TXID

	kmemos, err := repositories.TempReps.KmemoTempRep.GetKmemosByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kmemo by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetKmemoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	kcs, err := repositories.TempReps.KCTempRep.GetKCsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kc by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetKCError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyous, err := repositories.TempReps.IDFKyouTempRep.GetIDFKyousByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get idfkyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetIDFKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantanas, err := repositories.TempReps.LantanaTempRep.GetLantanasByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get lantana by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetLantanaError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	mis, err := repositories.TempReps.MiTempRep.GetMisByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get mi by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetMiError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlogs, err := repositories.TempReps.NlogTempRep.GetNlogsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get nlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetNlogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notifications, err := repositories.TempReps.NotificationTempRep.GetNotificationsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notification by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetNotificationError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyous, err := repositories.TempReps.ReKyouTempRep.GetReKyousByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get rekyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetReKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tags, err := repositories.TempReps.TagTempRep.GetTagsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get tag by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTagError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	texts, err := repositories.TempReps.TextTempRep.GetTextsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get text by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTextError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeiss, err := repositories.TempReps.TimeIsTempRep.GetTimeIssByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get timeis by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTimeIsError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlogs, err := repositories.TempReps.URLogTempRep.GetURLogsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get urlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetURLogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	for _, idfKyou := range idfKyous {
		err = repositories.WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), idfKyou)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// defer g.WebPushUpdatedData(r.Context(), userID, device, request.IDFKyou.ID)

		repName, err := repositories.WriteIDFKyouRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_UPDATED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.IDFKyouReps) == 1 && *gkill_options.CacheIDFKyouReps {
			err = repositories.IDFKyouReps[0].AddIDFKyouInfo(r.Context(), idfKyou)
			if err != nil {
				err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
		repositories.LatestDataRepositoryAddresses[idfKyou.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              idfKyou.IsDeleted,
			TargetID:                               idfKyou.ID,
			DataUpdateTime:                         idfKyou.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[idfKyou.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for idfKyou user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}
	for _, kc := range kcs {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), kc)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
			err = repositories.KCReps[0].AddKCInfo(r.Context(), kc)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteKCRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[kc.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              kc.IsDeleted,
			TargetID:                               kc.ID,
			DataUpdateTime:                         kc.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[kc.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, kmemo := range kmemos {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[kmemo.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              kmemo.IsDeleted,
			TargetID:                               kmemo.ID,
			DataUpdateTime:                         kmemo.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[kmemo.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, lantana := range lantanas {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[lantana.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              lantana.IsDeleted,
			TargetID:                               lantana.ID,
			DataUpdateTime:                         lantana.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[lantana.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, mi := range mis {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteMiRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[mi.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              mi.IsDeleted,
			TargetID:                               mi.ID,
			DataUpdateTime:                         mi.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[mi.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, nlog := range nlogs {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[nlog.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              nlog.IsDeleted,
			TargetID:                               nlog.ID,
			DataUpdateTime:                         nlog.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[nlog.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, notification := range notifications {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.NotificationReps) == 1 && *gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNotificationError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[notification.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              notification.IsDeleted,
			TargetID:                               notification.ID,
			TargetIDInData:                         &notification.TargetID,
			DataUpdateTime:                         notification.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[notification.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, rekyou := range rekyous {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), rekyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && *gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), rekyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[rekyou.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              rekyou.IsDeleted,
			TargetID:                               rekyou.ID,
			TargetIDInData:                         &rekyou.TargetID,
			DataUpdateTime:                         rekyou.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[rekyou.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, tag := range tags {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteTagRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[tag.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &tag.TargetID,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[tag.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, text := range texts {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteTextRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTextError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[text.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              text.IsDeleted,
			TargetID:                               text.ID,
			TargetIDInData:                         &text.TargetID,
			DataUpdateTime:                         text.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[text.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, timeis := range timeiss {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), timeis)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), timeis)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[timeis.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              timeis.IsDeleted,
			TargetID:                               timeis.ID,
			DataUpdateTime:                         timeis.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[timeis.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	for _, urlog := range urlogs {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), urlog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[urlog.ID] = gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              urlog.IsDeleted,
			TargetID:                               urlog.ID,
			DataUpdateTime:                         urlog.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}

		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[urlog.ID])
		if err != nil {
			err = fmt.Errorf("error at add or update latest data repository address for urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.CommitTxSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_URLOG_ADDED_GET_MESSAGE"}),
	})
}
