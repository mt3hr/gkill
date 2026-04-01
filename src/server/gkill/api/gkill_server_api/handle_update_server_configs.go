package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleUpdateServerConfigs(w http.ResponseWriter, r *http.Request) {
	defer func() {
		go g.ShutdownHTTPServer()
	}()
	func() {
		w.Header().Set("Content-Type", "application/json")
		request := &req_res.UpdateServerConfigsRequest{}
		response := &req_res.UpdateServerConfigsResponse{}

		defer func() {
			err := r.Body.Close()
			if err != nil {
				slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
			}
		}()
		defer func() {
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				err = fmt.Errorf("error at parse update server config response to json: %w", err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.InvalidUpdateServerConfigResponseDataError,
					ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}()

		err := json.NewDecoder(r.Body).Decode(request)
		if err != nil {
			err = fmt.Errorf("error at parse update server config request to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateServerConfigRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		auth := AuthFromContext(r.Context())
		userID := auth.UserID
		device := auth.Device

		// adminじゃなかったら弾く
		if !auth.Account.IsAdmin {
			err = fmt.Errorf("%s is not admin", userID)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountNotHasAdminError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "NO_ADMIN_PRIVILEGE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// TLS設定値がTRUEで設定されるとき、証明書ファイルが実在しない場合はエラー
		for _, serverConfig := range request.ServerConfigs {
			if serverConfig.EnableThisDevice {
				if !serverConfig.EnableTLS {
					continue
				}
				_, err := os.Stat(os.ExpandEnv(serverConfig.TLSCertFile))
				if err != nil {
					err = fmt.Errorf("not found tls cert file user id = %s device = %s: %w", userID, device, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.NotFoundTLSCertFileError,
						ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "CERT_FILE_NOT_CREATED_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
				_, err = os.Stat(os.ExpandEnv(serverConfig.TLSKeyFile))
				if err != nil {
					err = fmt.Errorf("not found tls key file user id = %s device = %s: %w", userID, device, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.NotFoundTLSCertFileError,
						ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "CERT_FILE_NOT_CREATED_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
			}
		}

		// mi通知用キーが空のものは登録する
		for _, serverConfig := range request.ServerConfigs {
			if serverConfig.GkillNotificationPrivateKey == "" {
				serverConfig.GkillNotificationPrivateKey, serverConfig.GkillNotificationPublicKey, err = webpush.GenerateVAPIDKeys()
				if err != nil {
					err = fmt.Errorf("error at generate vapid keys: %w", err)

					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.GenerateVAPIDKeysError,
						ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "KEY_GENERATION_ERROR_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
			}
		}

		// ServerConfigを更新する
		ok, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.DeleteWriteServerConfigs(r.Context(), request.ServerConfigs)
		if !ok || err != nil {
			if err != nil {
				err = fmt.Errorf("error at update server config user user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.UpdateServerConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		err = g.GkillDAOManager.Close()
		if err != nil {
			if err != nil {
				err = fmt.Errorf("error at close gkill dao manager: %w", err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.UpdateServerConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.Messages = append(response.Messages, &message.GkillMessage{
			MessageCode: message.UpdateServerConfigSuccessMessage,
			Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_SETTINGS_MESSAGE"}),
		})
		response.Messages = append(response.Messages, &message.GkillMessage{
			MessageCode: message.RebootingMessage,
			Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "JUST_A_MOMENT_MESSAGE"}),
		})
	}()
}
