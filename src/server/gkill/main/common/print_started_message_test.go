package common

// PrintStartedMessage が標準出力へ出す起動行の回帰テスト。
//
// "Access your record space at : <URL>" は Android の MainActivity（SERVER_URL_LINE_PREFIX）が
// 同梱 gkill_server の標準出力から拾い、WebView で開く URL にしている
// （documents/reverse/cross-boundary-map.md §11。受け付けるのは http / https でホストがループバック
// （localhost / 127.0.0.1 / ::1）の URL だけ）。
// プロトコルが実際の待ち受け（serve.go: EnableTLS && !DisableTLSForce のとき ListenAndServeTLS）と
// 食い違っても Go 側にはエラーが出ず、Android だけが繋がらない画面になる。
// ここでは os.Stdout を差し替えて行そのものを固定する。

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_server_api"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// startedMessageLinePrefix は MainActivity.kt の SERVER_URL_LINE_PREFIX と同じ字面（末尾の空白まで）。
const startedMessageLinePrefix = "Access your record space at : "

// startedMessageServerConfigDAO は有効な端末の設定を1件だけ返すスタブ。
// 埋め込みが nil なので、呼ばない他のメソッドを実装する必要がない（get_device_cache_test.go と同じ型）。
type startedMessageServerConfigDAO struct {
	server_config.ServerConfigDAO
	config *server_config.ServerConfig
}

func (s *startedMessageServerConfigDAO) GetAllServerConfigs(context.Context) ([]*server_config.ServerConfig, error) {
	return []*server_config.ServerConfig{
		{Device: "other-device", EnableThisDevice: false},
		s.config,
	}, nil
}

func (s *startedMessageServerConfigDAO) GetServerConfig(_ context.Context, device string) (*server_config.ServerConfig, error) {
	if device != s.config.Device {
		return nil, fmt.Errorf("unexpected device %q", device)
	}
	return s.config, nil
}

// startedMessageAccountDAO はアカウント無し（初回セットアップ URL の段落を出さない）を返すスタブ。
type startedMessageAccountDAO struct {
	account.AccountDAO
}

func (startedMessageAccountDAO) GetAllAccounts(context.Context) ([]*account.Account, error) {
	return nil, nil
}

// os.Stdout の差し替えは add_tag_test.go の captureStdout を使う。

// startedMessageLines は出力を行に分け、Android と同じく末尾の CR は落として返す。
func startedMessageLines(out string) []string {
	lines := strings.Split(out, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

func TestPrintStartedMessagePrintsServerURLLineForAndroid(t *testing.T) {
	originalDisableTLS := gkill_options.DisableTLSForce
	originalServerAddress := gkill_options.ServerAddress
	originalLogger := slog.Default()
	t.Cleanup(func() {
		gkill_options.DisableTLSForce = originalDisableTLS
		gkill_options.ServerAddress = originalServerAddress
		slog.SetDefault(originalLogger)
	})
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil))) // 起動ログの1行は標準エラーへ出さない

	cases := []struct {
		name            string
		enableTLS       bool
		disableTLSForce bool
		address         string
		addressOverride string
		wantURL         string
	}{
		{
			name:    "TLS 無効なら http",
			address: ":19999",
			wantURL: "http://localhost:19999",
		},
		{
			name:      "TLS 有効なら https（ホスト部は落としてポートだけ使う）",
			enableTLS: true,
			address:   "127.0.0.1:18443",
			wantURL:   "https://localhost:18443",
		},
		{
			name:            "設定で TLS 有効でも --disable_tls なら実際の待ち受けと同じ http",
			enableTLS:       true,
			disableTLSForce: true,
			address:         ":18443",
			wantURL:         "http://localhost:18443",
		},
		{
			name:            "--address の上書きは設定 DB ではなく実際に bind するポートを出す",
			address:         ":19999",
			addressOverride: "127.0.0.1:53271",
			wantURL:         "http://localhost:53271",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gkill_options.DisableTLSForce = c.disableTLSForce
			gkill_options.ServerAddress = c.addressOverride

			g := &gkill_server_api.GkillServerAPI{
				GkillDAOManager: &dao.GkillDAOManager{
					ConfigDAOs: &dao.ConfigDAOs{
						ServerConfigDAO: &startedMessageServerConfigDAO{config: &server_config.ServerConfig{
							Device:           "this-device",
							EnableThisDevice: true,
							Address:          c.address,
							EnableTLS:        c.enableTLS,
						}},
						AccountDAO: startedMessageAccountDAO{},
					},
				},
			}

			out := captureStdout(t, g.PrintStartedMessage)
			lines := startedMessageLines(out)

			var urlLines []string
			for _, line := range lines {
				if strings.HasPrefix(line, startedMessageLinePrefix) {
					urlLines = append(urlLines, line)
				}
			}
			if len(urlLines) != 1 {
				t.Fatalf("起動行（%q で始まる行）が %d 行ある。1行のはず。出力:\n%s", startedMessageLinePrefix, len(urlLines), out)
			}
			if got, want := urlLines[0], startedMessageLinePrefix+c.wantURL; got != want {
				t.Fatalf("起動行 = %q, want %q", got, want)
			}

			// Android の parseServerUrlLine が受け付ける範囲（http / https・ループバックのホスト）に入ること。
			// Go は常に localhost とポートで組むので、Android より厳しい「localhost:<ポート>」で確かめる。
			parsed, err := url.Parse(strings.TrimPrefix(urlLines[0], startedMessageLinePrefix))
			if err != nil {
				t.Fatalf("起動行の URL が読めない: %v", err)
			}
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				t.Fatalf("scheme = %q, want http か https", parsed.Scheme)
			}
			if parsed.Hostname() != "localhost" || parsed.Port() == "" {
				t.Fatalf("host = %q, want localhost:<port>", parsed.Host)
			}

			if !strings.Contains(out, "gkill server started.\n") {
				t.Fatalf("起動したことを知らせる行が無い。出力:\n%s", out)
			}
		})
	}
}
