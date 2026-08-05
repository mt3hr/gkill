package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// HandleGetPluginConfigHTML はプラグインの設定画面HTMLを取得します。
//
// POST /api/get_plugin_config_html（wrapAuth）
// req_res.GetPluginConfigHTMLRequest / req_res.GetPluginConfigHTMLResponse
//
// 設定画面の中身はプラグインが自分で組み立てます。gkill側は内容を解釈せずそのまま返します。
// RepNameに一致するプラグインが無い場合はエラーを返します。
func (g *GkillServerAPI) HandleGetPluginConfigHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetPluginConfigHTMLRequest{}
	response := &req_res.GetPluginConfigHTMLResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at encode get plugin config html response: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode get plugin config html request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetPluginConfigHTMLRequestDataError,
			ErrorMessage: "プラグイン設定HTML取得リクエストのパースに失敗しました",
		})
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID

	pm := g.GkillDAOManager.GetPluginManager(userID)
	pluginRepo := pm.GetPluginByRepName(request.RepName)
	if pluginRepo == nil {
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.GetPluginConfigHTMLError,
			ErrorMessage: fmt.Sprintf("プラグインが見つかりません: %s", request.RepName),
		})
		return
	}

	html, err := pluginRepo.GetConfigHTML(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get plugin config html %s: %w", request.RepName, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.GetPluginConfigHTMLError,
			ErrorMessage: fmt.Sprintf("プラグイン設定HTMLの取得に失敗しました: %s", err.Error()),
		})
		return
	}

	response.HTML = html
}
