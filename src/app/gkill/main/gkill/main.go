package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asticode/go-astikit"
	"github.com/asticode/go-astilectron"
	"github.com/mt3hr/gkill/src/app/gkill/main/common"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

func main() {
	if err := serverCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	cobra.MousetrapHelpText = "" // Windowsでマウスから起動しても怒られないようにする
	serverCmd.PersistentFlags().StringVar(&gkill_options.GkillHomeDir, "gkill_home_dir", gkill_options.GkillHomeDir, "")
	serverCmd.PersistentFlags().BoolVar(&gkill_options.IsOutputLog, "log", gkill_options.IsOutputLog, "")
	serverCmd.PersistentFlags().BoolVar(&gkill_options.DisableTLSForce, "disable_tls", gkill_options.DisableTLSForce, "")
	serverCmd.PersistentFlags().BoolVar(&gkill_options.IsCacheInMemory, "cache_in_memory", gkill_options.IsCacheInMemory, "")
	serverCmd.AddCommand(common.IDFCmd)
	serverCmd.AddCommand(common.DVNFCmd)
}

var (
	serverCmd = &cobra.Command{
		Use: "gkill",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			common.InitGkillOptions()
		},
		Run: func(_ *cobra.Command, _ []string) {
			var err error

			err = common.InitGkillServerAPI()
			if err != nil {
				log.Fatal(err)
			}
			go common.LaunchGkillServerAPI()

			for ; ; time.Sleep(time.Microsecond * 500) {
				api := common.GetGkillServerAPI()
				if api.GkillDAOManager == nil {
					continue
				}
				if api.GkillDAOManager.ConfigDAOs == nil {
					continue
				}
				if api.GkillDAOManager.ConfigDAOs.ServerConfigDAO == nil {
					continue
				}
				if serverConfigs, err := api.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.TODO()); len(serverConfigs) == 0 || err != nil {
					continue
				}
				break
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
			if serverConfig.EnableTLS {
				address += "https://localhost"
			} else {
				address += "http://localhost"
			}
			address += serverConfig.Address

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
				AppIconDefaultPath: "C:/Users/yamat/Git/gkill/public/favicon.png",
				AppIconDarwinPath:  "C:/Users/yamat/Git/gkill/public/favicon.ico",
				BaseDirectoryPath:  baseDirectoryPath,
				DataDirectoryPath:  dataDirectoryPath,
				SkipSetup:          false,
			})
			if err != nil {
				gkill_log.Info.Println("Electronが動かない環境であるかもしれません。その場合gkillは動きませんので変わりにgkill_serverを起動し、ブラウザからのアクセスを試みてください。")
				log.Fatal(err)
			}
			defer a.Close()

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
			w.OnMessage(func(m *astilectron.EventMessage) interface{} {
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
			os.Exit(0)
		},
	}
)
