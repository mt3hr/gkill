package main

import (
	"log"
	"os"

	"github.com/mt3hr/gkill/src/app/gkill/main/common"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

func main() {
	if err := AppCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	common.AppName = "gkill_server"
	cobra.MousetrapHelpText = "" // Windowsでマウスから起動しても怒られないようにする
	AppCmd.PersistentFlags().StringVar(&gkill_options.GkillHomeDir, "gkill_home_dir", gkill_options.GkillHomeDir, "")
	AppCmd.PersistentFlags().BoolVar(&gkill_options.IsOutputLog, "log", gkill_options.IsOutputLog, "")
	AppCmd.PersistentFlags().BoolVar(&gkill_options.DisableTLSForce, "disable_tls", gkill_options.DisableTLSForce, "")
	AppCmd.PersistentFlags().BoolVar(&gkill_options.IsCacheInMemory, "cache_in_memory", gkill_options.IsCacheInMemory, "")
	AppCmd.AddCommand(common.IDFCmd)
	AppCmd.AddCommand(common.DVNFCmd)
	AppCmd.AddCommand(common.VersionCommand)
}

var (
	AppCmd = &cobra.Command{
		Use: "gkill_server",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			common.InitGkillOptions()
		},
		Run: func(_ *cobra.Command, _ []string) {
			var err error

			err = common.InitGkillServerAPI()
			if err != nil {
				log.Fatal(err)
			}

			err = common.LaunchGkillServerAPI()
			if err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		},
	}
)
