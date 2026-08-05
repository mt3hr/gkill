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

// HandleDeleteShareKyouListInfos は共有ページの設定を削除して共有を取り消します。
// 名前は複数形ですが、1回で消せるのはShareKyouListInfo.ShareIDの1件だけです。
//
// POST /api/delete_share_kyou_list_infos（wrapAuthRepos）
// req_res.DeleteShareKyouListInfoRequest / req_res.DeleteShareKyouListInfosResponse
// （リクエスト型だけ単数形）
//
// DAOのDELETEはshare_idだけを条件にするため、共有IDを知っているだけの第三者に取り消されないよう、
// 削除前にここで所有者を確認します。共有が存在しない場合と所有者が別人の場合は、共有IDの存在有無を
// 問い合わせられないよう同じエラーで返します。
// 削除は追記ではなく行そのものの削除なので、Kyouの更新と違って履歴は残りません。
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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

	// 他人の共有を消せないよう所有者を確認する。DAOのDELETEはshare_idだけを条件にするので、
	// ここで弾かないと共有IDを知っているだけの第三者が取り消せてしまう。
	// 存在しない場合と所有者違いは区別せず、共有IDの存在有無が漏れないようにする。
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo == nil || existShareKyouListInfo.UserID != userID {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.NotExistShareKyouListInfoError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.DeleteKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete ShareKyouListInfos user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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
