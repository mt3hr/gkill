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

func (g *GkillServerAPI) HandleDeleteShareKyouListInfos(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DeleteShareKyouListInfoRequest{}
	response := &req_res.DeleteShareKyouListInfosResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete ShareKyouListInfos response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidDeleteShareKyouListInfosResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete ShareKyouListInfos request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDeleteShareKyouListInfosRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.DeleteKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete ShareKyouListInfos user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteShareKyouListInfosError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.DeleteShareKyouListInfosSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
	})
}
