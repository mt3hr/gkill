package main

import (
	"log"
	"os"

	"github.com/mt3hr/gkill/src/app/gkill/main/common"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

func main() {
	if err := ServerCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	common.AppName = "gkill_server"
	cobra.MousetrapHelpText = "" // Windowsでマウスから起動しても怒られないようにする
	ServerCmd.PersistentFlags().StringVar(&gkill_options.GkillHomeDir, "gkill_home_dir", gkill_options.GkillHomeDir, "")
	ServerCmd.PersistentFlags().BoolVar(&gkill_options.IsOutputLog, "log", gkill_options.IsOutputLog, "")
	ServerCmd.PersistentFlags().BoolVar(&gkill_options.DisableTLSForce, "disable_tls", gkill_options.DisableTLSForce, "")
	ServerCmd.PersistentFlags().BoolVar(&gkill_options.IsCacheInMemory, "cache_in_memory", gkill_options.IsCacheInMemory, "")
	ServerCmd.PersistentFlags().IntVar(&gkill_options.GoroutinePool, "goroutine_pool", gkill_options.GoroutinePool, "")
	ServerCmd.PersistentFlags().Int64Var(&gkill_options.CacheClearCountLimit, "cache_clear_count_limit", gkill_options.CacheClearCountLimit, "")
	ServerCmd.PersistentFlags().DurationVar(&gkill_options.CacheUpdateDuration, "cache_update_duration", gkill_options.CacheUpdateDuration, "")
	ServerCmd.AddCommand(common.IDFCmd)
	ServerCmd.AddCommand(common.DVNFCmd)
	ServerCmd.AddCommand(common.VersionCommand)
}

var (
	ServerCmd = &cobra.Command{
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
