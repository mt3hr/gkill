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

func (g *GkillServerAPI) HandlePostPluginConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.PostPluginConfigRequest{}
	response := &req_res.PostPluginConfigResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at encode post plugin config response: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode post plugin config request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidPostPluginConfigRequestDataError,
			ErrorMessage: "プラグイン設定保存リクエストのパースに失敗しました",
		})
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID

	pm := g.GkillDAOManager.GetPluginManager(userID)
	pluginRepo := pm.GetPluginByRepName(request.RepName)
	if pluginRepo == nil {
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.PostPluginConfigError,
			ErrorMessage: fmt.Sprintf("プラグインが見つかりません: %s", request.RepName),
		})
		return
	}

	if err := pluginRepo.PostConfig(r.Context(), request.FormData); err != nil {
		err = fmt.Errorf("error at post plugin config %s: %w", request.RepName, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.PostPluginConfigError,
			ErrorMessage: fmt.Sprintf("プラグイン設定の保存に失敗しました: %s", err.Error()),
		})
		return
	}
}
