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

func (g *GkillServerAPI) HandleGetPluginContentHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetPluginContentHTMLRequest{}
	response := &req_res.GetPluginContentHTMLResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at encode get plugin content html response: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode get plugin content html request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetPluginContentHTMLRequestDataError,
			ErrorMessage: "プラグインコンテンツHTML取得リクエストのパースに失敗しました",
		})
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID

	pm := g.GkillDAOManager.GetPluginManager(userID)
	pluginRepo := pm.GetPluginByRepName(request.RepName)
	if pluginRepo == nil {
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.GetPluginContentHTMLError,
			ErrorMessage: fmt.Sprintf("プラグインが見つかりません: %s", request.RepName),
		})
		return
	}

	html, err := pluginRepo.GetContentHTML(r.Context(), request.KyouID)
	if err != nil {
		err = fmt.Errorf("error at get plugin content html %s %s: %w", request.RepName, request.KyouID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.GetPluginContentHTMLError,
			ErrorMessage: fmt.Sprintf("プラグインコンテンツHTMLの取得に失敗しました: %s", err.Error()),
		})
		return
	}

	response.HTML = html
}
