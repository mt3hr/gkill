package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) HandleOpenDirectory(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.OpenDirectoryRequest{}
	response := &req_res.OpenDirectoryResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse open directory response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterOpenDirectoryResponse,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse open directory request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenDirectoryRequest,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	auth := AuthFromContext(r.Context())
	device := auth.Device

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if session == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if !session.IsLocalAppUser {
		err = fmt.Errorf("error at get login session session id = %s", request.SessionID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderNotLocalAccountError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(auth.UserID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories. userid = %s device = %s: %w", auth.UserID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	filename, err := repositories.Reps.GetPath(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get path. id = %s userid = %s device = %s: %w", request.TargetID, auth.UserID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	dirname := filepath.Dir(filename)

	cmd := os.Expand(serverConfig.OpenDirectoryCommand, func(str string) string {
		if str == "filename" {
			return filename
		}
		if str == "dirname" {
			return dirname
		}
		return ""
	})
	spl := strings.Split(cmd, " ")
	cmd, args := spl[0], spl[1:]

	err = exec.Command(cmd, args...).Start()
	if err != nil {
		err = fmt.Errorf("error at open file. device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.OpenDirectorySuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_OPEN_FOLDER_MESSAGE"}),
	})
}
