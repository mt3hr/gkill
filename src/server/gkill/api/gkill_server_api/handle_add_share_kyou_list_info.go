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

// HandleAddShareKyouListInfo は検索条件を共有リンクとして公開する設定を1件追加します。
//
// POST /api/add_share_kyou_list_info（wrapAuthRepos）
// req_res.AddShareKyouListInfoRequest / req_res.AddShareKyouListInfoResponse
//
// 保存先はリポジトリではなく設定DB（ShareKyouInfoDAO）なので、wrapAuthReposで得た
// repositoriesはここでは使わない。
// 共有の所有者UserID/Deviceはセッションから決め、リクエスト本文の値は使わない。
// リクエスト側を信じると他人のuser_idを指定した共有を作れてしまい、認証不要の
// /api/get_shared_kyousでその人のライフログを読み出せてしまうため。
// IDはサーバ側で採番する。ShareIDが既に使われている場合は上書きせず、
// AlreadyExistShareKyouListInfoErrorをerrorsに積んで返す。
// 成功時は設定DBから読み直したものをShareKyouListInfoに入れる。
func (g *GkillServerAPI) HandleAddShareKyouListInfo(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddShareKyouListInfoRequest{}
	response := &req_res.AddShareKyouListInfoResponse{}

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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddShareKyouListInfoResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add ShareKyouListInfo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddShareKyouListInfoRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	// 対象が存在する場合はエラー
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo != nil {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AlreadyExistShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 共有の所有者はリクエスト本文ではなくセッションから決める。
	// ここでrequest側の値を信じると、他人のuser_idを指定した共有を作って
	// 認証不要の /api/get_shared_kyous でその人のライフログを読み出せてしまう。
	shareKyouInfo := &share_kyou_info.ShareKyouInfo{
		ID:                   GenerateNewID(),
		ShareID:              request.ShareKyouListInfo.ShareID,
		UserID:               userID,
		Device:               device,
		ShareTitle:           request.ShareKyouListInfo.ShareTitle,
		FindQueryJSON:        request.ShareKyouListInfo.FindQueryJSON,
		ViewType:             request.ShareKyouListInfo.ViewType,
		IsShareTimeOnly:      request.ShareKyouListInfo.IsShareTimeOnly,
		IsShareWithTags:      request.ShareKyouListInfo.IsShareWithTags,
		IsShareWithTexts:     request.ShareKyouListInfo.IsShareWithTexts,
		IsShareWithTimeIss:   request.ShareKyouListInfo.IsShareWithTimeIss,
		IsShareWithLocations: request.ShareKyouListInfo.IsShareWithLocations,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.AddKyouShareInfo(r.Context(), shareKyouInfo)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add ShareKyouListInfo user id = %s device = %s ShareKyouListInfo = %#v: %w", userID, device, request.ShareKyouListInfo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfo = ShareKyouListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddShareKyouListInfoSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_ADDED_GET_MESSAGE"}),
	})
}
