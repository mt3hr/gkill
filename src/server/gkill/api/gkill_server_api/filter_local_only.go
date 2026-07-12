package gkill_server_api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

func (g *GkillServerAPI) filterLocalOnly(w http.ResponseWriter, r *http.Request) bool {
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		/*
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
			    ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		*/
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		/*
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		*/
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	if serverConfig == nil {
		err = fmt.Errorf("error at server config is nil device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	if !serverConfig.IsLocalOnlyAccess {
		return true
	}

	if isLocalRequest(r) {
		return true
	}
	w.WriteHeader(http.StatusForbidden)
	return false
}

// isLocalRequest はリクエスト元が同一マシンかどうかを返す。
// ファイル実パスなど、同一マシン上のクライアントにしか意味がなく、
// かつ外部に漏らしたくない情報の公開可否判定に使う。
func isLocalRequest(r *http.Request) bool {
	spl := strings.Split(r.RemoteAddr, ":")
	remoteHost := strings.Join(spl[:len(spl)-1], ":")
	switch remoteHost {
	case "localhost":
		fallthrough
	case "127.0.0.1":
		fallthrough
	case "[::1]":
		fallthrough
	case "::1":
		return true
	}
	return false
}
