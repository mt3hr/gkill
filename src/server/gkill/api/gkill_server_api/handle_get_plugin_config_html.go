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

// HandleGetPluginConfigHTML はプラグインの設定画面HTMLを取得します。
//
// POST /api/get_plugin_config_html（wrapAuth）
// req_res.GetPluginConfigHTMLRequest / req_res.GetPluginConfigHTMLResponse
//
// 管理者でなければ弾きます（自分自身を対象にする場合も同じ）。設定画面には取り込み元の
// フォルダ指定が出るので、一般利用者には開かせません。
// target_user_idが空なら自分自身を対象にします。
// 設定画面の中身はプラグインが自分で組み立てます。gkill側は内容を解釈せずそのまま返します。
// RepNameに一致するプラグインが無い場合はエラーを返します。
func (g *GkillServerAPI) HandleGetPluginConfigHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetPluginConfigHTMLRequest{}
	response := &req_res.GetPluginConfigHTMLResponse{}

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
			err = fmt.Errorf("error at encode get plugin config html response: %w", err)
			slog.Log(r.Context(), gkill_log.Error, "error at encode get plugin config html response", "error", fmt.Sprintf("%q", err))
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode get plugin config html request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at decode get plugin config html request", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetPluginConfigHTMLRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_PLUGIN_CONFIG_HTML_MESSAGE"}),
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
			ErrorCode:    message.GetPluginConfigHTMLError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "PLUGIN_NOT_FOUND_MESSAGE"}),
		})
		return
	}

	html, err := pluginRepo.GetConfigHTML(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get plugin config html %s: %w", request.RepName, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at get plugin config html", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.GetPluginConfigHTMLError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_PLUGIN_CONFIG_HTML_MESSAGE"}),
			Cause:        err,
		})
		return
	}

	response.HTML = html
}
