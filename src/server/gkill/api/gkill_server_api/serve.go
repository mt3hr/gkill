package gkill_server_api

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// securityHeadersMiddleware は全レスポンスに最小限の防御的セキュリティヘッダを付ける。
// clickjacking・MIMEスニッフィング・リファラ漏れへの defense-in-depth。
// CSP はSPA・Google Maps・プラグインiframe・PWA の許可設計が要るので、ここでは付けず
// 別途 report-only から段階導入する（利用者ファイル配信の CSP sandbox は
// withUserContentSecurityHeaders が個別に付けているので二重にはしない）。
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		if h.Get("X-Content-Type-Options") == "" {
			h.Set("X-Content-Type-Options", "nosniff")
		}
		if h.Get("X-Frame-Options") == "" {
			h.Set("X-Frame-Options", "SAMEORIGIN")
		}
		if h.Get("Referrer-Policy") == "" {
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		}
		next.ServeHTTP(w, r)
	})
}

func (g *GkillServerAPI) Serve(ctx context.Context) error {
	var err error
	router := g.GkillDAOManager.GetRouter()
	// 登録順の先頭が最外層になる(gorilla/mux)。
	// recoverMiddleware を最外層と最内層の両方に置いてあるのは、gzipMiddleware の
	// defer gzipWriter.Close() が panic の巻き戻しで先に走って暗黙200を確定させ、
	// 外側 recover の 500 が捨てられるのを防ぐため(理由の全文は recover_middleware.go)。
	router.Use(g.recoverMiddleware)
	router.Use(g.accessLogMiddleware)
	router.Use(securityHeadersMiddleware)
	router.Use(gzipMiddleware())
	router.Use(g.recoverMiddleware)
	// --- PathPrefix routes (wrapNoAuth) ---
	// 利用者のファイルをそのまま返す2経路には、下流(サムネイル・動画・ZIP展開物)まで
	// まとめて効くようルート側でセキュリティヘッダを付ける。
	router.PathPrefix("/files/").HandlerFunc(withUserContentSecurityHeaders(g.wrapNoAuth(g.HandleFileServe)))
	router.PathPrefix("/zip_cache/").HandlerFunc(withUserContentSecurityHeaders(g.wrapNoAuth(g.HandleZipCacheFileServe)))

	// --- /api/* ---
	// ルートの正本は gkill_server_api_address.go の apiRoutes() の表1つ。
	// ここへ HandleFunc を直に足さないこと（テストハーネスと本番で登録がずれる。ADR-0709）。
	if err := g.registerAPIRoutes(router); err != nil {
		return err
	}

	manualPage, err := fs.Sub(api.EmbedFS, "embed/manual")
	if err != nil {
		return err
	}
	router.PathPrefix("/resources/manual/").Handler(http.StripPrefix("/resources/manual/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			http.FileServer(http.FS(manualPage)).ServeHTTP(w, r)
		})))

	gkillPage, err := fs.Sub(api.EmbedFS, "embed/html")
	if err != nil {
		return err
	}
	router.PathPrefix(serviceWorkerJSPath).Handler(http.StripPrefix("",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/rykv").Handler(http.StripPrefix("/rykv",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/kftl").Handler(http.StripPrefix("/kftl",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/mi").Handler(http.StripPrefix("/mi",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/kyou").Handler(http.StripPrefix("/kyou",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/dashboard").Handler(http.StripPrefix("/dashboard",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	// ポート（開発コード rudbeckia）。SPAのパスはここに書かないと直リロードで404になる
	router.PathPrefix("/rudbeckia").Handler(http.StripPrefix("/rudbeckia",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/saihate").Handler(http.StripPrefix("/saihate",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/playing").Handler(http.StripPrefix("/playing",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/mkfl").Handler(http.StripPrefix("/mkfl",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/shared_page").Handler(http.StripPrefix("/shared_page",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/shared_mi").Handler(http.StripPrefix("/shared_mi",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/shared_rykv").Handler(http.StripPrefix("/shared_rykv",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/set_new_password").Handler(http.StripPrefix("/set_new_password",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))

	// 初回アカウント登録画面。ifRedirectResetAdminAccountIsNotFound を通さないこと
	// （通すとこの画面自身へのリダイレクトが無限ループする）。
	// /regist_first_account は旧パス。ブックマークや古い資料からの流入のために残してあり、
	// SPA 側（vue-router）が /register_first_account へリダイレクトする。
	router.PathPrefix("/register_first_account").Handler(http.StripPrefix("/register_first_account",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/regist_first_account").Handler(http.StripPrefix("/regist_first_account",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok := g.filterLocalOnly(w, r); !ok {
				return
			}
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.Path("/").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
			return
		}
		http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
	}))
	router.PathPrefix("/").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		if g.ifRedirectResetAdminAccountIsNotFound(w, r) {
			return
		}
		http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
	}))

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		return err
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		slog.Log(ctx, gkill_log.Error, "error at get server config device", "error", fmt.Sprintf("%q", err))
		return err
	}
	port := gkill_options.ResolveServerAddress(serverConfig.Address)

	g.PrintStartedMessage()
	serveCtx, serveCancel := context.WithCancel(ctx)
	defer serveCancel()
	g.server = &http.Server{
		Addr:    port,
		Handler: router,
		BaseContext: func(_ net.Listener) context.Context {
			return serveCtx
		},
		// slowloris・巨大ヘッダ対策。ReadTimeout/WriteTimeout は大容量アップロードや
		// 30万件応答・大ファイル配信を途中で切らないよう敢えて 0（無制限）のままにし、
		// ヘッダ読み込みとアイドル接続にだけ上限を設ける。
		ReadHeaderTimeout: 20 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}

	go func() {
		<-serveCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		g.server.Shutdown(shutdownCtx)
	}()

	if serverConfig.EnableTLS && !gkill_options.DisableTLSForce {
		certFileName, pemFileName, err := g.getTLSFileNames(device)
		if err != nil {
			slog.Log(ctx, gkill_log.Error, "error at tls file names", "error", fmt.Sprintf("%q", err))
			return err
		}
		certFileName, pemFileName = os.ExpandEnv(certFileName), os.ExpandEnv(pemFileName)
		certFileName, pemFileName = filepath.ToSlash(certFileName), filepath.ToSlash(pemFileName)
		err = g.server.ListenAndServeTLS(certFileName, pemFileName)
		return err
	} else {
		err = g.server.ListenAndServe()
		return err
	}
}
