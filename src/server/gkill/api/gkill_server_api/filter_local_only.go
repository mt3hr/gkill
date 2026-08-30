package gkill_server_api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func (g *GkillServerAPI) filterLocalOnly(w http.ResponseWriter, r *http.Request) bool {
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at get device name", "error", fmt.Sprintf("%q", err))
		writeGkillErrorResponse(r.Context(), w, &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: localeUnawareLocalizer().MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return false
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at get serverConfig device", "error", fmt.Sprintf("%q", err))
		writeGkillErrorResponse(r.Context(), w, &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: localeUnawareLocalizer().MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		})
		return false
	}
	if serverConfig == nil {
		err = fmt.Errorf("error at server config is nil device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at server config is nil device", "error", fmt.Sprintf("%q", err))
		writeGkillErrorResponse(r.Context(), w, &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: localeUnawareLocalizer().MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		})
		return false
	}
	if !serverConfig.IsLocalOnlyAccess {
		return true
	}

	if isLocalRequest(r) {
		return true
	}
	writeGkillErrorResponse(r.Context(), w, &message.GkillError{
		ErrorCode:    message.LocalOnlyAccessDeniedError,
		ErrorMessage: localeUnawareLocalizer().MustLocalizeMessage(&i18n.Message{ID: "LOCAL_ONLY_ACCESS_DENIED_MESSAGE"}),
	})
	return false
}

// localeUnawareLocalizer は locale_name を読む前に打ち切る経路のための localizer。
//
// filterLocalOnly はリクエストボディを読むより手前で走るので、利用者が選んだ言語が分からない。
// api.GetLocalizer("") は既定の言語にフォールバックするので、それを使う。
// (認証ミドルウェアはボディから locale_name だけ先読みできるので、あちらは利用者の言語で返せる)
func localeUnawareLocalizer() *i18n.Localizer {
	return api.GetLocalizer("")
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
