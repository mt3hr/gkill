// Package common はCLIサブコマンド定義とサーバ初期化の共通基盤。
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
	"slices"
	"strings"
	"time"

	_ "time/tzdata"

	_ "net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_server_api"
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

	gkillServerAPI *gkill_server_api.GkillServerAPI

	IDFCmd = &cobra.Command{
		Use:           "idf",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Usage()
			}

			targetDirs := args

			autoIDF := false
			router := mux.NewRouter()
			idfIgnore := gkill_options.IDFIgnore

			// 存在しない/gkillリポジトリでないディレクトリの skip は失敗ではないが、
			// IDF 処理そのものの失敗（idfKyouRep.IDF / glob の失敗）は握り潰さず、
			// stderr へ出したうえで exit code に反映する（監査 M-16）。
			var errs []error
			for _, filenamePattern := range targetDirs {
				filenamePattern = os.ExpandEnv(filenamePattern)
				matchFiles, err := zglob.Glob(filenamePattern)
				if err != nil {
					err = fmt.Errorf("error at glob %s: %w", filenamePattern, err)
					fmt.Printf("%s\n", err)
					errs = append(errs, err)
					continue
				}
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
					idfKyouRep, err := reps.NewIDFDirRep(context.TODO(), "", filename, idDBFilename, true, router, autoIDF, &idfIgnore, nil)
					if err != nil {
						err = fmt.Errorf("error at new idf dir rep: %w", err)
						slog.Log(cmd.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
						fmt.Printf("skip idf: %s\n", filename)
						continue
					}
					defer func(rep reps.IDFKyouRepository, name string) {
						if err := rep.Close(context.TODO()); err != nil {
							slog.Log(cmd.Context(), gkill_log.Debug, "error at close idf rep", "file", fmt.Sprintf("%q", name), "error", fmt.Sprintf("%q", err))
						}
					}(idfKyouRep, filename)
					if err := idfKyouRep.IDF(context.TODO()); err != nil {
						err = fmt.Errorf("error at idf %s: %w", filename, err)
						slog.Log(cmd.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
						fmt.Printf("%s\n", err)
						errs = append(errs, err)
					}
				}
			}
			// skip（存在しない/gkillリポジトリでないディレクトリ）は失敗ではないが、
			// glob や IDF 処理そのものの失敗があれば非0で返す（黙って成功にしない）。
			return errors.Join(errs...)
		},
		Short: `idf 'target_dir'`,
	}
	DVNFCmd = dvnf_cmd.DVNFCmd

	VersionCommand = &cobra.Command{
		Use:           "version",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := api.GetVersion()
			if err != nil {
				return fmt.Errorf("error at get version: %w", err)
			}
			fmt.Printf("%s:\t%s\n", AppName, version.Version)
			fmt.Printf("%s:\t%s\n", "build_time", version.BuildTime)
			fmt.Printf("%s:\t\t%s\n", "hash", version.CommitHash)
			return nil
		},
	}

	GenerateThumbCacheCmd = &cobra.Command{
		Use:           "generate_thumb_cache",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			gkill_options.LoadIDFRepOnly = true
			if len(args) == 0 {
				return cmd.Usage()
			}

			targetUserIDs := args

			if err := InitGkillServerAPI(); err != nil {
				return fmt.Errorf("error at init gkill server api: %w", err)
			}

			// 1ユーザが失敗しても残りは続け、最後にまとめて返す(1件でも失敗すれば exit 1)。
			var errs []error
			for _, targetUserID := range targetUserIDs {
				if err := GenerateThumbCache(cmd.Context(), targetUserID); err != nil {
					errs = append(errs, fmt.Errorf("error at generate thumb cache user id = %s: %w", targetUserID, err))
				}
			}
			return errors.Join(errs...)
		},
		Short: `generate_thumb_cache 'user_id'`,
	}

	GenerateVideoCacheCmd = &cobra.Command{
		Use:           "generate_video_cache",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			gkill_options.LoadIDFRepOnly = true
			if len(args) == 0 {
				return cmd.Usage()
			}

			targetUserIDs := args

			if err := InitGkillServerAPI(); err != nil {
				return fmt.Errorf("error at init gkill server api: %w", err)
			}

			// 1ユーザが失敗しても残りは続け、最後にまとめて返す(1件でも失敗すれば exit 1)。
			var errs []error
			for _, targetUserID := range targetUserIDs {
				if err := GenerateVideoCache(cmd.Context(), targetUserID); err != nil {
					errs = append(errs, fmt.Errorf("error at generate video cache user id = %s: %w", targetUserID, err))
				}
			}
			return errors.Join(errs...)
		},
		Short: `generate_video_cache 'user_id'`,
	}

	ClearCacheCmd = &cobra.Command{
		Use:           "clear_cache",
		Short:         `clear_cache <thumb|video|zip|plugin|all> <all|user_id...>`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 第1引数: 対象キャッシュ種別, 第2引数以降: 対象ユーザー(または all)。
			// 他のサブコマンド(generate_thumb_cache 等)と同様、対象は必須指定とする。
			if len(args) < 2 {
				return cmd.Usage()
			}

			mode := args[0]
			switch mode {
			case "thumb", "video", "zip", "plugin", "all":
				// ok
			default:
				return cmd.Usage()
			}

			targets := args[1:]

			// all を明示指定した場合: ディスク上の派生キャッシュを全ユーザー分まとめて削除する
			if slices.Contains(targets, "all") {
				cacheRootDir := os.ExpandEnv(gkill_options.CacheDir)
				var errs []error
				clearCacheDir := func(name string) {
					target := filepath.Join(cacheRootDir, name)
					if err := os.RemoveAll(target); err != nil {
						errs = append(errs, fmt.Errorf("error at clear cache %s: %w", target, err))
						return
					}
					fmt.Printf("cleared cache: %s\n", target)
				}
				switch mode {
				case "thumb":
					clearCacheDir("thumb_cache")
				case "video":
					clearCacheDir("video_cache")
				case "zip":
					clearCacheDir("zip_cache")
				case "plugin":
					clearCacheDir(pluginCacheDirName)
				case "all":
					clearCacheDir("thumb_cache")
					clearCacheDir("video_cache")
					clearCacheDir("zip_cache")
					clearCacheDir(pluginCacheDirName)
				}
				return errors.Join(errs...)
			}

			// プラグインのキャッシュはディスク上のディレクトリを消すだけでよく、
			// リポジトリを読み込む必要がない。重い初期化の前に済ませる
			if mode == "plugin" {
				var errs []error
				for _, targetUserID := range targets {
					if err := ClearPluginCache(targetUserID); err != nil {
						errs = append(errs, fmt.Errorf("error at clear plugin cache user id = %s: %w", targetUserID, err))
						continue
					}
					fmt.Printf("cleared cache: user id = %s mode = %s\n", targetUserID, mode)
				}
				return errors.Join(errs...)
			}

			// ユーザー指定: 指定ユーザーのリポジトリ分のキャッシュのみ削除する
			gkill_options.LoadIDFRepOnly = true
			if err := InitGkillServerAPI(); err != nil {
				return fmt.Errorf("error at init gkill server api: %w", err)
			}

			var errs []error
			for _, targetUserID := range targets {
				if err := ClearCache(cmd.Context(), targetUserID, mode); err != nil {
					errs = append(errs, fmt.Errorf("error at clear cache user id = %s: %w", targetUserID, err))
					continue
				}
				fmt.Printf("cleared cache: user id = %s mode = %s\n", targetUserID, mode)
			}
			return errors.Join(errs...)
		},
	}

	OptimizeCmd = &cobra.Command{
		Use:           "optimize",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			gkill_options.Optimize = true
			if len(args) == 0 {
				return cmd.Usage()
			}

			targetUserIDs := args

			if err := InitGkillServerAPI(); err != nil {
				return fmt.Errorf("error at init gkill server api: %w", err)
			}
			gkillServerAPI := GetGkillServerAPI()
			device, err := gkillServerAPI.GetDevice()
			if err != nil {
				return fmt.Errorf("error at get device: %w", err)
			}

			// 1ユーザが失敗しても残りは続け、最後にまとめて返す(1件でも失敗すれば exit 1)。
			var errs []error
			for _, targetUserID := range targetUserIDs {
				if _, err := gkillServerAPI.GkillDAOManager.GetRepositories(targetUserID, device); err != nil {
					errs = append(errs, fmt.Errorf("error at optimize (get repositories) user id = %s: %w", targetUserID, err))
				}
			}
			return errors.Join(errs...)
		},
		Short: `optimize 'user_id'`,
	}

	UpdateCacheCmd = &cobra.Command{
		Use:           "update_cache",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Usage()
			}

			targetUserIDs := args
			ctx := cmd.Context()

			configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
			endpoint, err := ResolveLocalServerEndpoint(ctx)
			if err != nil {
				return fmt.Errorf("error at resolve local server endpoint: %w", err)
			}
			httpClient := endpoint.Client

			// 認証: update_cacheは管理者セッションを要求する。
			// サブコマンドはサーバーと同一マシンで実行される前提なので、ローカルのDBを直接触って
			// 短命の管理者セッションを発行し、それを使う。
			// (以前は保存済みのパスワードハッシュを読んで /api/login に投げ直していたが、
			//  そのやり方は「DBを読めた者がログインできる」ことに依存していた。
			//  Argon2idで保存するようになったのでハッシュからのログインはできないし、
			//  そもそもそこに依存しないほうがよい)
			sessionID, cleanupSession, err := issueLocalAdminSession(ctx, configDBRootDir, endpoint.Device)
			if err != nil {
				return fmt.Errorf("error at issue local admin session: %w", err)
			}
			defer cleanupSession()

			address := endpoint.BaseURL + "/api/update_cache"
			requestBody := &req_res.UpdateCacheRequest{
				SessionID: sessionID,
				UserIDs:   targetUserIDs,
			}
			jsonBody, err := json.Marshal(requestBody)
			if err != nil {
				return fmt.Errorf("error at marshal update cache request: %w", err)
			}
			resp, err := httpClient.Post(address, "application/json", bytes.NewReader(jsonBody))
			if err != nil {
				return fmt.Errorf("error at post update cache request to %s: %w", address, err)
			}
			defer resp.Body.Close()

			response := &req_res.UpdateCacheResponse{}
			err = json.NewDecoder(resp.Body).Decode(response)
			if err != nil {
				return fmt.Errorf("error at decode update cache response: %w", err)
			}

			for _, msg := range response.Messages {
				fmt.Printf("%s\n", msg.Message)
			}

			// HTTPステータス異常、または応答のerrorsに中身があれば失敗として返す(exit 1)。
			// gkillはHTTP 200でもerrorsに中身を入れることがあるので両方を見る。
			var errs []error
			if resp.StatusCode != http.StatusOK {
				errs = append(errs, fmt.Errorf("error at update cache: status = %d", resp.StatusCode))
			}
			for _, errMsg := range response.Errors {
				if errMsg == nil {
					continue
				}
				errs = append(errs, fmt.Errorf("%s: %s", errMsg.ErrorCode, errMsg.ErrorMessage))
			}
			return errors.Join(errs...)
		},
		Short: `update_cache 'user_id...'`,
	}
)

func init() {
	if os.Getenv("HOME") == "" {
		os.Setenv("HOME", os.Getenv("HOMEDRIVE")+os.Getenv("HOMEPATH"))
	}
	fixTimezone()

	// pprofは環境変数で明示的に有効化したときだけ待ち受ける。
	//
	// ハンドラ自体は `_ "net/http/pprof"` の init で http.DefaultServeMux に登録済みだが、
	// gkill は gorilla/mux を使っており DefaultServeMux を配信しないので、
	// この待ち受けが無いと外からは届かない。
	// 未設定なら何も起きないので本番構成への影響は無い。
	//
	//	GKILL_PPROF_ADDR=localhost:6060 gkill_server ...
	if pprofAddr := os.Getenv("GKILL_PPROF_ADDR"); pprofAddr != "" {
		go func() {
			if err := http.ListenAndServe(pprofAddr, nil); err != nil {
				slog.Log(context.Background(), gkill_log.Error, "error at listen pprof", "address", pprofAddr, "error", err)
			}
		}()
	}

	IDFCmd.PersistentFlags().StringArrayVarP(&gkill_options.IDFIgnore, "ignore", "i", gkill_options.IDFIgnore, "ignore files")
}

// LocalServerEndpoint は稼働中のgkill_serverへAPIを投げるための宛先とクライアント。
type LocalServerEndpoint struct {
	// BaseURL は "http://localhost:9999" のようなスキーム込みの宛先。末尾にスラッシュは付かない
	BaseURL string

	// Device はこのサーバのデバイス名。セッション発行に使う
	Device string

	// Client は自己署名証明書を許容するHTTPクライアント
	Client *http.Client
}

// ResolveLocalServerEndpoint は設定DBを読んで、このマシンで動いているサーバの宛先を組み立てる。
//
// サブコマンドがサーバのAPIを叩くときの共通の入口。
// TLSが有効な場合、localhostの自己署名証明書へ繋ぐため証明書検証をスキップする。
func ResolveLocalServerEndpoint(ctx context.Context) (*LocalServerEndpoint, error) {
	configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
	serverConfigDAO, err := server_config.NewServerConfigDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "server_config.db"))
	if err != nil {
		return nil, fmt.Errorf("error at create server config dao: %w", err)
	}
	defer func() {
		if err := serverConfigDAO.Close(ctx); err != nil {
			slog.Log(ctx, gkill_log.Debug, "error at close server config dao", "error", err)
		}
	}()

	serverConfigs, err := serverConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("error at get all server configs: %w", err)
	}

	var currentServerConfig *server_config.ServerConfig
	for _, sc := range serverConfigs {
		if sc.EnableThisDevice {
			currentServerConfig = sc
			break
		}
	}
	if currentServerConfig == nil {
		return nil, fmt.Errorf("error: no enabled device found in server configs")
	}

	scheme := "http"
	if currentServerConfig.EnableTLS && !gkill_options.DisableTLSForce {
		scheme = "https"
	}
	portSuffix := gkill_options.ServerAddressPortSuffix(currentServerConfig.Address)

	httpClient := &http.Client{}
	if scheme == "https" {
		// localhostの自己署名証明書への接続のためTLS検証をスキップする
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
	}

	return &LocalServerEndpoint{
		BaseURL: fmt.Sprintf("%s://localhost%s", scheme, portSuffix),
		Device:  currentServerConfig.Device,
		Client:  httpClient,
	}, nil
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

	gkillServerAPI, err = gkill_server_api.NewGkillServerAPI()
	if err != nil {
		return err
	}
	return nil
}

// PreLoadRepositories は --pre_load_users で指定されたユーザのリポジトリを
// バックグラウンドで先に読み込んでおく。
//
// リポジトリの構築はユーザの規模によっては数分かかる。
// プリロードしていないとその待ち時間が最初のリクエストに丸ごと乗るため、
// 起動直後のアクセスが延々と待たされることになる。
// サーバのリッスン開始を遅らせないようゴルーチンで実行し、
// 失敗しても起動は続行する。
func PreLoadRepositories(ctx context.Context, gkillServerAPI *gkill_server_api.GkillServerAPI) {
	if len(gkill_options.PreLoadUserNames) == 0 || gkillServerAPI == nil {
		return
	}

	device, err := gkillServerAPI.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device for pre load users: %w", err)
		slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
		return
	}

	go func() {
		for _, userID := range gkill_options.PreLoadUserNames {
			if ctx.Err() != nil {
				return
			}
			startTime := time.Now()
			if _, err := gkillServerAPI.GkillDAOManager.GetRepositories(userID, device); err != nil {
				err = fmt.Errorf("error at pre load repositories. user id = %s device = %s: %w", userID, device, err)
				slog.Log(ctx, gkill_log.Error, "error", "error", fmt.Sprintf("%q", err))
				continue
			}
			slog.Log(ctx, gkill_log.Info, "pre loaded repositories", "userID", fmt.Sprintf("%q", userID), "device", fmt.Sprintf("%q", device), "elapsed", time.Since(startTime).String())
		}
	}()
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
		PreLoadRepositories(ctx, gkillServerAPI)
		err = gkillServerAPI.Serve(ctx)
		if err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				if ctx.Err() != nil {
					// SIGINT/SIGTERM → 終了
					return nil
				}
				// HandleUpdateServerConfigs → リスタート（ループ継続）
			} else {
				slog.Log(context.Background(), gkill_log.Error, "error at gkill server api serve", "error", fmt.Sprintf("%q", err))
				return err
			}
		}
		err = InitGkillServerAPI()
		if err != nil {
			return err
		}
	}
}

func GetGkillServerAPI() *gkill_server_api.GkillServerAPI {
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
		err = fmt.Errorf("error at generate video cache: %w", err)
		return err
	}
	return nil
}

// pluginCacheDirName はプラグインが作るキャッシュの置き場所。
// gkill_options.CacheDir 配下にユーザーごとのディレクトリを作る。
// プラグイン側(src/plugins/*/cache_path.go)も同じ場所を見ている。
const pluginCacheDirName = "plugin_cache"

// ClearPluginCache は指定ユーザーのプラグインキャッシュディレクトリを削除する。
// プラグインは次回起動時にキャッシュを作り直す。
func ClearPluginCache(userID string) error {
	if userID == "" || userID == "." || userID == ".." || strings.ContainsAny(userID, `/\`) {
		return fmt.Errorf("invalid user id for plugin cache path: %q", userID)
	}
	// ".." 除去。直前の検証で弾いているため実行時には常にno-opだが、
	// CodeQL の path-injection はこの形しか値サニタイザとして認識しない。
	safeUserID := strings.ReplaceAll(userID, "..", "")
	target := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), pluginCacheDirName, safeUserID)
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("error at clear plugin cache %s: %w", target, err)
	}
	return nil
}

// ClearCache は指定ユーザーの派生キャッシュ(thumb/video/zip/plugin)を削除する。
// mode は thumb|video|zip|plugin|all のいずれか。
func ClearCache(ctx context.Context, userID string, mode string) error {
	if mode == "plugin" {
		return ClearPluginCache(userID)
	}

	gkillServerAPI := GetGkillServerAPI()
	device, err := gkillServerAPI.GetDevice()
	if err != nil {
		return fmt.Errorf("error at get device: %w", err)
	}
	repositories, err := GetGkillServerAPI().GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		return fmt.Errorf("error at get gkill repositories: %w", err)
	}
	switch mode {
	case "thumb":
		return repositories.IDFKyouReps.ClearThumbCache(userID)
	case "video":
		return repositories.IDFKyouReps.ClearVideoCache(userID)
	case "zip":
		return repositories.IDFKyouReps.ClearZipCache(userID)
	case "all":
		return errors.Join(
			repositories.IDFKyouReps.ClearThumbCache(userID),
			repositories.IDFKyouReps.ClearVideoCache(userID),
			repositories.IDFKyouReps.ClearZipCache(userID),
			ClearPluginCache(userID),
		)
	}
	return nil
}
