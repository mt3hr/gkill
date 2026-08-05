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

// HandleGetPluginList はログイン中の利用者が使えるプラグインの一覧を返します。
//
// POST /api/get_plugin_list（wrapAuth）
// req_res.GetPluginListRequest / req_res.GetPluginListResponse
//
// 各要素はプラグインのmanifestの内容に、プロセスが生きているか（IsAlive）を添えたものです。
// プラグインが1つも無い場合はPluginsをnilのまま正常応答します。
func (g *GkillServerAPI) HandleGetPluginList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetPluginListRequest{}
	response := &req_res.GetPluginListResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at encode get plugin list response: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at decode get plugin list request: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.InvalidGetPluginListRequestDataError,
			ErrorMessage: "プラグイン一覧取得リクエストのパースに失敗しました",
		})
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID

	pm := g.GkillDAOManager.GetPluginManager(userID)
	for _, pluginRepo := range pm.GetPluginRepositories() {
		manifest := pluginRepo.GetManifest()
		info := req_res.PluginInfo{
			Name:        manifest.Name,
			Version:     manifest.Version,
			Description: manifest.Description,
			DataType:    manifest.DataType,
			RepName:     manifest.RepName,
			IsAlive:     pluginRepo.IsAlive(r.Context()),
		}
		response.Plugins = append(response.Plugins, info)
	}
}
