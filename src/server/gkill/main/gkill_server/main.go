package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mt3hr/gkill/src/server/gkill/main/common"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
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
	ServerCmd.PersistentFlags().BoolVar(&gkill_options.DisableTLSForce, "disable_tls", gkill_options.DisableTLSForce, "")
	ServerCmd.PersistentFlags().BoolVar(&gkill_options.IsCacheInMemory, "cache_in_memory", gkill_options.IsCacheInMemory, "")
	ServerCmd.PersistentFlags().BoolVar(&gkill_options.CacheRepsLocalStorage, "cache_reps_local", gkill_options.CacheRepsLocalStorage, "")
	ServerCmd.PersistentFlags().IntVar(&gkill_options.GoroutinePool, "goroutine_pool", gkill_options.GoroutinePool, "")
	ServerCmd.PersistentFlags().Int64Var(&gkill_options.CacheClearCountLimit, "cache_clear_count_limit", gkill_options.CacheClearCountLimit, "")
	ServerCmd.PersistentFlags().DurationVar(&gkill_options.CacheUpdateDuration, "cache_update_duration", gkill_options.CacheUpdateDuration, "")
	ServerCmd.PersistentFlags().StringArrayVar(&gkill_options.PreLoadUserNames, "pre_load_users", gkill_options.PreLoadUserNames, "")
	ServerCmd.PersistentFlags().StringVar(&gkill_log.LogLevelFromCmd, "log", gkill_log.LogLevelFromCmd, "")
	ServerCmd.AddCommand(common.IDFCmd)
	ServerCmd.AddCommand(common.DVNFCmd)
	ServerCmd.AddCommand(common.VersionCommand)
	ServerCmd.AddCommand(common.GenerateThumbCacheCmd)
	ServerCmd.AddCommand(common.GenerateVideoCacheCmd)
	ServerCmd.AddCommand(common.OptimizeCmd)
	ServerCmd.AddCommand(common.UpdateCacheCmd)
}

var (
	ServerCmd = &cobra.Command{
		Use: "gkill_server",
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

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			go func() {
				for _, preLoadUserNames := range gkill_options.PreLoadUserNames {
					userID := preLoadUserNames
					device, err := common.GetGkillServerAPI().GetDevice()
					if err != nil {
						err = fmt.Errorf("error at get device name: %w", err)
						slog.Log(ctx, gkill_log.Error, "error", "error", err)
						continue
					}
					common.GetGkillServerAPI().GkillDAOManager.GetRepositories(userID, device)
				}
			}()

			err = common.LaunchGkillServerAPI(ctx)
			if err != nil {
				log.Fatal(err)
			}
		},
	}
)
