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

func (g *GkillServerAPI) Serve(ctx context.Context) error {
	var err error
	router := g.GkillDAOManager.GetRouter()
	router.Use(g.accessLogMiddleware)
	// --- PathPrefix routes (wrapNoAuth) ---
	router.PathPrefix("/files/").HandlerFunc(g.wrapNoAuth(g.HandleFileServe))
	router.PathPrefix("/zip_cache/").HandlerFunc(g.wrapNoAuth(g.HandleZipCacheFileServe))

	// --- wrapNoAuth routes (no auth needed) ---
	router.HandleFunc(g.APIAddress.LoginAddress, g.wrapNoAuth(g.HandleLogin)).Methods(g.APIAddress.LoginMethod)
	router.HandleFunc(g.APIAddress.LogoutAddress, g.wrapNoAuth(g.HandleLogout)).Methods(g.APIAddress.LogoutMethod)
	router.HandleFunc(g.APIAddress.ResetPasswordAddress, g.wrapNoAuth(g.HandleResetPassword)).Methods(g.APIAddress.ResetPasswordMethod)
	router.HandleFunc(g.APIAddress.SetNewPasswordAddress, g.wrapNoAuth(g.HandleSetNewPassword)).Methods(g.APIAddress.SetNewPasswordMethod)
	router.HandleFunc(g.APIAddress.GetSharedKyousAddress, g.wrapNoAuth(g.HandleGetSharedKyous)).Methods(g.APIAddress.GetSharedKyousMethod)
	router.HandleFunc(g.APIAddress.UpdateCacheAddress, g.wrapNoAuth(g.HandleUpdateCache)).Methods(g.APIAddress.UpdateCacheMethod)
	router.HandleFunc(g.APIAddress.URLogBookmarkletAddress, g.wrapNoAuth(g.HandleURLogBookmarkletAddress)).Methods(g.APIAddress.URLogBookmarkletMethod)
	router.HandleFunc(g.APIAddress.GetKyousMCPAddress, g.wrapNoAuth(g.HandleGetKyousMCP)).Methods(g.APIAddress.GetKyousMCPMethod)
	router.HandleFunc(g.APIAddress.UploadFilesAddress, g.wrapNoAuth(g.HandleUploadFiles)).Methods(g.APIAddress.UploadFilesMethod)
	router.HandleFunc(g.APIAddress.UploadGPSLogFilesAddress, g.wrapNoAuth(g.HandleUploadGPSLogFiles)).Methods(g.APIAddress.UploadGPSLogFilesMethod)
	router.HandleFunc(g.APIAddress.BrowseZipContentsAddress, g.wrapNoAuth(g.HandleBrowseZipContents)).Methods(g.APIAddress.BrowseZipContentsMethod)

	// --- wrapAuth routes (auth only, no repos) ---
	router.HandleFunc(g.APIAddress.GetApplicationConfigAddress, g.wrapAuth(g.HandleGetApplicationConfig)).Methods(g.APIAddress.GetApplicationConfigMethod)
	router.HandleFunc(g.APIAddress.GetServerConfigsAddress, g.wrapAuth(g.HandleGetServerConfigs)).Methods(g.APIAddress.GetServerConfigsMethod)
	router.HandleFunc(g.APIAddress.UpdateApplicationConfigAddress, g.wrapAuth(g.HandleUpdateApplicationConfig)).Methods(g.APIAddress.UpdateApplicationConfigMethod)
	router.HandleFunc(g.APIAddress.UpdateAccountStatusAddress, g.wrapAuth(g.HandleUpdateAccountStatus)).Methods(g.APIAddress.UpdateAccountStatusMethod)
	router.HandleFunc(g.APIAddress.UpdateUserRepsAddress, g.wrapAuth(g.HandleUpdateUserReps)).Methods(g.APIAddress.UpdateUserRepsMethod)
	router.HandleFunc(g.APIAddress.UpdateServerConfigsAddress, g.wrapAuth(g.HandleUpdateServerConfigs)).Methods(g.APIAddress.UpdateServerConfigsMethod)
	router.HandleFunc(g.APIAddress.AddAccountAddress, g.wrapAuth(g.HandleAddAccount)).Methods(g.APIAddress.AddAccountMethod)
	router.HandleFunc(g.APIAddress.GenerateTLSFileAddress, g.wrapAuth(g.HandleGenerateTLSFile)).Methods(g.APIAddress.GenerateTLSFileMethod)
	router.HandleFunc(g.APIAddress.GetGkillNotificationPublicKeyAddress, g.wrapAuth(g.HandleGetGkillNotificationPublicKey)).Methods(g.APIAddress.GetGkillNotificationPublicKeyMethod)
	router.HandleFunc(g.APIAddress.RegisterGkillNotificationAddress, g.wrapAuth(g.HandleRegisterGkillNotification)).Methods(g.APIAddress.RegisterGkillNotificationMethod)
	router.HandleFunc(g.APIAddress.OpenDirectoryAddress, g.wrapAuth(g.HandleOpenDirectory)).Methods(g.APIAddress.OpenDirectoryMethod)
	router.HandleFunc(g.APIAddress.OpenFileAddress, g.wrapAuth(g.HandleOpenFile)).Methods(g.APIAddress.OpenFileMethod)
	router.HandleFunc(g.APIAddress.ReloadRepositoriesAddress, g.wrapAuth(g.HandleReloadRepositories)).Methods(g.APIAddress.ReloadRepositoriesMethod)
	router.HandleFunc(g.APIAddress.GetUpdatedDatasByTimeAddress, g.wrapAuth(g.HandleGetUpdatedDatasByTime)).Methods(g.APIAddress.GetUpdatedDatasByTimeMethod)

	// --- wrapAuthRepos routes (auth + repos) ---
	// Add handlers
	router.HandleFunc(g.APIAddress.AddTagAddress, g.wrapAuthRepos(g.HandleAddTag)).Methods(g.APIAddress.AddTagMethod)
	router.HandleFunc(g.APIAddress.AddTextAddress, g.wrapAuthRepos(g.HandleAddText)).Methods(g.APIAddress.AddTextMethod)
	router.HandleFunc(g.APIAddress.AddNotificationAddress, g.wrapAuthRepos(g.HandleAddNotification)).Methods(g.APIAddress.AddNotificationMethod)
	router.HandleFunc(g.APIAddress.AddKmemoAddress, g.wrapAuthRepos(g.HandleAddKmemo)).Methods(g.APIAddress.AddKmemoMethod)
	router.HandleFunc(g.APIAddress.AddKCAddress, g.wrapAuthRepos(g.HandleAddKC)).Methods(g.APIAddress.AddKCMethod)
	router.HandleFunc(g.APIAddress.AddURLogAddress, g.wrapAuthRepos(g.HandleAddURLog)).Methods(g.APIAddress.AddURLogMethod)
	router.HandleFunc(g.APIAddress.AddNlogAddress, g.wrapAuthRepos(g.HandleAddNlog)).Methods(g.APIAddress.AddNlogMethod)
	router.HandleFunc(g.APIAddress.AddTimeisAddress, g.wrapAuthRepos(g.HandleAddTimeis)).Methods(g.APIAddress.AddTimeisMethod)
	router.HandleFunc(g.APIAddress.AddMiAddress, g.wrapAuthRepos(g.HandleAddMi)).Methods(g.APIAddress.AddMiMethod)
	router.HandleFunc(g.APIAddress.AddLantanaAddress, g.wrapAuthRepos(g.HandleAddLantana)).Methods(g.APIAddress.AddLantanaMethod)
	router.HandleFunc(g.APIAddress.AddRekyouAddress, g.wrapAuthRepos(g.HandleAddRekyou)).Methods(g.APIAddress.AddRekyouMethod)
	// Update handlers
	router.HandleFunc(g.APIAddress.UpdateTagAddress, g.wrapAuthRepos(g.HandleUpdateTag)).Methods(g.APIAddress.UpdateTagMethod)
	router.HandleFunc(g.APIAddress.UpdateTextAddress, g.wrapAuthRepos(g.HandleUpdateText)).Methods(g.APIAddress.UpdateTextMethod)
	router.HandleFunc(g.APIAddress.UpdateNotificationAddress, g.wrapAuthRepos(g.HandleUpdateNotification)).Methods(g.APIAddress.UpdateNotificationMethod)
	router.HandleFunc(g.APIAddress.UpdateKmemoAddress, g.wrapAuthRepos(g.HandleUpdateKmemo)).Methods(g.APIAddress.UpdateKmemoMethod)
	router.HandleFunc(g.APIAddress.UpdateKCAddress, g.wrapAuthRepos(g.HandleUpdateKC)).Methods(g.APIAddress.UpdateKCMethod)
	router.HandleFunc(g.APIAddress.UpdateURLogAddress, g.wrapAuthRepos(g.HandleUpdateURLog)).Methods(g.APIAddress.UpdateURLogMethod)
	router.HandleFunc(g.APIAddress.UpdateNlogAddress, g.wrapAuthRepos(g.HandleUpdateNlog)).Methods(g.APIAddress.UpdateNlogMethod)
	router.HandleFunc(g.APIAddress.UpdateTimeisAddress, g.wrapAuthRepos(g.HandleUpdateTimeis)).Methods(g.APIAddress.UpdateTimeisMethod)
	router.HandleFunc(g.APIAddress.UpdateLantanaAddress, g.wrapAuthRepos(g.HandleUpdateLantana)).Methods(g.APIAddress.UpdateLantanaMethod)
	router.HandleFunc(g.APIAddress.UpdateIDFKyouAddress, g.wrapAuthRepos(g.HandleUpdateIDFKyou)).Methods(g.APIAddress.UpdateIDFKyouMethod)
	router.HandleFunc(g.APIAddress.UpdateMiAddress, g.wrapAuthRepos(g.HandleUpdateMi)).Methods(g.APIAddress.UpdateMiMethod)
	router.HandleFunc(g.APIAddress.UpdateRekyouAddress, g.wrapAuthRepos(g.HandleUpdateRekyou)).Methods(g.APIAddress.UpdateRekyouMethod)
	// Get handlers
	router.HandleFunc(g.APIAddress.GetKyousAddress, g.wrapAuthRepos(g.HandleGetKyous)).Methods(g.APIAddress.GetKyousMethod)
	router.HandleFunc(g.APIAddress.GetKyouAddress, g.wrapAuthRepos(g.HandleGetKyou)).Methods(g.APIAddress.GetKyouMethod)
	router.HandleFunc(g.APIAddress.GetKmemoAddress, g.wrapAuthRepos(g.HandleGetKmemo)).Methods(g.APIAddress.GetKmemoMethod)
	router.HandleFunc(g.APIAddress.GetKCAddress, g.wrapAuthRepos(g.HandleGetKC)).Methods(g.APIAddress.GetKCMethod)
	router.HandleFunc(g.APIAddress.GetURLogAddress, g.wrapAuthRepos(g.HandleGetURLog)).Methods(g.APIAddress.GetURLogMethod)
	router.HandleFunc(g.APIAddress.GetNlogAddress, g.wrapAuthRepos(g.HandleGetNlog)).Methods(g.APIAddress.GetNlogMethod)
	router.HandleFunc(g.APIAddress.GetTimeisAddress, g.wrapAuthRepos(g.HandleGetTimeis)).Methods(g.APIAddress.GetTimeisMethod)
	router.HandleFunc(g.APIAddress.GetMiAddress, g.wrapAuthRepos(g.HandleGetMi)).Methods(g.APIAddress.GetMiMethod)
	router.HandleFunc(g.APIAddress.GetLantanaAddress, g.wrapAuthRepos(g.HandleGetLantana)).Methods(g.APIAddress.GetLantanaMethod)
	router.HandleFunc(g.APIAddress.GetRekyouAddress, g.wrapAuthRepos(g.HandleGetRekyou)).Methods(g.APIAddress.GetRekyouMethod)
	router.HandleFunc(g.APIAddress.GetGitCommitLogAddress, g.wrapAuthRepos(g.HandleGetGitCommitLog)).Methods(g.APIAddress.GetGitCommitLogMethod)
	router.HandleFunc(g.APIAddress.GetIDFKyouAddress, g.wrapAuthRepos(g.HandleGetIDFKyou)).Methods(g.APIAddress.GetIDFKyouMethod)
	router.HandleFunc(g.APIAddress.GetMiBoardListAddress, g.wrapAuthRepos(g.HandleGetMiBoardList)).Methods(g.APIAddress.GetMiBoardListMethod)
	router.HandleFunc(g.APIAddress.GetAllTagNamesAddress, g.wrapAuthRepos(g.HandleGetAllTagNames)).Methods(g.APIAddress.GetAllTagNamesMethod)
	router.HandleFunc(g.APIAddress.GetAllRepNamesAddress, g.wrapAuthRepos(g.HandleGetAllRepNames)).Methods(g.APIAddress.GetAllRepNamesMethod)
	router.HandleFunc(g.APIAddress.GetTagsByTargetIDAddress, g.wrapAuthRepos(g.HandleGetTagsByTargetID)).Methods(g.APIAddress.GetTagsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetTagHistoriesByTagIDAddress, g.wrapAuthRepos(g.HandleGetTagHistoriesByTagID)).Methods(g.APIAddress.GetTagHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetTextsByTargetIDAddress, g.wrapAuthRepos(g.HandleGetTextsByTargetID)).Methods(g.APIAddress.GetTextsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetNotificationsByTargetIDAddress, g.wrapAuthRepos(g.HandleGetNotificationsByTargetID)).Methods(g.APIAddress.GetNotificationsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetTextHistoriesByTextIDAddress, g.wrapAuthRepos(g.HandleGetTextHistoriesByTextID)).Methods(g.APIAddress.GetTextHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetNotificationHistoriesByNotificationIDAddress, g.wrapAuthRepos(g.HandleGetNotificationHistoriesByNotificationID)).Methods(g.APIAddress.GetNotificationHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetGPSLogAddress, g.wrapAuthRepos(g.HandleGetGPSLog)).Methods(g.APIAddress.GetGPSLogMethod)
	// Share handlers
	router.HandleFunc(g.APIAddress.AddShareKyouListInfoAddress, g.wrapAuthRepos(g.HandleAddShareKyouListInfo)).Methods(g.APIAddress.AddShareKyouListInfoMethod)
	router.HandleFunc(g.APIAddress.UpdateShareKyouListInfoAddress, g.wrapAuthRepos(g.HandleUpdateShareKyouListInfo)).Methods(g.APIAddress.UpdateShareKyouListInfoMethod)
	router.HandleFunc(g.APIAddress.GetShareKyouListInfosAddress, g.wrapAuthRepos(g.HandleGetShareKyouListInfos)).Methods(g.APIAddress.GetShareKyouListInfosMethod)
	router.HandleFunc(g.APIAddress.DeleteShareKyouListInfosAddress, g.wrapAuthRepos(g.HandleDeleteShareKyouListInfos)).Methods(g.APIAddress.DeleteShareKyouListInfosMethod)
	// Repository and transaction handlers
	router.HandleFunc(g.APIAddress.GetRepositoriesAddress, g.wrapAuthRepos(g.HandleGetRepositories)).Methods(g.APIAddress.GetRepositoriesMethod)
	router.HandleFunc(g.APIAddress.CommitTXAddress, g.wrapAuthRepos(g.HandleCommitTx)).Methods(g.APIAddress.CommitTXMethod)
	router.HandleFunc(g.APIAddress.DiscardTXAddress, g.wrapAuthRepos(g.HandleDiscardTX)).Methods(g.APIAddress.DiscardTXMethod)
	router.HandleFunc(g.APIAddress.SubmitKFTLTextAddress, g.wrapAuthRepos(g.HandleSubmitKFTLText)).Methods(g.APIAddress.SubmitKFTLTextMethod)

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
	router.PathPrefix(g.APIAddress.GkillWebpushServiceWorkerJsAddress).Handler(http.StripPrefix("",
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
	router.PathPrefix("/plaing").Handler(http.StripPrefix("/plaing",
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
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return err
	}
	port := serverConfig.Address

	g.PrintStartedMessage()
	serveCtx, serveCancel := context.WithCancel(ctx)
	defer serveCancel()
	g.server = &http.Server{
		Addr:    port,
		Handler: router,
		BaseContext: func(_ net.Listener) context.Context {
			return serveCtx
		},
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
			slog.Log(ctx, gkill_log.Error, "error", "error", err)
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
