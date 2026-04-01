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

func (g *GkillServerAPI) HandleAddNlog(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddNlogRequest{}
	response := &req_res.AddNlogResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add nlog response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddNlogResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add nlog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddNlogRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	gkillErrors, err := g.UsecaseCtx.AddNlog(r.Context(), repositories, userID, device, request.LocaleName, request.Nlog, request.TXID)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddNlogError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if len(gkillErrors) > 0 {
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	if request.WantResponseKyou {
		nlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedNlog = nlog

		kyou, err := repositories.NlogReps.GetKyou(r.Context(), request.Nlog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNlogSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_NLOG_MESSAGE"}),
	})
}
