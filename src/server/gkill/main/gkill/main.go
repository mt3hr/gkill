// gkill はデスクトップアプリ版(go-astilectronウィンドウ)のエントリポイント。
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/mt3hr/gkill/src/server/gkill/main/common"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
	"github.com/spf13/cobra"
)

func main() {
	if err := AppCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	common.AppName = "gkill"
	cobra.MousetrapHelpText = "" // Windowsでマウスから起動しても怒られないようにする
	AppCmd.PersistentFlags().StringVar(&gkill_options.GkillHomeDir, "gkill_home_dir", gkill_options.GkillHomeDir, "")
	AppCmd.PersistentFlags().StringVar(&gkill_options.ServerAddress, "address", gkill_options.ServerAddress, "")
	AppCmd.PersistentFlags().BoolVar(&gkill_options.DisableTLSForce, "disable_tls", gkill_options.DisableTLSForce, "")
	AppCmd.PersistentFlags().BoolVar(&gkill_options.IsCacheInMemory, "cache_in_memory", gkill_options.IsCacheInMemory, "")
	AppCmd.PersistentFlags().BoolVar(&gkill_options.CacheRepsLocalStorage, "cache_reps_local", gkill_options.CacheRepsLocalStorage, "")
	AppCmd.PersistentFlags().IntVar(&gkill_options.GoroutinePool, "goroutine_pool", gkill_options.GoroutinePool, "")
	AppCmd.PersistentFlags().Int64Var(&gkill_options.CacheClearCountLimit, "cache_clear_count_limit", gkill_options.CacheClearCountLimit, "")
	AppCmd.PersistentFlags().DurationVar(&gkill_options.CacheUpdateDuration, "cache_update_duration", gkill_options.CacheUpdateDuration, "")
	AppCmd.PersistentFlags().StringArrayVar(&gkill_options.PreLoadUserNames, "pre_load_users", gkill_options.PreLoadUserNames, "")
	AppCmd.PersistentFlags().StringVar(&gkill_log.LogLevelFromCmd, "log", gkill_log.LogLevelFromCmd, "")
	AppCmd.AddCommand(common.DVNFCmd)
	AppCmd.AddCommand(common.VersionCommand)
	AppCmd.AddCommand(common.GenerateThumbCacheCmd)
	AppCmd.AddCommand(common.GenerateVideoCacheCmd)
	AppCmd.AddCommand(common.OptimizeCmd)
	AppCmd.AddCommand(common.UpdateCacheCmd)
	AppCmd.AddCommand(common.ClearCacheCmd)
	AppCmd.AddCommand(common.ResetPasswordCmd)
	AppCmd.AddCommand(common.AutoTagCmd)
}

var (
	AppCmd = &cobra.Command{
		Use: "gkill",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			common.InitGkillOptions()
			threads.Init()
			gkill_log.Init()
			if gkill_options.IsOutputLog {
				gkill_log.SetMinLevel(gkill_log.TraceSQL)
				gkill_log.SetMode(gkill_log.SplitOnly)
				gkill_log.SetStdoutMirror(false)
			}
		},
		Run: func(cmd *cobra.Command, _ []string) {
			var err error

			err = common.InitGkillServerAPI()
			if err != nil {
				log.Fatal(err)
			}

			serverCtx, serverCancel := context.WithCancel(cmd.Context())
			defer serverCancel()

			// --pre_load_users のプリロードは common.PreLoadRepositories が
			// LaunchGkillServerAPI の中で1回だけ走らせる。
			// ここでも同じことをすると起動のたびに2重に走る
			go func() {
				if err := common.LaunchGkillServerAPI(serverCtx); err != nil {
					slog.Log(serverCtx, gkill_log.Error, "error at launch gkill server api", "error", fmt.Sprintf("%q", err))
				}
			}()

			// DAO と server_config の準備を待つ。待っているのは DAO と設定の初期化だけで
			// (リポジトリ構築ではない)、上の goroutine の LaunchGkillServerAPI が失敗すると
			// 永遠に準備できない。以前は上限なしの 500µs busy-wait で、その場合フリーズしていた。
			// 上限を設けて超えたら fatal にする（60秒は初期化には十分。sleep も現実的な間隔に）。
			readyDeadline := time.Now().Add(60 * time.Second)
			for {
				api := common.GetGkillServerAPI()
				ready := api.GkillDAOManager != nil &&
					api.GkillDAOManager.ConfigDAOs != nil &&
					api.GkillDAOManager.ConfigDAOs.ServerConfigDAO != nil
				if ready {
					if serverConfigs, err := api.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.TODO()); len(serverConfigs) != 0 && err == nil {
						break
					}
				}
				if time.Now().After(readyDeadline) {
					log.Fatal("gkill server did not become ready within 60s (LaunchGkillServerAPI may have failed)")
				}
				time.Sleep(100 * time.Millisecond)
			}

			device, err := common.GetGkillServerAPI().GetDevice()
			if err != nil {
				log.Fatal(err)
			}
			serverConfig, err := common.GetGkillServerAPI().GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
			if err != nil {
				log.Fatal(err)
			}

			address := ""
			if serverConfig.EnableTLS && !gkill_options.DisableTLSForce {
				address += "https://localhost"
			} else {
				address += "http://localhost"
			}
			address += gkill_options.ServerAddressPortSuffix(serverConfig.Address)

			// Initialize astilectron
			baseDirectoryPath := filepath.Clean(os.ExpandEnv(gkill_options.LibDir))
			dataDirectoryPath := filepath.Clean(os.ExpandEnv(gkill_options.LibDir))
			directories := []string{
				baseDirectoryPath,
				dataDirectoryPath,
			}
			for _, dir := range directories {
				err := os.MkdirAll(dir, fs.ModePerm)
				if err != nil {
					err = fmt.Errorf("error at create directory %s: %w", dir, err)
					log.Fatal(err)
				}
			}

			a, err := astilectron.New(nil, astilectron.Options{
				AppName:            "gkill",
				VersionAstilectron: "0.51.0",
				VersionElectron:    "22.0.0",
				AppIconDefaultPath: "./public/favicon.png",
				AppIconDarwinPath:  "./public/favicon.ico",
				BaseDirectoryPath:  baseDirectoryPath,
				DataDirectoryPath:  dataDirectoryPath,
				SkipSetup:          false,
			})
			if err != nil {
				fmt.Printf("Electronが動かない環境であるかもしれません。その場合gkillは動きませんので変わりにgkill_serverを起動し、ブラウザからのアクセスを試みてください。")
				log.Fatal(err)
			}
			defer func() { a.Close() }()

			// Start astilectron
			a.Start()

			contextIsolation := false
			// Create a new window
			w, err := a.NewWindow(address, &astilectron.WindowOptions{
				Height: astikit.IntPtr(750),
				Width:  astikit.IntPtr(450),
				WebPreferences: &astilectron.WebPreferences{
					AllowRunningInsecureContent: &contextIsolation,
				},
			})
			if err != nil {
				err = fmt.Errorf("error at new window: %w", err)
				log.Fatal(err)
			}

			openInDefaultBrowserMessagePrefix := "open_in_default_browser:"
			w.OnMessage(func(m *astilectron.EventMessage) any {
				msg := ""
				m.Unmarshal(&msg)

				if strings.HasPrefix(msg, openInDefaultBrowserMessagePrefix) {
					url := strings.TrimSpace(strings.TrimPrefix(msg, openInDefaultBrowserMessagePrefix))
					common.Openbrowser(url)
					return nil
				}
				return nil
			})
			w.Create()
			w.ExecuteJavaScript(`// aタグがクリックされた時にelectronで開かず、デフォルトのブラウザで開く
document.addEventListener('click', (e) => {
  if (e.srcElement.href) {
    e.preventDefault()
	let href = e.srcElement.href
    astilectron.sendMessage('` + openInDefaultBrowserMessagePrefix + ` ' + href)
  }
})
`)

			// Blocking pattern
			a.Wait()
			serverCancel()
			os.Exit(0)
		},
	}
)
