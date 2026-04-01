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
	"github.com/mt3hr/gkill/src/server/gkill/dao/share_kyou_info"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleUpdateShareKyouListInfo(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateShareKyouListInfoRequest{}
	response := &req_res.UpdateShareKyouListInfoResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add ShareKyouListInfo response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateShareKyouListInfoResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add ShareKyouListInfo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateShareKyouListInfoRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	// 対象が存在しない
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo == nil {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotExistShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	shareKyouInfo := &share_kyou_info.ShareKyouInfo{
		ID:                   GenerateNewID(),
		ShareID:              request.ShareKyouListInfo.ShareID,
		UserID:               request.ShareKyouListInfo.UserID,
		Device:               request.ShareKyouListInfo.Device,
		ShareTitle:           request.ShareKyouListInfo.ShareTitle,
		FindQueryJSON:        request.ShareKyouListInfo.FindQueryJSON,
		ViewType:             request.ShareKyouListInfo.ViewType,
		IsShareTimeOnly:      request.ShareKyouListInfo.IsShareTimeOnly,
		IsShareWithTags:      request.ShareKyouListInfo.IsShareWithTags,
		IsShareWithTexts:     request.ShareKyouListInfo.IsShareWithTexts,
		IsShareWithTimeIss:   request.ShareKyouListInfo.IsShareWithTimeIss,
		IsShareWithLocations: request.ShareKyouListInfo.IsShareWithLocations,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.UpdateKyouShareInfo(r.Context(), shareKyouInfo)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add ShareKyouListInfo user id = %s device = %s ShareKyouListInfo = %#v: %w", userID, device, request.ShareKyouListInfo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfo = ShareKyouListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateShareKyouListInfoSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_SHARE_KYOU_LIST_INFO_UPDATED_GET_MESSAGE"}),
	})
}
