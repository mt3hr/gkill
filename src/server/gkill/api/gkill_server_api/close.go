package gkill_server_api

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

func (g *GkillServerAPI) Close() error {
	g.closeOnce.Do(func() {
		if g.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := g.server.Shutdown(shutdownCtx); err != nil {
				slog.Log(context.Background(), gkill_log.Warn, "error at shutdown http server", "error", err)
			}
		}
		if g.GkillDAOManager != nil {
			if err := g.GkillDAOManager.Close(); err != nil {
				g.closeErr = fmt.Errorf("error at close gkill dao manager: %w", err)
			}
		}
		if g.RebootServerCh != nil {
			close(g.RebootServerCh)
		}
		g.APIAddress = nil
		g.GkillDAOManager = nil
		g.FindFilter = nil
		g.RebootServerCh = nil
	})
	return g.closeErr
}

func (g *GkillServerAPI) ShutdownHTTPServer() {
	if g.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		g.server.Shutdown(shutdownCtx)
	}
}

func (g *GkillServerAPI) PrintStartedMessage() {
	ctx := context.Background()
	device, err := g.GetDevice()
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "Error getting device information", "error", err)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "Error getting server configuration", "error", err)
		return
	}

	port := serverConfig.Address
	protocol := "http"
	if serverConfig.EnableTLS && !gkill_options.DisableTLSForce {
		protocol = "https"
	}

	os.Stdout.WriteString("gkill server started.\n")
	os.Stdout.WriteString(fmt.Sprintf("Access your record space at : %s://localhost%s\n", protocol, port))
}
