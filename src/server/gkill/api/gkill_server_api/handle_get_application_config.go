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
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleGetApplicationConfig(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetApplicationConfigRequest{}
	response := &req_res.GetApplicationConfigResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get applicationConfig response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetApplicationConfigResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get applicationConfig request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetApplicationConfigRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil || applicationConfig == nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		err = fmt.Errorf("try create application config user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)

		defaultApplicationConfig := user_config.GetDefaultApplicationConfig(userID, device)
		_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(r.Context(), defaultApplicationConfig)

		if err != nil {
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		applicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil || applicationConfig == nil {
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if session == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sessions, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSessions(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get login sessions session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	for _, session := range sessions {
		if session.ApplicationName == "urlog_bookmarklet" {
			applicationConfig.URLogBookmarkletSession = session.SessionID
			break
		}
	}

	privateIP, err := privateIPv4s()
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}
	globalIP, err := globalIP(context.Background())
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}
	privateIPStr := ""
	if len(privateIP) != 0 {
		privateIPStr = privateIP[0].String()
	}

	version, err := api.GetVersion()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	applicationConfig.AccountIsAdmin = auth.Account.IsAdmin
	applicationConfig.SessionIsLocal = session.IsLocalAppUser
	response.ApplicationConfig = applicationConfig

	response.ApplicationConfig.UserID = userID
	response.ApplicationConfig.Device = device
	response.ApplicationConfig.UserIsAdmin = auth.Account.IsAdmin
	response.ApplicationConfig.CacheClearCountLimit = gkill_options.CacheClearCountLimit
	response.ApplicationConfig.GlobalIP = globalIP.String()
	response.ApplicationConfig.PrivateIP = privateIPStr

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}
	if serverConfig != nil {
		response.ApplicationConfig.LanHostname = serverConfig.LanHostname
		response.ApplicationConfig.GlobalHostname = serverConfig.GlobalHostname
	}
	response.ApplicationConfig.Version = version.Version
	response.ApplicationConfig.BuildTime = version.BuildTime
	response.ApplicationConfig.CommitHash = version.CommitHash

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetApplicationConfigSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_APPLICATION_CONFIG_MESSAGE"}),
	})
}
