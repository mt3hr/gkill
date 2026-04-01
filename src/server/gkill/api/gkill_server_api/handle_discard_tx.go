package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleDiscardTX(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DiscardTxRequest{}
	response := &req_res.DiscardTxResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse discart tx response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidDiscardTxResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISCARD_TRANSACTION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse discard tx request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidDiscardTxRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISCARD_TRANSACTION_MESSAGE"}),
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

	err = repositories.TempReps.IDFKyouTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete idfKyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteIDFKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.KCTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete kc by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteKCError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.KmemoTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete kmemo by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteKmemoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.LantanaTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete lantana by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteLantanaError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.MiTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete mi by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteMiError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.NlogTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete nlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteNlogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.NotificationTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete notification by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteNotificationError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.ReKyouTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete rekyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteReKyouError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.TagTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete tag by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTagError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.TextTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete text by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTextError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.TimeIsTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete timeis by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTimeIsError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.URLogTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete urlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteURLogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
}
