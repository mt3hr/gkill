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
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/app/gkill/api/find"
	"github.com/mt3hr/gkill/src/app/gkill/api/gpslogs"
	"github.com/mt3hr/gkill/src/app/gkill/api/message"
	"github.com/mt3hr/gkill/src/app/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/app/gkill/dao"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/gkill_notification"
	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/twpayne/go-gpx"
)

func NewGkillServerAPI() (*GkillServerAPI, error) {
	ctx := context.Background()
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
		passwordResetToken := GenerateNewID()
		adminAccount := &account.Account{
			UserID:             "admin",
			PasswordSha256:     nil,
			IsAdmin:            true,
			IsEnable:           true,
			PasswordResetToken: &passwordResetToken,
		}
		_, err := gkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(context.TODO(), adminAccount)
		if err != nil {
			err = fmt.Errorf("error at add admin account: %w", err)
			return nil, err
		}
	}

	serverConfigs, err := gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
	}

	if len(serverConfigs) == 0 {
		serverConfig := &server_config.ServerConfig{
			EnableThisDevice:     true,
			Device:               "gkill",
			IsLocalOnlyAccess:    false,
			Address:              ":9999",
			EnableTLS:            false,
			TLSCertFile:          gkill_options.TLSCertFileDefault,
			TLSKeyFile:           gkill_options.TLSKeyFileDefault,
			OpenDirectoryCommand: "explorer /select,$filename",
			OpenFileCommand:      "rundll32 url.dll,FileProtocolHandler $filename",
			URLogTimeout:         1 * time.Minute,
			URLogUserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36",
			UploadSizeLimitMonth: -1,
			UserDataDirectory:    gkill_options.DataDirectoryDefault,
		}
		serverConfig.GkillNotificationPrivateKey, serverConfig.GkillNotificationPublicKey, err = webpush.GenerateVAPIDKeys()
		if err != nil {
			err = fmt.Errorf("error at generate vapid keys: %w", err)
			return nil, err
		}

		_, err = gkillDAOManager.ConfigDAOs.ServerConfigDAO.AddServerConfig(context.TODO(), serverConfig)
		if err != nil {
			err = fmt.Errorf("error at add init data to server config db: %w", err)
			return nil, err
		}

		serverConfigs, err = gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.Background())
		if err != nil {
			err = fmt.Errorf("error at get all server configs: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			return nil, err
		}
	}

	var device *string
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			if device != nil {
				err = fmt.Errorf("invalid status. enable device count is not 1")
				return nil, err
			}
			device = &serverConfig.Device
		}
	}
	if device == nil {
		err = fmt.Errorf("invalid status. enable device count is not 1")
		return nil, err
	}

	applicationConfigs, err := gkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetAllApplicationConfigs(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get all application configs: %w", err)
		return nil, err
	}
	if len(applicationConfigs) == 0 {
		_, err = gkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddDefaultApplicationConfig(context.Background(), "admin", *device)
		if err != nil {
			err = fmt.Errorf("error at add application config admin: %w", err)
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
	server *http.Server

	APIAddress *GkillServerAPIAddress

	GkillDAOManager *dao.GkillDAOManager

	FindFilter *FindFilter

	RebootServerCh chan (struct{})

	device string
}

func (g *GkillServerAPI) Serve() error {
	var err error
	ctx := context.Background()
	router := g.GkillDAOManager.GetRouter()
	router.PathPrefix("/files/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleFileServe(w, r)
	})
	router.HandleFunc(g.APIAddress.LoginAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleLogin(w, r)
	}).Methods(g.APIAddress.LoginMethod)
	router.HandleFunc(g.APIAddress.LogoutAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleLogout(w, r)
	}).Methods(g.APIAddress.LogoutMethod)
	router.HandleFunc(g.APIAddress.ResetPasswordAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleResetPassword(w, r)
	}).Methods(g.APIAddress.ResetPasswordMethod)
	router.HandleFunc(g.APIAddress.SetNewPasswordAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleSetNewPassword(w, r)
	}).Methods(g.APIAddress.SetNewPasswordMethod)
	router.HandleFunc(g.APIAddress.AddTagAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddTag(w, r)
	}).Methods(g.APIAddress.AddTagMethod)
	router.HandleFunc(g.APIAddress.AddTextAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddText(w, r)
	}).Methods(g.APIAddress.AddTextMethod)
	router.HandleFunc(g.APIAddress.AddNotificationAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddNotification(w, r)
	}).Methods(g.APIAddress.AddNotificationMethod)
	router.HandleFunc(g.APIAddress.AddKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddKmemo(w, r)
	}).Methods(g.APIAddress.AddKmemoMethod)
	router.HandleFunc(g.APIAddress.AddKCAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddKC(w, r)
	}).Methods(g.APIAddress.AddKCMethod)
	router.HandleFunc(g.APIAddress.AddURLogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddURLog(w, r)
	}).Methods(g.APIAddress.AddURLogMethod)
	router.HandleFunc(g.APIAddress.AddNlogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddNlog(w, r)
	}).Methods(g.APIAddress.AddNlogMethod)
	router.HandleFunc(g.APIAddress.AddTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddTimeis(w, r)
	}).Methods(g.APIAddress.AddTimeisMethod)
	router.HandleFunc(g.APIAddress.AddMiAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddMi(w, r)
	}).Methods(g.APIAddress.AddMiMethod)
	router.HandleFunc(g.APIAddress.AddLantanaAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddLantana(w, r)
	}).Methods(g.APIAddress.AddLantanaMethod)
	router.HandleFunc(g.APIAddress.AddRekyouAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddRekyou(w, r)
	}).Methods(g.APIAddress.AddRekyouMethod)
	router.HandleFunc(g.APIAddress.UpdateTagAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateTag(w, r)
	}).Methods(g.APIAddress.UpdateTagMethod)
	router.HandleFunc(g.APIAddress.UpdateTextAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateText(w, r)
	}).Methods(g.APIAddress.UpdateTextMethod)
	router.HandleFunc(g.APIAddress.UpdateNotificationAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateNotification(w, r)
	}).Methods(g.APIAddress.UpdateNotificationMethod)
	router.HandleFunc(g.APIAddress.UpdateKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateKmemo(w, r)
	}).Methods(g.APIAddress.UpdateKmemoMethod)
	router.HandleFunc(g.APIAddress.UpdateKCAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateKC(w, r)
	}).Methods(g.APIAddress.UpdateKCMethod)
	router.HandleFunc(g.APIAddress.UpdateURLogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateURLog(w, r)
	}).Methods(g.APIAddress.UpdateURLogMethod)
	router.HandleFunc(g.APIAddress.UpdateNlogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateNlog(w, r)
	}).Methods(g.APIAddress.UpdateNlogMethod)
	router.HandleFunc(g.APIAddress.UpdateTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateTimeis(w, r)
	}).Methods(g.APIAddress.UpdateTimeisMethod)
	router.HandleFunc(g.APIAddress.UpdateLantanaAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateLantana(w, r)
	}).Methods(g.APIAddress.UpdateLantanaMethod)
	router.HandleFunc(g.APIAddress.UpdateIDFKyouAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateIDFKyou(w, r)
	}).Methods(g.APIAddress.UpdateIDFKyouMethod)
	router.HandleFunc(g.APIAddress.UpdateMiAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateMi(w, r)
	}).Methods(g.APIAddress.UpdateMiMethod)
	router.HandleFunc(g.APIAddress.UpdateRekyouAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateRekyou(w, r)
	}).Methods(g.APIAddress.UpdateRekyouMethod)
	router.HandleFunc(g.APIAddress.GetKyousAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetKyous(w, r)
	}).Methods(g.APIAddress.GetKyousMethod)
	router.HandleFunc(g.APIAddress.GetKyouAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetKyou(w, r)
	}).Methods(g.APIAddress.GetKyouMethod)
	router.HandleFunc(g.APIAddress.GetKmemoAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetKmemo(w, r)
	}).Methods(g.APIAddress.GetKmemoMethod)
	router.HandleFunc(g.APIAddress.GetKCAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetKC(w, r)
	}).Methods(g.APIAddress.GetKCMethod)
	router.HandleFunc(g.APIAddress.GetURLogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetURLog(w, r)
	}).Methods(g.APIAddress.GetURLogMethod)
	router.HandleFunc(g.APIAddress.GetNlogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetNlog(w, r)
	}).Methods(g.APIAddress.GetNlogMethod)
	router.HandleFunc(g.APIAddress.GetTimeisAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetTimeis(w, r)
	}).Methods(g.APIAddress.GetTimeisMethod)
	router.HandleFunc(g.APIAddress.GetMiAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetMi(w, r)
	}).Methods(g.APIAddress.GetMiMethod)
	router.HandleFunc(g.APIAddress.GetLantanaAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetLantana(w, r)
	}).Methods(g.APIAddress.GetLantanaMethod)
	router.HandleFunc(g.APIAddress.GetRekyouAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetRekyou(w, r)
	}).Methods(g.APIAddress.GetRekyouMethod)
	router.HandleFunc(g.APIAddress.GetGitCommitLogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetGitCommitLog(w, r)
	}).Methods(g.APIAddress.GetGitCommitLogMethod)
	router.HandleFunc(g.APIAddress.GetIDFKyouAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetIDFKyou(w, r)
	}).Methods(g.APIAddress.GetIDFKyouMethod)
	router.HandleFunc(g.APIAddress.GetMiBoardListAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetMiBoardList(w, r)
	}).Methods(g.APIAddress.GetMiBoardListMethod)
	router.HandleFunc(g.APIAddress.GetAllTagNamesAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetAllTagNames(w, r)
	}).Methods(g.APIAddress.GetAllTagNamesMethod)
	router.HandleFunc(g.APIAddress.GetAllRepNamesAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetAllRepNames(w, r)
	}).Methods(g.APIAddress.GetAllRepNamesMethod)
	router.HandleFunc(g.APIAddress.GetTagsByTargetIDAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetTagsByTargetID(w, r)
	}).Methods(g.APIAddress.GetTagsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetTagHistoriesByTagIDAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetTagHistoriesByTagID(w, r)
	}).Methods(g.APIAddress.GetTagHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetTextsByTargetIDAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetTextsByTargetID(w, r)
	}).Methods(g.APIAddress.GetTextsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetNotificationsByTargetIDAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetNotificationsByTargetID(w, r)
	}).Methods(g.APIAddress.GetNotificationsByTargetIDMethod)
	router.HandleFunc(g.APIAddress.GetTextHistoriesByTextIDAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetTextHistoriesByTextID(w, r)
	}).Methods(g.APIAddress.GetTextHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetNotificationHistoriesByNotificationIDAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetNotificationHistoriesByNotificationID(w, r)
	}).Methods(g.APIAddress.GetNotificationHistoriesByTagIDMethod)
	router.HandleFunc(g.APIAddress.GetApplicationConfigAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetApplicationConfig(w, r)
	}).Methods(g.APIAddress.GetApplicationConfigMethod)
	router.HandleFunc(g.APIAddress.GetServerConfigsAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetServerConfigs(w, r)
	}).Methods(g.APIAddress.GetServerConfigsMethod)
	router.HandleFunc(g.APIAddress.UploadFilesAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUploadFiles(w, r)
	}).Methods(g.APIAddress.UploadFilesMethod)
	router.HandleFunc(g.APIAddress.UploadGPSLogFilesAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUploadGPSLogFiles(w, r)
	}).Methods(g.APIAddress.UploadGPSLogFilesMethod)
	router.HandleFunc(g.APIAddress.UpdateApplicationConfigAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateApplicationConfig(w, r)
	}).Methods(g.APIAddress.UpdateApplicationConfigMethod)
	router.HandleFunc(g.APIAddress.UpdateAccountStatusAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateAccountStatus(w, r)
	}).Methods(g.APIAddress.UpdateAccountStatusMethod)
	router.HandleFunc(g.APIAddress.UpdateUserRepsAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateUserReps(w, r)
	}).Methods(g.APIAddress.UpdateUserRepsMethod)
	router.HandleFunc(g.APIAddress.UpdateServerConfigsAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateServerConfigs(w, r)
	}).Methods(g.APIAddress.UpdateServerConfigsMethod)
	router.HandleFunc(g.APIAddress.AddAccountAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddAccount(w, r)
	}).Methods(g.APIAddress.AddAccountMethod)
	router.HandleFunc(g.APIAddress.GenerateTLSFileAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGenerateTLSFile(w, r)
	}).Methods(g.APIAddress.GenerateTLSFileMethod)
	router.HandleFunc(g.APIAddress.GetGPSLogAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetGPSLog(w, r)
	}).Methods(g.APIAddress.GetGPSLogMethod)
	router.HandleFunc(g.APIAddress.GetShareKyouListInfosAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetShareKyouListInfos(w, r)
	}).Methods(g.APIAddress.GetShareKyouListInfosMethod)
	router.HandleFunc(g.APIAddress.AddShareKyouListInfoAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleAddShareKyouListInfo(w, r)
	}).Methods(g.APIAddress.AddShareKyouListInfoMethod)
	router.HandleFunc(g.APIAddress.UpdateShareKyouListInfoAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateShareKyouListInfo(w, r)
	}).Methods(g.APIAddress.UpdateShareKyouListInfoMethod)
	router.HandleFunc(g.APIAddress.DeleteShareKyouListInfosAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleDeleteShareKyouListInfos(w, r)
	}).Methods(g.APIAddress.DeleteShareKyouListInfosMethod)
	router.HandleFunc(g.APIAddress.GetSharedKyousAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetSharedKyous(w, r)
	}).Methods(g.APIAddress.GetSharedKyousMethod)
	router.HandleFunc(g.APIAddress.GetRepositoriesAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetRepositories(w, r)
	}).Methods(g.APIAddress.GetRepositoriesMethod)
	router.HandleFunc(g.APIAddress.GetGkillNotificationPublicKeyAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetGkillNotificationPublicKey(w, r)
	}).Methods(g.APIAddress.GetGkillNotificationPublicKeyMethod)
	router.HandleFunc(g.APIAddress.RegisterGkillNotificationAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleRegisterGkillNotification(w, r)
	}).Methods(g.APIAddress.RegisterGkillNotificationMethod)
	router.HandleFunc(g.APIAddress.OpenDirectoryAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleOpenDirectory(w, r)
	}).Methods(g.APIAddress.OpenDirectoryMethod)
	router.HandleFunc(g.APIAddress.OpenFileAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleOpenFile(w, r)
	}).Methods(g.APIAddress.OpenFileMethod)
	router.HandleFunc(g.APIAddress.ReloadRepositoriesAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleReloadRepositories(w, r)
	}).Methods(g.APIAddress.ReloadRepositoriesMethod)
	router.HandleFunc(g.APIAddress.URLogBookmarkletAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleURLogBookmarkletAddress(w, r)
	}).Methods(g.APIAddress.URLogBookmarkletMethod)
	router.HandleFunc(g.APIAddress.GetUpdatedDatasByTimeAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetUpdatedDatasByTime(w, r)
	}).Methods(g.APIAddress.GetUpdatedDatasByTimeMethod)
	router.HandleFunc(g.APIAddress.CommitTXAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleCommitTx(w, r)
	}).Methods(g.APIAddress.CommitTXMethod)
	router.HandleFunc(g.APIAddress.DiscardTXAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleDiscardTX(w, r)
	}).Methods(g.APIAddress.DiscardTXMethod)

	gkillPage, err := fs.Sub(EmbedFS, "embed/html")
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
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		return err
	}
	port := serverConfig.Address

	g.PrintStartedMessage()
	g.server = &http.Server{Addr: port, Handler: router}
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidLoginResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse login request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidLoginRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 存在するアカウントを取得
	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INVALID_USER_ID_OR_PASSWORD"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント有効確認
	if !account.IsEnable {
		err = fmt.Errorf("error at account is not enable = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountIsNotEnableError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "ACCOUNT_DISABLED_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードリセット処理実施中のアカウントはログインから弾く
	if account.PasswordResetToken != nil {
		err = fmt.Errorf("error at password reset token is not nil = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountPasswordResetTokenIsNotNilError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "REQUESTED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワード不一致を弾く
	if account.PasswordSha256 != nil && *account.PasswordSha256 != request.PasswordSha256 {
		err = fmt.Errorf("error at account invalid password = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidPasswordError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_MESSAGE"}),
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
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
			err = fmt.Errorf("error add login session user_id = %s: %w", request.UserID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountLoginInternalServerError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// URLogブックマークレット用のセッションがもしなければ作成する
	loginSessions, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetAllLoginSessions(r.Context())
	if err != nil {
		if err != nil {
			err = fmt.Errorf("error get login sessions = %s: %w", request.UserID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountSessionsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	var urlogBookmarkletSession *account_state.LoginSession
	for _, loginSession := range loginSessions {
		if loginSession.ApplicationName == "urlog_bookmarklet" && loginSession.UserID == request.UserID {
			urlogBookmarkletSession = loginSession
			break
		}
	}
	if urlogBookmarkletSession == nil {
		loginSession := &account_state.LoginSession{
			ID:              GenerateNewID(),
			UserID:          request.UserID,
			Device:          device,
			ApplicationName: "urlog_bookmarklet",
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
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogLoginSessionError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGIN_INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.SessionID = loginSession.SessionID
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.LoginSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_LOGIN_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidLogoutResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGOUT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse logout request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidLogoutRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGOUT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.CloseDatabase {
		account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
		if err != nil {
			if err != nil {
				err = fmt.Errorf("error account from session id = %s: %w", request.SessionID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		device, err := g.GetDevice()
		if err != nil {
			err = fmt.Errorf("error at get device name: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		_, err = g.GkillDAOManager.CloseUserRepositories(account.UserID, device)
		if err != nil {
			err = fmt.Errorf("error at close repository user id = %s device = %s: %w", account.UserID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.DeleteLoginSession(r.Context(), request.SessionID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error add logout session id = %s: %w", request.SessionID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountLogoutInternalServerError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_LOGOUT_INTERNAL_SERVER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.LogoutSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_LOGOUT_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidResetPasswordResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse reset password request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidResetPasswordRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// パスワードリセット操作をしたユーザを特定
	requesterSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), requesterSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", requesterSession.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if requesterAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterSession.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.PasswordResetSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_PASSWORD_RESET_MESSAGE"}),
	})
	response.PasswordResetPathWithoutHost = *updateTargetAccount.PasswordResetToken
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse login response to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidSetNewPasswordResponseDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得してパスワード設定
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// リセットトークンがあっているか確認
	if targetAccount.PasswordResetToken == nil || request.ResetToken != *targetAccount.PasswordResetToken {
		err = fmt.Errorf("error at reset token is not match user id = %s requested token = %s: %w", request.UserID, request.ResetToken, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidPasswordResetTokenError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInfoUpdateError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SET_NEW_PASSWORD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.SetNewPasswordSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_SET_NEW_PASSWORD_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddTagResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add tag request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddTagRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existTag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTag != nil {
		err = fmt.Errorf("exist tag id = %s", request.Tag.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), request.Tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), request.Tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TagTempRep.AddTagInfo(r.Context(), request.Tag, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteTagRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Tag.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Tag.IsDeleted,
		TargetID:                               request.Tag.ID,
		TargetIDInData:                         &request.Tag.TargetID,
		DataUpdateTime:                         request.Tag.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Tag.ID])
		if err != nil {
			err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	tag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedTag = tag
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddTagSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_TAG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddTextResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add text request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddTextRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existText, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existText != nil {
		err = fmt.Errorf("exist text id = %s", request.Text.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), request.Text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), request.Text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TextTempRep.AddTextInfo(r.Context(), request.Text, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteTextRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Text.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Text.IsDeleted,
		TargetID:                               request.Text.ID,
		TargetIDInData:                         &request.Text.TargetID,
		DataUpdateTime:                         request.Text.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Text.ID])
		if err != nil {
			err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	text, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedText = text
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddTextSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_TEXT_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleAddNotification(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddNotificationRequest{}
	response := &req_res.AddNotificationResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add notification response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddNotificationResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add notification request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddNotificationRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existNotification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNotification != nil {
		err = fmt.Errorf("exist notification id = %s", request.Notification.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), request.Notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NotificationReps) == 1 && *gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), request.Notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NotificationTempRep.AddNotificationInfo(r.Context(), request.Notification, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Notification.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Notification.IsDeleted,
		TargetID:                               request.Notification.ID,
		TargetIDInData:                         &request.Notification.TargetID,
		DataUpdateTime:                         request.Notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Notification.ID])
		if err != nil {
			err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	notification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 通知情報を更新する
	notificator, err := g.GkillDAOManager.GetNotificator(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notificator: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = notificator.UpdateNotificationTargets(context.Background())
	if err != nil {
		err = fmt.Errorf("error at update notification targetrs: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedNotification = notification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNotificationSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_NOTIFICATION_MESSAGE"}),
	})

	repName, err = repositories.WriteNotificationRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Notification.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Notification.IsDeleted,
		TargetID:                               request.Notification.ID,
		TargetIDInData:                         &request.Notification.TargetID,
		DataUpdateTime:                         request.Notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Notification.ID])
		if err != nil {
			err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	response.AddedNotification = notification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNotificationSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_NOTIFICATION_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddKmemoResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add kmemo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddKmemoRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existKmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKmemo != nil {
		err = fmt.Errorf("exist kmemo id = %s", request.Kmemo.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), request.Kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), request.Kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KmemoTempRep.AddKmemoInfo(r.Context(), request.Kmemo, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Kmemo.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Kmemo.IsDeleted,
		TargetID:                               request.Kmemo.ID,
		DataUpdateTime:                         request.Kmemo.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Kmemo.ID])
		if err != nil {
			err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		kmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKmemo = kmemo

		kyou, err := repositories.KmemoReps.GetKyou(r.Context(), request.Kmemo.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kyou user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddKmemoSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_KMEMO_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleAddKC(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddKCRequest{}
	response := &req_res.AddKCResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add kc response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddKCResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add kc request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddKCRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existKC, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKC != nil {
		err = fmt.Errorf("exist kc id = %s", request.KC.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), request.KC)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
			err = repositories.KCReps[0].AddKCInfo(r.Context(), request.KC)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KCTempRep.AddKCInfo(r.Context(), request.KC, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteKCRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.KC.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.KC.IsDeleted,
		TargetID:                               request.KC.ID,
		DataUpdateTime:                         request.KC.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.KC.ID])
		if err != nil {
			err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		kc, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKC = kc
		kyou, err := repositories.KCReps.GetKyou(r.Context(), request.KC.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddKCSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_KC_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddURLogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add kmemo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddURLogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existURLog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", request.URLog.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// applicationConfigを取得
	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = request.URLog.FillURLogField(serverConfig, applicationConfig)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}

	if request.TXID == nil {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), request.URLog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), request.URLog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.URLogTempRep.AddURLogInfo(r.Context(), request.URLog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.URLog.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.URLog.IsDeleted,
		TargetID:                               request.URLog.ID,
		DataUpdateTime:                         request.URLog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.URLog.ID])
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		urlog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedURLog = urlog

		kyou, err := repositories.URLogReps.GetKyou(r.Context(), request.URLog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddURLogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_URLOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddNlogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add nlog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddNlogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existNlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNlog != nil {
		err = fmt.Errorf("exist nlog id = %s", request.Nlog.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), request.Nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), request.Nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NlogTempRep.AddNlogInfo(r.Context(), request.Nlog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Nlog.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Nlog.IsDeleted,
		TargetID:                               request.Nlog.ID,
		DataUpdateTime:                         request.Nlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Nlog.ID])
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		nlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedNlog = nlog

		kyou, err := repositories.NlogReps.GetKyou(r.Context(), request.Nlog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNlogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_NLOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddTimeIsResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add timeis request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddTimeIsRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTimeIs != nil {
		err = fmt.Errorf("exist timeis id = %s", request.TimeIs.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), request.TimeIs)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), request.TimeIs)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(r.Context(), request.TimeIs, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.TimeIs.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.TimeIs.IsDeleted,
		TargetID:                               request.TimeIs.ID,
		DataUpdateTime:                         request.TimeIs.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.TimeIs.ID])
		if err != nil {
			err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		timeis, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedTimeis = timeis

		kyou, err := repositories.TimeIsReps.GetKyou(r.Context(), request.TimeIs.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddTimeIsSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_TIMEIS_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddLantanaResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add lantana request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddLantanaRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existLantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if existLantana != nil {
		err = fmt.Errorf("exist lantana id = %s", request.Lantana.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), request.Lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), request.Lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.LantanaTempRep.AddLantanaInfo(r.Context(), request.Lantana, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Lantana.ID)

	repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Lantana.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Lantana.IsDeleted,
		TargetID:                               request.Lantana.ID,
		DataUpdateTime:                         request.Lantana.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Lantana.ID])
		if err != nil {
			err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		lantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedLantana = lantana

		kyou, err := repositories.LantanaReps.GetKyou(r.Context(), request.Lantana.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		response.AddedKyou = kyou
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddLantanaSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_LANTANA_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddMiResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add mi request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddMiRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existMi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existMi != nil {
		err = fmt.Errorf("exist mi id = %s", request.Mi.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), request.Mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), request.Mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.MiTempRep.AddMiInfo(r.Context(), request.Mi, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteMiRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Mi.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Mi.IsDeleted,
		TargetID:                               request.Mi.ID,
		DataUpdateTime:                         request.Mi.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Mi.ID])
		if err != nil {
			err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		mi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedMi = mi

		kyou, err := repositories.MiReps.GetKyou(r.Context(), request.Mi.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddMiSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_MI_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddReKyouResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add rekyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddReKyouRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existReKyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existReKyou != nil {
		err = fmt.Errorf("exist rekyou id = %s", request.ReKyou.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), request.ReKyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && *gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), request.ReKyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.ReKyouTempRep.AddReKyouInfo(r.Context(), request.ReKyou, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.ReKyou.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.ReKyou.IsDeleted,
		TargetID:                               request.ReKyou.ID,
		TargetIDInData:                         &request.ReKyou.TargetID,
		DataUpdateTime:                         request.ReKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.ReKyou.ID])
		if err != nil {
			err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	if request.WantResponseKyou {
		rekyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedReKyou = rekyou

		kyou, err := repositories.ReKyouReps.GetKyou(r.Context(), request.ReKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.AddedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddReKyouSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_REKYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTagResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update tag request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTagRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), request.Tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), request.Tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TagTempRep.AddTagInfo(r.Context(), request.Tag, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteTagRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Tag.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Tag.IsDeleted,
		TargetID:                               request.Tag.ID,
		TargetIDInData:                         &request.Tag.TargetID,
		DataUpdateTime:                         request.Tag.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Tag.ID])
		if err != nil {
			err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	tag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existTag, err := repositories.GetTag(r.Context(), request.Tag.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTag == nil {
		err = fmt.Errorf("not exist tag id = %s", request.Tag.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TAG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedTag = tag
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTagSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_TAG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTextResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update text request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTextRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), request.Text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), request.Text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TextTempRep.AddTextInfo(r.Context(), request.Text, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteTextRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Text.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Text.IsDeleted,
		TargetID:                               request.Text.ID,
		TargetIDInData:                         &request.Text.TargetID,
		DataUpdateTime:                         request.Text.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Text.ID])
		if err != nil {
			err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	text, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existText, err := repositories.GetText(r.Context(), request.Text.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existText == nil {
		err = fmt.Errorf("not exist text id = %s", request.Text.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedText = text
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTextSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_TEXT_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleUpdateNotification(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateNotificationRequest{}
	response := &req_res.UpdateNotificationResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update notification response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateNotificationResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update notification request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateNotificationRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), request.Notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NotificationReps) == 1 && *gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), request.Notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NotificationTempRep.AddNotificationInfo(r.Context(), request.Notification, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Notification.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Notification.IsDeleted,
		TargetID:                               request.Notification.ID,
		TargetIDInData:                         &request.Notification.TargetID,
		DataUpdateTime:                         request.Notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Notification.ID])
		if err != nil {
			err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	notification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existNotification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNotification == nil {
		err = fmt.Errorf("not exist notification id = %s", request.Notification.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 通知情報を更新する
	notificator, err := g.GkillDAOManager.GetNotificator(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notificator: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = notificator.UpdateNotificationTargets(context.Background())
	if err != nil {
		err = fmt.Errorf("error at update notification targetrs: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedNotification = notification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateNotificationSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_NOTIFICATION_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateKmemoResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update kmemo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateKmemoRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), request.Kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), request.Kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KmemoTempRep.AddKmemoInfo(r.Context(), request.Kmemo, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Kmemo.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Kmemo.IsDeleted,
		TargetID:                               request.Kmemo.ID,
		DataUpdateTime:                         request.Kmemo.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Kmemo.ID])
		if err != nil {
			err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	kmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existKmemo, err := repositories.KmemoReps.GetKmemo(r.Context(), request.Kmemo.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKmemo == nil {
		err = fmt.Errorf("not exist kmemo id = %s", request.Kmemo.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedKmemo = kmemo
		kyou, err := repositories.KmemoReps.GetKyou(r.Context(), request.Kmemo.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateKmemoSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_KMEMO_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleUpdateKC(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateKCRequest{}
	response := &req_res.UpdateKCResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update kc response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateKCResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update kc request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateKCRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), request.KC)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
			err = repositories.WriteKCRep.AddKCInfo(r.Context(), request.KC)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.KCTempRep.AddKCInfo(r.Context(), request.KC, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteKCRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.KC.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.KC.IsDeleted,
		TargetID:                               request.KC.ID,
		DataUpdateTime:                         request.KC.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.KC.ID])
		if err != nil {
			err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	kc, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existKC, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKC == nil {
		err = fmt.Errorf("not exist kc id = %s", request.KC.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedKC = kc
		kyou, err := repositories.KCReps.GetKyou(r.Context(), request.KC.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateKCSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_KC_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateURLogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update urlog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateURLogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.ReGetURLogContent {
		var currentServerConfig *server_config.ServerConfig
		serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		for _, serverConfig := range serverConfigs {
			if serverConfig.Device == device {
				currentServerConfig = serverConfig
				break
			}
		}
		if currentServerConfig == nil {
			err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil || applicationConfig == nil {
			err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
			err = fmt.Errorf("try create application config user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)

			defaultApplicationConfig := user_config.GetDefaultApplicationConfig(userID, device)
			_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(context.TODO(), defaultApplicationConfig)

			if err != nil {
				gkillError := &message.GkillError{
					ErrorCode:    message.GetApplicationConfigError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			applicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
			if err != nil {
				gkillError := &message.GkillError{
					ErrorCode:    message.GetApplicationConfigError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		err = request.URLog.FillURLogField(currentServerConfig, applicationConfig)
		if err != nil {
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	if request.TXID == nil {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), request.URLog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), request.URLog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.URLogTempRep.AddURLogInfo(r.Context(), request.URLog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.URLog.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.URLog.IsDeleted,
		TargetID:                               request.URLog.ID,
		DataUpdateTime:                         request.URLog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.URLog.ID])
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	urlog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existURLog, err := repositories.URLogReps.GetURLog(r.Context(), request.URLog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog == nil {
		err = fmt.Errorf("not exist urlog id = %s", request.URLog.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedURLog = urlog
		kyou, err := repositories.URLogReps.GetKyou(r.Context(), request.URLog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateURLogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_URLOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateNlogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update nlog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateNlogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), request.Nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), request.Nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.NlogTempRep.AddNlogInfo(r.Context(), request.Nlog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Nlog.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Nlog.IsDeleted,
		TargetID:                               request.Nlog.ID,
		DataUpdateTime:                         request.Nlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Nlog.ID])
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	nlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existNlog, err := repositories.NlogReps.GetNlog(r.Context(), request.Nlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNlog == nil {
		err = fmt.Errorf("not exist nlog id = %s", request.Nlog.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedNlog = nlog
		kyou, err := repositories.NlogReps.GetKyou(r.Context(), request.Nlog.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateNlogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_NLOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateTimeIsResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update timeis request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateTimeIsRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), request.TimeIs)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), request.TimeIs)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(r.Context(), request.TimeIs, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.TimeIs.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.TimeIs.IsDeleted,
		TargetID:                               request.TimeIs.ID,
		DataUpdateTime:                         request.TimeIs.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.TimeIs.ID])
		if err != nil {
			err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	timeis, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existTimeIs, err := repositories.TimeIsReps.GetTimeIs(r.Context(), request.TimeIs.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTimeIs == nil {
		err = fmt.Errorf("not exist timeis id = %s", request.TimeIs.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedTimeis = timeis
		kyou, err := repositories.TimeIsReps.GetKyou(r.Context(), request.TimeIs.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTimeIsSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_TIMEIS_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateLantanaResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update lantana request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateLantanaRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), request.Lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), request.Lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.LantanaTempRep.AddLantanaInfo(r.Context(), request.Lantana, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Lantana.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Lantana.IsDeleted,
		TargetID:                               request.Lantana.ID,
		DataUpdateTime:                         request.Lantana.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Lantana.ID])
		if err != nil {
			err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	lantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existLantana, err := repositories.LantanaReps.GetLantana(r.Context(), request.Lantana.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existLantana == nil {
		err = fmt.Errorf("not exist lantana id = %s", request.Lantana.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedLantana = lantana
		kyou, err := repositories.LantanaReps.GetKyou(r.Context(), request.Lantana.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateLantanaSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_LANTANA_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateIDFKyouResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update idfKyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateIDFKyouRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), request.IDFKyou)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.IDFKyouReps) == 1 && *gkill_options.CacheIDFKyouReps {
			err = repositories.IDFKyouReps[0].AddIDFKyouInfo(r.Context(), request.IDFKyou)
			if err != nil {
				err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.IDFKyouTempRep.AddIDFKyouInfo(r.Context(), request.IDFKyou, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteIDFKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.IDFKyou.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.IDFKyou.IsDeleted,
		TargetID:                               request.IDFKyou.ID,
		DataUpdateTime:                         request.IDFKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.IDFKyou.ID])
		if err != nil {
			err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	idfKyou, err := repositories.IDFKyouReps.GetIDFKyou(r.Context(), request.IDFKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existIDFKyou, err := repositories.IDFKyouReps.GetIDFKyou(r.Context(), request.IDFKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existIDFKyou == nil {
		err = fmt.Errorf("not exist idfKyou id = %s", request.IDFKyou.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedIDFKyou = idfKyou
		kyou, err := repositories.IDFKyouReps.GetKyou(r.Context(), request.IDFKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateIDFKyouSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_IDFKYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateMiResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update mi request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateMiRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existMi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existMi == nil {
		err = fmt.Errorf("not exist mi id = %s", request.Mi.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), request.Mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), request.Mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.MiTempRep.AddMiInfo(r.Context(), request.Mi, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteMiRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.Mi.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Mi.IsDeleted,
		TargetID:                               request.Mi.ID,
		DataUpdateTime:                         request.Mi.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.Mi.ID])
		if err != nil {
			err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	mi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if request.WantResponseKyou {
		response.UpdatedMi = mi
		kyou, err := repositories.MiReps.GetKyou(r.Context(), request.Mi.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_MI_UPDATED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateMiSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_MI_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateReKyouResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update rekyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateReKyouRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), request.ReKyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && *gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), request.ReKyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	} else {
		err = repositories.TempReps.ReKyouTempRep.AddReKyouInfo(r.Context(), request.ReKyou, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[request.ReKyou.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.ReKyou.IsDeleted,
		TargetID:                               request.ReKyou.ID,
		TargetIDInData:                         &request.ReKyou.TargetID,
		DataUpdateTime:                         request.ReKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[request.ReKyou.ID])
		if err != nil {
			err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	rekyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existReKyou, err := repositories.ReKyouReps.GetReKyou(r.Context(), request.ReKyou.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existReKyou == nil {
		err = fmt.Errorf("not exist rekyou id = %s", request.ReKyou.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.WantResponseKyou {
		response.UpdatedReKyou = rekyou
		kyou, err := repositories.ReKyouReps.GetKyou(r.Context(), request.ReKyou.ID, nil)
		if err != nil {
			err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.UpdatedKyou = kyou
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateReKyouSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_REKYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyousResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyousRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOUS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	trueValue := true
	request.Query.OnlyLatestData = &trueValue

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kyous, gkillErrors, err := g.FindFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, request.Query)
	if len(gkillErrors) != 0 || err != nil {
		if err != nil {
			err = fmt.Errorf("error at find kyous: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		response.Errors = append(response.Errors, gkillErrors...)
		return
	}

	response.Kyous = kyous
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyousSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOUS_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKyouResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKyouRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var kyouHistories []*reps.Kyou
	if request.UpdateTime != nil {
		var kyou *reps.Kyou
		kyou, err = repositories.GetKyou(r.Context(), request.ID, request.UpdateTime)
		kyouHistories = []*reps.Kyou{kyou}
	} else {
		kyouHistories, err = repositories.Reps.GetKyouHistoriesByRepName(r.Context(), request.ID, request.RepName)
	}

	if err != nil {
		err = fmt.Errorf("error at get kyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KyouHistories = kyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKyouSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOU_MESSAGE"}),
	})
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKmemoResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kmemo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKmemoRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kmemoHistories, err := repositories.KmemoReps.GetKmemoHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KmemoHistories = kmemoHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKmemoSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KMEMO_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleGetKC(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKCRequest{}
	response := &req_res.GetKCResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kc response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKCResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kc request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKCRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kcHistories, err := repositories.KCReps.GetKCHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KC_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KCHistories = kcHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKCSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KC_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetURLogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get urlog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetURLogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlogHistories, err := repositories.URLogReps.GetURLogHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_URLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.URLogHistories = urlogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetURLogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_URLOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNlogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get nlog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNlogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlogHistories, err := repositories.NlogReps.GetNlogHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NLOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.NlogHistories = nlogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNlogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_NLOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTimeIsResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get timeis request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTimeIsRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeisHistories, err := repositories.TimeIsReps.GetTimeIsHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TimeisHistories = timeisHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTimeIsSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_TIMEIS_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get mi request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miHistories, err := repositories.MiReps.GetMiHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MI_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.MiHistories = miHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_MI_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetLantanaResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get lantana request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetLantanaRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantanaHistories, err := repositories.LantanaReps.GetLantanaHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LANTANA_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.LantanaHistories = lantanaHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetLantanaSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_LANTANA_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetReKyouResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get rekyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetReKyouRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyouHistories, err := repositories.ReKyouReps.GetReKyouHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ReKyouHistories = rekyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetReKyouSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_REKYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetGitCommitLogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get gitCommitLog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetGitCommitLogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	gitCommitLog, err := repositories.GitCommitLogReps.GetGitCommitLog(r.Context(), request.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get gitCommitLog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetGitCommitLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GIT_COMMIT_LOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GitCommitLogHistories = []*reps.GitCommitLog{gitCommitLog}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGitCommitLogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_GIT_COMMIT_LOG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetIDFKyouResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get idfKyou request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetIDFKyouRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyouHistories, err := repositories.IDFKyouReps.GetIDFKyouHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_IDFKYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.IDFKyouHistories = idfKyouHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetIDFKyouSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_IDFKYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiBoardNamesResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MIBOARD_NAMES_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get mi board names request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiBoardNamesRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MIBOARD_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_MIBOARD_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miBoardNames, err := repositories.MiReps.GetBoardNames(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get mi board names user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiBoardNamesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Boards = miBoardNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiBoardNamesSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_MIBOARD_NAMES_MESSAGE"}),
	})
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetAllTagNamesResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetAllTagNamesRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	allTagNames, err := repositories.GetAllTagNames(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all tag names user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllTagNamesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_TAG_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TagNames = allTagNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetAllTagNamesSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_ALL_TAG_NAMES_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetAllRepNamesResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_REP_NAMES_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kyous request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetAllRepNamesRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_REP_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_REP_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	allRepNames, err := repositories.GetAllRepNames(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get all rep names user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllRepNamesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_REP_NAMES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.RepNames = allRepNames
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetAllRepNamesSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_ALL_REP_NAMES_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTagsByTargetIDResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAGS_BY_TARGET_ID_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get tags by target id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTagsByTargetIDRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAGS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tags, err := repositories.GetTagsByTargetID(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get tags by target id user id = %s device = %s target id = %s: %w", userID, device, request.TargetID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagsByTargetIDError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Tags = tags
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTagsByTargetIDSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_TAGS_BY_TARGET_ID_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTagHistoriesByTagIDResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAG_HISTORIES_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get tag histories by tag id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTagHistoriesByTagIDRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAG_HISTORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAG_HISTORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var tags []*reps.Tag
	if request.UpdateTime != nil {
		var tag *reps.Tag
		tag, err = repositories.GetTag(r.Context(), request.ID, request.UpdateTime)
		tags = []*reps.Tag{tag}
	} else {
		tags, err = repositories.TagReps.GetTagHistoriesByRepName(r.Context(), request.ID, request.RepName)
	}

	if err != nil {
		err = fmt.Errorf("error at get tag histories by tag id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagHistoriesByTagIDError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAG_HISTORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TagHistories = tags
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTagHistoriesByTagIDSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_TAG_HISTORIES_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTextsByTargetIDResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get texts by target id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTextsByTargetIDRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	texts, err := repositories.GetTextsByTargetID(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get texts by target id user id = %s device = %s target id = %s: %w", userID, device, request.TargetID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextsByTargetIDError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Texts = texts
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTextsByTargetIDSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleGetNotificationsByTargetID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNotificationsByTargetIDRequest{}
	response := &req_res.GetNotificationsByTargetIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get notifications by target id response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNotificationsByTargetIDResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get notifications by target id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNotificationsByTargetIDRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notifications, err := repositories.GetNotificationsByTargetID(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get notifications by target id user id = %s device = %s target id = %s: %w", userID, device, request.TargetID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationsByTargetIDError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Notifications = notifications
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNotificationsByTargetIDSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_NOTIFICATION_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetTextHistoriesByTextIDResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get text histories by text id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetTextHistoriesByTextIDRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var texts []*reps.Text
	if request.UpdateTime != nil {
		var text *reps.Text
		text, err = repositories.GetText(r.Context(), request.ID, request.UpdateTime)
		texts = []*reps.Text{text}
	} else {
		texts, err = repositories.TextReps.GetTextHistoriesByRepName(r.Context(), request.ID, request.RepName)
	}

	if err != nil {
		err = fmt.Errorf("error at get text histories by text id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextHistoriesByTextIDError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.TextHistories = texts
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetTextHistoriesByTextIDSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleGetNotificationHistoriesByNotificationID(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNotificationHistoryByNotificationIDRequest{}
	response := &req_res.GetNotificationHistoryByNotificationIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get notification histories by notification id response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNotificationHistoriesByNotificationIDResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get notification histories by notification id request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNotificationHistoriesByNotificationIDRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	var notifications []*reps.Notification
	if request.UpdateTime != nil {
		var notification *reps.Notification
		notification, err = repositories.GetNotification(r.Context(), request.ID, request.UpdateTime)
		notifications = []*reps.Notification{notification}
	} else {
		notifications, err = repositories.GetNotificationHistories(r.Context(), request.ID)
	}

	if err != nil {
		err = fmt.Errorf("error at get notification histories by notification id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationHistoriesByNotificationIDError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.NotificationHistories = notifications
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNotificationHistoriesByNotificationIDSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_NOTIFICATION_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetApplicationConfigResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get applicationConfig request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetApplicationConfigRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil || applicationConfig == nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		err = fmt.Errorf("try create application config user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)

		defaultApplicationConfig := user_config.GetDefaultApplicationConfig(userID, device)
		_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(context.TODO(), defaultApplicationConfig)

		if err != nil {
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		applicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil {
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sessions, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSessions(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get login sessions session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	for _, session := range sessions {
		if session.ApplicationName == "urlog_bookmarklet" {
			applicationConfig.URLogBookmarkletSession = session.SessionID
			break
		}
	}

	privateIP, err := privateIPv4s()
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}
	globalIP, err := globalIP(context.Background())
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
	}
	privateIPStr := ""
	if len(privateIP) != 0 {
		privateIPStr = privateIP[0].String()
	}

	version, err := GetVersion()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	applicationConfig.AccountIsAdmin = account.IsAdmin
	applicationConfig.SessionIsLocal = session.IsLocalAppUser
	response.ApplicationConfig = applicationConfig

	response.ApplicationConfig.UserID = userID
	response.ApplicationConfig.Device = device
	response.ApplicationConfig.UserIsAdmin = account.IsAdmin
	response.ApplicationConfig.CacheClearCountLimit = gkill_options.CacheClearCountLimit
	response.ApplicationConfig.GlobalIP = globalIP.String()
	response.ApplicationConfig.PrivateIP = privateIPStr
	response.ApplicationConfig.Version = version.Version
	response.ApplicationConfig.BuildTime = version.BuildTime
	response.ApplicationConfig.CommitHash = version.CommitHash

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetApplicationConfigSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_APPLICATION_CONFIG_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetServerConfigResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get serverConfig request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetServerConfigRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !account.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", account.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	for _, serverConfig := range serverConfigs {
		accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get all account config")
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetAllAccountConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ACCOUNT_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		serverConfig.Accounts = accounts

		repositories, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.GetAllRepositories(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get all repositories")
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetAllRepositoriesError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ALL_REPOSITORIES_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		serverConfig.Repositories = repositories
	}

	response.ServerConfigs = serverConfigs
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetServerConfigSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_APPLICATION_CONFIG_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleUploadFiles(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UploadFilesRequest{}
	response := &req_res.UploadFilesResponse{}

	g.GkillDAOManager.SetSkipIDF(true)
	defer g.GkillDAOManager.SetSkipIDF(false)

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse upload files response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUploadFilesResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse upload files request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadFilesRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	g.GkillDAOManager.SetSkipIDF(true)
	defer g.GkillDAOManager.SetSkipIDF(false)

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidStatusGetRepNameError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTargetIDFRepError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// ファイル名解決
		estimateCreateFileName, err := g.resolveFileName(repDir, fileInfo.FileName, request.ConflictBehavior)
		if err != nil {
			err := fmt.Errorf("error at resolve save file name at %s filename= %s: %w", request.TargetRepName, fileInfo.FileName, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg.Add(1)
		go func(filename string, base64Data string) {
			defer wg.Done()
			var gkillError *message.GkillError
			parts := strings.SplitN(base64Data, ",", 2)
			encoded := parts[len(parts)-1]
			base64Reader := bufio.NewReader(strings.NewReader(encoded))
			decoder := base64.NewDecoder(base64.StdEncoding, base64Reader)

			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				err := fmt.Errorf("error at open file filename= %s: %w", filename, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError = &message.GkillError{
					ErrorCode:    message.GetRepPathError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
				}
				gkillErrorCh <- gkillError
				return
			}
			defer file.Close()
			io.Copy(file, decoder)
			os.Chtimes(filename, time.Now(), fileInfo.LastModified)

			// idfKyouを作る
			idfKyou := &reps.IDFKyou{
				IsDeleted:    false,
				ID:           GenerateNewID(),
				RelatedTime:  fileInfo.LastModified,
				CreateTime:   time.Now(),
				CreateApp:    "gkill",
				CreateDevice: device,
				CreateUser:   userID,
				UpdateTime:   time.Now(),
				UpdateApp:    "gkill",
				UpdateUser:   userID,
				UpdateDevice: device,
				TargetFile:   filepath.Base(filename),
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError = &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
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
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError = &message.GkillError{
						ErrorCode:    message.GetRepPathError,
						ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}

				// defer g.WebPushUpdatedData(r.Context(), userID, device, idfKyou.ID)
				repositories.LatestDataRepositoryAddresses[idfKyou.ID] = &account_state.LatestDataRepositoryAddress{
					IsDeleted:                              idfKyou.IsDeleted,
					TargetID:                               idfKyou.ID,
					DataUpdateTime:                         idfKyou.UpdateTime,
					LatestDataRepositoryName:               repName,
					LatestDataRepositoryAddressUpdatedTime: time.Now(),
				}
				go func() {
					_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[idfKyou.ID])
					if err != nil {
						err = fmt.Errorf("error at update or add latest data repository address: %w", err)
						slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					}
				}()
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError = &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_FILE_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
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
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPLOAD_FILE_GET_KYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUploadGPSLogFilesResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse upload files request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadGPSLogFilesRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidStatusGetRepNameError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
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
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTargetGPSLogRepError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ファイルを保存/GPSLogを追加する
	gkillErrors := []*message.GkillError{}
	gpsLogsCh := make(chan []*reps.GPSLog, len(request.GPSLogFiles))
	gkillErrorCh := make(chan *message.GkillError, len(request.GPSLogFiles))
	defer close(gpsLogsCh)
	defer close(gkillErrorCh)
	wg := &sync.WaitGroup{}
	repDir := ""
	for _, fileInfo := range request.GPSLogFiles {
		repDir, err = targetRep.GetPath(r.Context(), "")
		if err != nil {
			err := fmt.Errorf("error at get target rep path at %s: %w", request.TargetRepName, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg.Add(1)
		go func(filename string, base64Data string) {
			// テンポラリファイル書き込み
			defer wg.Done()
			base64Reader := bufio.NewReader(strings.NewReader(strings.SplitN(base64Data, ",", 2)[1]))
			decoder := base64.NewDecoder(base64.RawStdEncoding, base64Reader)
			base64DataBytes, err := io.ReadAll(decoder)
			if err != nil {
				err := fmt.Errorf("error at load gps log file content filename = %s: %w", filename, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
				}
				gkillErrorCh <- gkillError
				return
			}

			var gkillError *message.GkillError
			// gpsLogsを作る
			gpsLogs, err := gpslogs.GPSLogFileAsGPSLogs(repDir, filename, request.ConflictBehavior, string(base64DataBytes))
			if err != nil {
				err := fmt.Errorf("error at gps log file as gpx file filename = %s: %w", filename, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
				}
				gkillErrorCh <- gkillError
				return
			}
			gpsLogsCh <- gpsLogs
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
			if len(gpsLogs) != 0 {
				uploadedGPSLogs = append(uploadedGPSLogs, gpsLogs...)
			}
		default:
			break loop
		}
	}

	// 日ごとに分ける
	const dateFormat = "20060102"
	gpsLogDateMap := map[string][]*reps.GPSLog{}
	fileCount := 0
	for _, gpsLog := range uploadedGPSLogs {
		if _, exist := gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)]; !exist {
			gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)] = []*reps.GPSLog{}
		}
		gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)] = append(gpsLogDateMap[gpsLog.RelatedTime.Format(dateFormat)], gpsLog)
	}
	for range gpsLogDateMap {
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg2.Add(1)
		go func(filename string, gpsLogs []*reps.GPSLog) {
			defer wg2.Done()
			// Mergeだったら既存のデータも混ぜる
			if request.ConflictBehavior == req_res.Merge {
				startTime, err := time.Parse(dateFormat, datestr)
				if err != nil {
					err = fmt.Errorf("error at parse date string %s: %w", datestr, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError = &message.GkillError{
						ErrorCode:    message.ConvertGPSLogError,
						ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
				}
				endTime := startTime.Add(time.Hour * 24).Add(-time.Millisecond)
				existGPSLogs, err := targetRep.GetGPSLogs(r.Context(), &startTime, &endTime)
				if err != nil {
					err = fmt.Errorf("error at exist gpx datas %s: %w", datestr, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillErrorCh2 <- gkillError
					return
				}
				gpsLogs = append(gpsLogs, existGPSLogs...)
			}

			gpxFileContent, err := g.generateGPXFileContent(gpsLogs)
			if err != nil {
				err := fmt.Errorf("error at generate gpx file content filename = %s: %w", filename, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError = &message.GkillError{
					ErrorCode:    message.GenerateGPXFileContentError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
				}
				gkillErrorCh2 <- gkillError
				return
			}
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				err := fmt.Errorf("error at open file filename= %s: %w", filename, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError = &message.GkillError{
					ErrorCode:    message.GetRepPathError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
				}
				gkillErrorCh <- gkillError
				return
			}
			defer file.Close()
			_, err = file.WriteString(gpxFileContent)
			if err != nil {
				err := fmt.Errorf("error at write gpx content to file filename= %s: %w", filename, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError = &message.GkillError{
					ErrorCode:    message.WriteGPXFileError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPLOAD_GPSLOG_FILE_MESSAGE"}),
				}
				gkillErrorCh <- gkillError
				return
			}
		}(estimateCreateFileName, gpsLogs)
	}
	wg2.Wait()

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

func (g *GkillServerAPI) HandleUpdateApplicationConfig(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateApplicationConfigRequest{}
	response := &req_res.UpdateApplicationConfigResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update application config response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateApplicationconfigResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update application config request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateApplicationConfigRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
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
			err = fmt.Errorf("error at update application config user user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateApplicationConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateApplicationConfigSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_SETTINGS_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateAccountStatusResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update accountStatus request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateAccountStatusRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	requesterAccount, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := requesterAccount.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterAccount.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_PASSWORD_RESET_MESSAGE"}),
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
			err = fmt.Errorf("error at update users account user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateUsersAccountStatusError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_ACCOUNT_STATUS_STRUCT_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateAccountStatusSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_ACCOUNT_STATUS_STRUCT_UPDATED_GET_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateUserRepsResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update userReps request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateUserRepsRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	requesterAccount, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterAccount.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象のアカウント情報を取得して更新
	targetAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.TargetUserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.TargetUserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := targetAccount.UserID

	ok, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.DeleteWriteRepositories(r.Context(), userID, request.UpdatedReps)
	if !ok || err != nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUpdatedRepositoriesByUser,
			ErrorMessage: fmt.Sprintf("%s%s", GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_REP_WITH_ERROR_MESSAGE"}), err),
		}
		response.Errors = append(response.Errors, gkillError)
		if err != nil {
			err = fmt.Errorf("error at delete add all repositories by users user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}

		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateRepositoriesSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_REP_WITH_ERROR_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleUpdateServerConfigs(w http.ResponseWriter, r *http.Request) {
	defer func() {
		go g.server.Shutdown(context.Background())
	}()
	func() {
		w.Header().Set("Content-Type", "application/json")
		request := &req_res.UpdateServerConfigsRequest{}
		response := &req_res.UpdateServerConfigsResponse{}

		defer r.Body.Close()
		defer func() {
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				err = fmt.Errorf("error at parse update server config response to json: %w", err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.InvalidUpdateServerConfigResponseDataError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}()

		err := json.NewDecoder(r.Body).Decode(request)
		if err != nil {
			err = fmt.Errorf("error at parse update server config request to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateServerConfigRequestDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// アカウントを取得
		account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
		if err != nil {
			response.Errors = append(response.Errors, gkillError)
			return
		}

		userID := account.UserID
		device, err := g.GetDevice()
		if err != nil {
			err = fmt.Errorf("error at get device name: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// adminじゃなかったら弾く
		if !account.IsAdmin {
			err = fmt.Errorf("%s is not admin", userID)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountNotHasAdminError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "NO_ADMIN_PRIVILEGE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// TLS設定値がTRUEで設定されるとき、証明書ファイルが実在しない場合はエラー
		for _, serverConfig := range request.ServerConfigs {
			if serverConfig.EnableThisDevice {
				if !serverConfig.EnableTLS {
					continue
				}
				_, err := os.Stat(os.ExpandEnv(serverConfig.TLSCertFile))
				if err != nil {
					err = fmt.Errorf("not found tls cert file user id = %s device = %s: %w", userID, device, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.NotFoundTLSCertFileError,
						ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "CERT_FILE_NOT_CREATED_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
				_, err = os.Stat(os.ExpandEnv(serverConfig.TLSKeyFile))
				if err != nil {
					err = fmt.Errorf("not found tls key file user id = %s device = %s: %w", userID, device, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.NotFoundTLSCertFileError,
						ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "CERT_FILE_NOT_CREATED_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
			}
		}

		// mi通知用キーが空のものは登録する
		for _, serverConfig := range request.ServerConfigs {
			if serverConfig.GkillNotificationPrivateKey == "" {
				serverConfig.GkillNotificationPrivateKey, serverConfig.GkillNotificationPublicKey, err = webpush.GenerateVAPIDKeys()
				if err != nil {
					err = fmt.Errorf("error at generate vapid keys: %w", err)

					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.GenerateVAPIDKeysError,
						ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "KEY_GENERATION_ERROR_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
			}
		}

		// ServerConfigを更新する
		ok, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.DeleteWriteServerConfigs(r.Context(), request.ServerConfigs)
		if !ok || err != nil {
			if err != nil {
				err = fmt.Errorf("error at update server config user user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.UpdateServerConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		err = g.GkillDAOManager.Close()
		if err != nil {
			if err != nil {
				err = fmt.Errorf("error at close gkill dao manager: %w", err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.UpdateServerConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SETTINGS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		response.Messages = append(response.Messages, &message.GkillMessage{
			MessageCode: message.UpdateServerConfigSuccessMessage,
			Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_SETTINGS_MESSAGE"}),
		})
		response.Messages = append(response.Messages, &message.GkillMessage{
			MessageCode: message.RebootingMessage,
			Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "JUST_A_MOMENT_MESSAGE"}),
		})
	}()
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddAccountResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add account request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddAccountRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	requesterAccount, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := requesterAccount.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", userID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.AccountInfo.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user device = %s id = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existAccount != nil {
		err = fmt.Errorf("exist account id = %s", userID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistAccountError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウント情報を追加
	defaultApplicationConfig := user_config.GetDefaultApplicationConfig(request.AccountInfo.UserID, device)
	_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(context.TODO(), defaultApplicationConfig)
	if err != nil {
		err = fmt.Errorf("error at add application config user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddApplicationConfig,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	passwordResetToken := GenerateNewID()
	account := &account.Account{
		UserID:             request.AccountInfo.UserID,
		IsAdmin:            request.AccountInfo.IsAdmin,
		IsEnable:           request.AccountInfo.IsEnable,
		PasswordResetToken: &passwordResetToken,
	}
	_, err = g.GkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(r.Context(), account)
	if err != nil {
		err = fmt.Errorf("error at add account user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddApplicationConfig,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err = g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), request.AccountInfo.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if requesterAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.DoInitialize {
		err := g.initializeNewUserReps(r.Context(), requesterAccount)
		if err != nil {
			err = fmt.Errorf("error at initialize new user reps user id = %s device = %s account = %#v: %w", userID, device, request.AccountInfo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddAccountError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_ACCOUNT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.AddedAccountInfo = requesterAccount
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddAccountSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_ACCOUNT_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidGenerateTLSFileResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse generate tls request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidGenerateTLSFileRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// TLSファイル作成操作をしたユーザを特定
	requesterSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	requesterAccount, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(r.Context(), requesterSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", requesterSession.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if requesterAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 管理者権限がなければ弾く
	if !requesterAccount.IsAdmin {
		err = fmt.Errorf("account not has admin user id = %s: %w", requesterSession.UserID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotHasAdminError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_NO_AUTH_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certFileName, pemFileName, err := g.getTLSFileNames(device)
	if err != nil {
		err = fmt.Errorf("error at get tls file names: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTLSFileNamesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.RemoveCertFileError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	if _, err := os.Stat(pemFileName); err == nil {
		err := os.Remove(pemFileName)
		if err != nil {
			err = fmt.Errorf("error at remove pem file %s: %w", pemFileName, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.RemovePemFileError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
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
	isCABool := true
	host := &hostStr
	ecdsaCurve := &ecdsaCurveStr
	ed25519Key := &ed25519KeyBool
	rsaBits := &rsaBitsInt
	validFrom := &validFromStr
	validFor := &validForDuration
	isCA := &isCABool
	if len(*host) == 0 {
		slog.Log(r.Context(), gkill_log.Trace, "finish Missing required --host parameter")
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
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
		slog.Log(r.Context(), gkill_log.Trace, "finish Unrecognized elliptic", "curve", *ecdsaCurve)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to generate private key", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
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
			slog.Log(r.Context(), gkill_log.Trace, "finish Failed to parse creation date", "error", err)
			err = fmt.Errorf("error at generate tls files")
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GenerateTLSFilesError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	notAfter := notBefore.Add(*validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to generate serial number", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
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
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to create certificate", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	parentDirCert, parentDirKey := filepath.Dir(certFileName), filepath.Dir(pemFileName)
	parentDirCert, parentDirKey = filepath.ToSlash(parentDirCert), filepath.ToSlash((parentDirKey))

	err = os.MkdirAll(parentDirCert, os.ModePerm)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to open cert.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = os.MkdirAll(parentDirKey, os.ModePerm)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to open cert.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certOut, err := os.Create(certFileName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to open cert.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to write data to cert.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := certOut.Close(); err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Error closing cert.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	keyOut, err := os.OpenFile(pemFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to open key.pem for writing", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Unable to marshal private key", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Failed to write data to key.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := keyOut.Close(); err != nil {
		slog.Log(r.Context(), gkill_log.Trace, "finish Error closing key.pem", "error", err)
		err = fmt.Errorf("error at generate tls files")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_CREATE_TLS_FILE_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	message := &message.GkillMessage{
		MessageCode: message.TLSFileCreateSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_CREATE_TLS_FILE_MESSAGE"}),
	}
	response.Messages = append(response.Messages, message)
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetGPSLogResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GPS_LOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get gpsLog request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetGPSLogRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GPS_LOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GPS_LOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	gpsLogHistories, err := repositories.GPSLogReps.GetGPSLogs(r.Context(), &request.StartDate, &request.EndDate)
	if err != nil {
		err = fmt.Errorf("error at get gpsLog user id = %s device = %s start time = %s end time = %s: %w", userID, device, request.StartDate.Format(time.RFC3339), request.EndDate.Format(time.RFC3339), err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetGPSLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_GPS_LOG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GPSLogs = gpsLogHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGPSLogSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_GPS_LOG_MESSAGE"}),
	})
}

func privateIPv4s() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, iface := range ifaces {
		// down / loopback は除外
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			ip4 := ip.To4()
			if ip4 == nil {
				continue // IPv4のみ
			}

			// 169.254.x.x (link-local) などは除外
			if ip4.IsLinkLocalUnicast() {
				continue
			}

			if isPrivateIPv4(ip4) {
				ips = append(ips, ip4)
			}
		}
	}
	return ips, nil
}

func isPrivateIPv4(ip net.IP) bool {
	// ip must be 4 bytes (To4済み想定)
	// RFC1918: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	switch {
	case ip[0] == 10:
		return true
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return true
	case ip[0] == 192 && ip[1] == 168:
		return true
	default:
		return false
	}
}

func globalIP(ctx context.Context) (net.IP, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	s := strings.TrimSpace(string(b))
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip response: %q", s)
	}
	return ip, nil
}

func (g *GkillServerAPI) HandleAddShareKyouListInfo(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddShareKyouListInfoRequest{}
	response := &req_res.AddShareKyouListInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add ShareKyouListInfo response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddShareKyouListInfoResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add ShareKyouListInfo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddShareKyouListInfoRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo != nil {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	shareKyouInfo := &share_kyou_info.ShareKyouInfo{
		ID:                   GenerateNewID(),
		ShareID:              request.ShareKyouListInfo.ShareID,
		UserID:               request.ShareKyouListInfo.UserID,
		Device:               request.ShareKyouListInfo.Device,
		ShareTitle:           request.ShareKyouListInfo.ShareTitle,
		FindQueryJSON:        request.ShareKyouListInfo.FindQueryJSON,
		ViewType:             request.ShareKyouListInfo.ViewType,
		IsShareTimeOnly:      request.ShareKyouListInfo.IsShareTimeOnly,
		IsShareWithTags:      request.ShareKyouListInfo.IsShareWithTags,
		IsShareWithTexts:     request.ShareKyouListInfo.IsShareWithTexts,
		IsShareWithTimeIss:   request.ShareKyouListInfo.IsShareWithTimeIss,
		IsShareWithLocations: request.ShareKyouListInfo.IsShareWithLocations,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.AddKyouShareInfo(r.Context(), shareKyouInfo)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add ShareKyouListInfo user id = %s device = %s ShareKyouListInfo = %#v: %w", userID, device, request.ShareKyouListInfo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_ADDED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfo = ShareKyouListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddShareKyouListInfoSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_SHARE_KYOU_LIST_INFO_ADDED_GET_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleUpdateShareKyouListInfo(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateShareKyouListInfoRequest{}
	response := &req_res.UpdateShareKyouListInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add ShareKyouListInfo response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateShareKyouListInfoResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add ShareKyouListInfo request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateShareKyouListInfoRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo == nil {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.NotExistShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	shareKyouInfo := &share_kyou_info.ShareKyouInfo{
		ID:                   GenerateNewID(),
		ShareID:              request.ShareKyouListInfo.ShareID,
		UserID:               request.ShareKyouListInfo.UserID,
		Device:               request.ShareKyouListInfo.Device,
		ShareTitle:           request.ShareKyouListInfo.ShareTitle,
		FindQueryJSON:        request.ShareKyouListInfo.FindQueryJSON,
		ViewType:             request.ShareKyouListInfo.ViewType,
		IsShareTimeOnly:      request.ShareKyouListInfo.IsShareTimeOnly,
		IsShareWithTags:      request.ShareKyouListInfo.IsShareWithTags,
		IsShareWithTexts:     request.ShareKyouListInfo.IsShareWithTexts,
		IsShareWithTimeIss:   request.ShareKyouListInfo.IsShareWithTimeIss,
		IsShareWithLocations: request.ShareKyouListInfo.IsShareWithLocations,
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.UpdateKyouShareInfo(r.Context(), shareKyouInfo)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add ShareKyouListInfo user id = %s device = %s ShareKyouListInfo = %#v: %w", userID, device, request.ShareKyouListInfo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_SHARE_KYOU_LIST_INFO_UPDATED_GET_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfo = ShareKyouListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateShareKyouListInfoSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_UPDATE_SHARE_KYOU_LIST_INFO_UPDATED_GET_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleGetShareKyouListInfos(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetShareKyouListInfosRequest{}
	response := &req_res.GetShareKyouListInfosResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get ShareKyouListInfos response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetShareKyouListInfosResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get ShareKyouListInfos request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetShareKyouListInfosRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouList, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfos(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfos user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfosError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfos = ShareKyouList
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetShareKyouListInfosSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleDeleteShareKyouListInfos(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DeleteShareKyouListInfoRequest{}
	response := &req_res.DeleteShareKyouListInfosResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete ShareKyouListInfos response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidDeleteShareKyouListInfosResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete ShareKyouListInfos request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDeleteShareKyouListInfosRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを削除
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.DeleteKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete ShareKyouListInfos user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteShareKyouListInfosError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.DeleteShareKyouListInfosSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_DELETE_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleGetSharedKyous(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSharedKyousRequest{}
	response := &req_res.GetSharedKyousResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete ShareKyouListInfos response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiSharedTasksResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete ShareKyouListInfos request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTasksRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.SharedID)
	if err != nil || sharedKyouInfo == nil {
		err = fmt.Errorf("error at get ShareKyouListInfos shared id = %s: %w", request.SharedID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiSharedTasksError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SHARE_KYOU_LIST_INFOS_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := sharedKyouInfo.UserID
	device := sharedKyouInfo.Device

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	findQuery := &find.FindQuery{}
	err = json.Unmarshal([]byte(sharedKyouInfo.FindQueryJSON), findQuery)
	if err != nil {
		err = fmt.Errorf("error at parse query json at find kyous %#v: %w", findQuery, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTaskRequest,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	trueValue := true
	falseValue := false
	findQuery.OnlyLatestData = &trueValue

	// Kyou
	findFilter := &FindFilter{}
	kyous, _, err := findFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, findQuery)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	findQueryValueForKyouInstances := *findQuery
	findQueryForKyouInstances := &findQueryValueForKyouInstances
	findQueryForKyouInstances.UseIDs = &trueValue
	findQueryForKyouInstances.IncludeCreateMi = &trueValue
	findQueryForKyouInstances.IncludeStartMi = &trueValue
	findQueryForKyouInstances.IncludeCheckMi = &trueValue
	findQueryForKyouInstances.IncludeEndMi = &trueValue
	findQueryForKyouInstances.IncludeLimitMi = &trueValue
	findQueryForKyouInstances.IncludeEndTimeIs = &trueValue
	findQueryForKyouInstances.IDs = &[]string{}
	for _, kyou := range kyous {
		*findQueryForKyouInstances.IDs = append(*findQueryForKyouInstances.IDs, kyou.ID)
	}
	findQueryForKyouInstances.OnlyLatestData = &falseValue

	// Mi
	mis, err := repositories.MiReps.FindMi(r.Context(), findQueryForKyouInstances)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// GPSLogs
	gpsLogs := []*reps.GPSLog{}
	if sharedKyouInfo.IsShareWithLocations {
		gpsLogs, err = repositories.GPSLogReps.GetGPSLogs(r.Context(), findQuery.CalendarStartDate, findQuery.CalendarEndDate)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	kmemos := []*reps.Kmemo{}
	kcs := []*reps.KC{}
	timeiss := []*reps.TimeIs{}
	nlogs := []*reps.Nlog{}
	lantanas := []*reps.Lantana{}
	urlogs := []*reps.URLog{}
	idfKyous := []*reps.IDFKyou{}
	rekyous := []*reps.ReKyou{}
	gitCommitLogs := []*reps.GitCommitLog{}
	if sharedKyouInfo.ViewType != "mi" {
		// Kmemo
		kmemos, err = repositories.KmemoReps.FindKmemo(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// KC
		kcs, err = repositories.KCReps.FindKC(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// TimeIs
		timeiss, err = repositories.TimeIsReps.FindTimeIs(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// Nlog
		nlogs, err = repositories.NlogReps.FindNlog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// Lantana
		lantanas, err = repositories.LantanaReps.FindLantana(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// URLogs
		urlogs, err = repositories.URLogReps.FindURLog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// IDFKyou
		idfKyous, err = repositories.IDFKyouReps.FindIDFKyou(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// ReKyou
		rekyous, err = repositories.ReKyouReps.FindReKyou(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// GitCommitLog
		gitCommitLogs, err = repositories.GitCommitLogReps.FindGitCommitLog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_KYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// AttachedTag
	tags := []*reps.Tag{}
	tagSet := map[string]*reps.Tag{}
	if sharedKyouInfo.IsShareWithTags {
		for _, kyou := range kyous {
			tagsRelatedID, err := repositories.GetTagsByTargetID(r.Context(), kyou.ID)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTagsShareKyouError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TAGS_BY_TARGET_ID_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			for _, tag := range tagsRelatedID {
				tagSet[tag.ID] = tag
			}
		}
		for _, tag := range tagSet {
			tags = append(tags, tag)
		}
	}

	// AttachedText
	texts := []*reps.Text{}
	textSet := map[string]*reps.Text{}
	if sharedKyouInfo.IsShareWithTexts {
		for _, kyou := range kyous {
			textsRelatedID, err := repositories.GetTextsByTargetID(r.Context(), kyou.ID)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTextsShareKyouError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			for _, text := range textsRelatedID {
				textSet[text.ID] = text
			}
		}
		for _, text := range textSet {
			texts = append(texts, text)
		}
	}

	// AttachedTimeIs
	attachedTimeisKyous := []*reps.Kyou{}
	attachedTimeiss := []*reps.TimeIs{}
	if sharedKyouInfo.IsShareWithTimeIss {
		trueValue := true
		attachedTimeIsKyousMap := map[string]*reps.Kyou{}
		attachedTimeIssMap := map[string]*reps.TimeIs{}
		queries := []*find.FindQuery{}

		timeisQueryValue := *findQuery
		timeisQuery := &timeisQueryValue
		timeisQuery.UseRepTypes = &trueValue
		timeisQuery.RepTypes = &[]string{"timeis"}
		timeisQuery.OnlyLatestData = &trueValue
		queries = append(queries, timeisQuery)

		if timeisQuery.UseCalendar != nil && *timeisQuery.UseCalendar && timeisQuery.CalendarStartDate != nil {
			timeisPlaingHeadQuery := &find.FindQuery{}
			timeisPlaingHeadQuery.UsePlaing = &trueValue
			timeisPlaingHeadQuery.PlaingTime = timeisQuery.CalendarStartDate
			queries = append(queries, timeisPlaingHeadQuery)
		}

		if timeisQuery.UseCalendar != nil && *timeisQuery.UseCalendar && timeisQuery.CalendarEndDate != nil {
			timeisPlaingHipQuery := &find.FindQuery{}
			timeisPlaingHipQuery.UsePlaing = &trueValue
			timeisPlaingHipQuery.PlaingTime = timeisQuery.CalendarEndDate
			queries = append(queries, timeisPlaingHipQuery)
		}

		for _, query := range queries {
			findFilter := &FindFilter{}
			matchPlaingKyous, _, err := findFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, query)
			if err != nil {
				err = fmt.Errorf("error at find tags user id = %s device = %s: %w", userID, device, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTextsShareKyouError,
					ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TEXTS_BY_TARGET_ID_MESSAGE"}),
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			for _, timeisKyou := range matchPlaingKyous {
				if existKyou, exist := attachedTimeIsKyousMap[timeisKyou.ID]; exist {
					if timeisKyou.UpdateTime.After(existKyou.UpdateTime) {
						attachedTimeIsKyousMap[timeisKyou.ID] = timeisKyou
					}
				} else {
					attachedTimeIsKyousMap[timeisKyou.ID] = timeisKyou
				}
			}

			ids := []string{}
			for id := range attachedTimeIsKyousMap {
				ids = append(ids, id)
			}
			if len(ids) != 0 {
				plaingTimeIss, err := repositories.TimeIsReps.FindTimeIs(r.Context(), query)
				if err != nil {
					err = fmt.Errorf("error at find plaing timeis user id = %s device = %s: %w", userID, device, err)
					slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
					gkillError := &message.GkillError{
						ErrorCode:    message.FindTextsShareKyouError,
						ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_TIMEIS_MESSAGE"}),
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
				for _, timeis := range plaingTimeIss {
					attachedTimeIssMap[timeis.ID] = timeis
					if existTimeIs, exist := attachedTimeIssMap[timeis.ID]; exist {
						if timeis.UpdateTime.After(existTimeIs.UpdateTime) {
							attachedTimeIssMap[timeis.ID] = timeis
						}
					} else {
						attachedTimeIssMap[timeis.ID] = timeis
					}
				}
			}

			for _, kyou := range attachedTimeIsKyousMap {
				attachedTimeisKyous = append(attachedTimeisKyous, kyou)
			}
			for _, timeis := range attachedTimeIssMap {
				attachedTimeiss = append(attachedTimeiss, timeis)
			}
		}
	}

	response.Kyous = kyous
	response.Mis = mis
	response.Kmemos = kmemos
	response.KCs = kcs
	response.TimeIss = timeiss
	response.Nlogs = nlogs
	response.Lantanas = lantanas
	response.URLogs = urlogs
	response.IDFKyous = idfKyous
	response.ReKyous = rekyous
	response.GitCommitLogs = gitCommitLogs
	response.GPSLogs = gpsLogs
	response.AttachedTags = tags
	response.AttachedTexts = texts
	response.Title = sharedKyouInfo.ShareTitle
	response.ViewType = sharedKyouInfo.ViewType
	response.AttachedTimeIss = attachedTimeiss
	response.AttachedTimeIsKyous = attachedTimeisKyous
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSharedTasksSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_KYOU_MESSAGE"}),
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
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetRepositoriesResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get repositories request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetRepositoriesRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.GetRepositories(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_REPOSITORIES_MESSAGE"}),
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
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_GET_REPOSITORIES_MESSAGE"}),
	})
}

func (g *GkillServerAPI) getAccountFromSessionIDWithApplicationName(ctx context.Context, sessionID string, applicationName string, localeName string) (*account.Account, *message.GkillError, error) {
	loginSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(ctx, sessionID)
	if loginSession == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", sessionID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}
	if loginSession.ApplicationName != applicationName {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, loginSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ACCOUNT_AUTH_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	if !account.IsEnable {
		err = fmt.Errorf("error at disable account user id = %s: %w", loginSession.UserID, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountDisabledError,
			ErrorMessage: GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "ACCOUNT_DISABLED_MESSAGE"}),
		}
		return nil, gkillError, err
	}

	return account, nil, nil

}

func (g *GkillServerAPI) getAccountFromSessionID(ctx context.Context, sessionID string, localeName string) (*account.Account, *message.GkillError, error) {
	return g.getAccountFromSessionIDWithApplicationName(ctx, sessionID, "gkill", localeName)
}

func GenerateNewID() string {
	return uuid.New().String()
}

func (g *GkillServerAPI) resolveFileName(repDir string, filename string, behavior req_res.FileUploadConflictBehavior) (string, error) {
	fullFilename := filepath.Join(repDir, filename)
	_, err := os.Stat(fullFilename)
	if err != nil {
		return fullFilename, nil
	} else {
		switch string(behavior) {
		case string(req_res.Override):
			return fullFilename, nil
		case string(req_res.Rename):
			// カッコのついていないファイル名。例えば「hogehoge (1).txt」なら「hogehoge.txt」。
			planeFileName := g.planeFileName(fullFilename)
			ext := filepath.Ext(planeFileName)
			withoutExt := planeFileName[:len(planeFileName)-len(ext)]

			// ファイルが存在しない名前になるまでカッコ内の数字をインクリメントし続ける
			// targetFilenameは最終的な移動先ファイル名
			fullFilename = planeFileName
			for count := 1; ; count++ {
				if _, err := os.Stat(fullFilename); err != nil {
					break
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
		case string(req_res.Merge):
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

	buf := bytes.NewBufferString("")
	writer := bufio.NewWriter(buf)
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

	return buf.String(), nil
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

	userDataRootDirectory := filepath.Join(os.ExpandEnv(serverConfig.UserDataDirectory), account.UserID)
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

	repositories := []*user_config.Repository{}

	repTypeFileNameMap := map[string]string{}
	repTypeFileNameMap["kmemo"] = "Kmemo.db"
	repTypeFileNameMap["kc"] = "KC.db"
	repTypeFileNameMap["urlog"] = "URLog.db"
	repTypeFileNameMap["timeis"] = "TimeIs.db"
	repTypeFileNameMap["mi"] = "Mi.db"
	repTypeFileNameMap["nlog"] = "Nlog.db"
	repTypeFileNameMap["lantana"] = "Lantana.db"
	repTypeFileNameMap["tag"] = "Tag.db"
	repTypeFileNameMap["text"] = "Text.db"
	repTypeFileNameMap["notification"] = "Notification.db"
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
			ID:                        GenerateNewID(),
			UserID:                    account.UserID,
			Device:                    device,
			Type:                      repType,
			File:                      repFileFullName,
			UseToWrite:                true,
			IsExecuteIDFWhenReload:    true,
			IsWatchTargetForUpdateRep: false,
			IsEnable:                  true,
		}
		repositories = append(repositories, repository)
	}

	repType, repFileName := "directory", "Files"
	repFileFullName := filepath.Join(userDataRootDirectory, repFileName)
	err = os.MkdirAll(os.ExpandEnv(repFileFullName), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository create directory reptype = %s repdirname = %s: %w", repType, repFileFullName, err)
		return err
	}
	repository := &user_config.Repository{
		ID:                        GenerateNewID(),
		UserID:                    account.UserID,
		Device:                    device,
		Type:                      repType,
		File:                      repFileFullName,
		UseToWrite:                true,
		IsExecuteIDFWhenReload:    true,
		IsWatchTargetForUpdateRep: false,
		IsEnable:                  true,
	}
	repositories = append(repositories, repository)

	repType, repFileName = "gpslog", "GPSLog"
	repFileFullName = filepath.Join(userDataRootDirectory, repFileName)
	err = os.MkdirAll(os.ExpandEnv(repFileFullName), fs.ModePerm)
	if err != nil {
		err = fmt.Errorf("error at initialize new user reps. error at add repository create directory reptype = %s repdirname = %s: %w", repType, repFileFullName, err)
		return err
	}
	repository = &user_config.Repository{
		ID:                        GenerateNewID(),
		UserID:                    account.UserID,
		Device:                    device,
		Type:                      repType,
		File:                      repFileFullName,
		UseToWrite:                true,
		IsExecuteIDFWhenReload:    true,
		IsWatchTargetForUpdateRep: false,
		IsEnable:                  true,
	}
	repositories = append(repositories, repository)

	ok, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.DeleteWriteRepositories(ctx, account.UserID, repositories)
	if !ok || err != nil {
		err = fmt.Errorf("error at delete write repositories: %w", err)
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
	ctx := context.Background()
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		return "", "", err
	}
	return serverConfig.TLSCertFile, serverConfig.TLSKeyFile, nil
}

func (g *GkillServerAPI) HandleFileServe(w http.ResponseWriter, r *http.Request) {

	sessionID := ""
	sharedID := ""

	// クッキーを見て認証する
	sessionIDCookie, err := r.Cookie("gkill_session_id")
	if err != nil {
		sharedIDCookie, err := r.Cookie("gkill_shared_id")
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
			return
		}
		sharedID = strings.ReplaceAll(sharedIDCookie.Value, "shared_id", "")
	} else {
		sessionID = sessionIDCookie.Value
	}

	// アカウントを取得
	// NGであれば403でreturn
	userID := ""
	if sessionID != "" {
		account, gkillError, err := g.getAccountFromSessionID(r.Context(), sessionID, "")
		if account == nil || gkillError != nil || err != nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
			return
		}
		userID = account.UserID
	} else if sharedID != "" {
		sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), sharedID)
		if err != nil || sharedKyouInfo == nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
			return
		}
		userID = sharedKyouInfo.UserID
	} else {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
		return
	}

	// リクエストPathから対象Rep名を抽出
	targetRepName := strings.SplitN(r.URL.Path, "/", 4)[2]

	// OKであればRepNameが一致するIDFRepを探す
	var targetIDFRep reps.IDFKyouRepository
	idfRepImpls, err := repositories.IDFKyouReps.UnWrapTyped()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
		return
	}
	for _, idfRep := range idfRepImpls {
		repName, err := idfRep.GetRepName(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("error at handle file serve: %w", err)
			slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
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
		slog.Log(r.Context(), gkill_log.Trace, "finish", "error", err)
		return
	}

	// StripPrefixしてIDFサーバのハンドラにわたす
	rootAddress := "/files/" + targetRepName
	http.StripPrefix(rootAddress, http.HandlerFunc(targetIDFRep.HandleFileServe)).ServeHTTP(w, r)
}

func (g *GkillServerAPI) HandleGetGkillNotificationPublicKey(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGkillNotificationPublicKeyRequest{}
	response := &req_res.GetGkillNotificationPublicKeyResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi task notification public key response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiTaskNotificationPublicKeyResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get get mi task notification public key request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiTaskNotificationPublicKeyRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.Device == device {
			currentServerConfig = serverConfig
			break
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GkillNotificationPublicKey = currentServerConfig.GkillNotificationPublicKey
}
func (g *GkillServerAPI) HandleRegisterGkillNotification(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.RegisterGkillNotificationRequest{}
	response := &req_res.RegisterGkillNotificationResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse register mi task notification response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterMiTaskNotificationResponse,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get register mi task notification request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterMiTaskNotificationRequest,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID

	gkillNotificationTarget := &gkill_notification.GkillNotificateTarget{
		ID:           sqlite3impl.GenerateNewID(),
		UserID:       userID,
		PublicKey:    request.PublicKey,
		Subscription: gkill_notification.JSONString(request.Subscription),
	}

	_, err = g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.AddGkillNotificationTarget(r.Context(), gkillNotificationTarget)
	if err != nil {
		err = fmt.Errorf("error at add mi notification target : %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddGkillNotificationTargetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTagSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_REGIST_MI_TASK_NOTIFICATION_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleOpenDirectory(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.OpenDirectoryRequest{}
	response := &req_res.OpenDirectoryResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse open directory response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterOpenDirectoryResponse,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse open directory request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenDirectoryRequest,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if !session.IsLocalAppUser {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderNotLocalAccountError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(session.UserID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories. userid = %s device = %s: %w", session.UserID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	filename, err := repositories.Reps.GetPath(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get path. id = %s userid = %s device = %s: %w", request.TargetID, session.UserID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	dirname := filepath.Dir(filename)

	cmd := os.Expand(serverConfig.OpenDirectoryCommand, func(str string) string {
		if str == "filename" {
			return filename
		}
		if str == "dirname" {
			return dirname
		}
		return ""
	})
	spl := strings.Split(cmd, " ")
	cmd, args := spl[0], spl[1:]

	err = exec.Command(cmd, args...).Start()
	if err != nil {
		err = fmt.Errorf("error at open file. device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FOLDER_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.OpenDirectorySuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_OPEN_FOLDER_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleOpenFile(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.OpenFileRequest{}
	response := &req_res.OpenFileResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse open file response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterOpenFileResponse,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse open file request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenFileRequest,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if !session.IsLocalAppUser {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderNotLocalAccountError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(session.UserID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories. userid = %s device = %s: %w", session.UserID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	filename, err := repositories.Reps.GetPath(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get path. id = %s userid = %s device = %s: %w", request.TargetID, session.UserID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	dirname := filepath.Dir(filename)

	cmd := os.Expand(serverConfig.OpenFileCommand, func(str string) string {
		if str == "filename" {
			return filename
		}
		if str == "dirname" {
			return dirname
		}
		return ""
	})
	spl := strings.Split(cmd, " ")
	cmd, args := spl[0], spl[1:]

	err = exec.Command(cmd, args...).Start()
	if err != nil {
		err = fmt.Errorf("error at open file. device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_OPEN_FILE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.OpenFileSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_OPEN_FILE_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleReloadRepositories(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ReloadRepositoriesRequest{}
	response := &req_res.ReloadRepositoriesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse reload repositories response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidReloadRepositoriesResponse,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse reload repositories request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenFileRequest,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	_, err = repositories.LatestDataRepositoryAddressDAO.DeleteAllLatestDataRepositoryAddress(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.ClearThumbCache {
		err = repositories.IDFKyouReps.ClearThumbCache()
		if err != nil {
			err = fmt.Errorf("error at clear thumb cache: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.RepositoriesGetError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	_, err = g.GkillDAOManager.CloseUserRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	_, err = g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_RELOAD_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.ReloadRepositoriesSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_RELOAD_MESSAGE"}),
	})
}

func (g *GkillServerAPI) ifRedirectResetAdminAccountIsNotFound(w http.ResponseWriter, r *http.Request) bool {
	// GET 以外は対象外
	if r.Method != http.MethodGet {
		return false
	}

	// ブラウザの通常ナビゲーション(HTMLドキュメント)の時だけ
	if d := r.Header.Get("Sec-Fetch-Dest"); d != "" && d != "document" {
		return false
	}
	if m := r.Header.Get("Sec-Fetch-Mode"); m != "" && m != "navigate" && m != "nested-navigate" {
		return false
	}

	p := r.URL.Path
	if strings.HasPrefix(p, "/assets/") ||
		strings.HasSuffix(p, ".js") ||
		strings.HasSuffix(p, ".css") ||
		strings.HasSuffix(p, ".map") ||
		strings.HasSuffix(p, ".png") ||
		strings.HasSuffix(p, ".svg") ||
		strings.HasSuffix(p, ".ico") ||
		strings.HasSuffix(p, ".webmanifest") {
		return false
	}

	accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get all account config")
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllAccountConfigError,
			ErrorMessage: GetLocalizer("").MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_ACCOUNT_CONFIG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return false
	}

	if len(accounts) == 1 {
		if accounts[0].UserID != "admin" || accounts[0].PasswordSha256 != nil {
			return false
		}

		http.Redirect(w, r, fmt.Sprintf("/regist_first_account?reset_token=%s", *accounts[0].PasswordResetToken), http.StatusTemporaryRedirect)
		// http.Redirect(w, r, fmt.Sprintf("/set_new_password?reset_token=%s&user_id=%s", *accounts[0].PasswordResetToken, accounts[0].UserID), http.StatusTemporaryRedirect)
		return true
	}
	return false
}

func (g *GkillServerAPI) HandleURLogBookmarkletAddress(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	request := &req_res.URLogBookmarkletRequest{}

	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse urlog bookmarklet request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddKmemoRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		// response.Errors = append(response.Errors, gkillError)
		_ = gkillError
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionIDWithApplicationName(r.Context(), request.SessionID, "urlog_bookmarklet", request.LocaleName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	var imgBase64 string
	if request.ImageURL != "" {
		imgBase64, err = httpGetBase64Data(request.ImageURL)
		if err != nil {
			err = fmt.Errorf("error at http get base 64 data from %s: %w", request.ImageURL, err)
			log.Printf("err = %+v\n", err)
		}
	}
	var faviconBase64 string
	if request.FaviconURL != "" {
		faviconBase64, err = httpGetBase64Data(request.FaviconURL)
		if err != nil {
			err = fmt.Errorf("error at http get base 64 data from %s: %w", request.FaviconURL, err)
			log.Printf("err = %+v\n", err)
		}
	}

	urlog := &reps.URLog{
		IsDeleted:      false,
		ID:             GenerateNewID(),
		RelatedTime:    time.Now(),
		CreateTime:     time.Now(),
		CreateApp:      "urlog_bookmarklet",
		CreateDevice:   device,
		CreateUser:     userID,
		UpdateTime:     time.Now(),
		UpdateApp:      "urlog_bookmarklet",
		UpdateUser:     userID,
		UpdateDevice:   device,
		URL:            request.URL,
		Title:          request.Title,
		Description:    request.Description,
		FaviconImage:   faviconBase64,
		ThumbnailImage: imgBase64,
	}

	// 対象が存在する場合はエラー
	existURLog, err := repositories.URLogReps.GetURLog(r.Context(), urlog.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", urlog.ID)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// applicationConfigを取得
	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_APPLICATION_CONFIG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	urlog.FillURLogField(serverConfig, applicationConfig)

	err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), urlog)
	if err != nil {
		err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AddURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, urlog.ID)

	if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
		err = repositories.URLogReps[0].AddURLogInfo(r.Context(), urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	repositories.LatestDataRepositoryAddresses[urlog.ID] = &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              urlog.IsDeleted,
		TargetID:                               urlog.ID,
		DataUpdateTime:                         urlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	}
	go func() {
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[urlog.ID])
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
	}()

	// 通知する
	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}

	// 送信対象を取得する
	notificationTargets, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(r.Context(), userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}

	content := &struct {
		Content        string    `json:"content"`
		URL            string    `json:"url"`
		Time           time.Time `json:"time"`
		IsNotification bool      `json:"is_notification"`
	}{
		Content:        urlog.Title,
		URL:            "/kyou?kyou_id=" + urlog.ID,
		Time:           urlog.CreateTime,
		IsNotification: true,
	}
	contentJSONb, err := json.Marshal(content)
	if err != nil {
		err = fmt.Errorf("error at marshal webpush content: %w", err)
		slog.Log(r.Context(), gkill_log.Error, "error", "error", err)
		return
	}

	for _, notificationTarget := range notificationTargets {
		subscription := string(notificationTarget.Subscription)
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subscription), s)
		resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  currentServerConfig.GkillNotificationPublicKey,
			VAPIDPrivateKey: currentServerConfig.GkillNotificationPrivateKey,
			TTL:             0,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(r.Context(), notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
	}
}

func (g *GkillServerAPI) HandleGetUpdatedDatasByTime(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	request := &req_res.GetUpdatedDatasByTimeRequest{}
	response := &req_res.GetUpdatedDatasByTimeResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get updated data by time response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetUpdatedDatasByTimeResponse,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LATEST_INFO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get updated data by time request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetUpdatedDatasByTimeRequest,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LATEST_INFO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LATEST_INFO_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	limit := gkill_options.CacheClearCountLimit + 1
	updatedInfos, err := repositories.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressByUpdateTimeAfter(r.Context(), request.LastUpdatedTime, limit)
	if err != nil {
		err = fmt.Errorf("error at get latest data repositories data user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLatestDataRepositoryAddressByUpdateTimeAfterError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_LATEST_INFO_MESSAGE2"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	for _, updatedInfo := range updatedInfos {
		response.UpdatedIDs = append(response.UpdatedIDs, updatedInfo.TargetID)
		if updatedInfo.TargetIDInData != nil {
			response.UpdatedIDs = append(response.UpdatedIDs, *updatedInfo.TargetIDInData)
		}
	}
}

func (g *GkillServerAPI) GetDevice() (string, error) {
	ctx := context.Background()
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		return "", err
	}

	var device *string
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			if device != nil {
				err = fmt.Errorf("invalid status. enable device count is not 1")
				return "", err
			}
			device = &serverConfig.Device
		}
	}
	if device == nil {
		err = fmt.Errorf("invalid status. enable device count is not 1")
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

func httpGetBase64Data(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		err = fmt.Errorf("error at new http get request: %w", err)
		return "", err
	}
	req.Header.Set("Referer", url)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("error at http get %s: %w", url, err)
		return "", err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("error at read all body %s: %w", url, err)
		return "", err
	}

	base64Data := base64.StdEncoding.EncodeToString(b)
	return base64Data, nil
}

func (g *GkillServerAPI) filterLocalOnly(w http.ResponseWriter, r *http.Request) bool {
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		/*
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
			    ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		*/
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		/*
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_GET_SERVER_CONFIG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		*/
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	if serverConfig == nil {
		err = fmt.Errorf("error at server config is nil device = %s: %w", device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	if !serverConfig.IsLocalOnlyAccess {
		return true
	}

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
		return true
	}
	w.WriteHeader(http.StatusForbidden)
	return false
}

func (g *GkillServerAPI) WebPushUpdatedData(ctx context.Context, userID string, device string, kyouID string) {
	// 通知する
	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return
	}

	// 送信対象を取得する
	notificationTargets, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(ctx, userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return
	}

	content := &struct {
		IsUpdatedDataNotify bool   `json:"is_updated_data_notify"`
		ID                  string `json:"id"`
	}{
		IsUpdatedDataNotify: true,
		ID:                  kyouID,
	}
	contentJSONb, err := json.Marshal(content)
	if err != nil {
		err = fmt.Errorf("error at marshal webpush content: %w", err)
		slog.Log(ctx, gkill_log.Error, "error", "error", err)
		return
	}

	for _, notificationTarget := range notificationTargets {
		subscription := string(notificationTarget.Subscription)
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subscription), s)
		resp, err := webpush.SendNotification(contentJSONb, s, &webpush.Options{
			Subscriber:      "example@example.com",
			VAPIDPublicKey:  currentServerConfig.GkillNotificationPublicKey,
			VAPIDPrivateKey: currentServerConfig.GkillNotificationPrivateKey,
			TTL:             0,
		})
		if err != nil {
			err = fmt.Errorf("error at send gkill notification: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", err)
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(ctx, notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				slog.Log(ctx, gkill_log.Debug, "error", "error", err)
			}
		}
	}
}

func (g *GkillServerAPI) HandleCommitTx(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.CommitTxRequest{}
	response := &req_res.CommitTxResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse commit tx response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidCommitTxResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse commit tx request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidCommitTxRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	txID := request.TXID

	kmemos, err := repositories.TempReps.KmemoTempRep.GetKmemosByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kmemo by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	kcs, err := repositories.TempReps.KCTempRep.GetKCsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kc by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyous, err := repositories.TempReps.IDFKyouTempRep.GetIDFKyousByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get idfkyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantanas, err := repositories.TempReps.LantanaTempRep.GetLantanasByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get lantana by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	mis, err := repositories.TempReps.MiTempRep.GetMisByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get mi by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlogs, err := repositories.TempReps.NlogTempRep.GetNlogsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get nlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notifications, err := repositories.TempReps.NotificationTempRep.GetNotificationsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notification by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyous, err := repositories.TempReps.ReKyouTempRep.GetReKyousByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get rekyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tags, err := repositories.TempReps.TagTempRep.GetTagsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get tag by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	texts, err := repositories.TempReps.TextTempRep.GetTextsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get text by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeiss, err := repositories.TempReps.TimeIsTempRep.GetTimeIssByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get timeis by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlogs, err := repositories.TempReps.URLogTempRep.GetURLogsByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get urlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	for _, idfKyou := range idfKyous {
		err = repositories.WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), idfKyou)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// defer g.WebPushUpdatedData(r.Context(), userID, device, request.IDFKyou.ID)

		repName, err := repositories.WriteIDFKyouRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_UPDATE_IDFKYOU_UPDATED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.IDFKyouReps) == 1 && *gkill_options.CacheIDFKyouReps {
			err = repositories.IDFKyouReps[0].AddIDFKyouInfo(r.Context(), idfKyou)
			if err != nil {
				err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}
		repositories.LatestDataRepositoryAddresses[idfKyou.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              idfKyou.IsDeleted,
			TargetID:                               idfKyou.ID,
			DataUpdateTime:                         idfKyou.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[idfKyou.ID])
			if err != nil {
				err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}
	for _, kc := range kcs {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), kc)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
			err = repositories.KCReps[0].AddKCInfo(r.Context(), kc)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteKCRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KC_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[kc.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              kc.IsDeleted,
			TargetID:                               kc.ID,
			DataUpdateTime:                         kc.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[kc.ID])
			if err != nil {
				err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, kmemo := range kmemos {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_KMEMO_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[kmemo.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              kmemo.IsDeleted,
			TargetID:                               kmemo.ID,
			DataUpdateTime:                         kmemo.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[kmemo.ID])
			if err != nil {
				err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, lantana := range lantanas {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_LANTANA_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[lantana.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              lantana.IsDeleted,
			TargetID:                               lantana.ID,
			DataUpdateTime:                         lantana.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[lantana.ID])
			if err != nil {
				err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, mi := range mis {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteMiRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_MI_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[mi.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              mi.IsDeleted,
			TargetID:                               mi.ID,
			DataUpdateTime:                         mi.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[mi.ID])
			if err != nil {
				err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, nlog := range nlogs {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[nlog.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              nlog.IsDeleted,
			TargetID:                               nlog.ID,
			DataUpdateTime:                         nlog.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[nlog.ID])
			if err != nil {
				err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, notification := range notifications {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.NotificationReps) == 1 && *gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNotificationError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_NOTIFICATION_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[notification.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              notification.IsDeleted,
			TargetID:                               notification.ID,
			TargetIDInData:                         &notification.TargetID,
			DataUpdateTime:                         notification.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[notification.ID])
			if err != nil {
				err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, rekyou := range rekyous {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), rekyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && *gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), rekyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_REKYOU_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[rekyou.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              rekyou.IsDeleted,
			TargetID:                               rekyou.ID,
			TargetIDInData:                         &rekyou.TargetID,
			DataUpdateTime:                         rekyou.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[rekyou.ID])
			if err != nil {
				err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, tag := range tags {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && *gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteTagRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TAG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[tag.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &tag.TargetID,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[tag.ID])
			if err != nil {
				err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, text := range texts {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TextReps) == 1 && *gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteTextRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTextError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TEXT_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[text.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              text.IsDeleted,
			TargetID:                               text.ID,
			TargetIDInData:                         &text.TargetID,
			DataUpdateTime:                         text.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[text.ID])
			if err != nil {
				err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, timeis := range timeiss {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), timeis)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), timeis)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_TIMEIS_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[timeis.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              timeis.IsDeleted,
			TargetID:                               timeis.ID,
			DataUpdateTime:                         timeis.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[timeis.ID])
			if err != nil {
				err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	for _, urlog := range urlogs {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), urlog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}

		repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_ADD_URLOG_ADDED_GET_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		repositories.LatestDataRepositoryAddresses[urlog.ID] = &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              urlog.IsDeleted,
			TargetID:                               urlog.ID,
			DataUpdateTime:                         urlog.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		go func() {
			_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), repositories.LatestDataRepositoryAddresses[urlog.ID])
			if err != nil {
				err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
				slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			}
		}()
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.CommitTxSuccessMessage,
		Message:     GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_URLOG_ADDED_GET_MESSAGE"}),
	})
}

func (g *GkillServerAPI) HandleDiscardTX(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DiscardTxRequest{}
	response := &req_res.DiscardTxResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse discart tx response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidDiscardTxResponseDataError,
				ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISCARD_TRANSACTION_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse discard tx request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidDiscardTxRequestDataError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISCARD_TRANSACTION_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID, request.LocaleName)
	if err != nil {
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	txID := request.TXID
	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories. userid = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = repositories.TempReps.IDFKyouTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete idfKyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteIDFKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.KCTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete kc by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteKCError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.KmemoTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete kmemo by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteKmemoError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.LantanaTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete lantana by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteLantanaError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.MiTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete mi by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteMiError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.NlogTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete nlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteNlogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.NotificationTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete notification by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteNotificationError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.ReKyouTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete rekyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteReKyouError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.TagTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete tag by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTagError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.TextTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete text by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTextError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.TimeIsTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete timeis by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTimeIsError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
	err = repositories.TempReps.URLogTempRep.DeleteByTXID(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete urlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", err)
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteURLogError,
			ErrorMessage: GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
		})
		return
	}
}

func (g *GkillServerAPI) PrintStartedMessage() {
	ctx := context.Background()
	device, err := g.GetDevice()
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "Error getting device information", "error", err)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		slog.Log(ctx, gkill_log.Debug, "Error getting server configuration", "error", err)
		return
	}

	port := serverConfig.Address
	protocol := "http"
	if serverConfig.EnableTLS && !gkill_options.DisableTLSForce {
		protocol = "https"
	}

	os.Stdout.WriteString("gkill server started.\n")
	os.Stdout.WriteString(fmt.Sprintf("Access your record space at : %s://localhost%s\n", protocol, port))
}
