package api

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/api/gpslogs"
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/twpayne/go-gpx"
)

func NewGkillServerAPI() (*GkillServerAPI, error) {
	gkillDAOManager, err := dao.NewGkillDAOManager()
	if err != nil {
		err = fmt.Errorf("error at create gkill dao manager: %w", err)
		return nil, err
	}

	// 初回起動の場合、Adminアカウントデータを作成する
	accounts, err := gkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get all account: %w", err)
		return nil, err
	}
	if len(accounts) == 0 {
		passwordSha256Admin := "8c6976e5b5410415bde908bd4dee15dfb167a9c873fc4bb8a81f6f2ab448a918"
		adminAccount := &account.Account{
			UserID:             "admin",
			PasswordSha256:     &passwordSha256Admin,
			IsAdmin:            true,
			IsEnable:           true,
			PasswordResetToken: nil,
		}
		_, err := gkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(context.TODO(), adminAccount)
		if err != nil {
			err = fmt.Errorf("error at add admin account: %w", err)
			return nil, err
		}
	}

	applicationConfigs, err := gkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetAllApplicationConfigs(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get all application configs: %w", err)
		return nil, err
	}
	if len(applicationConfigs) == 0 {
		applicationConfig := &user_config.ApplicationConfig{
			UserID:                    "admin",
			Device:                    "gkill",
			EnableBrowserCache:        false,
			GoogleMapAPIKey:           "",
			RykvImageListColumnNumber: 3,
			RykvHotReload:             false,
			MiDefaultBoard:            "Inbox",
		}
		_, err := gkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(context.TODO(), applicationConfig)
		if err != nil {
			err = fmt.Errorf("error at add application config admin: %w", err)
			return nil, err
		}

	}

	serverConfigs, err := gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get server configs: %w", err)
		return nil, err
	}
	if len(serverConfigs) == 0 {
		serverConfig := &server_config.ServerConfig{
			EnableThisDevice:     true,
			Device:               "gkill",
			IsLocalOnlyAccess:    false,
			Address:              ":9999",
			EnableTLS:            false,
			TLSCertFile:          "$HOME/gkill/tls/cert.cer",
			TLSKeyFile:           "$HOME/gkill/tls/key.pem",
			OpenDirectoryCommand: "explorer /select,$filename",
			OpenFileCommand:      "rundll32 url.dll,FileProtocolHandler $filename",
			URLogTimeout:         1 * time.Minute,
			URLogUserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36",
			UploadSizeLimitMonth: -1,
			UserDataDirectory:    "$HOME/gkill/datas",
		}
		_, err = gkillDAOManager.ConfigDAOs.ServerConfigDAO.AddServerConfig(context.TODO(), serverConfig)
		if err != nil {
			err = fmt.Errorf("error at add init data to server config db: %w", err)
			return nil, err
		}
	}

	return &GkillServerAPI{
		APIAddress:      NewGKillAPIAddress(),
		GkillDAOManager: gkillDAOManager,
		FindFilter:      &FindFilter{},
		RebootServerCh:  make(chan struct{}),
	}, nil
}

type GkillServerAPI struct {
	APIAddress *GkillServerAPIAddress

	GkillDAOManager *dao.GkillDAOManager

	FindFilter *FindFilter

	RebootServerCh chan (struct{})

	device string
}

func (g *GkillServerAPI) Serve() error {
	router := g.GkillDAOManager.GetRouter()
	router.PathPrefix("/files/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.HandleFileServe(w, r)
	})
	router.HandleFunc(g.APIAddress.LoginAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleLogin(w, r)
	}).Methods(g.APIAddress.LoginMethod)
	router.HandleFunc(g.APIAddress.LogoutAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleLogout(w, r)
	}).Methods(g.APIAddress.LogoutMethod)
	router.HandleFunc(g.APIAddress.ResetPasswordAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleResetPassword(w, r)
	}).Methods(g.APIAddress.ResetPasswordMethod)
	router.HandleFunc(g.APIAddress.SetNewPasswordAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleSetNewPassword(w, r)
	}).Methods(g.APIAddress.SetNewPasswordMethod)
	router.HandleFunc(g.APIAddress.AddTagAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddTag(w, r)
	}).Methods(g.APIAddress.AddTagMethod)
	router.HandleFunc(g.APIAddress.AddTextAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddText(w, r)
	}).Methods(g.APIAddress.AddTextMethod)
	router.HandleFunc(g.APIAddress.AddKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddKmemo(w, r)
	}).Methods(g.APIAddress.AddKmemoMethod)
	router.HandleFunc(g.APIAddress.AddURLogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddURLog(w, r)
	}).Methods(g.APIAddress.AddURLogMethod)
	router.HandleFunc(g.APIAddress.AddNlogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddNlog(w, r)
	}).Methods(g.APIAddress.AddNlogMethod)
	router.HandleFunc(g.APIAddress.AddTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddTimeis(w, r)
	}).Methods(g.APIAddress.AddTimeisMethod)
	router.HandleFunc(g.APIAddress.AddMiAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddMi(w, r)
	}).Methods(g.APIAddress.AddMiMethod)
	router.HandleFunc(g.APIAddress.AddLantanaAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddLantana(w, r)
	}).Methods(g.APIAddress.AddLantanaMethod)
	router.HandleFunc(g.APIAddress.AddRekyouAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddRekyou(w, r)
	}).Methods(g.APIAddress.AddRekyouMethod)
	router.HandleFunc(g.APIAddress.UpdateTagAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateTag(w, r)
	}).Methods(g.APIAddress.UpdateTagMethod)
	router.HandleFunc(g.APIAddress.UpdateTextAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateText(w, r)
	}).Methods(g.APIAddress.UpdateTextMethod)
	router.HandleFunc(g.APIAddress.UpdateKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateKmemo(w, r)
	}).Methods(g.APIAddress.UpdateKmemoMethod)
	router.HandleFunc(g.APIAddress.UpdateURLogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateURLog(w, r)
	}).Methods(g.APIAddress.UpdateURLogMethod)
	router.HandleFunc(g.APIAddress.UpdateNlogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateNlog(w, r)
	}).Methods(g.APIAddress.UpdateNlogMethod)
	router.HandleFunc(g.APIAddress.UpdateTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateTimeis(w, r)
	}).Methods(g.APIAddress.UpdateTimeisMethod)
	router.HandleFunc(g.APIAddress.UpdateLantanaAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateLantana(w, r)
	}).Methods(g.APIAddress.UpdateLantanaMethod)
	router.HandleFunc(g.APIAddress.UpdateIDFKyouAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateIDFKyou(w, r)
	}).Methods(g.APIAddress.UpdateIDFKyouMethod)
	router.HandleFunc(g.APIAddress.UpdateMiAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateMi(w, r)
	}).Methods(g.APIAddress.UpdateMiMethod)
	router.HandleFunc(g.APIAddress.UpdateRekyouAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateRekyou(w, r)
	}).Methods(g.APIAddress.UpdateRekyouMethod)
	router.HandleFunc(g.APIAddress.GetKyousAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetKyous(w, r)
	}).Methods(g.APIAddress.GetKyousMethod)
	router.HandleFunc(g.APIAddress.GetKyouAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetKyou(w, r)
	}).Methods(g.APIAddress.GetKyouMethod)
	router.HandleFunc(g.APIAddress.GetKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetKmemo(w, r)
	}).Methods(g.APIAddress.GetKmemoMethod)
	router.HandleFunc(g.APIAddress.GetURLogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetURLog(w, r)
	}).Methods(g.APIAddress.GetURLogMethod)
	router.HandleFunc(g.APIAddress.GetNlogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetNlog(w, r)
	}).Methods(g.APIAddress.GetNlogMethod)
	router.HandleFunc(g.APIAddress.GetTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetTimeis(w, r)
	}).Methods(g.APIAddress.GetTimeisMethod)
	router.HandleFunc(g.APIAddress.GetMiAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetMi(w, r)
	}).Methods(g.APIAddress.GetMiMethod)
	router.HandleFunc(g.APIAddress.GetLantanaAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetLantana(w, r)
	}).Methods(g.APIAddress.GetLantanaMethod)
	router.HandleFunc(g.APIAddress.GetRekyouAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetRekyou(w, r)
	}).Methods(g.APIAddress.GetRekyouMethod)
	router.HandleFunc(g.APIAddress.GetGitCommitLogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetGitCommitLog(w, r)
	}).Methods(g.APIAddress.GetGitCommitLogMethod)
	router.HandleFunc(g.APIAddress.GetIDFKyouAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetIDFKyou(w, r)
	}).Methods(g.APIAddress.GetIDFKyouMethod)
	router.HandleFunc(g.APIAddress.GetMiBoardListAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetMiBoardList(w, r)
	}).Methods(g.APIAddress.GetMiBoardListMethod)
	router.HandleFunc(g.APIAddress.GetPlaingTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetPlaingTimeis(w, r)
	}).Methods(g.APIAddress.GetPlaingTimeisMethod)
	router.HandleFunc(g.APIAddress.GetAllTagNamesAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetAllTagNames(w, r)
	}).Methods(g.APIAddress.GetAllTagNamesMethod)
	router.HandleFunc(g.APIAddress.GetAllRepNamesAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetAllRepNames(w, r)
	}).Methods(g.APIAddress.GetAllRepNamesMethod)
	router.HandleFunc(g.APIAddress.GetTagsByTargetIDAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetTagsByTargetID(w, r)
	}).Methods(g.APIAddress.GetTagsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetTagHistoriesByTagIDAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetTagHistoriesByTagID(w, r)
	}).Methods(g.APIAddress.GetTagHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetTextsByTargetIDAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetTextsByTargetID(w, r)
	}).Methods(g.APIAddress.GetTextsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetTextHistoriesByTextIDAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetTextHistoriesByTextID(w, r)
	}).Methods(g.APIAddress.GetTextHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetApplicationConfigAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetApplicationConfig(w, r)
	}).Methods(g.APIAddress.GetApplicationConfigMethod)
	router.HandleFunc(g.APIAddress.GetServerConfigsAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetServerConfigs(w, r)
	}).Methods(g.APIAddress.GetServerConfigsMethod)
	router.HandleFunc(g.APIAddress.UploadFilesAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUploadFiles(w, r)
	}).Methods(g.APIAddress.UploadFilesMethod)
	router.HandleFunc(g.APIAddress.UploadGPSLogFilesAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUploadGPSLogFiles(w, r)
	}).Methods(g.APIAddress.UploadGPSLogFilesMethod)
	router.HandleFunc(g.APIAddress.UpdateApplicationConfigAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateApplicationConfig(w, r)
	}).Methods(g.APIAddress.UpdateApplicationConfigMethod)
	router.HandleFunc(g.APIAddress.UpdateTagStructAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateTagStruct(w, r)
	}).Methods(g.APIAddress.UpdateTagStructMethod)
	router.HandleFunc(g.APIAddress.UpdateRepStructAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateRepStruct(w, r)
	}).Methods(g.APIAddress.UpdateRepStructMethod)
	router.HandleFunc(g.APIAddress.UpdateDeviceStructAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateDeviceStruct(w, r)
	}).Methods(g.APIAddress.UpdateDeviceStructMethod)
	router.HandleFunc(g.APIAddress.UpdateRepTypeStructAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateRepTypeStruct(w, r)
	}).Methods(g.APIAddress.UpdateRepTypeStructMethod)
	router.HandleFunc(g.APIAddress.UpdateKFTLTemplateAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateKFTLTemplate(w, r)
	}).Methods(g.APIAddress.UpdateKFTLTemplateStructMethod)
	router.HandleFunc(g.APIAddress.UpdateAccountStatusAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateAccountStatus(w, r)
	}).Methods(g.APIAddress.UpdateAccountStatusMethod)
	router.HandleFunc(g.APIAddress.UpdateUserRepsAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateUserReps(w, r)
	}).Methods(g.APIAddress.UpdateUserRepsMethod)
	router.HandleFunc(g.APIAddress.UpdateServerConfigsAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateServerConfigs(w, r)
	}).Methods(g.APIAddress.UpdateServerConfigsMethod)
	router.HandleFunc(g.APIAddress.AddAccountAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddAccount(w, r)
	}).Methods(g.APIAddress.AddAccountMethod)
	router.HandleFunc(g.APIAddress.GenerateTLSFileAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGenerateTLSFile(w, r)
	}).Methods(g.APIAddress.GenerateTLSFileMethod)
	router.HandleFunc(g.APIAddress.GetGPSLogAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetGPSLog(w, r)
	}).Methods(g.APIAddress.GetGPSLogMethod)
	router.HandleFunc(g.APIAddress.GetKFTLTemplateAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetKFTLTemplate(w, r)
	}).Methods(g.APIAddress.GetKFTLTemplateMethod)
	router.HandleFunc(g.APIAddress.GetGkillInfoAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetGkillInfo(w, r)
	}).Methods(g.APIAddress.GetGkillInfoMethod)
	router.HandleFunc(g.APIAddress.GetShareMiTaskListInfosAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetShareMiTaskListInfos(w, r)
	}).Methods(g.APIAddress.GetShareMiTaskListInfosMethod)
	router.HandleFunc(g.APIAddress.AddShareMiTaskListInfoAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleAddShareMiTaskListInfo(w, r)
	}).Methods(g.APIAddress.AddShareMiTaskListInfoMethod)
	router.HandleFunc(g.APIAddress.
		UpdateShareMiTaskListInfoAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleUpdateShareMiTaskListInfo(w, r)
	}).Methods(g.APIAddress.UpdateShareMiTaskListInfoMethod)
	router.HandleFunc(g.APIAddress.DeleteShareMiTaskListInfosAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleDeleteShareMiTaskListInfos(w, r)
	}).Methods(g.APIAddress.DeleteShareMiTaskListInfosMethod)
	router.HandleFunc(g.APIAddress.GetMiSharedTasksAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetMiSharedTask(w, r)
	}).Methods(g.APIAddress.GetMiSharedTasksMethod)
	router.HandleFunc(g.APIAddress.GetRepositoriesAddress, func(w http.ResponseWriter, r *http.Request) {
		g.HandleGetRepositories(w, r)
	}).Methods(g.APIAddress.GetRepositoriesMethod)

	gkillPage, err := fs.Sub(htmlFS, "embed/html")
	if err != nil {
		return err
	}
	router.PathPrefix("/rykv").Handler(http.StripPrefix("/rykv",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/kftl").Handler(http.StripPrefix("/kftl",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/mi").Handler(http.StripPrefix("/mi",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/kyou").Handler(http.StripPrefix("/kyou",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/saihate").Handler(http.StripPrefix("/saihate",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/plaing").Handler(http.StripPrefix("/plaing",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/mkfl").Handler(http.StripPrefix("/mkfl",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/shared_mi").Handler(http.StripPrefix("/shared_mi",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/set_new_password").Handler(http.StripPrefix("/set_new_password",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.FileServer(http.FS(gkillPage)).ServeHTTP(w, r)
		})))
	router.PathPrefix("/").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		gkill_log.Debug.Printf(err.Error())
		return err
	}
	port := serverConfig.Address
	if serverConfig.EnableTLS {
		certFileName, pemFileName, err := g.getTLSFileNames(device)
		if err != nil {
			gkill_log.Debug.Fatal(err)
			return err
		}
		certFileName, pemFileName = os.ExpandEnv(certFileName), os.ExpandEnv(pemFileName)
		certFileName, pemFileName = filepath.ToSlash(certFileName), filepath.ToSlash(pemFileName)
		err = http.ListenAndServeTLS(port, certFileName, pemFileName, router)
	} else {
		err = http.ListenAndServe(port, router)
	}
	return err
}

func (g *GkillServerAPI) Close() error {
	var err error
	err = g.GkillDAOManager.Close()
	if err != nil {
		err = fmt.Errorf("error at close gkill dbo manager: %w", err)
		return err
	}

	close(g.RebootServerCh)
	g.APIAddress = nil
	g.GkillDAOManager = nil
	g.FindFilter = nil
	g.RebootServerCh = nil

	return nil
}

func (g *GkillServerAPI) HandleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.LoginRequest{}
	response := &req_res.LoginResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse login response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidLoginResponseDataError,
				ErrorMessage: "ログインに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse login request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidLoginRequestDataError,
			ErrorMessage: "ログインに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 存在するアカウントを取得
	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "ログインに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "ユーザIDまたはパスワードが違います",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント有効確認
	if !account.IsEnable {
		err = fmt.Errorf("error at account is not enable = %s: %w", request.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountIsNotEnableError,
			ErrorMessage: "ログインに失敗しました。アカウントが無効化されています",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードリセット処理実施中のアカウントはログインから弾く
	if account.PasswordResetToken != nil {
		err = fmt.Errorf("error at password reset token is not nil = %s: %w", request.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountPasswordResetTokenIsNotNilError,
			ErrorMessage: "パスワードリセットを完了してください",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワード不一致を弾く
	if account.PasswordSha256 != nil && *account.PasswordSha256 != request.PasswordSha256 {
		err = fmt.Errorf("error at account invalid password = %s: %w", request.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: "ログインに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ログインセッション追加
	isLocalAppUser := false
	spl := strings.Split(r.RemoteAddr, ":")
	remoteHost := strings.Join(spl[:len(spl)-1], ":")
	switch remoteHost {
	case "localhost":
		fallthrough
	case "127.0.0.1":
		fallthrough
	case "[::1]":
		fallthrough
	case "::1":
		isLocalAppUser = true
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	loginSession := &account_state.LoginSession{
		ID:              GenerateNewID(),
		UserID:          request.UserID,
		Device:          device,
		ApplicationName: "gkill",
		SessionID:       GenerateNewID(),
		ClientIPAddress: remoteHost,
		LoginTime:       time.Now(),
		ExpirationTime:  time.Now().Add(time.Hour * 24 * 30), // 1ヶ月
		IsLocalAppUser:  isLocalAppUser,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.AddLoginSession(r.Context(), loginSession)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error add login session = %s: %w", request.UserID, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountLoginInternalServerError,
			ErrorMessage: "ログインに失敗しました（サーバ内部エラー）",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.SessionID = loginSession.SessionID
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.LoginSuccessMessage,
		Message:     "ログインしました",
	})
}

func (g *GkillServerAPI) HandleLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.LogoutRequest{}
	response := &req_res.LogoutResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse logout request to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidLogoutResponseDataError,
				ErrorMessage: "ログアウトに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse logout request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidLogoutRequestDataError,
			ErrorMessage: "ログアウトに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.DeleteLoginSession(r.Context(), request.SessionID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error add logout session id = %s: %w", request.SessionID, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountLogoutInternalServerError,
			ErrorMessage: "ログインに失敗しました（サーバ内部エラー）",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.LogoutSuccessMessage,
		Message:     "ログアウトしました",
	})
}

func (g *GkillServerAPI) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ResetPasswordRequest{}
	response := &req_res.ResetPasswordResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse reset password to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidResetPasswordResponseDataError,
				ErrorMessage: "パスワードリセット処理に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse reset password request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidResetPasswordRequestDataError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードリセット操作をしたユーザを特定
	requesterSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), requesterSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", requesterSession.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterSession.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "パスワードリセット処理に失敗しました。権限がありません。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	passwordResetToken := GenerateNewID()
	updateTargetAccount := &account.Account{
		UserID:             targetAccount.UserID,
		IsAdmin:            targetAccount.IsAdmin,
		IsEnable:           targetAccount.IsEnable,
		PasswordSha256:     nil,
		PasswordResetToken: &passwordResetToken,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), updateTargetAccount)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update account user id = %s: %w", request.TargetUserID, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.PasswordResetSuccessMessage,
		Message:     "パスワードリセット処理を完了しました",
	})
	response.PasswordResetPathWithoutHost = fmt.Sprintf("%s", *updateTargetAccount.PasswordResetToken)
}

func (g *GkillServerAPI) HandleSetNewPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.SetNewPasswordRequest{}
	response := &req_res.SetNewPasswordResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse set new password response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
				ErrorMessage: "パスワード設定に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse login response to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
			ErrorMessage: "パスワード設定に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得してパスワード設定
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワード設定に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// リセットトークンがあっているか確認
	if targetAccount.PasswordResetToken == nil || request.ResetToken != *targetAccount.PasswordResetToken {
		err = fmt.Errorf("error at reset token is not match user id = %s requested token = %s: %w", request.UserID, request.ResetToken, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidPasswordResetTokenError,
			ErrorMessage: "パスワード設定に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	updateTargetAccount := &account.Account{
		UserID:             targetAccount.UserID,
		IsAdmin:            targetAccount.IsAdmin,
		IsEnable:           targetAccount.IsEnable,
		PasswordSha256:     &request.NewPasswordSha256,
		PasswordResetToken: nil,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), updateTargetAccount)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update account user id = %s: %w", request.UserID, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: "パスワード設定に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.SetNewPasswordSuccessMessage,
		Message:     "パスワード設定処理が完了しました",
	})
}

func (g *GkillServerAPI) HandleAddTag(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddTagRequest{}
	response := &req_res.AddTagResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add tag response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddTagResponseDataError,
				ErrorMessage: "タグ追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add tag request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddTagRequestDataError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existTag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTag != nil {
		err = fmt.Errorf("exist tag id = %s", request.Tag.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTagError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteTagRep.AddTagInfo(r.Context(), request.Tag)
	if err != nil {
		err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddTagError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteTagRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Tag.IsDeleted,
		TargetID:                 request.Tag.ID,
		DataUpdateTime:           request.Tag.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedTag = tag
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddTagSuccessMessage,
		Message:     "タグを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddTextRequest{}
	response := &req_res.AddTextResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add text response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddTextResponseDataError,
				ErrorMessage: "テキスト追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add text request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddTextRequestDataError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existText, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existText != nil {
		err = fmt.Errorf("exist text id = %s", request.Text.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTextError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteTextRep.AddTextInfo(r.Context(), request.Text)
	if err != nil {
		err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddTextError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteTextRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Text.IsDeleted,
		TargetID:                 request.Text.ID,
		DataUpdateTime:           request.Text.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	text, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedText = text
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddTextSuccessMessage,
		Message:     "テキストを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddKmemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddKmemoRequest{}
	response := &req_res.AddKmemoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add kmemo response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddKmemoResponseDataError,
				ErrorMessage: "kmemo追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add kmemo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddKmemoRequestDataError,
			ErrorMessage: "kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existKmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKmemo != nil {
		err = fmt.Errorf("exist kmemo id = %s", request.Kmemo.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistKmemoError,
			ErrorMessage: "Kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), request.Kmemo)
	if err != nil {
		err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddKmemoError,
			ErrorMessage: "Kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Kmemo.IsDeleted,
		TargetID:                 request.Kmemo.ID,
		DataUpdateTime:           request.Kmemo.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "kmemo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedKmemo = kmemo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddKmemoSuccessMessage,
		Message:     "kmemoを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddURLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddURLogRequest{}
	response := &req_res.AddURLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add urlog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddURLogResponseDataError,
				ErrorMessage: "URLog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add kmemo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddURLogRequestDataError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existURLog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", request.URLog.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// applicationConfigを取得
	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "ApplicationConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ServerConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	request.URLog.FillURLogField(serverConfig, applicationConfig)

	err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), request.URLog)
	if err != nil {
		err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.URLog.IsDeleted,
		TargetID:                 request.URLog.ID,
		DataUpdateTime:           request.URLog.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedURLog = urlog
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddURLogSuccessMessage,
		Message:     "URLogを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddNlog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddNlogRequest{}
	response := &req_res.AddNlogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add nlog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddNlogResponseDataError,
				ErrorMessage: "Nlog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add urlog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddNlogRequestDataError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existNlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNlog != nil {
		err = fmt.Errorf("exist nlog id = %s", request.Nlog.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistNlogError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), request.Nlog)
	if err != nil {
		err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddNlogError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Nlog.IsDeleted,
		TargetID:                 request.Nlog.ID,
		DataUpdateTime:           request.Nlog.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedNlog = nlog
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNlogSuccessMessage,
		Message:     "Nlogを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddTimeis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddTimeIsRequest{}
	response := &req_res.AddTimeIsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add timeis response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddTimeIsResponseDataError,
				ErrorMessage: "TimeIs追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add timeis request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddTimeIsRequestDataError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTimeIs != nil {
		err = fmt.Errorf("exist timeis id = %s", request.TimeIs.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTimeIsError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), request.TimeIs)
	if err != nil {
		err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddTimeIsError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.TimeIs.IsDeleted,
		TargetID:                 request.TimeIs.ID,
		DataUpdateTime:           request.TimeIs.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeis, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedTimeis = timeis
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddTimeIsSuccessMessage,
		Message:     "TimeIsを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddLantana(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddLantanaRequest{}
	response := &req_res.AddLantanaResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add lantana response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddLantanaResponseDataError,
				ErrorMessage: "Lantana追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add lantana request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddLantanaRequestDataError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existLantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if existLantana != nil {
		err = fmt.Errorf("exist lantana id = %s", request.Lantana.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistLantanaError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), request.Lantana)
	if err != nil {
		err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddLantanaError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Lantana.IsDeleted,
		TargetID:                 request.Lantana.ID,
		DataUpdateTime:           request.Lantana.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedLantana = lantana
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddLantanaSuccessMessage,
		Message:     "Lantanaを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddMi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddMiRequest{}
	response := &req_res.AddMiResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add mi response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddMiResponseDataError,
				ErrorMessage: "Mi追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add mi request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddMiRequestDataError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existMi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existMi != nil {
		err = fmt.Errorf("exist mi id = %s", request.Mi.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistMiError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteMiRep.AddMiInfo(r.Context(), request.Mi)
	if err != nil {
		err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddMiError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteMiRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Mi.IsDeleted,
		TargetID:                 request.Mi.ID,
		DataUpdateTime:           request.Mi.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	mi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedMi = mi
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddMiSuccessMessage,
		Message:     "Miを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddRekyou(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddReKyouRequest{}
	response := &req_res.AddReKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add rekyou response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddReKyouResponseDataError,
				ErrorMessage: "ReKyou追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add rekyou request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddReKyouRequestDataError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existReKyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existReKyou != nil {
		err = fmt.Errorf("exist rekyou id = %s", request.ReKyou.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistReKyouError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), request.ReKyou)
	if err != nil {
		err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddReKyouError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.ReKyou.IsDeleted,
		TargetID:                 request.ReKyou.ID,
		DataUpdateTime:           request.ReKyou.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedReKyou = rekyou
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddReKyouSuccessMessage,
		Message:     "ReKyouを追加しました",
	})
}

func (g *GkillServerAPI) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTagRequest{}
	response := &req_res.UpdateTagResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update tag response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTagResponseDataError,
				ErrorMessage: "タグ更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update tag request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTagRequestDataError,
			ErrorMessage: "タグ更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "タグ更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteTagRep.AddTagInfo(r.Context(), request.Tag)
	if err != nil {
		err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddTagError,
			ErrorMessage: "タグ更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteTagRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Tag.IsDeleted,
		TargetID:                 request.Tag.ID,
		DataUpdateTime:           request.Tag.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existTag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTag == nil {
		err = fmt.Errorf("not exist tag id = %s", request.Tag.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTagError,
			ErrorMessage: "タグ更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedTag = tag
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTagSuccessMessage,
		Message:     "タグを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTextRequest{}
	response := &req_res.UpdateTextResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update text response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTextResponseDataError,
				ErrorMessage: "テキスト更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update text request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTextRequestDataError,
			ErrorMessage: "テキスト更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "テキスト更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteTextRep.AddTextInfo(r.Context(), request.Text)
	if err != nil {
		err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddTextError,
			ErrorMessage: "テキスト更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteTextRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Text.IsDeleted,
		TargetID:                 request.Text.ID,
		DataUpdateTime:           request.Text.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	text, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existText, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existText == nil {
		err = fmt.Errorf("not exist text id = %s", request.Text.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTextError,
			ErrorMessage: "テキスト更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedText = text
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTextSuccessMessage,
		Message:     "テキストを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateKmemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateKmemoRequest{}
	response := &req_res.UpdateKmemoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update kmemo response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateKmemoResponseDataError,
				ErrorMessage: "Kmemo更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update kmemo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateKmemoRequestDataError,
			ErrorMessage: "Kmemo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Kmemo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), request.Kmemo)
	if err != nil {
		err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddKmemoError,
			ErrorMessage: "Kmemo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Kmemo.IsDeleted,
		TargetID:                 request.Kmemo.ID,
		DataUpdateTime:           request.Kmemo.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existKmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKmemo == nil {
		err = fmt.Errorf("not exist kmemo id = %s", request.Kmemo.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundKmemoError,
			ErrorMessage: "Kmemo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedKmemo = kmemo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateKmemoSuccessMessage,
		Message:     "Kmemoを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateURLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateURLogRequest{}
	response := &req_res.UpdateURLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update urlog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateURLogResponseDataError,
				ErrorMessage: "URLog更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update urlog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateURLogRequestDataError,
			ErrorMessage: "URLog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "URLog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), request.URLog)
	if err != nil {
		err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddURLogError,
			ErrorMessage: "URLog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.URLog.IsDeleted,
		TargetID:                 request.URLog.ID,
		DataUpdateTime:           request.URLog.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existURLog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog == nil {
		err = fmt.Errorf("not exist urlog id = %s", request.URLog.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundURLogError,
			ErrorMessage: "URLog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedURLog = urlog
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateURLogSuccessMessage,
		Message:     "URLogを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateNlog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateNlogRequest{}
	response := &req_res.UpdateNlogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update nlog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateNlogResponseDataError,
				ErrorMessage: "Nlog更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update nlog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateNlogRequestDataError,
			ErrorMessage: "Nlog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Nlog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), request.Nlog)
	if err != nil {
		err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddNlogError,
			ErrorMessage: "Nlog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Nlog.IsDeleted,
		TargetID:                 request.Nlog.ID,
		DataUpdateTime:           request.Nlog.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existNlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNlog == nil {
		err = fmt.Errorf("not exist nlog id = %s", request.Nlog.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundNlogError,
			ErrorMessage: "Nlog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedNlog = nlog
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateNlogSuccessMessage,
		Message:     "Nlogを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateTimeis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTimeisRequest{}
	response := &req_res.UpdateTimeisResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update timeis response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTimeIsResponseDataError,
				ErrorMessage: "TimeIs更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update timeis request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTimeIsRequestDataError,
			ErrorMessage: "TimeIs更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "TimeIs更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), request.TimeIs)
	if err != nil {
		err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddTimeIsError,
			ErrorMessage: "TimeIs更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.TimeIs.IsDeleted,
		TargetID:                 request.TimeIs.ID,
		DataUpdateTime:           request.TimeIs.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeis, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTimeIs == nil {
		err = fmt.Errorf("not exist timeis id = %s", request.TimeIs.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTimeIsError,
			ErrorMessage: "TimeIs更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedTimeis = timeis
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTimeIsSuccessMessage,
		Message:     "TimeIsを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateLantana(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateLantanaRequest{}
	response := &req_res.UpdateLantanaResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update lantana response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateLantanaResponseDataError,
				ErrorMessage: "Lantana更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update lantana request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateLantanaRequestDataError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existLantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), request.Lantana)
	if err != nil {
		err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddLantanaError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Lantana.IsDeleted,
		TargetID:                 request.Lantana.ID,
		DataUpdateTime:           request.Lantana.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existLantana, err = repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existLantana == nil {
		err = fmt.Errorf("not exist lantana id = %s", request.Lantana.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundLantanaError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedLantana = lantana
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateLantanaSuccessMessage,
		Message:     "Lantanaを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateIDFKyou(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateIDFKyouRequest{}
	response := &req_res.UpdateIDFKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update idfKyou response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateIDFKyouResponseDataError,
				ErrorMessage: "IDFKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update idfKyou request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateIDFKyouRequestDataError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existIDFKyou, err := repositories.IDFKyouReps.GetIDFKyou(r.Context(), request.IDFKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), request.IDFKyou)
	if err != nil {
		err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddIDFKyouError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteIDFKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.IDFKyou.IsDeleted,
		TargetID:                 request.IDFKyou.ID,
		DataUpdateTime:           request.IDFKyou.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyou, err := repositories.IDFKyouReps.GetIDFKyou(r.Context(), request.IDFKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existIDFKyou, err = repositories.IDFKyouReps.GetIDFKyou(r.Context(), request.IDFKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existIDFKyou == nil {
		err = fmt.Errorf("not exist idfKyou id = %s", request.IDFKyou.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundIDFKyouError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedIDFKyou = idfKyou
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateIDFKyouSuccessMessage,
		Message:     "IDFKyouを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateMi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateMiRequest{}
	response := &req_res.UpdateMiResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update mi response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateMiResponseDataError,
				ErrorMessage: "Mi更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update mi request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateMiRequestDataError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteMiRep.AddMiInfo(r.Context(), request.Mi)
	if err != nil {
		err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddMiError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteMiRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.Mi.IsDeleted,
		TargetID:                 request.Mi.ID,
		DataUpdateTime:           request.Mi.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	mi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existMi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existMi == nil {
		err = fmt.Errorf("not exist mi id = %s", request.Mi.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundMiError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedMi = mi
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateMiSuccessMessage,
		Message:     "Miを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateRekyou(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateReKyouRequest{}
	response := &req_res.UpdateReKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update rekyou response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateReKyouResponseDataError,
				ErrorMessage: "ReKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update rekyou request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateReKyouRequestDataError,
			ErrorMessage: "ReKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "ReKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), request.ReKyou)
	if err != nil {
		err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddReKyouError,
			ErrorMessage: "ReKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                request.ReKyou.IsDeleted,
		TargetID:                 request.ReKyou.ID,
		DataUpdateTime:           request.ReKyou.UpdateTime,
		LatestDataRepositoryName: repName,
	})
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existReKyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existReKyou == nil {
		err = fmt.Errorf("not exist rekyou id = %s", request.ReKyou.ID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundReKyouError,
			ErrorMessage: "ReKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedReKyou = rekyou
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateReKyouSuccessMessage,
		Message:     "ReKyouを更新しました",
	})
}

func (g *GkillServerAPI) HandleGetKyous(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyousRequest{}
	response := &req_res.GetKyousResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousResponseDataError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyousRequestDataError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kyous, gkillErrors, err := g.FindFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, request.Query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			gkill_log.Debug.Printf(err.Error())
		}
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.Kyous = kyous
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyousSuccessMessage,
		Message:     "検索完了",
	})
}

func (g *GkillServerAPI) HandleGetKyou(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyouRequest{}
	response := &req_res.GetKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyou response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyouResponseDataError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyou request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyouRequestDataError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	kyouHistories := []*reps.Kyou{}
	if request.UpdateTime != nil {
		kyou := &reps.Kyou{}
		kyou, err = repositories.GetKyou(r.Context(), request.ID, request.UpdateTime)
		kyouHistories = []*reps.Kyou{kyou}
	} else {
		kyouHistories, err = repositories.GetKyouHistories(r.Context(), request.ID)
	}

	if err != nil {
		err = fmt.Errorf("error at get kyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKyouError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KyouHistories = kyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyouSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetKmemos(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetKmemosResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetURLogs(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetURLogsResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetNlogs(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetNlogsResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetTimeiss(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetTimeissResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetRekyous(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetReKyousResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetMis(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetMisResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetLantanas(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetLantanasResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetKmemo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKmemoRequest{}
	response := &req_res.GetKmemoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kmemo response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKmemoResponseDataError,
				ErrorMessage: "Kmemo取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kmemo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKmemoRequestDataError,
			ErrorMessage: "Kmemo取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Kmemo取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kmemoHistories, err := repositories.KmemoReps.GetKmemoHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KmemoHistories = kmemoHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKmemoSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetURLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetURLogRequest{}
	response := &req_res.GetURLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get urlog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetURLogResponseDataError,
				ErrorMessage: "URLog取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get urlog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetURLogRequestDataError,
			ErrorMessage: "URLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "URLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlogHistories, err := repositories.URLogReps.GetURLogHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.URLogHistories = urlogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetURLogSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetNlog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNlogRequest{}
	response := &req_res.GetNlogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get nlog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNlogResponseDataError,
				ErrorMessage: "Nlog取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get nlog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNlogRequestDataError,
			ErrorMessage: "Nlog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Nlog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlogHistories, err := repositories.NlogReps.GetNlogHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.NlogHistories = nlogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNlogSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetTimeis(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTimeisRequest{}
	response := &req_res.GetTimeisResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get timeis response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTimeIsResponseDataError,
				ErrorMessage: "TimeIs取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get timeis request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTimeIsRequestDataError,
			ErrorMessage: "TimeIs取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "TimeIs取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeisHistories, err := repositories.TimeIsReps.GetTimeIsHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TimeisHistories = timeisHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTimeIsSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetMi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetMiRequest{}
	response := &req_res.GetMiResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiResponseDataError,
				ErrorMessage: "Mi取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get mi request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiRequestDataError,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miHistories, err := repositories.MiReps.GetMiHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.MiHistories = miHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetLantana(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetLantanaRequest{}
	response := &req_res.GetLantanaResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get lantana response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetLantanaResponseDataError,
				ErrorMessage: "Lantana取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get lantana request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetLantanaRequestDataError,
			ErrorMessage: "Lantana取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Lantana取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantanaHistories, err := repositories.LantanaReps.GetLantanaHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.LantanaHistories = lantanaHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetLantanaSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetRekyou(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetReKyouRequest{}
	response := &req_res.GetReKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get rekyou response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetReKyouResponseDataError,
				ErrorMessage: "rekyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get rekyou request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetReKyouRequestDataError,
			ErrorMessage: "rekyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "rekyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyouHistories, err := repositories.ReKyouReps.GetReKyouHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "rekyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ReKyouHistories = rekyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetReKyouSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetGitCommitLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGitCommitLogRequest{}
	response := &req_res.GetGitCommitLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gitCommitLog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetGitCommitLogResponseDataError,
				ErrorMessage: "GitCommitLog取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get gitCommitLog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetGitCommitLogRequestDataError,
			ErrorMessage: "GitCommitLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "GitCommitLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	gitCommitLog, err := repositories.GitCommitLogReps.GetGitCommitLog(r.Context(), request.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get gitCommitLog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetGitCommitLogError,
			ErrorMessage: "GitCommitLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GitCommitLogHistories = []*reps.GitCommitLog{gitCommitLog}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGitCommitLogSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetIDFKyou(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetIDFKyouRequest{}
	response := &req_res.GetIDFKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get idfKyou response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetIDFKyouResponseDataError,
				ErrorMessage: "IDFKyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get idfKyou request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetIDFKyouRequestDataError,
			ErrorMessage: "IDFKyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "IDFKyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyouHistories, err := repositories.IDFKyouReps.GetIDFKyouHistories(r.Context(), request.ID)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.IDFKyouHistories = idfKyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetIDFKyouSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetMiBoardList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetMiBoardRequest{}
	response := &req_res.GetMiBoardResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi board names response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiBoardNamesResponseDataError,
				ErrorMessage: "MiBoardList取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get mi board names request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiBoardNamesRequestDataError,
			ErrorMessage: "MiBoardList取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "MiBoardList取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miBoardNames, err := repositories.MiReps.GetBoardNames(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get mi board names user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiBoardNamesError,
			ErrorMessage: "MiBoardList取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Boards = miBoardNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiBoardNamesSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetPlaingTimeis(w http.ResponseWriter, r *http.Request) {
	response := &req_res.GetPlaingTimeisResponse{}
	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.NotImplementsError,
				ErrorMessage: "機能が実装されていません",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := fmt.Errorf("not implements")
	gkill_log.Debug.Printf(err.Error())
	gkillError := &message.GkillError{
		ErrorCode:    message.NotImplementsError,
		ErrorMessage: "機能が実装されていません",
	}
	response.Errors = append(response.Errors, gkillError)
	return
}

func (g *GkillServerAPI) HandleGetAllTagNames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetAllTagNamesRequest{}
	response := &req_res.GetAllTagNamesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetAllTagNamesResponseDataError,
				ErrorMessage: "タグ名全件取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetAllTagNamesRequestDataError,
			ErrorMessage: "タグ名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "タグ名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	allTagNames, err := repositories.GetAllTagNames(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get all tag names user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllTagNamesError,
			ErrorMessage: "タグ名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TagNames = allTagNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetAllTagNamesSuccessMessage,
		Message:     "検索完了",
	})
}

func (g *GkillServerAPI) HandleGetAllRepNames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetAllRepNamesRequest{}
	response := &req_res.GetAllRepNamesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetAllRepNamesResponseDataError,
				ErrorMessage: "Rep名全件取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetAllRepNamesRequestDataError,
			ErrorMessage: "Rep名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Rep名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	allRepNames, err := repositories.GetAllRepNames(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get all rep names user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllRepNamesError,
			ErrorMessage: "Rep名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.RepNames = allRepNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetAllRepNamesSuccessMessage,
		Message:     "検索完了",
	})
}

func (g *GkillServerAPI) HandleGetTagsByTargetID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTagsByTargetIDRequest{}
	response := &req_res.GetTagsByTargetIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get tags by target id response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTagsByTargetIDResponseDataError,
				ErrorMessage: "タグ取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get tags by target id request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTagsByTargetIDRequestDataError,
			ErrorMessage: "タグ取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "タグ取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tags, err := repositories.GetTagsByTargetID(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get tags by target id user id = %s device = %s target id = %s: %w", userID, device, request.TargetID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagsByTargetIDError,
			ErrorMessage: "タグ取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Tags = tags
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTagsByTargetIDSuccessMessage,
		Message:     "タグ取得完了",
	})
}

func (g *GkillServerAPI) HandleGetTagHistoriesByTagID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTagHistoryByTagIDRequest{}
	response := &req_res.GetTagHistoryByTagIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get tag histories by tag id response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTagHistoriesByTagIDResponseDataError,
				ErrorMessage: "タグ取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get tag histories by tag id request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTagHistoriesByTagIDRequestDataError,
			ErrorMessage: "タグ取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "タグ取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	tags := []*reps.Tag{}
	if request.UpdateTime != nil {
		tag := &reps.Tag{}
		tag, err = repositories.GetTag(r.Context(), request.ID, request.UpdateTime)
		tags = []*reps.Tag{tag}
	} else {
		tags, err = repositories.GetTagHistories(r.Context(), request.ID)
	}

	if err != nil {
		err = fmt.Errorf("error at get tag histories by tag id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagHistoriesByTagIDError,
			ErrorMessage: "タグ取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TagHistories = tags
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTagHistoriesByTagIDSuccessMessage,
		Message:     "タグ取得完了",
	})
}

func (g *GkillServerAPI) HandleGetTextsByTargetID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTextsByTargetIDRequest{}
	response := &req_res.GetTextsByTargetIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get texts by target id response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTextsByTargetIDResponseDataError,
				ErrorMessage: "テキスト取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get texts by target id request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTextsByTargetIDRequestDataError,
			ErrorMessage: "テキスト取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "テキスト取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	texts, err := repositories.GetTextsByTargetID(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get texts by target id user id = %s device = %s target id = %s: %w", userID, device, request.TargetID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextsByTargetIDError,
			ErrorMessage: "テキスト取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Texts = texts
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTextsByTargetIDSuccessMessage,
		Message:     "テキスト取得完了",
	})
}

func (g *GkillServerAPI) HandleGetTextHistoriesByTextID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTextHistoryByTextIDRequest{}
	response := &req_res.GetTextHistoryByTextIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get text histories by text id response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTextHistoriesByTextIDResponseDataError,
				ErrorMessage: "テキスト取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get text histories by text id request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTextHistoriesByTextIDRequestDataError,
			ErrorMessage: "テキスト取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "テキスト取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	texts := []*reps.Text{}
	if request.UpdateTime != nil {
		text := &reps.Text{}
		text, err = repositories.GetText(r.Context(), request.ID, request.UpdateTime)
		texts = []*reps.Text{text}
	} else {
		texts, err = repositories.GetTextHistories(r.Context(), request.ID)
	}

	if err != nil {
		err = fmt.Errorf("error at get text histories by text id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextHistoriesByTextIDError,
			ErrorMessage: "テキスト取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TextHistories = texts
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTextHistoriesByTextIDSuccessMessage,
		Message:     "テキスト取得完了",
	})
}

func (g *GkillServerAPI) HandleGetApplicationConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetApplicationConfigRequest{}
	response := &req_res.GetApplicationConfigResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get applicationConfig response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetApplicationConfigResponseDataError,
				ErrorMessage: "ApplicationConfig取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get applicationConfig request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetApplicationConfigRequestDataError,
			ErrorMessage: "ApplicationConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil || applicationConfig == nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		err = fmt.Errorf("try create application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())

		newApplicationConfig := &user_config.ApplicationConfig{
			UserID:                    userID,
			Device:                    device,
			EnableBrowserCache:        false,
			GoogleMapAPIKey:           "",
			RykvImageListColumnNumber: 3,
			RykvHotReload:             false,
			MiDefaultBoard:            "Inbox",
		}
		_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(r.Context(), newApplicationConfig)
		if err != nil {
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: "ApplicationConfig取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		applicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil {
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: "ApplicationConfig取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	kftlTemplates, err := g.GkillDAOManager.ConfigDAOs.KFTLTemplateDAO.GetKFTLTemplates(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kftlTemplates user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKFTLTemplateError,
			ErrorMessage: "KFTLTemplate取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tagStructs, err := g.GkillDAOManager.ConfigDAOs.TagStructDAO.GetTagStructs(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get tagStructs user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagStructError,
			ErrorMessage: "TagStruct取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repStructs, err := g.GkillDAOManager.ConfigDAOs.RepStructDAO.GetRepStructs(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repStructs user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepStructError,
			ErrorMessage: "RepStruct取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	deviceStructs, err := g.GkillDAOManager.ConfigDAOs.DeviceStructDAO.GetDeviceStructs(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get deviceStructs user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceStructError,
			ErrorMessage: "DeviceStruct取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repTypeStructs, err := g.GkillDAOManager.ConfigDAOs.RepTypeStructDAO.GetRepTypeStructs(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repTypeStructs user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepTypeStructError,
			ErrorMessage: "RepTypeStruct取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	applicationConfig.KFTLTemplate = kftlTemplates
	applicationConfig.TagStruct = tagStructs
	applicationConfig.RepStruct = repStructs
	applicationConfig.DeviceStruct = deviceStructs
	applicationConfig.RepTypeStruct = repTypeStructs
	applicationConfig.AccountIsAdmin = account.IsAdmin
	response.ApplicationConfig = applicationConfig
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetApplicationConfigSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetServerConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetServerConfigsRequest{}
	response := &req_res.GetServerConfigsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get serverConfig response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetServerConfigResponseDataError,
				ErrorMessage: "ServerConfig取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get serverConfig request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetServerConfigRequestDataError,
			ErrorMessage: "ServerConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !account.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", account.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "サーバコンフィグ取得処理二失敗しました。権限がありません。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ServerConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	for _, serverConfig := range serverConfigs {
		accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get all account config")
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetAllAccountConfigError,
				ErrorMessage: "アカウント設定情報の取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		serverConfig.Accounts = accounts

		repositories, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.GetAllRepositories(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get all repositories")
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetAllRepositoriesError,
				ErrorMessage: "Repository全件取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		serverConfig.Repositories = repositories
	}

	response.ServerConfigs = serverConfigs
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetServerConfigSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleUploadFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UploadFilesRequest{}
	response := &req_res.UploadFilesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse upload files response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUploadFilesResponseDataError,
				ErrorMessage: "ファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse upload files request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadFilesRequestDataError,
			ErrorMessage: "ファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "ファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// repNameが一致するIDFRepを取得する
	var targetRep reps.IDFKyouRepository
	for _, idfRep := range repositories.IDFKyouReps {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name from idf rep: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidStatusGetRepNameError,
				ErrorMessage: "ファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if repName == request.TargetRepName {
			targetRep = idfRep
			break
		}
	}

	if targetRep == nil {
		err := fmt.Errorf("error at not found target idf rep %s: %w", request.TargetRepName, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTargetIDFRepError,
			ErrorMessage: "ファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ファイルを保存/IDFを追加する
	savedIDFKyouIDs := []string{}
	gkillErrors := []*message.GkillError{}
	idfKyouCh := make(chan *reps.IDFKyou, len(request.Files))
	gkillErrorCh := make(chan *message.GkillError, len(request.Files))
	defer close(idfKyouCh)
	defer close(gkillErrorCh)
	wg := &sync.WaitGroup{}
	for _, fileInfo := range request.Files {
		repDir, err := targetRep.GetPath(r.Context(), "")
		if err != nil {
			err := fmt.Errorf("error at get target rep path at %s: %w", request.TargetRepName, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "ファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// ファイル名解決
		estimateCreateFileName, err := g.resolveFileName(repDir, fileInfo.FileName, request.ConflictBehavior)
		if err != nil {
			err := fmt.Errorf("error at resolve save file name at %s filename= %s: %w", request.TargetRepName, fileInfo.FileName, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "ファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg.Add(1)
		go func(filename string, base64Data string) {
			// ファイル書き込み
			defer wg.Done()
			var gkillError *message.GkillError
			base64Reader := bufio.NewReader(strings.NewReader(base64Data))
			decoder := base64.NewDecoder(base64.RawStdEncoding.Strict(), base64Reader)

			file, err := os.OpenFile(filename, os.O_CREATE, os.ModePerm)
			if err != nil {
				err := fmt.Errorf("error at open file filename= %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.GetRepPathError,
					ErrorMessage: "ファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}
			defer file.Close()
			io.Copy(file, decoder)

			// idfKyouを作る
			idfKyou := &reps.IDFKyou{
				IsDeleted:    false,
				ID:           GenerateNewID(),
				RelatedTime:  time.Now(),
				CreateTime:   time.Now(),
				CreateApp:    "gkill",
				CreateDevice: device,
				CreateUser:   userID,
				UpdateTime:   time.Now(),
				UpdateApp:    "gkill",
				UpdateUser:   userID,
				UpdateDevice: device,
				TargetFile:   filename,
				RepName:      request.TargetRepName, // 無視される
				DataType:     "idf",                 // 無視される
				FileURL:      "",                    // 無視される
				IsImage:      false,                 //無視される
			}
			idfKyouCh <- idfKyou
		}(estimateCreateFileName, fileInfo.DataBase64)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case gkillError := <-gkillErrorCh:
			if gkillError != nil {
				gkillErrors = append(gkillErrors, gkillError)
			}
		default:
			break errloop
		}
	}
	if len(gkillErrors) != 0 {
		response.Errors = gkillErrors
		return
	}

	repName, err := targetRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError = &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: "ファイルアップロードに失敗しました",
		}
		return
	}

	// idfKyou集約
loop:
	for {
		select {
		case idfKyou := <-idfKyouCh:
			if idfKyou != nil {
				savedIDFKyouIDs = append(savedIDFKyouIDs, idfKyou.ID)
				err = targetRep.AddIDFKyouInfo(r.Context(), idfKyou)
				if err != nil {
					err := fmt.Errorf("error at add idf kyou info at %s: %w", request.TargetRepName, err)
					gkill_log.Debug.Printf(err.Error())
					gkillError = &message.GkillError{
						ErrorCode:    message.GetRepPathError,
						ErrorMessage: "ファイルアップロードに失敗しました",
					}
					return
				}
				_, err = repositories.LatestDataRepositoryAddressDAO.UpdateOrAddLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
					IsDeleted:                idfKyou.IsDeleted,
					TargetID:                 idfKyou.ID,
					DataUpdateTime:           idfKyou.UpdateTime,
					LatestDataRepositoryName: repName,
				})
			}
		default:
			break loop
		}
	}

	kyous := []*reps.Kyou{}
	for _, idfKyouID := range savedIDFKyouIDs {
		kyou, err := targetRep.GetKyou(r.Context(), idfKyouID, nil)
		if err != nil {
			err := fmt.Errorf("error at get kyou at %s: %w", request.TargetRepName, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError = &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "ファイルアップロード後Kyou取得に失敗しました",
			}
			return
		}
		kyous = append(kyous, kyou)
	}

	sort.Slice(kyous, func(i, j int) bool {
		return kyous[i].RelatedTime.After(kyous[j].RelatedTime)
	})

	response.UploadedKyous = kyous
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UploadFilesSuccessMessage,
		Message:     "ファイルアップロードが完了しました",
	})
}

func (g *GkillServerAPI) HandleUploadGPSLogFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UploadGPSLogFilesRequest{}
	response := &req_res.UploadGPSLogFilesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse upload files response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUploadGPSLogFilesResponseDataError,
				ErrorMessage: "GPSLogファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse upload files request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadGPSLogFilesRequestDataError,
			ErrorMessage: "GPSLogファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "GPSLogファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// repNameが一致するGPSLogRepを取得する
	var targetRep reps.GPSLogRepository
	for _, gpsLogRep := range repositories.GPSLogReps {
		repName, err := gpsLogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name from gpsLog rep: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidStatusGetRepNameError,
				ErrorMessage: "GPSLogファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if repName == request.TargetRepName {
			targetRep = gpsLogRep
			break
		}
	}

	if targetRep == nil {
		err := fmt.Errorf("error at not found target gpsLog rep %s: %w", request.TargetRepName, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTargetGPSLogRepError,
			ErrorMessage: "GPSLogファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ファイルを保存/GPSLogを追加する
	repDir := ""
	gkillErrors := []*message.GkillError{}
	gpsLogsCh := make(chan []*reps.GPSLog, len(request.GPSLogFiles))
	gkillErrorCh := make(chan *message.GkillError, len(request.GPSLogFiles))
	defer close(gpsLogsCh)
	defer close(gkillErrorCh)
	wg := &sync.WaitGroup{}
	for _, fileInfo := range request.GPSLogFiles {
		repDir, err = targetRep.GetPath(r.Context(), "")
		if err != nil {
			err := fmt.Errorf("error at get target rep path at %s: %w", request.TargetRepName, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "GPSLogファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg.Add(1)
		go func(filename string, base64Data string) {
			// テンポラリファイル書き込み
			defer wg.Done()
			tempFile, err := os.CreateTemp("gkill_gpx_temp", filename)
			if err != nil {
				err = fmt.Errorf("error at create temp file %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}
			defer tempFile.SetDeadline(time.Now())
			base64Reader := bufio.NewReader(strings.NewReader(base64Data))
			decoder := base64.NewDecoder(base64.RawStdEncoding.Strict(), base64Reader)
			_, err = io.Copy(tempFile, decoder)
			if err != nil {
				err = fmt.Errorf("error at write temp file %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}

			var gkillError *message.GkillError
			// gpsLogを作る
			gpsLog, err := gpslogs.GPSLogFileAsGPSLogs(repDir, filename, request.ConflictBehavior, base64Data)
			if err != nil {
				err := fmt.Errorf("error at gps log file as gpx file filename = %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}
			gpsLogsCh <- gpsLog
		}(fileInfo.FileName, fileInfo.DataBase64)
	}
	wg.Wait()

	// エラー集約
errloop:
	for {
		select {
		case gkillError := <-gkillErrorCh:
			if gkillError != nil {
				gkillErrors = append(gkillErrors, gkillError)
			}
		default:
			break errloop
		}
	}
	if len(gkillErrors) != 0 {
		response.Errors = gkillErrors
		return
	}
	// GPSLogの集約
	uploadedGPSLogs := []*reps.GPSLog{}
loop:
	for {
		select {
		case gpsLogs := <-gpsLogsCh:
			if gpsLogs != nil && len(gpsLogs) != 0 {
				uploadedGPSLogs = append(uploadedGPSLogs, gpsLogs...)
			}
		default:
			break loop
		}
	}

	// 日ごとに分ける
	const dateFormat = "20220401"
	gpsLogDateMap := map[string][]*reps.GPSLog{}
	fileCount := 0
	for _, gpsLog := range uploadedGPSLogs {
		if _, exist := gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)]; !exist {
			gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)] = []*reps.GPSLog{}
		}
		gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)] = append(gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)], gpsLog)
	}
	for _ = range gpsLogDateMap {
		fileCount++
	}

	wg2 := &sync.WaitGroup{}
	gkillErrorCh2 := make(chan *message.GkillError, fileCount)
	defer close(gkillErrorCh2)
	for datestr, gpsLogs := range gpsLogDateMap {
		// ファイル名解決
		filename := fmt.Sprintf("%s.gpx", datestr)
		estimateCreateFileName, err := g.resolveFileName(repDir, filename, request.ConflictBehavior)
		if err != nil {
			err := fmt.Errorf("error at resolve save file name at %s filename= %s: %w", request.TargetRepName, filename, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "GPSLogファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg2.Add(1)
		go func(filename string, gpsLogs []*reps.GPSLog) {
			// Mergeだったら既存のデータも混ぜる
			if request.ConflictBehavior == req_res.Merge {
				startTime, err := time.Parse(dateFormat, datestr)
				endTime := startTime.Add(time.Hour * 24).Add(-time.Millisecond)
				existGPSLogs, err := targetRep.GetGPSLogs(r.Context(), &startTime, &endTime)
				if err != nil {
					err = fmt.Errorf("error at exist gpx datas %s: %w", datestr, err)
					gkillErrorCh2 <- gkillError
					return
				}
				gpsLogs = append(gpsLogs, existGPSLogs...)
			}

			gpxFileContent, err := g.generateGPXFileContent(gpsLogs)
			if err != nil {
				err := fmt.Errorf("error at generate gpx file content filename = %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.GenerateGPXFileContentError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh2 <- gkillError
				return
			}
			file, err := os.OpenFile(filename, os.O_CREATE, os.ModePerm)
			if err != nil {
				err := fmt.Errorf("error at open file filename= %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.GetRepPathError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}
			defer file.Close()
			_, err = file.WriteString(gpxFileContent)
			if err != nil {
				err := fmt.Errorf("error at write gpx content to file filename= %s: %w", filename, err)
				gkill_log.Debug.Printf(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.WriteGPXFileError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}

		}(estimateCreateFileName, gpsLogs)
	}

	// エラー集約
errloop2:
	for {
		select {
		case gkillError := <-gkillErrorCh2:
			if gkillError != nil {
				gkillErrors = append(gkillErrors, gkillError)
			}
		default:
			break errloop2
		}
	}
	if len(gkillErrors) != 0 {
		response.Errors = gkillErrors
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UploadGPSLogFilesSuccessMessage,
		Message:     "GPSLogファイルアップロードが完了しました",
	})
}

func (g *GkillServerAPI) HandleUpdateTagStruct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTagStructRequest{}
	response := &req_res.UpdateTagStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update tagStruct response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTagStructResponseDataError,
				ErrorMessage: "タグ構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update tagStruct request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTagStructRequestDataError,
			ErrorMessage: "タグ構造更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 他ユーザのものが紛れていたら弾く
	for _, tagStruct := range request.TagStruct {
		if tagStruct.UserID != userID {
			err := fmt.Errorf("error at invalid user id user id = %s tag struct user id = %s device = %s: %w", userID, tagStruct.UserID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.TagStructInvalidUserID,
				ErrorMessage: "タグ構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// 一回全部消して全部いれる
	ok, err := g.GkillDAOManager.ConfigDAOs.TagStructDAO.DeleteUsersTagStructs(r.Context(), userID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete users tag structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteUsersTagStructError,
			ErrorMessage: "タグ構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	ok, err = g.GkillDAOManager.ConfigDAOs.TagStructDAO.AddTagStructs(r.Context(), request.TagStruct)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add tag structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUsersTagStructError,
			ErrorMessage: "タグ構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ApplicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "タグ構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTagStructSuccessMessage,
		Message:     "タグ構造を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateApplicationConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateApplicationConfigRequest{}
	response := &req_res.UpdateApplicationConfigResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update application config response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateApplicationconfigResponseDataError,
				ErrorMessage: "設定更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update application config request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateApplicationConfigRequestDataError,
			ErrorMessage: "設定更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	request.ApplicationConfig.UserID = userID
	request.ApplicationConfig.Device = device

	// ApplicationConfigを更新する
	ok, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.UpdateApplicationConfig(r.Context(), &request.ApplicationConfig)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update application config user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateApplicationConfigError,
			ErrorMessage: "設定更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateApplicationConfigSuccessMessage,
		Message:     "設定を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateRepStruct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateRepStructRequest{}
	response := &req_res.UpdateRepStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update repStruct response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateRepStructResponseDataError,
				ErrorMessage: "Rep構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update repStruct request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateRepStructRequestDataError,
			ErrorMessage: "Rep構造更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 他ユーザのものが紛れていたら弾く
	for _, repStruct := range request.RepStruct {
		if repStruct.UserID != userID {
			err := fmt.Errorf("error at invalid user id user id = %s rep struct user id = %s device = %s: %w", userID, repStruct.UserID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.RepStructInvalidUserID,
				ErrorMessage: "Rep構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// 一回全部消して全部いれる
	ok, err := g.GkillDAOManager.ConfigDAOs.RepStructDAO.DeleteUsersRepStructs(r.Context(), userID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete users rep structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteUsersRepStructError,
			ErrorMessage: "Rep構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	ok, err = g.GkillDAOManager.ConfigDAOs.RepStructDAO.AddRepStructs(r.Context(), request.RepStruct)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add rep structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUsersRepStructError,
			ErrorMessage: "Rep構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ApplicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "Rep構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateRepStructSuccessMessage,
		Message:     "Rep構造を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateDeviceStruct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateDeviceStructRequest{}
	response := &req_res.UpdateDeviceStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update deviceStruct response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateDeviceStructResponseDataError,
				ErrorMessage: "Device構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update deviceStruct request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateDeviceStructRequestDataError,
			ErrorMessage: "Device構造更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 他ユーザのものが紛れていたら弾く
	for _, deviceStruct := range request.DeviceStruct {
		if deviceStruct.UserID != userID {
			err := fmt.Errorf("error at invalid user id user id = %s device struct user id = %s device = %s: %w", userID, deviceStruct.UserID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.DeviceStructInvalidUserID,
				ErrorMessage: "Device構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// 一回全部消して全部いれる
	ok, err := g.GkillDAOManager.ConfigDAOs.DeviceStructDAO.DeleteUsersDeviceStructs(r.Context(), userID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete users device structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteUsersDeviceStructError,
			ErrorMessage: "Device構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	ok, err = g.GkillDAOManager.ConfigDAOs.DeviceStructDAO.AddDeviceStructs(r.Context(), request.DeviceStruct)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add device structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUsersDeviceStructError,
			ErrorMessage: "Device構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ApplicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "Device構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateDeviceStructSuccessMessage,
		Message:     "Device構造を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateRepTypeStruct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateRepTypeStructRequest{}
	response := &req_res.UpdateRepTypeStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update repTypeStruct response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateRepTypeStructResponseDataError,
				ErrorMessage: "RepType構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update repTypeStruct request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateRepTypeStructRequestDataError,
			ErrorMessage: "RepType構造更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 他ユーザのものが紛れていたら弾く
	for _, repTypeStruct := range request.RepTypeStruct {
		if repTypeStruct.UserID != userID {
			err := fmt.Errorf("error at invalid user id user id = %s repType struct user id = %s device = %s: %w", userID, repTypeStruct.UserID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.RepTypeStructInvalidUserID,
				ErrorMessage: "RepType構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// 一回全部消して全部いれる
	ok, err := g.GkillDAOManager.ConfigDAOs.RepTypeStructDAO.DeleteUsersRepTypeStructs(r.Context(), userID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete users repType structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteUsersRepTypeStructError,
			ErrorMessage: "RepType構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	ok, err = g.GkillDAOManager.ConfigDAOs.RepTypeStructDAO.AddRepTypeStructs(r.Context(), request.RepTypeStruct)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add repType structs user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUsersRepTypeStructError,
			ErrorMessage: "RepType構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ApplicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "RepType構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateRepTypeStructSuccessMessage,
		Message:     "RepType構造を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateKFTLTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateKFTLTemplateRequest{}
	response := &req_res.UpdateKFTLTemplateResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update kftl template response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateKFTLTemplateResponseDataError,
				ErrorMessage: "KFTLテンプレート構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update kftl template request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateKFTLTemplateRequestDataError,
			ErrorMessage: "KFTLテンプレート構造更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 他ユーザのものが紛れていたら弾く
	for _, kftlTemplate := range request.KFTLTemplates {
		if kftlTemplate.UserID != userID {
			err := fmt.Errorf("error at invalid user id user id = %s kftl template user id = %s device = %s: %w", userID, kftlTemplate.UserID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.KFTLTemplateStructInvalidUserID,
				ErrorMessage: "KFTLテンプレート構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// 一回全部消して全部いれる
	ok, err := g.GkillDAOManager.ConfigDAOs.KFTLTemplateDAO.DeleteUsersKFTLTemplates(r.Context(), userID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete users kftl tempates user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteUsersKFTLTemplateError,
			ErrorMessage: "KFTLテンプレート構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	ok, err = g.GkillDAOManager.ConfigDAOs.KFTLTemplateDAO.AddKFTLTemplates(r.Context(), request.KFTLTemplates)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add kftl templates user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUsersKFTLTemplateError,
			ErrorMessage: "KFTLテンプレート構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ApplicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "KFTLテンプレート構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateKFTLTemplateSuccessMessage,
		Message:     "KFTLテンプレート構造を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateAccountStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateAccountStatusRequest{}
	response := &req_res.UpdateAccountStatusResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update accountStatus response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateAccountStatusResponseDataError,
				ErrorMessage: "アカウントステータス構造更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update accountStatus request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateAccountStatusRequestDataError,
			ErrorMessage: "アカウントステータス構造更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	requesterAccount, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := requesterAccount.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterAccount.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "アカウントステータス更新処理に失敗しました。権限がありません。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	targetAccountUpdated := &account.Account{
		UserID:             targetAccount.UserID,
		PasswordSha256:     targetAccount.PasswordSha256,
		IsAdmin:            targetAccount.IsAdmin,
		IsEnable:           request.Enable,
		PasswordResetToken: targetAccount.PasswordResetToken,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.UpdateAccount(r.Context(), targetAccountUpdated)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update users account user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateUsersAccountStatusError,
			ErrorMessage: "アカウントステータス構造更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateAccountStatusSuccessMessage,
		Message:     "アカウントステータス構造を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateUserReps(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateUserRepsRequest{}
	response := &req_res.UpdateUserRepsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update userReps response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateUserRepsResponseDataError,
				ErrorMessage: "Rep更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update userReps request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateUserRepsRequestDataError,
			ErrorMessage: "Rep更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	requesterAccount, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := requesterAccount.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterAccount.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "Rep更新に失敗しました。権限がありません",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "Rep更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	userID = targetAccount.UserID
	device = ""

	ok, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.DeleteAllRepositoriesByUser(r.Context(), userID, device)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete all repositories by users user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteAllRepositoriesByUserError,
			ErrorMessage: "Re",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ok, err = g.GkillDAOManager.ConfigDAOs.RepositoryDAO.AddRepositories(r.Context(), request.UpdatedReps)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete add all repositories by users user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUpdatedRepositoriesByUser,
			ErrorMessage: "Re",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateRepositoriesSuccessMessage,
		Message:     "Rep更新に成功しました",
	})
}

func (g *GkillServerAPI) HandleUpdateServerConfigs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateServerConfigsRequest{}
	response := &req_res.UpdateServerConfigsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update server config response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateServerConfigResponseDataError,
				ErrorMessage: "設定更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update server config request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateServerConfigRequestDataError,
			ErrorMessage: "設定更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// adminじゃなかったら弾く
	if !account.IsAdmin {
		err = fmt.Errorf("%s is not admin", userID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "管理者権限を所有していません",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ServerConfigを更新する
	ok, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.DeleteWriteServerConfigs(r.Context(), request.ServerConfigs)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at update server config user user id = %s device = %s id = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateServerConfigError,
			ErrorMessage: "設定更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateServerConfigSuccessMessage,
		Message:     "設定を更新しました",
	})
}

func (g *GkillServerAPI) HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddAccountRequest{}
	response := &req_res.AddAccountResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add account response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddAccountResponseDataError,
				ErrorMessage: "account追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add account request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddAccountRequestDataError,
			ErrorMessage: "account追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	requesterAccount, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := requesterAccount.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", userID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "アカウント追加に失敗しました。権限がありません。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.AccountInfo.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user device = %s id = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: "Account追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existAccount != nil {
		err = fmt.Errorf("exist account id = %s", userID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistAccountError,
			ErrorMessage: "Account追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント情報を追加
	passwordResetToken := GenerateNewID()
	newAccount := &account.Account{
		UserID:             request.AccountInfo.UserID,
		IsAdmin:            false,
		IsEnable:           true,
		PasswordResetToken: &passwordResetToken,
	}
	ok, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(r.Context(), newAccount)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add account user id = %s device = %s account = %#v: %w", userID, device, request.AccountInfo, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddAccountError,
			ErrorMessage: "Account追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err = g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.AccountInfo.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: "account追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	applicationConfig := &user_config.ApplicationConfig{
		UserID:                    request.AccountInfo.UserID,
		Device:                    device,
		EnableBrowserCache:        false,
		GoogleMapAPIKey:           "",
		RykvImageListColumnNumber: 3,
		RykvHotReload:             false,
		MiDefaultBoard:            "Inbox",
	}
	_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(context.TODO(), applicationConfig)
	if err != nil {
		err = fmt.Errorf("error at add application config user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddApplicationConfig,
			ErrorMessage: "account追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.DoInitialize {
		err := g.initializeNewUserReps(r.Context(), requesterAccount)
		if err != nil {
			err = fmt.Errorf("error at initialize new user reps user id = %s device = %s account = %#v: %w", userID, device, request.AccountInfo, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddAccountError,
				ErrorMessage: "Account追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.AddedAccountInfo = requesterAccount
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddAccountSuccessMessage,
		Message:     "accountを追加しました",
	})
}

func (g *GkillServerAPI) HandleGenerateTLSFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GenerateTLSFileRequest{}
	response := &req_res.GenerateTLSFileResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse generate tls to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidGenerateTLSFileResponseDataError,
				ErrorMessage: "TLSファイル作成処理に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse generate tls request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidGenerateTLSFileRequestDataError,
			ErrorMessage: "TLSファイル作成処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// TLSファイル作成操作をしたユーザを特定
	requesterSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: "TLSファイル作成処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), requesterSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", requesterSession.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "TLSファイル作成処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterSession.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。権限がありません。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certFileName, pemFileName, err := g.getTLSFileNames(device)
	if err != nil {
		err = fmt.Errorf("error at get tls file names: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTLSFileNamesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	certFileName, pemFileName = os.ExpandEnv(certFileName), os.ExpandEnv(pemFileName)
	certFileName, pemFileName = filepath.ToSlash(certFileName), filepath.ToSlash(pemFileName)

	// あったら消す
	if _, err := os.Stat(certFileName); err == nil {
		err := os.Remove(certFileName)
		if err != nil {
			err = fmt.Errorf("error at remove cert file %s: %w", certFileName, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.RemoveCertFileError,
				ErrorMessage: "TLSファイル作成処理に失敗しました。",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	if _, err := os.Stat(pemFileName); err == nil {
		err := os.Remove(pemFileName)
		if err != nil {
			err = fmt.Errorf("error at remove pem file %s: %w", pemFileName, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.RemovePemFileError,
				ErrorMessage: "TLSファイル作成処理に失敗しました。",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	hostStr := "localhost"
	ecdsaCurveStr := ""
	ed25519KeyBool := false
	rsaBitsInt := 2048
	validFromStr := ""
	validForDuration := 365 * 24 * time.Hour
	isCABool := false
	host := &hostStr
	ecdsaCurve := &ecdsaCurveStr
	ed25519Key := &ed25519KeyBool
	rsaBits := &rsaBitsInt
	validFrom := &validFromStr
	validFor := &validForDuration
	isCA := &isCABool
	if len(*host) == 0 {
		gkill_log.Debug.Printf("Missing required --host parameter")
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	var priv any
	switch *ecdsaCurve {
	case "":
		if *ed25519Key {
			_, priv, err = ed25519.GenerateKey(rand.Reader)
		} else {
			priv, err = rsa.GenerateKey(rand.Reader, *rsaBits)
		}
	case "P224":
		priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		gkill_log.Debug.Printf("Unrecognized elliptic curve: %q", *ecdsaCurve)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err != nil {
		gkill_log.Debug.Printf("Failed to generate private key: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ECDSA, ED25519 and RSA subject keys should have the DigitalSignature
	// KeyUsage bits set in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature
	// Only RSA subject keys should have the KeyEncipherment KeyUsage bits set. In
	// the context of TLS this KeyUsage is particular to RSA key exchange and
	// authentication.
	if _, isRSA := priv.(*rsa.PrivateKey); isRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}

	var notBefore time.Time
	if len(*validFrom) == 0 {
		notBefore = time.Now()
	} else {
		notBefore, err = time.Parse("Jan 2 15:04:05 2006", *validFrom)
		if err != nil {
			gkill_log.Debug.Printf("Failed to parse creation date: %v", err)
			err = fmt.Errorf("error at generate tls files")
			gkillError := &message.GkillError{
				ErrorCode:    message.GenerateTLSFilesError,
				ErrorMessage: "TLSファイル作成処理に失敗しました。",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	notAfter := notBefore.Add(*validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		gkill_log.Debug.Printf("Failed to generate serial number: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(*host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	if *isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		gkill_log.Debug.Printf("Failed to create certificate: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	parentDirCert, parentDirKey := filepath.Dir(certFileName), filepath.Dir(pemFileName)
	parentDirCert, parentDirKey = filepath.ToSlash(parentDirCert), filepath.ToSlash((parentDirKey))

	err = os.MkdirAll(parentDirCert, os.ModePerm)
	if err != nil {
		gkill_log.Debug.Printf("Failed to open cert.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = os.MkdirAll(parentDirKey, os.ModePerm)
	if err != nil {
		gkill_log.Debug.Printf("Failed to open cert.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certOut, err := os.Create(certFileName)
	if err != nil {
		gkill_log.Debug.Printf("Failed to open cert.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		gkill_log.Debug.Printf("Failed to write data to cert.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := certOut.Close(); err != nil {
		gkill_log.Debug.Printf("Error closing cert.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	log.Print("wrote cert.pem\n")

	keyOut, err := os.OpenFile(pemFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		gkill_log.Debug.Printf("Failed to open key.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		gkill_log.Debug.Printf("Unable to marshal private key: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		gkill_log.Debug.Printf("Failed to write data to key.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := keyOut.Close(); err != nil {
		gkill_log.Debug.Printf("Error closing key.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	log.Print("wrote key.pem\n")
}

func (g *GkillServerAPI) HandleGetGPSLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGPSLogRequest{}
	response := &req_res.GetGPSLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gpsLog response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetGPSLogResponseDataError,
				ErrorMessage: "GPSLog取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get gpsLog request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetGPSLogRequestDataError,
			ErrorMessage: "GPSLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "GPSLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	gpsLogHistories, err := repositories.GPSLogReps.GetGPSLogs(r.Context(), &request.StartDate, &request.EndDate)
	if err != nil {
		err = fmt.Errorf("error at get gpsLog user id = %s device = %s start time = %s end time = %s: %w", userID, device, request.StartDate.Format(time.RFC3339), request.EndDate.Format(time.RFC3339), err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetGPSLogError,
			ErrorMessage: "GPSLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GPSLogs = gpsLogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGPSLogSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetKFTLTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKFTLTemplatesRequest{}
	response := &req_res.GetKFTLTemplateResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kftl template response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKFTLTemplateResponseDataError,
				ErrorMessage: "kftlTemplate取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kftl template request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKFTLTemplateRequestDataError,
			ErrorMessage: "kftlTemplate取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kftlTemplates, err := g.GkillDAOManager.ConfigDAOs.KFTLTemplateDAO.GetKFTLTemplates(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kftlTemplates user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKFTLTemplateError,
			ErrorMessage: "KFTLTemplate取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KFTLTemplates = kftlTemplates
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetApplicationConfigSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetGkillInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGkillInfoRequest{}
	response := &req_res.GetGkillInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gkillInfo response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetGkillInfoResponseDataError,
				ErrorMessage: "GkillInfo取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get gkillInfo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetGkillInfoRequestDataError,
			ErrorMessage: "GkillInfo取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UserID = userID
	response.Device = device
	response.UserIsAdmin = account.IsAdmin
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGkillInfoSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleAddShareMiTaskListInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddShareMiTaskListInfoRequest{}
	response := &req_res.AddShareMiTaskListInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add shareMiTaskListInfo response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddShareMiTaskListInfoResponseDataError,
				ErrorMessage: "shareMiTaskListInfo追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add shareMiTaskListInfo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddShareMiTaskListInfoRequestDataError,
			ErrorMessage: "shareMiTaskListInfo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existShareMiTaskListInfo, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.GetMiShareInfo(r.Context(), request.ShareMiTaskListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get shareMiTaskListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareMiTaskListInfo.ShareID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareMiTaskListInfoError,
			ErrorMessage: "ShareMiTaskListInfo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareMiTaskListInfo != nil {
		err = fmt.Errorf("not exist shareMiTaskListInfo id = %s", request.ShareMiTaskListInfo.ShareID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistShareMiTaskListInfoError,
			ErrorMessage: "ShareMiTaskListInfo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miShareInfo := &mi_share_info.MiShareInfo{
		ID:            GenerateNewID(),
		UserID:        request.ShareMiTaskListInfo.UserID,
		Device:        request.ShareMiTaskListInfo.Device,
		ShareTitle:    request.ShareMiTaskListInfo.ShareTitle,
		IsShareDetail: request.ShareMiTaskListInfo.IsShareDetail,
		ShareID:       request.ShareMiTaskListInfo.ShareID,
		FindQueryJSON: request.ShareMiTaskListInfo.FindQueryJSON,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.AddMiShareInfo(r.Context(), miShareInfo)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add shareMiTaskListInfo user id = %s device = %s shareMiTaskListInfo = %#v: %w", userID, device, request.ShareMiTaskListInfo, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddShareMiTaskListInfoError,
			ErrorMessage: "ShareMiTaskListInfo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	shareMiTaskListInfo, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.GetMiShareInfo(r.Context(), request.ShareMiTaskListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get shareMiTaskListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareMiTaskListInfo.ShareID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareMiTaskListInfoError,
			ErrorMessage: "shareMiTaskListInfo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareMiTaskListInfo = shareMiTaskListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddShareMiTaskListInfoSuccessMessage,
		Message:     "shareMiTaskListInfoを追加しました",
	})
}

func (g *GkillServerAPI) HandleUpdateShareMiTaskListInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateShareMiTaskListInfoRequest{}
	response := &req_res.UpdateShareMiTaskListInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add shareMiTaskListInfo response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateShareMiTaskListInfoResponseDataError,
				ErrorMessage: "shareMiTaskListInfo更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add shareMiTaskListInfo request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateShareMiTaskListInfoRequestDataError,
			ErrorMessage: "shareMiTaskListInfo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない
	existShareMiTaskListInfo, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.GetMiShareInfo(r.Context(), request.ShareMiTaskListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get shareMiTaskListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareMiTaskListInfo.ShareID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareMiTaskListInfoError,
			ErrorMessage: "ShareMiTaskListInfo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareMiTaskListInfo == nil {
		err = fmt.Errorf("not exist shareMiTaskListInfo id = %s", request.ShareMiTaskListInfo.ShareID)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotExistShareMiTaskListInfoError,
			ErrorMessage: "ShareMiTaskListInfo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miShareInfo := &mi_share_info.MiShareInfo{
		ID:            GenerateNewID(),
		UserID:        request.ShareMiTaskListInfo.UserID,
		Device:        request.ShareMiTaskListInfo.Device,
		ShareTitle:    request.ShareMiTaskListInfo.ShareTitle,
		IsShareDetail: request.ShareMiTaskListInfo.IsShareDetail,
		ShareID:       request.ShareMiTaskListInfo.ShareID,
		FindQueryJSON: request.ShareMiTaskListInfo.FindQueryJSON,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.UpdateMiShareInfo(r.Context(), miShareInfo)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add shareMiTaskListInfo user id = %s device = %s shareMiTaskListInfo = %#v: %w", userID, device, request.ShareMiTaskListInfo, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateShareMiTaskListInfoError,
			ErrorMessage: "ShareMiTaskListInfo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	shareMiTaskListInfo, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.GetMiShareInfo(r.Context(), request.ShareMiTaskListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get shareMiTaskListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareMiTaskListInfo.ShareID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareMiTaskListInfoError,
			ErrorMessage: "shareMiTaskListInfo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareMiTaskListInfo = shareMiTaskListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateShareMiTaskListInfoSuccessMessage,
		Message:     "shareMiTaskListInfoを更新しました",
	})
}

func (g *GkillServerAPI) HandleGetShareMiTaskListInfos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetShareMiTaskListInfosRequest{}
	response := &req_res.GetShareMiTaskListInfosResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get shareMiTaskListInfos response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetShareMiTaskListInfosResponseDataError,
				ErrorMessage: "ShareMiTaskListInfos取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get shareMiTaskListInfos request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetShareMiTaskListInfosRequestDataError,
			ErrorMessage: "ShareMiTaskListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	shareMiTaskList, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.GetMiShareInfos(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get shareMiTaskListInfos user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareMiTaskListInfosError,
			ErrorMessage: "ShareMiTaskListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareMiTaskListInfos = shareMiTaskList
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetShareMiTaskListInfosSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleDeleteShareMiTaskListInfos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DeleteShareMiTaskListInfoRequest{}
	response := &req_res.DeleteShareMiTaskListInfosResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete shareMiTaskListInfos response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidDeleteShareMiTaskListInfosResponseDataError,
				ErrorMessage: "ShareMiTaskListInfos削除に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete shareMiTaskListInfos request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDeleteShareMiTaskListInfosRequestDataError,
			ErrorMessage: "ShareMiTaskListInfos削除に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを削除
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.DeleteMiShareInfo(r.Context(), request.ShareMiTaskListInfo.ShareID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete shareMiTaskListInfos user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteShareMiTaskListInfosError,
			ErrorMessage: "ShareMiTaskListInfos削除に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.DeleteShareMiTaskListInfosSuccessMessage,
		Message:     "削除完了",
	})
}

func (g *GkillServerAPI) HandleGetMiSharedTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSharedMiTasksRequest{}
	response := &req_res.GetSharedMiTasksResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete shareMiTaskListInfos response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiSharedTasksResponseDataError,
				ErrorMessage: "ShareMiTaskListInfos取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete shareMiTaskListInfos request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTasksRequestDataError,
			ErrorMessage: "ShareMiTaskListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sharedMiInfo, err := g.GkillDAOManager.ConfigDAOs.MiShareInfoDAO.GetMiShareInfo(r.Context(), request.SharedID)
	if err != nil {
		err = fmt.Errorf("error at get shareMiTaskListInfos shared id = %s: %w", request.SharedID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiSharedTasksError,
			ErrorMessage: "ShareMiTaskListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := sharedMiInfo.UserID
	device := sharedMiInfo.Device

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	findQuery := &find.FindQuery{}
	err = json.Unmarshal([]byte(sharedMiInfo.FindQueryJSON), findQuery)
	if err != nil {
		err = fmt.Errorf("error at parse query json at find kyous %s: %w", findQuery, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTaskRequest,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// Kyou
	kyous, err := repositories.MiReps.FindKyous(r.Context(), findQuery)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareMiError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// Mi
	mis, err := repositories.MiReps.FindMi(r.Context(), findQuery)
	if err != nil {
		err = fmt.Errorf("error at find Mis user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.FindMisShareMiError,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// Tag
	tags := []*reps.Tag{}
	for _, mi := range mis {
		tagsRelatedID, err := repositories.GetTagsByTargetID(r.Context(), mi.ID)
		if err != nil {
			err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindTagsShareMiError,
				ErrorMessage: "タグ取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		tags = append(tags, tagsRelatedID...)
	}

	// Text
	texts := []*reps.Text{}
	for _, mi := range mis {
		textsRelatedID, err := repositories.GetTextsByTargetID(r.Context(), mi.ID)
		if err != nil {
			err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindTextsShareMiError,
				ErrorMessage: "テキスト取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		texts = append(texts, textsRelatedID...)
	}

	// TimeIs
	// not implements
	timeiss := []*reps.TimeIs{}

	response.MiKyous = kyous
	response.Mis = mis
	response.Tags = tags
	response.Texts = texts
	response.TimeIss = timeiss
	response.Title = sharedMiInfo.ShareTitle
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSharedTasksSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetRepositories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetRepositoriesRequest{}
	response := &req_res.GetRepositoriesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get repositories response to json: %w", err)
			gkill_log.Debug.Printf(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetRepositoriesResponseDataError,
				ErrorMessage: "Repositoriesの取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get repositories request to json: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetRepositoriesRequestDataError,
			ErrorMessage: "Repositoriesの取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.GetRepositories(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: "Repositoriesの取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	for _, repository := range repositories {
		repository.File = ""
	}

	response.Repositories = repositories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetRepositoriesSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) getAccountFromSessionID(ctx context.Context, sessionID string) (*account.Account, *message.GkillError, error) {

	loginSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(ctx, sessionID)
	if loginSession == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", sessionID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: "アカウント認証に失敗しました",
		}
		return nil, gkillError, err
	}

	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, loginSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "アカウント認証に失敗しました",
		}
		return nil, gkillError, err
	}

	if !account.IsEnable {
		err = fmt.Errorf("error at disable account user id = %s: %w", loginSession.UserID, err)
		gkill_log.Debug.Printf(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountDisabledError,
			ErrorMessage: "アカウントが無効化されています",
		}
		return nil, gkillError, err
	}

	return account, nil, nil
}

func GenerateNewID() string {
	return uuid.New().String()
}

func (g *GkillServerAPI) resolveFileName(repDir string, filename string, behavior req_res.FileUploadConflictBehavior) (string, error) {
	fullFilename := filepath.Join(repDir, filename)
	_, err := os.Stat(fullFilename)
	if err == nil {
		return fullFilename, nil
	} else {
		switch behavior {
		case req_res.Override:
			return fullFilename, nil
		case req_res.Rename:
			// カッコのついていないファイル名。例えば「hogehoge (1).txt」なら「hogehoge.txt」。
			planeFileName := g.planeFileName(fullFilename)
			ext := filepath.Ext(planeFileName)
			withoutExt := planeFileName[:len(planeFileName)-len(ext)]

			// ファイルが存在しない名前になるまでカッコ内の数字をインクリメントし続ける
			// targetFilenameは最終的な移動先ファイル名
			fullFilename = planeFileName
			for count := 0; ; count++ {
				if _, err := os.Stat(fullFilename); err != nil {
					break
				}
				// 初回は無視。count:=0としないほうがfor文としてきれいに収まる
				if count == 0 {
					continue
				}
				fullFilename = os.Expand("${name} (${count})${ext}", func(str string) string {
					switch str {
					case "name":
						return withoutExt
					case "count":
						return strconv.Itoa(count)
					case "ext":
						return ext
					}
					return ""
				})
			}
			return fullFilename, nil
		case req_res.Merge:
			return fullFilename, nil
		}
	}
	err = fmt.Errorf("does not set file upload conflict behavior")
	return "", err
}

func (g *GkillServerAPI) generateGPXFileContent(gpsLogs []*reps.GPSLog) (string, error) {
	gpxData := &gpx.GPX{}
	gpxData.Trk = []*gpx.TrkType{&gpx.TrkType{}}
	gpxData.Trk[0].TrkSeg = []*gpx.TrkSegType{&gpx.TrkSegType{}}
	trkPts := []*gpx.WptType{}
	for _, gpslog := range gpsLogs {
		trkPts = append(trkPts, &gpx.WptType{
			Time: gpslog.RelatedTime,
			Lat:  gpslog.Latitude,
			Lon:  gpslog.Longitude,
		})
	}
	gpxData.Trk[0].TrkSeg[0].TrkPt = trkPts

	gpsDataResult := ""
	writer := bufio.NewWriter(bytes.NewBufferString(gpsDataResult))
	err := gpxData.Write(writer)
	if err != nil {
		err = fmt.Errorf("error at write gpx data: %w", err)
		return "", err
	}

	err = writer.Flush()
	if err != nil {
		err = fmt.Errorf("error at write gpx data flush: %w", err)
		return "", err
	}

	return gpsDataResult, nil
}

func (g *GkillServerAPI) initializeNewUserReps(ctx context.Context, account *account.Account) error {
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		return err
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(ctx, device)
	if err != nil {
		err = fmt.Errorf("error at get server config: %w", err)
		return err
	}

	userDataRootDirectory := filepath.Join(serverConfig.UserDataDirectory, account.UserID)
	if _, err := os.Stat(os.ExpandEnv(userDataRootDirectory)); err == nil {
		err := fmt.Errorf("error at initialize new user reps. user root directory aleady exist %s: %w", userDataRootDirectory, err)
		return err
	} else {
		err := os.MkdirAll(os.ExpandEnv(userDataRootDirectory), fs.ModePerm)
		if err != nil {
			err = fmt.Errorf("error at initialize new user reps. error at create directory %s: %w", userDataRootDirectory, err)
			return err
		}
	}

	repTypeFileNameMap := map[string]string{}
	repTypeFileNameMap["kmemo"] = "Kmemo.db"
	repTypeFileNameMap["urlog"] = "URLog.db"
	repTypeFileNameMap["timeis"] = "TimeIs.db"
	repTypeFileNameMap["mi"] = "Mi.db"
	repTypeFileNameMap["nlog"] = "Nlog.db"
	repTypeFileNameMap["lantana"] = "Lantana.db"
	repTypeFileNameMap["tag"] = "Tag.db"
	repTypeFileNameMap["text"] = "Text.db"
	repTypeFileNameMap["rekyou"] = "ReKyou.db"

	for repType, repFileName := range repTypeFileNameMap {
		repFileFullName := filepath.Join(userDataRootDirectory, repFileName)
		repFile, err := os.Create(os.ExpandEnv(repFileFullName))
		if err != nil {
			err = fmt.Errorf("error at create rep file %s: %w", repFileFullName, err)
			return err
		}
		err = repFile.Close()
		if err != nil {
			err = fmt.Errorf("error at close rep file %s: %w", repFileFullName, err)
			return err
		}

		repository := &user_config.Repository{
			ID:                     GenerateNewID(),
			UserID:                 account.UserID,
			Device:                 device,
			Type:                   repType,
			File:                   repFileFullName,
			UseToWrite:             true,
			IsExecuteIDFWhenReload: true,
			IsEnable:               true,
		}
		_, err = g.GkillDAOManager.ConfigDAOs.RepositoryDAO.AddRepository(ctx, repository)
		if err != nil {
			err = fmt.Errorf("error at initialize new user reps. error at add repository reptype = %s repfilename = %s: %w", repType, repFileFullName, err)
			return err
		}
	}

	repType, repFileName := "directory", "Files"
	repFileFullName := filepath.Join(userDataRootDirectory, repFileName)
	err = os.MkdirAll(os.ExpandEnv(repFileFullName), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository create directory reptype = %s repdirname = %s: %w", repType, repFileFullName, err)
		return err
	}
	repository := &user_config.Repository{
		ID:                     GenerateNewID(),
		UserID:                 account.UserID,
		Device:                 device,
		Type:                   repType,
		File:                   repFileFullName,
		UseToWrite:             true,
		IsExecuteIDFWhenReload: true,
		IsEnable:               true,
	}
	_, err = g.GkillDAOManager.ConfigDAOs.RepositoryDAO.AddRepository(ctx, repository)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository reptype = %s repfilename = %s: %w", repType, repFileFullName, err)
		return err
	}

	repType, repFileName = "gpslog", "GPSLog"
	repFileFullName = filepath.Join(userDataRootDirectory, repFileName)
	err = os.MkdirAll(os.ExpandEnv(repFileFullName), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository create directory reptype = %s repdirname = %s: %w", repType, repFileFullName, err)
		return err
	}
	repository = &user_config.Repository{
		ID:                     GenerateNewID(),
		UserID:                 account.UserID,
		Device:                 device,
		Type:                   repType,
		File:                   repFileFullName,
		UseToWrite:             true,
		IsExecuteIDFWhenReload: true,
		IsEnable:               true,
	}
	_, err = g.GkillDAOManager.ConfigDAOs.RepositoryDAO.AddRepository(ctx, repository)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository reptype = %s repfilename = %s: %w", repType, repFileFullName, err)
		return err
	}

	return nil
}

// ファイル名に(n)がついていたら除去して返します。
// hogehoge.txt (1) (1) (1)とかにならないように。
// Windowsのファイル重複時Suffixに対応しています。？
func (g *GkillServerAPI) planeFileName(filename string) (fixedfilename string) {
	_ = "${name} (${count})${ext}" //このフォーマットが対象です。

	ext := filepath.Ext(filename)
	fnwithoutext := filename[:len(filename)-len(ext)]

	//それぞれLastIndex
	lindexP := strings.LastIndexAny(fnwithoutext, " (") //スペースがあります。
	lindexS := strings.LastIndexAny(fnwithoutext, ")")
	if lindexP != -1 && lindexS != -1 && //(と)が含まれていて、
		lindexS == len(fnwithoutext)-1 && //)が一番最後で、
		lindexP < lindexS { //)よりも(が前にあり、
		//その上括弧の間が数字であるとき、それは${count}でつけられたsuffixでありえる。
		num := fnwithoutext[lindexP+1 : lindexS] //スペース分+1
		_, err := strconv.Atoi(num)
		if err == nil {
			//${count}部分を除去して返す
			fnwithoutext = fnwithoutext[:len(fnwithoutext)-(len(num)+3)] //+3はカッコ2つとスペース分
			filename = fnwithoutext + ext
			return filename
		}
	}
	//${count}部分がなければそのまま返す
	return filename
}

func (g *GkillServerAPI) getTLSFileNames(device string) (certFileName string, pemFileName string, err error) {
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		gkill_log.Debug.Printf(err.Error())
		return "", "", err
	}
	return serverConfig.TLSCertFile, serverConfig.TLSKeyFile, nil
}

func (g *GkillServerAPI) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	// クッキーを見て認証する
	sessionIDCookie, err := r.Cookie("gkill_session_id")
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Debug.Printf("%#v", err)
		return
	}
	sessionID := sessionIDCookie.Value

	// アカウントを取得
	// NGであれば403でreturn
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), sessionID)
	if account == nil || gkillError != nil || err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Debug.Printf("%#v", err)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Debug.Printf("%#v", err)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Debug.Printf("%#v", err)
		return
	}

	// リクエストPathから対象Rep名を抽出
	targetRepName := strings.SplitN(r.URL.Path, "/", 4)[2]

	// OKであればRepNameが一致するIDFRepを探す
	var targetIDFRep reps.IDFKyouRepository
	for _, idfRep := range repositories.IDFKyouReps {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("error at handle file serve: %w", err)
			gkill_log.Debug.Printf("%#v", err)
			return
		}
		if repName == targetRepName {
			targetIDFRep = idfRep
			break
		}
	}

	if targetIDFRep == nil {
		w.WriteHeader(http.StatusNotFound)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Debug.Printf("%#v", err)
		return
	}

	// StripPrefixしてIDFサーバのハンドラにわたす
	rootAddress := "/files/" + targetRepName
	http.StripPrefix(rootAddress, http.HandlerFunc(targetIDFRep.HandleFileServe)).ServeHTTP(w, r)
}

func (g *GkillServerAPI) GetDevice() (string, error) {
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		gkill_log.Debug.Printf(err.Error())
		return "", err
	}

	var device *string
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			if device != nil {
				err = fmt.Errorf("invalid status. enable device count is not 1.")
				return "", err
			}
			device = &serverConfig.Device
		}
	}
	if device == nil {
		err = fmt.Errorf("invalid status. enable device count is not 1.")
		return "", err
	}
	g.device = *device
	return g.device, nil
}

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}
