package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "time/tzdata"

	_ "net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/hide_files"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	dvnf_cmd "github.com/mt3hr/gkill/src/server/gkill/dvnf/cmd"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
)

var (
	AppName = "gkill_server"

	gkillServerAPI *api.GkillServerAPI

	IDFCmd = &cobra.Command{
		Use: "idf",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Usage()
				return
			}

			targetDirs := args

			autoIDF := false
			router := mux.NewRouter()
			idfIgnore := gkill_options.IDFIgnore

			for _, filenamePattern := range targetDirs {
				filenamePattern = os.ExpandEnv(filenamePattern)
				matchFiles, _ := zglob.Glob(filenamePattern)
				for _, filename := range matchFiles {
					if _, err := os.Stat(filename); os.IsNotExist(err) {
						fmt.Printf("Directory not found. skip idf: %s\n", filename)
						continue
					}
					parentDir := filepath.Join(filename, ".gkill")
					err := os.MkdirAll(os.ExpandEnv(parentDir), os.ModePerm)
					if err != nil {
						err = fmt.Errorf("error at make directory %s: %w", parentDir, err)
						fmt.Printf("%s\n", err)
						fmt.Printf("skip idf: %s\n", filename)
						continue
					}
					hide_files.HideFolder(parentDir)
					idDBFilename := filepath.Join(parentDir, "gkill_id.db")
					idfKyouRep, err := reps.NewIDFDirRep(context.TODO(), filename, idDBFilename, true, router, autoIDF, &idfIgnore, nil)
					if err != nil {
						err = fmt.Errorf("error at new idf dir rep: %w", err)
						slog.Log(cmd.Context(), gkill_log.Debug, "error", "error", err)
						fmt.Printf("skip idf: %s\n", filename)
						continue
					}
					defer idfKyouRep.Close(context.TODO())
					idfKyouRep.IDF(context.TODO())
				}
			}
		},
		Short: `idf 'target_dir'`,
	}
	DVNFCmd = dvnf_cmd.DVNFCmd

	VersionCommand = &cobra.Command{
		Use: "version",
		Run: func(cmd *cobra.Command, args []string) {
			version, err := api.GetVersion()
			if err != nil {
				println(err.Error())
				return
			}
			fmt.Printf("%s:\t%s\n", AppName, version.Version)
			fmt.Printf("%s:\t%s\n", "build_time", version.BuildTime)
			fmt.Printf("%s:\t\t%s\n", "hash", version.CommitHash)
		},
	}

	GenerateThumbCacheCmd = &cobra.Command{
		Use: "generate_thumb_cache",
		Run: func(cmd *cobra.Command, args []string) {
			gkill_options.LoadIDFRepOnly = true
			if len(args) == 0 {
				cmd.Usage()
				return
			}

			targetUserIDs := args

			err := InitGkillServerAPI()
			if err != nil {
				slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
			}

			for _, targetUserID := range targetUserIDs {
				err := GenerateThumbCache(cmd.Context(), targetUserID)
				if err != nil {
					err = fmt.Errorf("error at generate thumb cache user id = %s: %w", targetUserID, err)
					slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
				}
			}
		},
		Short: `generate_thumb_cache 'user_id'`,
	}

	GenerateVideoCacheCmd = &cobra.Command{
		Use: "generate_video_cache",
		Run: func(cmd *cobra.Command, args []string) {
			gkill_options.LoadIDFRepOnly = true
			if len(args) == 0 {
				cmd.Usage()
				return
			}

			targetUserIDs := args

			err := InitGkillServerAPI()
			if err != nil {
				slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
			}

			for _, targetUserID := range targetUserIDs {
				err := GenerateVideoCache(cmd.Context(), targetUserID)
				if err != nil {
					err = fmt.Errorf("error at generate video cache user id = %s: %w", targetUserID, err)
					slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
				}
			}
		},
		Short: `generate_video_cache 'user_id'`,
	}

	OptimizeCmd = &cobra.Command{
		Use: "optimize",
		Run: func(cmd *cobra.Command, args []string) {
			gkill_options.Optimize = true
			if len(args) == 0 {
				cmd.Usage()
				return
			}

			targetUserIDs := args

			err := InitGkillServerAPI()
			if err != nil {
				slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
			}
			gkillServerAPI := GetGkillServerAPI()
			device, err := gkillServerAPI.GetDevice()
			if err != nil {
				slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
			}

			for _, targetUserID := range targetUserIDs {
				_, err = gkillServerAPI.GkillDAOManager.GetRepositories(targetUserID, device)
				if err != nil {
					slog.Log(cmd.Context(), gkill_log.Error, "error", "error", err)
				}
			}
		},
		Short: `optimize 'user_id'`,
	}

	UpdateCacheCmd = &cobra.Command{
		Use: "update_cache",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Usage()
				return
			}

			targetUserIDs := args
			ctx := cmd.Context()

			configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
			serverConfigDAO, err := server_config.NewServerConfigDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "server_config.db"))
			if err != nil {
				err = fmt.Errorf("error at create server config dao: %w", err)
				slog.Log(ctx, gkill_log.Error, "error", "error", err)
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			defer serverConfigDAO.Close(ctx)

			serverConfigs, err := serverConfigDAO.GetAllServerConfigs(ctx)
			if err != nil {
				err = fmt.Errorf("error at get all server configs: %w", err)
				slog.Log(ctx, gkill_log.Error, "error", "error", err)
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}

			var currentServerConfig *server_config.ServerConfig
			for _, sc := range serverConfigs {
				if sc.EnableThisDevice {
					currentServerConfig = sc
					break
				}
			}
			if currentServerConfig == nil {
				err = fmt.Errorf("error: no enabled device found in server configs")
				slog.Log(ctx, gkill_log.Error, "error", "error", err)
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}

			scheme := "http"
			if currentServerConfig.EnableTLS && !gkill_options.DisableTLSForce {
				scheme = "https"
			}
			address := fmt.Sprintf("%s://localhost%s/api/update_cache", scheme, currentServerConfig.Address)

			requestBody := &req_res.UpdateCacheRequest{
				UserIDs: targetUserIDs,
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				err = fmt.Errorf("error at marshal update cache request: %w", err)
				slog.Log(ctx, gkill_log.Error, "error", "error", err)
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}

			httpClient := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
			resp, err := httpClient.Post(address, "application/json", bytes.NewReader(jsonBody))
			if err != nil {
				err = fmt.Errorf("error at post update cache request to %s: %w", address, err)
				slog.Log(ctx, gkill_log.Error, "error", "error", err)
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}
			defer resp.Body.Close()

			response := &req_res.UpdateCacheResponse{}
			err = json.NewDecoder(resp.Body).Decode(response)
			if err != nil {
				err = fmt.Errorf("error at decode update cache response: %w", err)
				slog.Log(ctx, gkill_log.Error, "error", "error", err)
				fmt.Fprintf(os.Stderr, "%s\n", err)
				return
			}

			for _, msg := range response.Messages {
				fmt.Printf("%s\n", msg.Message)
			}
			for _, errMsg := range response.Errors {
				fmt.Fprintf(os.Stderr, "%s: %s\n", errMsg.ErrorCode, errMsg.ErrorMessage)
			}
		},
		Short: `update_cache 'user_id'`,
	}
)

func init() {
	if os.Getenv("HOME") == "" {
		os.Setenv("HOME", os.Getenv("HOMEDRIVE")+os.Getenv("HOMEPATH"))
	}
	fixTimezone()

	/*
		go func() {
			http.ListenAndServe("localhost:6060", nil) // pprof用
		}()
	*/

	IDFCmd.PersistentFlags().StringArrayVarP(&gkill_options.IDFIgnore, "ignore", "i", gkill_options.IDFIgnore, "ignore files")
}

func InitGkillOptions() {
	os.Setenv("GKILL_HOME", filepath.Clean(os.ExpandEnv(gkill_options.GkillHomeDir)))
	gkill_options.LibDir = fmt.Sprintf("%s/lib/base_directory", gkill_options.GkillHomeDir)
	gkill_options.CacheDir = fmt.Sprintf("%s/caches", gkill_options.GkillHomeDir)
	gkill_options.LogDir = fmt.Sprintf("%s/logs", gkill_options.GkillHomeDir)
	gkill_options.ConfigDir = fmt.Sprintf("%s/configs", gkill_options.GkillHomeDir)
	gkill_options.TLSCertFileDefault = fmt.Sprintf("%s/tls/cert.cer", gkill_options.GkillHomeDir)
	gkill_options.TLSKeyFileDefault = fmt.Sprintf("%s/tls/key.pem", gkill_options.GkillHomeDir)
	gkill_options.DataDirectoryDefault = fmt.Sprintf("%s/datas", gkill_options.GkillHomeDir)
}

func fixTimezone() {
	if runtime.GOOS == "android" {
		out, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
		if err != nil {
			return
		}
		z, err := time.LoadLocation(strings.TrimSpace(string(out)))
		if err != nil {
			return
		}
		time.Local = z
	}
}

func InitGkillServerAPI() error {
	var err error

	gkillServerAPI, err = api.NewGkillServerAPI()
	if err != nil {
		return err
	}
	return nil
}

func LaunchGkillServerAPI(ctx context.Context) error {
	defer func() {
		err := gkillServerAPI.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	var err error
	for {
		err = gkillServerAPI.Serve(ctx)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				if ctx.Err() != nil {
					// SIGINT/SIGTERM → 終了
					return nil
				}
				// HandleUpdateServerConfigs → リスタート（ループ継続）
			} else {
				slog.Log(context.Background(), gkill_log.Error, "error at gkill server api serve", "error", err)
				return err
			}
		}
		err = InitGkillServerAPI()
		if err != nil {
			return err
		}
	}
}

func GetGkillServerAPI() *api.GkillServerAPI {
	return gkillServerAPI
}

func Openbrowser(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func GenerateThumbCache(ctx context.Context, userID string) error {
	gkillServerAPI := GetGkillServerAPI()
	device, err := gkillServerAPI.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device: %w", err)
		return err
	}
	repositories, err := GetGkillServerAPI().GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get gkill repositories: %w", err)
		return err
	}
	err = repositories.IDFKyouReps.GenerateThumbCache(ctx)
	if err != nil {
		err = fmt.Errorf("error at generate thumb cache: %w", err)
		return err
	}
	return nil
}

func GenerateVideoCache(ctx context.Context, userID string) error {
	gkillServerAPI := GetGkillServerAPI()
	device, err := gkillServerAPI.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device: %w", err)
		return err
	}
	repositories, err := GetGkillServerAPI().GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get gkill repositories: %w", err)
		return err
	}
	err = repositories.IDFKyouReps.GenerateVideoCache(ctx)
	if err != nil {
		err = fmt.Errorf("error at generate thumb cache: %w", err)
		return err
	}
	return nil
}
