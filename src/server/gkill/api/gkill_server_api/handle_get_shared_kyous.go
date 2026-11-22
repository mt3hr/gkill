package gkill_server_api

// 所有者を本文の user_id から取らない理由:
// documents/adr/0702-share-owner-from-session.md

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetSharedKyous は、共有IDに紐づく共有設定の検索条件を実行して、
// 共有ページに表示するKyouと付随データをまとめて返します。
//
// POST /api/get_shared_kyous（wrapNoAuth）
// req_res.GetSharedKyousRequest / req_res.GetSharedKyousResponse
//
// 認証はSessionIDではなくSharedIDで行います。共有設定に記録された
// user_id/deviceのリポジトリを、保存された検索条件 (FindQueryJSON) ごと再現して引くので、
// 共有IDを知っていれば誰でも同じ結果を取得できます。
// Kyou一覧は最新版のみ (OnlyLatestData = true) を引き、型別データ・付随データは
// そのIDに対して版を絞らずに引きます。
// 共有設定のViewTypeが "mi" のときはMi以外の型別データを引きません。
// IsShareWithLocations / IsShareWithTags / IsShareWithTexts / IsShareWithTimeIss が
// 偽の項目は、それぞれ空のまま返します。
func (g *GkillServerAPI) HandleGetSharedKyous(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSharedKyousRequest{}
	response := &req_res.GetSharedKyousResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close request body", "error", fmt.Sprintf("%q", err))
		}
	}()
	defer func() {
		writeErrorStatus(r.Context(), w, response.Errors)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get shared kyous response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse get shared kyous response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiSharedTasksResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get shared kyous request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse get shared kyous request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTasksRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.SharedID)
	if err != nil || sharedKyouInfo == nil {
		err = fmt.Errorf("error at get ShareKyouListInfos shared id = %s: %w", request.SharedID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at get ShareKyouListInfos shared id", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiSharedTasksError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := sharedKyouInfo.UserID
	device := sharedKyouInfo.Device

	findQuery := &find.FindQuery{}
	err = json.Unmarshal([]byte(sharedKyouInfo.FindQueryJSON), findQuery)
	if err != nil {
		err = fmt.Errorf("error at parse query json at find kyous %#v: %w", findQuery, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse query json at find kyous %#v", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTaskRequest,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	bundle, gkillErr := g.collectKyouBundle(r.Context(), &kyouBundleOptions{
		UserID:           userID,
		Device:           device,
		LocaleName:       request.LocaleName,
		FindQuery:        findQuery,
		ViewType:         sharedKyouInfo.ViewType,
		IncludeLocations: sharedKyouInfo.IsShareWithLocations,
		IncludeTags:      sharedKyouInfo.IsShareWithTags,
		IncludeTexts:     sharedKyouInfo.IsShareWithTexts,
		IncludeTimeIss:   sharedKyouInfo.IsShareWithTimeIss,
	})
	if gkillErr != nil {
		response.Errors = append(response.Errors, gkillErr)
		return
	}

	response.Kyous = bundle.Kyous
	response.Mis = bundle.Mis
	response.Kmemos = bundle.Kmemos
	response.KCs = bundle.KCs
	response.TimeIss = bundle.TimeIss
	response.Nlogs = bundle.Nlogs
	response.Lantanas = bundle.Lantanas
	response.URLogs = bundle.URLogs
	response.IDFKyous = bundle.IDFKyous
	response.ReKyous = bundle.ReKyous
	response.GitCommitLogs = bundle.GitCommitLogs
	response.GPSLogs = bundle.GPSLogs
	response.AttachedTags = bundle.AttachedTags
	response.AttachedTexts = bundle.AttachedTexts
	response.AttachedTimeIss = bundle.AttachedTimeIss
	response.AttachedTimeIsKyous = bundle.AttachedTimeIsKyous
	response.Title = sharedKyouInfo.ShareTitle
	response.ViewType = sharedKyouInfo.ViewType
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSharedTasksSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOU_MESSAGE"}),
	})
}
