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
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleGetGkillNotificationPublicKey はこの端末の通知用公開鍵（VAPID）を返します。
//
// POST /api/get_gkill_notification_public_key（wrapAuth）
// req_res.GetGkillNotificationPublicKeyRequest / req_res.GetGkillNotificationPublicKeyResponse
//
// 鍵はサーバ設定に端末ごとに保存されています。
// この端末のサーバ設定が見つからない場合はエラーを返します。
func (g *GkillServerAPI) HandleGetGkillNotificationPublicKey(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGkillNotificationPublicKeyRequest{}
	response := &req_res.GetGkillNotificationPublicKeyResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi task notification public key response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiTaskNotificationPublicKeyResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGISTER_MI_TASK_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get get mi task notification public key request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiTaskNotificationPublicKeyRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGISTER_MI_TASK_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGISTER_MI_TASK_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.Device == device {
			currentServerConfig = serverConfig
			break
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GkillNotificationPublicKey = currentServerConfig.GkillNotificationPublicKey
}
