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

// HandlePostPluginConfig は設定画面から送られたフォーム値をプラグインに渡し、保存させます。
//
// POST /api/post_plugin_config（wrapAuth）
// req_res.PostPluginConfigRequest / req_res.PostPluginConfigResponse
//
// 管理者でなければ弾きます（自分自身を対象にする場合も同じ）。取り込み元のフォルダ指定を
// 書き換えられるので、一般利用者には触らせません。
// target_user_idが空なら自分自身を対象にします。
// 保存処理の実体はプラグイン側にあり、gkillはFormDataを解釈せずそのまま中継します。
// RepNameに一致するプラグインが無い場合はエラーを返します。
// 成功時はメッセージを何も詰めず、errorsが空の応答を返すだけです。
func (g *GkillServerAPI) HandlePostPluginConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.PostPluginConfigRequest{}
	response := &req_res.PostPluginConfigResponse{}

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
			err = fmt.Errorf("error at encode post plugin config response: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "error at encode post plugin config response", "error", fmt.Sprintf("%q", err))
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode post plugin config request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at decode post plugin config request", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidPostPluginConfigRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_POST_PLUGIN_CONFIG_MESSAGE"}),
			Cause:        err,
		})
		return
	}

	userID, targetUserErrors := g.resolvePluginTargetUser(r.Context(), request.TargetUserID, request.LocaleName, true)
	if len(targetUserErrors) != 0 {
		response.Errors = append(response.Errors, targetUserErrors...)
		return
	}

	pm := g.GkillDAOManager.GetPluginManager(userID)
	pluginRepo := pm.GetPluginByRepName(r.Context(), request.RepName)
	if pluginRepo == nil {
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.PostPluginConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "PLUGIN_NOT_FOUND_MESSAGE"}),
		})
		return
	}

	if err := pluginRepo.PostConfig(r.Context(), request.FormData); err != nil {
		err = fmt.Errorf("error at post plugin config %s: %w", request.RepName, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at post plugin config", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.PostPluginConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_POST_PLUGIN_CONFIG_MESSAGE"}),
			Cause:        err,
		})
		return
	}

	// 設定が変わると本文の作られ方も変わりうるので、キャッシュ済みのHTMLを捨てる
	g.pluginContentHTMLCacheOf().InvalidateUser(userID)
}
