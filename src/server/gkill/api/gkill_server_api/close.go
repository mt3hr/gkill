package gkill_server_api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strings"
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
				slog.Log(context.Background(), gkill_log.Warn, "error at shutdown http server", "error", fmt.Sprintf("%q", err))
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
		slog.Log(ctx, gkill_log.Debug, "Error getting device information", "error", fmt.Sprintf("%q", err))
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "Error getting server configuration", "error", fmt.Sprintf("%q", err))
		return
	}

	port := gkill_options.ServerAddressPortSuffix(serverConfig.Address)
	protocol := "http"
	if serverConfig.EnableTLS && !gkill_options.DisableTLSForce {
		protocol = "https"
	}

	os.Stdout.WriteString("gkill server started.\n")
	// この行は Android の MainActivity が標準出力から拾ってWebViewのURLを決めている。
	// 文言を変えたり順番を入れ替えたりしないこと
	os.Stdout.WriteString(fmt.Sprintf("Access your record space at : %s://localhost%s\n", protocol, port))

	// --address は設定DBを書き換えない実行時オーバーライドなので、
	// 実際にbindするアドレスは ResolveServerAddress で解決する
	g.printInsecureBindWarning(gkill_options.ResolveServerAddress(serverConfig.Address), protocol)
	g.printInitialSetupURLs(ctx, protocol, port)
}

// printInsecureBindWarning はTLS無効のまま外部から届くアドレスで待ち受けているときに警告する。
// 起動メッセージが localhost しか出さないため、実際には全インターフェースで
// 待ち受けていることに気づきにくい。
func (g *GkillServerAPI) printInsecureBindWarning(address string, protocol string) {
	if protocol == "https" {
		return
	}
	if isLoopbackOnlyBindAddress(address) {
		return
	}
	os.Stdout.WriteString("----------------------------------------------------------------\n")
	os.Stdout.WriteString(fmt.Sprintf("警告: TLSが無効な状態で %s で待ち受けています。\n", address))
	os.Stdout.WriteString("      パスワードとセッションIDが平文でネットワークを流れます。\n")
	os.Stdout.WriteString("      外部に公開する場合はTLSを有効にするか、\n")
	os.Stdout.WriteString("      --address 127.0.0.1:<port> でループバックに限定してください。\n")
	os.Stdout.WriteString("----------------------------------------------------------------\n")
}

// printInitialSetupURLs はパスワードが未設定のアカウントについて、
// パスワード設定用のURLを標準出力に出す。
//
// リセットトークンはアカウントを丸ごと取れてしまう秘密なので、
// ネットワーク越し (307リダイレクト) には同一マシンからのアクセスにしか渡さない。
// ループバック以外から初回セットアップする場合は、ここに出るURLを使うことになる。
func (g *GkillServerAPI) printInitialSetupURLs(ctx context.Context, protocol string, port string) {
	accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at get all accounts for initial setup message", "error", fmt.Sprintf("%q", err))
		return
	}

	now := time.Now()
	printedHeader := false
	for _, targetAccount := range accounts {
		if targetAccount.PasswordHash != nil || targetAccount.PasswordResetToken == nil {
			continue
		}
		if !printedHeader {
			os.Stdout.WriteString("----------------------------------------------------------------\n")
			os.Stdout.WriteString("パスワードが未設定のアカウントがあります。\n")
			os.Stdout.WriteString("下記のURLからパスワードを設定してください。\n")
			os.Stdout.WriteString("期限が切れた場合は `gkill_server reset_password <user_id>` で再発行できます。\n")
			printedHeader = true
		}
		// 期限切れのトークンでURLを出しても開けないので、そのことがわかるようにする
		if targetAccount.PasswordResetTokenExpiration != nil && now.After(*targetAccount.PasswordResetTokenExpiration) {
			fmt.Fprintf(os.Stdout, "  %s : リセットトークンの期限切れ (`gkill_server reset_password %s` で再発行)\n",
				targetAccount.UserID, targetAccount.UserID)
			continue
		}
		fmt.Fprintf(os.Stdout, "  %s : %s://localhost%s/set_new_password?user_id=%s&reset_token=%s\n",
			targetAccount.UserID, protocol, port,
			url.QueryEscape(targetAccount.UserID), url.QueryEscape(*targetAccount.PasswordResetToken))
	}
	if printedHeader {
		os.Stdout.WriteString("----------------------------------------------------------------\n")
	}
}

// isLoopbackOnlyBindAddress は待ち受けアドレスがループバックに限定されているかを返す。
// ホスト部分が空 (":9999" など) の場合は全インターフェースなのでfalse。
func isLoopbackOnlyBindAddress(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
