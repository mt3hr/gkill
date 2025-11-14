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
	"os/exec"
	"path/filepath"
	"runtime"
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
	"github.com/mt3hr/gkill/src/app/gkill/main/common/threads"
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

	applicationConfigs, err := gkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetAllApplicationConfigs(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get all application configs: %w", err)
		return nil, err
	}
	if len(applicationConfigs) == 0 {
		applicationConfig := &user_config.ApplicationConfig{
			UserID:                    "admin",
			Device:                    "gkill",
			UseDarkTheme:              false,
			GoogleMapAPIKey:           "",
			RykvImageListColumnNumber: 3,
			RykvHotReload:             false,
			MiDefaultBoard:            "Inbox",
			RykvDefaultPeriod:         json.Number("-1"),
			MiDefaultPeriod:           json.Number("-1"),
			IsShowShareFooter:         false,
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
	router.HandleFunc(g.APIAddress.UpdateTagStructAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateTagStruct(w, r)
	}).Methods(g.APIAddress.UpdateTagStructMethod)
	router.HandleFunc(g.APIAddress.UpdateDnoteJSONDataAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateDnoteJSONData(w, r)
	}).Methods(g.APIAddress.UpdateDnoteJSONDataMethod)

	router.HandleFunc(g.APIAddress.UpdateRepStructAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateRepStruct(w, r)
	}).Methods(g.APIAddress.UpdateRepStructMethod)
	router.HandleFunc(g.APIAddress.UpdateDeviceStructAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateDeviceStruct(w, r)
	}).Methods(g.APIAddress.UpdateDeviceStructMethod)
	router.HandleFunc(g.APIAddress.UpdateRepTypeStructAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateRepTypeStruct(w, r)
	}).Methods(g.APIAddress.UpdateRepTypeStructMethod)
	router.HandleFunc(g.APIAddress.UpdateKFTLTemplateAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleUpdateKFTLTemplate(w, r)
	}).Methods(g.APIAddress.UpdateKFTLTemplateStructMethod)
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
	router.HandleFunc(g.APIAddress.GetKFTLTemplateAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetKFTLTemplate(w, r)
	}).Methods(g.APIAddress.GetKFTLTemplateMethod)
	router.HandleFunc(g.APIAddress.GetGkillInfoAddress, func(w http.ResponseWriter, r *http.Request) {
		if ok := g.filterLocalOnly(w, r); !ok {
			return
		}
		g.HandleGetGkillInfo(w, r)
	}).Methods(g.APIAddress.GetGkillInfoMethod)
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

	gkillPage, err := fs.Sub(HTMLFS, "embed/html")
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
		gkill_log.Debug.Println(err.Error())
		return err
	}
	port := serverConfig.Address

	g.PrintStartedMessage()
	g.server = &http.Server{Addr: port, Handler: router}
	if serverConfig.EnableTLS && !gkill_options.DisableTLSForce {
		certFileName, pemFileName, err := g.getTLSFileNames(device)
		if err != nil {
			gkill_log.Debug.Fatal(err)
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.LoginRequest{}
	response := &req_res.LoginResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse login response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "ログインに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: %w", request.UserID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.LogoutRequest{}
	response := &req_res.LogoutResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse logout request to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ResetPasswordRequest{}
	response := &req_res.ResetPasswordResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse reset password to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if requesterAccount == nil {
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if targetAccount == nil {
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
			gkill_log.Debug.Println(err.Error())
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
	response.PasswordResetPathWithoutHost = *updateTargetAccount.PasswordResetToken
}

func (g *GkillServerAPI) HandleSetNewPassword(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.SetNewPasswordRequest{}
	response := &req_res.SetNewPasswordResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse set new password response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワード設定に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "パスワードリセット処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// リセットトークンがあっているか確認
	if targetAccount.PasswordResetToken == nil || request.ResetToken != *targetAccount.PasswordResetToken {
		err = fmt.Errorf("error at reset token is not match user id = %s requested token = %s: %w", request.UserID, request.ResetToken, err)
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddTagRequest{}
	response := &req_res.AddTagResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add tag response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTag != nil {
		err = fmt.Errorf("exist tag id = %s", request.Tag.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTagError,
			ErrorMessage: "タグ追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), request.Tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: "タグ追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), request.Tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTagError,
					ErrorMessage: "タグ追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.TagTempRep.AddTagInfo(r.Context(), request.Tag, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: "タグ追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Tag.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Tag.TargetID)

	repName, err := repositories.WriteTagRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Tag.IsDeleted,
		TargetID:                               request.Tag.ID,
		TargetIDInData:                         &request.Tag.TargetID,
		DataUpdateTime:                         request.Tag.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddTextRequest{}
	response := &req_res.AddTextResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add text response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existText != nil {
		err = fmt.Errorf("exist text id = %s", request.Text.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTextError,
			ErrorMessage: "テキスト追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), request.Text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: "テキスト追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TextReps) == 1 && gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), request.Text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTextError,
					ErrorMessage: "テキスト追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.TextTempRep.AddTextInfo(r.Context(), request.Text, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: "テキスト追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Text.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Text.TargetID)

	repName, err := repositories.WriteTextRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Text.IsDeleted,
		TargetID:                               request.Text.ID,
		TargetIDInData:                         &request.Text.TargetID,
		DataUpdateTime:                         request.Text.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleAddNotification(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddNotificationRequest{}
	response := &req_res.AddNotificationResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add notification response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddNotificationResponseDataError,
				ErrorMessage: "通知追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add notification request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddNotificationRequestDataError,
			ErrorMessage: "通知追加に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "通知追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existNotification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNotification != nil {
		err = fmt.Errorf("exist notification id = %s", request.Notification.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistNotificationError,
			ErrorMessage: "通知追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), request.Notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: "通知追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NotificationReps) == 1 && gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), request.Notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddNotificationError,
					ErrorMessage: "通知追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.NotificationTempRep.AddNotificationInfo(r.Context(), request.Notification, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: "通知追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Notification.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Notification.TargetID)

	repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Notification.IsDeleted,
		TargetID:                               request.Notification.ID,
		TargetIDInData:                         &request.Notification.TargetID,
		DataUpdateTime:                         request.Notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 通知情報を更新する
	notificator, err := g.GkillDAOManager.GetNotificator(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notificator: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = notificator.UpdateNotificationTargets(context.Background())
	if err != nil {
		err = fmt.Errorf("error at update notification targetrs: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedNotification = notification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNotificationSuccessMessage,
		Message:     "通知を追加しました",
	})
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Notification.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Notification.TargetID)

	repName, err = repositories.WriteNotificationRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Notification.IsDeleted,
		TargetID:                               request.Notification.ID,
		TargetIDInData:                         &request.Notification.TargetID,
		DataUpdateTime:                         request.Notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedNotification = notification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddNotificationSuccessMessage,
		Message:     "通知を追加しました",
	})
}

func (g *GkillServerAPI) HandleAddKmemo(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddKmemoRequest{}
	response := &req_res.AddKmemoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add kmemo response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKmemo != nil {
		err = fmt.Errorf("exist kmemo id = %s", request.Kmemo.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistKmemoError,
			ErrorMessage: "Kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), request.Kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: "Kmemo追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.KmemoReps) == 1 && gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), request.Kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddKmemoError,
					ErrorMessage: "Kmemo追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.KmemoTempRep.AddKmemoInfo(r.Context(), request.Kmemo, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: "Kmemo追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Kmemo.ID)

	repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Kmemo.IsDeleted,
		TargetID:                               request.Kmemo.ID,
		DataUpdateTime:                         request.Kmemo.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleAddKC(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddKCRequest{}
	response := &req_res.AddKCResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add kc response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidAddKCResponseDataError,
				ErrorMessage: "kc追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add kc request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddKCRequestDataError,
			ErrorMessage: "kc追加に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "kc追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existKC, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKC != nil {
		err = fmt.Errorf("exist kc id = %s", request.KC.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistKCError,
			ErrorMessage: "KC追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), request.KC)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: "KC追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KCReps) == 1 && gkill_options.CacheKCReps {
			err = repositories.KCReps[0].AddKCInfo(r.Context(), request.KC)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddKCError,
					ErrorMessage: "KC追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.KCTempRep.AddKCInfo(r.Context(), request.KC, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: "KC追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.KC.ID)

	repName, err := repositories.WriteKCRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.KC.IsDeleted,
		TargetID:                               request.KC.ID,
		DataUpdateTime:                         request.KC.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kc, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "kc追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.AddedKC = kc
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddKCSuccessMessage,
		Message:     "kcを追加しました",
	})
}

func (g *GkillServerAPI) HandleAddURLog(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddURLogRequest{}
	response := &req_res.AddURLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add urlog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", request.URLog.ID)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ServerConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	err = request.URLog.FillURLogField(serverConfig, applicationConfig)
	if err != nil {
		gkill_log.Debug.Println(err.Error())
	}

	if request.TXID == nil {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), request.URLog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: "URLog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.URLogReps) == 1 && gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), request.URLog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddURLogError,
					ErrorMessage: "URLog追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.URLogTempRep.AddURLogInfo(r.Context(), request.URLog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: "URLog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.URLog.ID)

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.URLog.IsDeleted,
		TargetID:                               request.URLog.ID,
		DataUpdateTime:                         request.URLog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddNlogRequest{}
	response := &req_res.AddNlogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add nlog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		err = fmt.Errorf("error at parse add nlog request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNlog != nil {
		err = fmt.Errorf("exist nlog id = %s", request.Nlog.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistNlogError,
			ErrorMessage: "Nlog追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), request.Nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: "Nlog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NlogReps) == 1 && gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), request.Nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddNlogError,
					ErrorMessage: "Nlog追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.NlogTempRep.AddNlogInfo(r.Context(), request.Nlog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: "Nlog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Nlog.ID)

	repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Nlog.IsDeleted,
		TargetID:                               request.Nlog.ID,
		DataUpdateTime:                         request.Nlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddTimeIsRequest{}
	response := &req_res.AddTimeIsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add timeis response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTimeIs != nil {
		err = fmt.Errorf("exist timeis id = %s", request.TimeIs.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistTimeIsError,
			ErrorMessage: "TimeIs追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), request.TimeIs)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: "TimeIs追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TimeIsReps) == 1 && gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), request.TimeIs)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTimeIsError,
					ErrorMessage: "TimeIs追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.TimeIsTempRep.AddTimeIsInfo(r.Context(), request.TimeIs, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: "TimeIs追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.TimeIs.ID)

	repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.TimeIs.IsDeleted,
		TargetID:                               request.TimeIs.ID,
		DataUpdateTime:                         request.TimeIs.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddLantanaRequest{}
	response := &req_res.AddLantanaResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add lantana response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if existLantana != nil {
		err = fmt.Errorf("exist lantana id = %s", request.Lantana.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistLantanaError,
			ErrorMessage: "Lantana追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), request.Lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: "Lantana追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.LantanaReps) == 1 && gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), request.Lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddLantanaError,
					ErrorMessage: "Lantana追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.LantanaTempRep.AddLantanaInfo(r.Context(), request.Lantana, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: "Lantana追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Lantana.ID)

	repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Lantana.IsDeleted,
		TargetID:                               request.Lantana.ID,
		DataUpdateTime:                         request.Lantana.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddMiRequest{}
	response := &req_res.AddMiResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add mi response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existMi != nil {
		err = fmt.Errorf("exist mi id = %s", request.Mi.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistMiError,
			ErrorMessage: "Mi追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), request.Mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: "Mi追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.MiReps) == 1 && gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), request.Mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddMiError,
					ErrorMessage: "Mi追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.MiTempRep.AddMiInfo(r.Context(), request.Mi, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: "Mi追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Mi.ID)

	repName, err := repositories.WriteMiRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Mi.IsDeleted,
		TargetID:                               request.Mi.ID,
		DataUpdateTime:                         request.Mi.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddReKyouRequest{}
	response := &req_res.AddReKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add rekyou response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existReKyou != nil {
		err = fmt.Errorf("exist rekyou id = %s", request.ReKyou.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistReKyouError,
			ErrorMessage: "ReKyou追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), request.ReKyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: "ReKyou追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), request.ReKyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddReKyouError,
					ErrorMessage: "ReKyou追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.ReKyouTempRep.AddReKyouInfo(r.Context(), request.ReKyou, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: "ReKyou追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.ReKyou.ID)

	repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.ReKyou.IsDeleted,
		TargetID:                               request.ReKyou.ID,
		TargetIDInData:                         &request.ReKyou.TargetID,
		DataUpdateTime:                         request.ReKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTagRequest{}
	response := &req_res.UpdateTagResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update tag response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), request.Tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: "タグ更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), request.Tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTagError,
					ErrorMessage: "タグ追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.TagTempRep.AddTagInfo(r.Context(), request.Tag, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, request.Tag, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: "タグ更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Tag.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Tag.TargetID)

	repName, err := repositories.WriteTagRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Tag.IsDeleted,
		TargetID:                               request.Tag.ID,
		TargetIDInData:                         &request.Tag.TargetID,
		DataUpdateTime:                         request.Tag.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, request.Tag.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTagError,
			ErrorMessage: "タグ更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTag == nil {
		err = fmt.Errorf("not exist tag id = %s", request.Tag.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTextRequest{}
	response := &req_res.UpdateTextResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update text response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), request.Text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: "テキスト更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.TextReps) == 1 && gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), request.Text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTextError,
					ErrorMessage: "テキスト更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.TextTempRep.AddTextInfo(r.Context(), request.Text, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, request.Text, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: "テキスト更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Text.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Text.TargetID)

	repName, err := repositories.WriteTextRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Text.IsDeleted,
		TargetID:                               request.Text.ID,
		TargetIDInData:                         &request.Text.TargetID,
		DataUpdateTime:                         request.Text.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, request.Text.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTextError,
			ErrorMessage: "テキスト更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existText == nil {
		err = fmt.Errorf("not exist text id = %s", request.Text.ID)
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleUpdateNotification(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateNotificationRequest{}
	response := &req_res.UpdateNotificationResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update notification response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateNotificationResponseDataError,
				ErrorMessage: "通知更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update notification request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateNotificationRequestDataError,
			ErrorMessage: "通知更新に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), request.Notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: "通知更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NotificationReps) == 1 && gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), request.Notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddNotificationError,
					ErrorMessage: "通知更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.NotificationTempRep.AddNotificationInfo(r.Context(), request.Notification, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, request.Notification, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: "通知更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Notification.ID)
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Notification.TargetID)

	repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Notification.IsDeleted,
		TargetID:                               request.Notification.ID,
		TargetIDInData:                         &request.Notification.TargetID,
		DataUpdateTime:                         request.Notification.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existNotification, err := repositories.GetNotification(r.Context(), request.Notification.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, request.Notification.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNotification == nil {
		err = fmt.Errorf("not exist notification id = %s", request.Notification.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundNotificationError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 通知情報を更新する
	notificator, err := g.GkillDAOManager.GetNotificator(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notificator: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = notificator.UpdateNotificationTargets(context.Background())
	if err != nil {
		err = fmt.Errorf("error at update notification targetrs: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificatorError,
			ErrorMessage: "通知更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedNotification = notification
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateNotificationSuccessMessage,
		Message:     "通知を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateKmemo(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateKmemoRequest{}
	response := &req_res.UpdateKmemoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update kmemo response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), request.Kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: "Kmemo更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.KmemoReps) == 1 && gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), request.Kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddKmemoError,
					ErrorMessage: "Kmemo更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.KmemoTempRep.AddKmemoInfo(r.Context(), request.Kmemo, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, request.Kmemo, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: "Kmemo更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Kmemo.ID)

	repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Kmemo.IsDeleted,
		TargetID:                               request.Kmemo.ID,
		DataUpdateTime:                         request.Kmemo.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.Kmemo.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKmemoError,
			ErrorMessage: "Kmemo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKmemo == nil {
		err = fmt.Errorf("not exist kmemo id = %s", request.Kmemo.ID)
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleUpdateKC(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateKCRequest{}
	response := &req_res.UpdateKCResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update kc response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateKCResponseDataError,
				ErrorMessage: "KC更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update kc request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateKCRequestDataError,
			ErrorMessage: "KC更新に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "KC更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// すでに存在する場合はエラー
	_, err = repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), request.KC)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: "KC更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.KCReps) == 1 && gkill_options.CacheKCReps {
			err = repositories.WriteKCRep.AddKCInfo(r.Context(), request.KC)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddKCError,
					ErrorMessage: "KC更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.KCTempRep.AddKCInfo(r.Context(), request.KC, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, request.KC, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: "KC更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.KC.ID)

	repName, err := repositories.WriteKCRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.KC.IsDeleted,
		TargetID:                               request.KC.ID,
		DataUpdateTime:                         request.KC.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kc, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existKC, err := repositories.KCReps.GetKC(r.Context(), request.KC.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.KC.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existKC == nil {
		err = fmt.Errorf("not exist kc id = %s", request.KC.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundKCError,
			ErrorMessage: "KC更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.UpdatedKC = kc
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateKCSuccessMessage,
		Message:     "KCを更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateURLog(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateURLogRequest{}
	response := &req_res.UpdateURLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update urlog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.ReGetURLogContent {
		var currentServerConfig *server_config.ServerConfig
		serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: "Kyouタスク通知登録に失敗しました",
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: "ServerConfig取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil || applicationConfig == nil {
			err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
			err = fmt.Errorf("try create application config user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())

			newApplicationConfig := &user_config.ApplicationConfig{
				UserID:                    userID,
				Device:                    device,
				UseDarkTheme:              false,
				GoogleMapAPIKey:           "",
				RykvImageListColumnNumber: 3,
				RykvHotReload:             false,
				MiDefaultBoard:            "Inbox",
				RykvDefaultPeriod:         json.Number("-1"),
				MiDefaultPeriod:           json.Number("-1"),
				IsShowShareFooter:         false,
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

		err = request.URLog.FillURLogField(currentServerConfig, applicationConfig)
		if err != nil {
			gkill_log.Debug.Println(err.Error())
		}
	}

	if request.TXID == nil {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), request.URLog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: "URLog更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.URLogReps) == 1 && gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), request.URLog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddURLogError,
					ErrorMessage: "URLog更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.URLogTempRep.AddURLogInfo(r.Context(), request.URLog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, request.URLog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: "URLog更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.URLog.ID)

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.URLog.IsDeleted,
		TargetID:                               request.URLog.ID,
		DataUpdateTime:                         request.URLog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.URLog.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog == nil {
		err = fmt.Errorf("not exist urlog id = %s", request.URLog.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateNlogRequest{}
	response := &req_res.UpdateNlogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update nlog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), request.Nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: "Nlog更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.NlogReps) == 1 && gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), request.Nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddNlogError,
					ErrorMessage: "Nlog更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.NlogTempRep.AddNlogInfo(r.Context(), request.Nlog, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, request.Nlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: "Nlog更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Nlog.ID)

	repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Nlog.IsDeleted,
		TargetID:                               request.Nlog.ID,
		DataUpdateTime:                         request.Nlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.Nlog.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNlogError,
			ErrorMessage: "Nlog更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existNlog == nil {
		err = fmt.Errorf("not exist nlog id = %s", request.Nlog.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTimeisRequest{}
	response := &req_res.UpdateTimeisResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update timeis response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), request.TimeIs)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: "TimeIs更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TimeIsReps) == 1 && gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), request.TimeIs)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTimeIsError,
					ErrorMessage: "TimeIs更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.TimeIsTempRep.AddTimeIsInfo(r.Context(), request.TimeIs, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, request.TimeIs, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: "TimeIs更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.TimeIs.ID)

	repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.TimeIs.IsDeleted,
		TargetID:                               request.TimeIs.ID,
		DataUpdateTime:                         request.TimeIs.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.TimeIs.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetTimeIsError,
			ErrorMessage: "TimeIs更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existTimeIs == nil {
		err = fmt.Errorf("not exist timeis id = %s", request.TimeIs.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateLantanaRequest{}
	response := &req_res.UpdateLantanaResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update lantana response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), request.Lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: "Lantana更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.LantanaReps) == 1 && gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), request.Lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddLantanaError,
					ErrorMessage: "Lantana更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.LantanaTempRep.AddLantanaInfo(r.Context(), request.Lantana, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, request.Lantana, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: "Lantana更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Lantana.ID)

	repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Lantana.IsDeleted,
		TargetID:                               request.Lantana.ID,
		DataUpdateTime:                         request.Lantana.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.Lantana.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLantanaError,
			ErrorMessage: "Lantana更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existLantana == nil {
		err = fmt.Errorf("not exist lantana id = %s", request.Lantana.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateIDFKyouRequest{}
	response := &req_res.UpdateIDFKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update idfKyou response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), request.IDFKyou)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: "IDFKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.IDFKyouReps) == 1 && gkill_options.CacheIDFKyouReps {
			err = repositories.IDFKyouReps[0].AddIDFKyouInfo(r.Context(), request.IDFKyou)
			if err != nil {
				err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddIDFKyouError,
					ErrorMessage: "IDFKyou更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.IDFKyouTempRep.AddIDFKyouInfo(r.Context(), request.IDFKyou, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, request.IDFKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: "IDFKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.IDFKyou.ID)

	repName, err := repositories.WriteIDFKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.IDFKyou.IsDeleted,
		TargetID:                               request.IDFKyou.ID,
		DataUpdateTime:                         request.IDFKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.IDFKyou.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetIDFKyouError,
			ErrorMessage: "IDFKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existIDFKyou == nil {
		err = fmt.Errorf("not exist idfKyou id = %s", request.IDFKyou.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateMiRequest{}
	response := &req_res.UpdateMiResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update mi response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない場合はエラー
	existMi, err := repositories.MiReps.GetMi(r.Context(), request.Mi.ID, nil)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existMi == nil {
		err = fmt.Errorf("not exist mi id = %s", request.Mi.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundMiError,
			ErrorMessage: "Mi更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), request.Mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: "Mi更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.MiReps) == 1 && gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), request.Mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddMiError,
					ErrorMessage: "Mi更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.MiTempRep.AddMiInfo(r.Context(), request.Mi, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, request.Mi, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: "Mi更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.Mi.ID)

	repName, err := repositories.WriteMiRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.Mi.IsDeleted,
		TargetID:                               request.Mi.ID,
		DataUpdateTime:                         request.Mi.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.Mi.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiError,
			ErrorMessage: "Mi追加後取得に失敗しました",
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateReKyouRequest{}
	response := &req_res.UpdateReKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update rekyou response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if request.TXID == nil {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), request.ReKyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: "ReKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), request.ReKyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddReKyouError,
					ErrorMessage: "ReKyou更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
	} else {
		err = g.GkillDAOManager.TempReps.ReKyouTempRep.AddReKyouInfo(r.Context(), request.ReKyou, *request.TXID, userID, device)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, request.ReKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: "ReKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, request.ReKyou.ID)

	repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              request.ReKyou.IsDeleted,
		TargetID:                               request.ReKyou.ID,
		TargetIDInData:                         &request.ReKyou.TargetID,
		DataUpdateTime:                         request.ReKyou.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ReKyou.ID, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetReKyouError,
			ErrorMessage: "ReKyou更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existReKyou == nil {
		err = fmt.Errorf("not exist rekyou id = %s", request.ReKyou.ID)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyousRequest{}
	response := &req_res.GetKyousResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKyouRequest{}
	response := &req_res.GetKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyou response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		var kyou *reps.Kyou
		kyou, err = repositories.GetKyou(r.Context(), request.ID, request.UpdateTime)
		kyouHistories = []*reps.Kyou{kyou}
	} else {
		kyouHistories, err = repositories.Reps.GetKyouHistoriesByRepName(r.Context(), request.ID, request.RepName)
	}

	if err != nil {
		err = fmt.Errorf("error at get kyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleGetKmemo(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKmemoRequest{}
	response := &req_res.GetKmemoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kmemo response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Kmemo取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kmemoHistories, err := repositories.KmemoReps.GetKmemoHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleGetKC(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKCRequest{}
	response := &req_res.GetKCResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kc response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetKCResponseDataError,
				ErrorMessage: "KC取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get kc request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetKCRequestDataError,
			ErrorMessage: "KC取得に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "KC取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	kcHistories, err := repositories.KCReps.GetKCHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetKCError,
			ErrorMessage: "KC取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.KCHistories = kcHistories
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetKCSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetURLog(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetURLogRequest{}
	response := &req_res.GetURLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get urlog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "URLog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlogHistories, err := repositories.URLogReps.GetURLogHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNlogRequest{}
	response := &req_res.GetNlogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get nlog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Nlog取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlogHistories, err := repositories.NlogReps.GetNlogHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTimeisRequest{}
	response := &req_res.GetTimeisResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get timeis response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "TimeIs取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeisHistories, err := repositories.TimeIsReps.GetTimeIsHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetMiRequest{}
	response := &req_res.GetMiResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Mi取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	miHistories, err := repositories.MiReps.GetMiHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetLantanaRequest{}
	response := &req_res.GetLantanaResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get lantana response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Lantana取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantanaHistories, err := repositories.LantanaReps.GetLantanaHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetReKyouRequest{}
	response := &req_res.GetReKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get rekyou response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "rekyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyouHistories, err := repositories.ReKyouReps.GetReKyouHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGitCommitLogRequest{}
	response := &req_res.GetGitCommitLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gitCommitLog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetIDFKyouRequest{}
	response := &req_res.GetIDFKyouResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get idfKyou response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "IDFKyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyouHistories, err := repositories.IDFKyouReps.GetIDFKyouHistoriesByRepName(r.Context(), request.ID, request.RepName)
	if err != nil {
		err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetMiBoardRequest{}
	response := &req_res.GetMiBoardResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi board names response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleGetAllTagNames(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetAllTagNamesRequest{}
	response := &req_res.GetAllTagNamesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "タグ名全件取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// allTagNames, err := repositories.GetAllTagNames(r.Context())
	allTagNames, err := repositories.GetAllTagNames(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all tag names user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetAllRepNamesRequest{}
	response := &req_res.GetAllRepNamesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kyous response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTagsByTargetIDRequest{}
	response := &req_res.GetTagsByTargetIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get tags by target id response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTagHistoryByTagIDRequest{}
	response := &req_res.GetTagHistoryByTagIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get tag histories by tag id response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		var tag *reps.Tag
		tag, err = repositories.GetTag(r.Context(), request.ID, request.UpdateTime)
		tags = []*reps.Tag{tag}
	} else {
		tags, err = repositories.TagReps.GetTagHistoriesByRepName(r.Context(), request.ID, request.RepName)
	}

	if err != nil {
		err = fmt.Errorf("error at get tag histories by tag id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTextsByTargetIDRequest{}
	response := &req_res.GetTextsByTargetIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get texts by target id response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleGetNotificationsByTargetID(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNotificationsByTargetIDRequest{}
	response := &req_res.GetNotificationsByTargetIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get notifications by target id response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNotificationsByTargetIDResponseDataError,
				ErrorMessage: "通知取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get notifications by target id request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNotificationsByTargetIDRequestDataError,
			ErrorMessage: "通知取得に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "通知取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notifications, err := repositories.GetNotificationsByTargetID(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get notifications by target id user id = %s device = %s target id = %s: %w", userID, device, request.TargetID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationsByTargetIDError,
			ErrorMessage: "通知取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Notifications = notifications
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNotificationsByTargetIDSuccessMessage,
		Message:     "通知取得完了",
	})
}

func (g *GkillServerAPI) HandleGetTextHistoriesByTextID(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetTextHistoryByTextIDRequest{}
	response := &req_res.GetTextHistoryByTextIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get text histories by text id response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		var text *reps.Text
		text, err = repositories.GetText(r.Context(), request.ID, request.UpdateTime)
		texts = []*reps.Text{text}
	} else {
		texts, err = repositories.TextReps.GetTextHistoriesByRepName(r.Context(), request.ID, request.RepName)
	}

	if err != nil {
		err = fmt.Errorf("error at get text histories by text id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleGetNotificationHistoriesByNotificationID(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetNotificationHistoryByNotificationIDRequest{}
	response := &req_res.GetNotificationHistoryByNotificationIDResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get notification histories by notification id response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetNotificationHistoriesByNotificationIDResponseDataError,
				ErrorMessage: "通知取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get notification histories by notification id request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetNotificationHistoriesByNotificationIDRequestDataError,
			ErrorMessage: "通知取得に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "通知取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// UpdateTimeが指定されていれば一致するものを、そうでなければIDが一致する履歴全部を取得する
	notifications := []*reps.Notification{}
	if request.UpdateTime != nil {
		var notification *reps.Notification
		notification, err = repositories.GetNotification(r.Context(), request.ID, request.UpdateTime)
		notifications = []*reps.Notification{notification}
	} else {
		notifications, err = repositories.GetNotificationHistories(r.Context(), request.ID)
	}

	if err != nil {
		err = fmt.Errorf("error at get notification histories by notification id user id = %s device = %s target id = %s: %w", userID, device, request.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetNotificationHistoriesByNotificationIDError,
			ErrorMessage: "通知取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.NotificationHistories = notifications
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetNotificationHistoriesByNotificationIDSuccessMessage,
		Message:     "通知取得完了",
	})
}

func (g *GkillServerAPI) HandleGetApplicationConfig(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetApplicationConfigRequest{}
	response := &req_res.GetApplicationConfigResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get applicationConfig response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())

		newApplicationConfig := &user_config.ApplicationConfig{
			UserID:                    userID,
			Device:                    device,
			UseDarkTheme:              false,
			GoogleMapAPIKey:           "",
			RykvImageListColumnNumber: 3,
			RykvHotReload:             false,
			MiDefaultBoard:            "Inbox",
			RykvDefaultPeriod:         json.Number("-1"),
			MiDefaultPeriod:           json.Number("-1"),
			IsShowShareFooter:         false,
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepTypeStructError,
			ErrorMessage: "RepTypeStruct取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	dnoteJSONData, err := g.GkillDAOManager.ConfigDAOs.DnoteDataDAO.GetDnoteData(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get dnote json data user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDnoteJSONDataError,
			ErrorMessage: "集計条件情報取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "ApplicationConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	var dnoteJSONDataValue json.RawMessage
	if len(dnoteJSONData) != 0 {
		dnoteJSONDataValue = dnoteJSONData[0].DnoteJSONData
	}

	applicationConfig.KFTLTemplate = kftlTemplates
	applicationConfig.TagStruct = tagStructs
	applicationConfig.RepStruct = repStructs
	applicationConfig.DeviceStruct = deviceStructs
	applicationConfig.RepTypeStruct = repTypeStructs
	applicationConfig.DnoteJSONData = dnoteJSONDataValue
	applicationConfig.AccountIsAdmin = account.IsAdmin
	applicationConfig.SessionIsLocal = session.IsLocalAppUser
	response.ApplicationConfig = applicationConfig
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetApplicationConfigSuccessMessage,
		Message:     "設定データ取得完了",
	})
}

func (g *GkillServerAPI) HandleGetServerConfigs(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetServerConfigsRequest{}
	response := &req_res.GetServerConfigsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get serverConfig response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
		Message:     "サーバ設定データ取得完了",
	})
}

func (g *GkillServerAPI) HandleUploadFiles(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
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
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUploadFilesRequestDataError,
			ErrorMessage: "ファイルアップロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	g.GkillDAOManager.SetSkipIDF(true)
	defer g.GkillDAOManager.SetSkipIDF(false)

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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "ファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg.Add(1)
		done := threads.AllocateThread()
		go func(filename string, base64Data string) {
			defer done()
			// ファイル書き込み
			defer wg.Done()
			var gkillError *message.GkillError
			parts := strings.SplitN(base64Data, ",", 2)
			encoded := parts[len(parts)-1]
			base64Reader := bufio.NewReader(strings.NewReader(encoded))
			decoder := base64.NewDecoder(base64.StdEncoding, base64Reader)

			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				err := fmt.Errorf("error at open file filename= %s: %w", filename, err)
				gkill_log.Debug.Println(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.GetRepPathError,
					ErrorMessage: "ファイルアップロードに失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError = &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: "ファイルアップロードに失敗しました",
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
					gkill_log.Debug.Println(err.Error())
					gkillError = &message.GkillError{
						ErrorCode:    message.GetRepPathError,
						ErrorMessage: "ファイルアップロードに失敗しました",
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}

				// defer g.WebPushUpdatedData(r.Context(), userID, device, idfKyou.ID)
				_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
					IsDeleted:                              idfKyou.IsDeleted,
					TargetID:                               idfKyou.ID,
					DataUpdateTime:                         idfKyou.UpdateTime,
					LatestDataRepositoryName:               repName,
					LatestDataRepositoryAddressUpdatedTime: time.Now(),
				})
				if err != nil {
					err = fmt.Errorf("error at update or add latest data repository address: %w", err)
					gkill_log.Debug.Println(err.Error())
					gkillError = &message.GkillError{
						ErrorCode:    message.UpdateRepositoryAddressError,
						ErrorMessage: "ファイルアップロードに失敗しました",
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
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
			gkill_log.Debug.Println(err.Error())
			gkillError = &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "ファイルアップロード後Kyou取得に失敗しました",
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
		Message:     "ファイルアップロードが完了しました",
	})
}

func (g *GkillServerAPI) HandleUploadGPSLogFiles(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UploadGPSLogFilesRequest{}
	response := &req_res.UploadGPSLogFilesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse upload files response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotFoundTargetGPSLogRepError,
			ErrorMessage: "GPSLogファイルアップロードに失敗しました",
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "GPSLogファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg.Add(1)
		done := threads.AllocateThread()
		go func(filename string, base64Data string) {
			defer done()
			// テンポラリファイル書き込み
			defer wg.Done()
			base64Reader := bufio.NewReader(strings.NewReader(strings.SplitN(base64Data, ",", 2)[1]))
			decoder := base64.NewDecoder(base64.RawStdEncoding, base64Reader)
			base64DataBytes, err := io.ReadAll(decoder)
			if err != nil {
				err := fmt.Errorf("error at load gps log file content filename = %s: %w", filename, err)
				gkill_log.Debug.Println(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh <- gkillError
				return
			}

			var gkillError *message.GkillError
			// gpsLogsを作る
			gpsLogs, err := gpslogs.GPSLogFileAsGPSLogs(repDir, filename, request.ConflictBehavior, string(base64DataBytes))
			if err != nil {
				err := fmt.Errorf("error at gps log file as gpx file filename = %s: %w", filename, err)
				gkill_log.Debug.Println(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.ConvertGPSLogError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetRepPathError,
				ErrorMessage: "GPSLogファイルアップロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		wg2.Add(1)
		done := threads.AllocateThread()
		go func(filename string, gpsLogs []*reps.GPSLog) {
			defer done()
			defer wg2.Done()
			// Mergeだったら既存のデータも混ぜる
			if request.ConflictBehavior == req_res.Merge {
				startTime, err := time.Parse(dateFormat, datestr)
				if err != nil {
					err = fmt.Errorf("error at parse date string %s: %w", datestr, err)
					gkill_log.Debug.Println(err.Error())
					gkillError = &message.GkillError{
						ErrorCode:    message.ConvertGPSLogError,
						ErrorMessage: "GPSLogファイルアップロードに失敗しました",
					}
					response.Errors = append(response.Errors, gkillError)
				}
				endTime := startTime.Add(time.Hour * 24).Add(-time.Millisecond)
				existGPSLogs, err := targetRep.GetGPSLogs(r.Context(), &startTime, &endTime)
				if err != nil {
					err = fmt.Errorf("error at exist gpx datas %s: %w", datestr, err)
					gkill_log.Debug.Println(err.Error())
					gkillErrorCh2 <- gkillError
					return
				}
				gpsLogs = append(gpsLogs, existGPSLogs...)
			}

			gpxFileContent, err := g.generateGPXFileContent(gpsLogs)
			if err != nil {
				err := fmt.Errorf("error at generate gpx file content filename = %s: %w", filename, err)
				gkill_log.Debug.Println(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.GenerateGPXFileContentError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
				}
				gkillErrorCh2 <- gkillError
				return
			}
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				err := fmt.Errorf("error at open file filename= %s: %w", filename, err)
				gkill_log.Debug.Println(err.Error())
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
				gkill_log.Debug.Println(err.Error())
				gkillError = &message.GkillError{
					ErrorCode:    message.WriteGPXFileError,
					ErrorMessage: "GPSLogファイルアップロードに失敗しました",
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

func (g *GkillServerAPI) HandleUpdateTagStruct(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateTagStructRequest{}
	response := &req_res.UpdateTagStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update tagStruct response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at delete users tag structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Print(err.Error())
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
			err = fmt.Errorf("error at add tag structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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

func (g *GkillServerAPI) HandleUpdateDnoteJSONData(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateDnoteJSONDataRequest{}
	response := &req_res.UpdateDnoteJSONDataResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update dnote json data response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateDnoteJSONDataResponseDataError,
				ErrorMessage: "集計ビュー条件情報更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse update dnote json data request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateDnoteJSONDataRequestDataError,
			ErrorMessage: "集計ビュー条件情報更新に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 一回全部消して全部いれる
	ok, err := g.GkillDAOManager.ConfigDAOs.DnoteDataDAO.DeleteUsersDnoteData(r.Context(), userID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete users dnote json data user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Print(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteUsersDnoteDataError,
			ErrorMessage: "集計ビュー条件情報更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	ok, err = g.GkillDAOManager.ConfigDAOs.DnoteDataDAO.AddDnoteData(r.Context(), &user_config.DnoteData{
		ID:            uuid.NewString(),
		UserID:        userID,
		Device:        device,
		DnoteJSONData: request.DnoteJSONData,
	})
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at add dnote json data user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUsersDnoteDataError,
			ErrorMessage: "集計ビュー条件情報更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ApplicationConfig, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "集計ビュー条件情報更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTagStructSuccessMessage,
		Message:     "集計ビュー条件情報を更新しました",
	})
}

func (g *GkillServerAPI) HandleUpdateApplicationConfig(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateApplicationConfigRequest{}
	response := &req_res.UpdateApplicationConfigResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update application config response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at update application config user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Print(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateRepStructRequest{}
	response := &req_res.UpdateRepStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update repStruct response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at delete users rep structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at add rep structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateDeviceStructRequest{}
	response := &req_res.UpdateDeviceStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update deviceStruct response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at delete users device structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at add device structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateRepTypeStructRequest{}
	response := &req_res.UpdateRepTypeStructResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update repTypeStruct response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at delete users repType structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at add repType structs user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateKFTLTemplateRequest{}
	response := &req_res.UpdateKFTLTemplateResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update kftl template response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at delete users kftl tempates user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at add kftl templates user user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateAccountStatusRequest{}
	response := &req_res.UpdateAccountStatusResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update accountStatus response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			err = fmt.Errorf("error at update users account user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateUserRepsRequest{}
	response := &req_res.UpdateUserRepsResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse update userReps response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "Rep更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if targetAccount == nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "Rep更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := targetAccount.UserID

	ok, err := g.GkillDAOManager.ConfigDAOs.RepositoryDAO.DeleteWriteRepositories(r.Context(), userID, request.UpdatedReps)
	if !ok || err != nil {
		gkillError := &message.GkillError{
			ErrorCode:    message.AddUpdatedRepositoriesByUser,
			ErrorMessage: fmt.Sprintf("Rep更新に失敗しました。%s", err.Error()),
		}
		response.Errors = append(response.Errors, gkillError)
		if err != nil {
			err = fmt.Errorf("error at delete add all repositories by users user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
		}

		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateRepositoriesSuccessMessage,
		Message:     "Rep更新に成功しました",
	})
}

func (g *GkillServerAPI) HandleUpdateServerConfigs(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	defer func() {
		done := threads.AllocateThread()
		defer done()
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
				gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountNotHasAdminError,
				ErrorMessage: "管理者権限を所有していません",
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
					gkill_log.Debug.Println(err.Error())
					gkillError := &message.GkillError{
						ErrorCode:    message.NotFoundTLSCertFileError,
						ErrorMessage: "証明書ファイルが作成されていません",
					}
					response.Errors = append(response.Errors, gkillError)
					return
				}
				_, err = os.Stat(os.ExpandEnv(serverConfig.TLSKeyFile))
				if err != nil {
					err = fmt.Errorf("not found tls key file user id = %s device = %s: %w", userID, device, err)
					gkill_log.Debug.Println(err.Error())
					gkillError := &message.GkillError{
						ErrorCode:    message.NotFoundTLSCertFileError,
						ErrorMessage: "証明書ファイルが作成されていません",
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

					gkill_log.Debug.Println(err.Error())
					gkillError := &message.GkillError{
						ErrorCode:    message.GenerateVAPIDKeysError,
						ErrorMessage: "鍵生成エラー",
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
				gkill_log.Debug.Println(err.Error())
			}
			gkillError := &message.GkillError{
				ErrorCode:    message.UpdateServerConfigError,
				ErrorMessage: "設定更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		err = g.GkillDAOManager.Close()
		if err != nil {
			if err != nil {
				err = fmt.Errorf("error at close gkill dao manager: %w", err)
				gkill_log.Debug.Println(err.Error())
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
		response.Messages = append(response.Messages, &message.GkillMessage{
			MessageCode: message.RebootingMessage,
			Message:     "少し待ってリロードしてください",
		})
	}()
}

func (g *GkillServerAPI) HandleAddAccount(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddAccountRequest{}
	response := &req_res.AddAccountResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add account response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: "Account追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existAccount != nil {
		err = fmt.Errorf("exist account id = %s", userID)
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAccountError,
			ErrorMessage: "account追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if requesterAccount == nil {
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
		UseDarkTheme:              false,
		GoogleMapAPIKey:           "",
		RykvImageListColumnNumber: 3,
		RykvHotReload:             false,
		MiDefaultBoard:            "Inbox",
		RykvDefaultPeriod:         json.Number("-1"),
		MiDefaultPeriod:           json.Number("-1"),
		IsShowShareFooter:         false,
	}
	_, err = g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.AddApplicationConfig(context.TODO(), applicationConfig)
	if err != nil {
		err = fmt.Errorf("error at add application config user id = %s id = %s: %w", userID, request.AccountInfo.UserID, err)
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GenerateTLSFileRequest{}
	response := &req_res.GenerateTLSFileResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse generate tls to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "TLSファイル作成処理に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if requesterAccount == nil {
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
			gkill_log.Debug.Println(err.Error())
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
	isCABool := true
	host := &hostStr
	ecdsaCurve := &ecdsaCurveStr
	ed25519Key := &ed25519KeyBool
	rsaBits := &rsaBitsInt
	validFrom := &validFromStr
	validFor := &validForDuration
	isCA := &isCABool
	if len(*host) == 0 {
		gkill_log.Trace.Printf("finish Missing required --host parameter")
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Trace.Printf("finish Unrecognized elliptic curve: %q", *ecdsaCurve)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err != nil {
		gkill_log.Trace.Printf("finish Failed to generate private key: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
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
			gkill_log.Trace.Printf("finish Failed to parse creation date: %v", err)
			err = fmt.Errorf("error at generate tls files")
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Trace.Printf("finish Failed to generate serial number: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Trace.Printf("finish Failed to create certificate: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Trace.Printf("finish Failed to open cert.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	err = os.MkdirAll(parentDirKey, os.ModePerm)
	if err != nil {
		gkill_log.Trace.Printf("finish Failed to open cert.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	certOut, err := os.Create(certFileName)
	if err != nil {
		gkill_log.Trace.Printf("finish Failed to open cert.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		gkill_log.Trace.Printf("finish Failed to write data to cert.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := certOut.Close(); err != nil {
		gkill_log.Trace.Printf("finish Error closing cert.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	keyOut, err := os.OpenFile(pemFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		gkill_log.Trace.Printf("finish Failed to open key.pem for writing: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		gkill_log.Trace.Printf("finish Unable to marshal private key: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		gkill_log.Trace.Printf("finish Failed to write data to key.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if err := keyOut.Close(); err != nil {
		gkill_log.Trace.Printf("finish Error closing key.pem: %v", err)
		err = fmt.Errorf("error at generate tls files")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GenerateTLSFilesError,
			ErrorMessage: "TLSファイル作成処理に失敗しました。",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	message := &message.GkillMessage{
		MessageCode: message.TLSFileCreateSuccessMessage,
		Message:     "TLSファイル作成完了",
	}
	response.Messages = append(response.Messages, message)
}

func (g *GkillServerAPI) HandleGetGPSLog(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGPSLogRequest{}
	response := &req_res.GetGPSLogResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gpsLog response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		Message:     "GPSLog取得完了",
	})
}

func (g *GkillServerAPI) HandleGetKFTLTemplate(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetKFTLTemplatesRequest{}
	response := &req_res.GetKFTLTemplateResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get kftl template response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		Message:     "KFTLテンプレートデータ取得完了",
	})
}

func (g *GkillServerAPI) HandleGetGkillInfo(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGkillInfoRequest{}
	response := &req_res.GetGkillInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get gkillInfo response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
	response.CacheClearCountLimit = gkill_options.CacheClearCountLimit
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetGkillInfoSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleAddShareKyouListInfo(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.AddShareKyouListInfoRequest{}
	response := &req_res.AddShareKyouListInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add ShareKyouListInfo response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidAddShareKyouListInfoResponseDataError,
				ErrorMessage: "ShareKyouListInfo追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add ShareKyouListInfo request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidAddShareKyouListInfoRequestDataError,
			ErrorMessage: "ShareKyouListInfo追加に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在する場合はエラー
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo != nil {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo追加に失敗しました",
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
			gkill_log.Debug.Println(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.AddShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo追加後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfo = ShareKyouListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.AddShareKyouListInfoSuccessMessage,
		Message:     "ShareKyouListInfoを追加しました",
	})
}

func (g *GkillServerAPI) HandleUpdateShareKyouListInfo(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.UpdateShareKyouListInfoRequest{}
	response := &req_res.UpdateShareKyouListInfoResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse add ShareKyouListInfo response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidUpdateShareKyouListInfoResponseDataError,
				ErrorMessage: "ShareKyouListInfo更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse add ShareKyouListInfo request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidUpdateShareKyouListInfoRequestDataError,
			ErrorMessage: "ShareKyouListInfo更新に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 対象が存在しない
	existShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	if existShareKyouListInfo == nil {
		err = fmt.Errorf("not exist ShareKyouListInfo id = %s", request.ShareKyouListInfo.ShareID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.NotExistShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo更新に失敗しました",
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
			gkill_log.Debug.Println(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.UpdateShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo更新に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouListInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfo user id = %s device = %s id = %s: %w", userID, device, request.ShareKyouListInfo.ShareID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfoError,
			ErrorMessage: "ShareKyouListInfo更新後取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfo = ShareKyouListInfo
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateShareKyouListInfoSuccessMessage,
		Message:     "ShareKyouListInfoを更新しました",
	})
}

func (g *GkillServerAPI) HandleGetShareKyouListInfos(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetShareKyouListInfosRequest{}
	response := &req_res.GetShareKyouListInfosResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get ShareKyouListInfos response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetShareKyouListInfosResponseDataError,
				ErrorMessage: "ShareKyouListInfos取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get ShareKyouListInfos request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetShareKyouListInfosRequestDataError,
			ErrorMessage: "ShareKyouListInfos取得に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ShareKyouList, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfos(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get ShareKyouListInfos user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetShareKyouListInfosError,
			ErrorMessage: "ShareKyouListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.ShareKyouListInfos = ShareKyouList
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetShareKyouListInfosSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleDeleteShareKyouListInfos(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DeleteShareKyouListInfoRequest{}
	response := &req_res.DeleteShareKyouListInfosResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete ShareKyouListInfos response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidDeleteShareKyouListInfosResponseDataError,
				ErrorMessage: "ShareKyouListInfos削除に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete ShareKyouListInfos request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidDeleteShareKyouListInfosRequestDataError,
			ErrorMessage: "ShareKyouListInfos削除に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	ok, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.DeleteKyouShareInfo(r.Context(), request.ShareKyouListInfo.ShareID)
	if !ok || err != nil {
		if err != nil {
			err = fmt.Errorf("error at delete ShareKyouListInfos user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
		}
		gkillError := &message.GkillError{
			ErrorCode:    message.DeleteShareKyouListInfosError,
			ErrorMessage: "ShareKyouListInfos削除に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.DeleteShareKyouListInfosSuccessMessage,
		Message:     "削除完了",
	})
}

func (g *GkillServerAPI) HandleGetSharedKyous(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetSharedKyousRequest{}
	response := &req_res.GetSharedKyousResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse delete ShareKyouListInfos response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiSharedTasksResponseDataError,
				ErrorMessage: "ShareKyouListInfos取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse delete ShareKyouListInfos request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTasksRequestDataError,
			ErrorMessage: "ShareKyouListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), request.SharedID)
	if err != nil || sharedKyouInfo == nil {
		err = fmt.Errorf("error at get ShareKyouListInfos shared id = %s: %w", request.SharedID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetMiSharedTasksError,
			ErrorMessage: "ShareKyouListInfos取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := sharedKyouInfo.UserID
	device := sharedKyouInfo.Device

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	findQuery := &find.FindQuery{}
	err = json.Unmarshal([]byte(sharedKyouInfo.FindQueryJSON), findQuery)
	if err != nil {
		err = fmt.Errorf("error at parse query json at find kyous %#v: %w", findQuery, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiSharedTaskRequest,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// Kyou
	findFilter := &FindFilter{}
	kyous, _, err := findFilter.FindKyous(r.Context(), userID, device, g.GkillDAOManager, findQuery)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareKyouError,
			ErrorMessage: "Kyou取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	trueValue := true
	findQueryValueForKyouInstances := *findQuery
	findQueryForKyouInstances := &findQueryValueForKyouInstances
	findQueryForKyouInstances.UseIDs = &trueValue
	findQueryForKyouInstances.IDs = &[]string{}
	for _, kyou := range kyous {
		*findQueryForKyouInstances.IDs = append(*findQueryForKyouInstances.IDs, kyou.ID)
	}

	// Mi
	mis, err := repositories.MiReps.FindMi(r.Context(), findQueryForKyouInstances)
	if err != nil {
		err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.FindKyousShareKyouError,
			ErrorMessage: "Kyou取得に失敗しました",
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
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
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTagsShareKyouError,
					ErrorMessage: "タグ取得に失敗しました",
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
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTextsShareKyouError,
					ErrorMessage: "テキスト取得に失敗しました",
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
	timeiss := []*reps.TimeIs{}
	if sharedKyouInfo.IsShareWithTimeIss {
		trueValue := true
		for _, kyou := range kyous {
			findQueryForPlaingTimeIs := &find.FindQuery{}
			findQueryForPlaingTimeIs.UsePlaing = &trueValue
			findQueryForPlaingTimeIs.PlaingTime = &kyou.RelatedTime
			plaingTimeIss, err := repositories.TimeIsReps.FindTimeIs(r.Context(), findQueryForPlaingTimeIs)
			if err != nil {
				err = fmt.Errorf("error at find plaing timeis user id = %s device = %s: %w", userID, device, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.FindTextsShareKyouError,
					ErrorMessage: "TimeIs取得に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
			timeiss = append(timeiss, plaingTimeIss...)
		}
	}

	kmemos := []*reps.Kmemo{}
	kcs := []*reps.KC{}
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// KC
		kcs, err = repositories.KCReps.FindKC(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// Nlog
		nlogs, err = repositories.NlogReps.FindNlog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// Lantana
		lantanas, err = repositories.LantanaReps.FindLantana(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// URLogs
		urlogs, err = repositories.URLogReps.FindURLog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// IDFKyou
		idfKyous, err = repositories.IDFKyouReps.FindIDFKyou(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// ReKyou
		rekyous, err = repositories.ReKyouReps.FindReKyou(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// GitCommitLog
		gitCommitLogs, err = repositories.GitCommitLogReps.FindGitCommitLog(r.Context(), findQueryForKyouInstances)
		if err != nil {
			err = fmt.Errorf("error at find Kyous user id = %s device = %s: %w", userID, device, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.FindKyousShareKyouError,
				ErrorMessage: "Kyou取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.Kyous = kyous
	response.Mis = mis
	response.Kmemos = kmemos
	response.KCs = kcs
	response.Nlogs = nlogs
	response.Lantanas = lantanas
	response.URLogs = urlogs
	response.IDFKyous = idfKyous
	response.ReKyous = rekyous
	response.GitCommitLogs = gitCommitLogs
	response.GPSLogs = gpsLogs
	response.AttachedTags = tags
	response.AttachedTexts = texts
	response.AttachedTimeIss = timeiss
	response.Title = sharedKyouInfo.ShareTitle
	response.ViewType = sharedKyouInfo.ViewType
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.GetMiSharedTasksSuccessMessage,
		Message:     "取得完了",
	})
}

func (g *GkillServerAPI) HandleGetRepositories(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetRepositoriesRequest{}
	response := &req_res.GetRepositoriesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get repositories response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
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
		Message:     "記録保存先データ取得完了",
	})
}

func (g *GkillServerAPI) getAccountFromSessionID(ctx context.Context, sessionID string) (*account.Account, *message.GkillError, error) {
	loginSession, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(ctx, sessionID)
	if loginSession == nil || err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", sessionID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountSessionNotFoundError,
			ErrorMessage: "アカウント認証に失敗しました",
		}
		return nil, gkillError, err
	}

	account, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAccount(ctx, loginSession.UserID)
	if err != nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "アカウント認証に失敗しました",
		}
		return nil, gkillError, err
	}

	if account == nil {
		err = fmt.Errorf("error at get account user id = %s: %w", loginSession.UserID, err)
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountNotFoundError,
			ErrorMessage: "アカウント認証に失敗しました",
		}
		return nil, gkillError, err
	}

	if !account.IsEnable {
		err = fmt.Errorf("error at disable account user id = %s: %w", loginSession.UserID, err)
		gkill_log.Debug.Println(err.Error())
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
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		gkill_log.Debug.Println(err.Error())
		return "", "", err
	}
	return serverConfig.TLSCertFile, serverConfig.TLSKeyFile, nil
}

func (g *GkillServerAPI) HandleFileServe(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()

	sessionID := ""
	sharedID := ""

	// クッキーを見て認証する
	sessionIDCookie, err := r.Cookie("gkill_session_id")
	if err != nil {
		sharedIDCookie, err := r.Cookie("gkill_shared_id")
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			gkill_log.Trace.Printf("finish %#v", err)
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
		account, gkillError, err := g.getAccountFromSessionID(r.Context(), sessionID)
		if account == nil || gkillError != nil || err != nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			gkill_log.Trace.Printf("finish %#v", err)
			return
		}
		userID = account.UserID
	} else if sharedID != "" {
		sharedKyouInfo, err := g.GkillDAOManager.ConfigDAOs.ShareKyouInfoDAO.GetKyouShareInfo(r.Context(), sharedID)
		if err != nil || sharedKyouInfo == nil {
			w.WriteHeader(http.StatusForbidden)
			err = fmt.Errorf("error at handle file serve: %w", err)
			gkill_log.Trace.Printf("finish %#v", err)
			return
		}
		userID = sharedKyouInfo.UserID
	} else {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Trace.Printf("finish %#v", err)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Trace.Printf("finish %#v", err)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		err = fmt.Errorf("error at handle file serve: %w", err)
		gkill_log.Trace.Printf("finish %#v", err)
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
			gkill_log.Trace.Printf("finish %#v", err)
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
		gkill_log.Trace.Printf("finish %#v", err)
		return
	}

	// StripPrefixしてIDFサーバのハンドラにわたす
	rootAddress := "/files/" + targetRepName
	http.StripPrefix(rootAddress, http.HandlerFunc(targetIDFRep.HandleFileServe)).ServeHTTP(w, r)
}

func (g *GkillServerAPI) HandleGetGkillNotificationPublicKey(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.GetGkillNotificationPublicKeyRequest{}
	response := &req_res.GetGkillNotificationPublicKeyResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse get mi task notification public key response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetMiTaskNotificationPublicKeyResponseDataError,
				ErrorMessage: "Miタスク通知登録に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get get mi task notification public key request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetMiTaskNotificationPublicKeyRequestDataError,
			ErrorMessage: "Miタスク通知登録に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "Miタスク通知登録に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ServerConfig取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.GkillNotificationPublicKey = currentServerConfig.GkillNotificationPublicKey
}
func (g *GkillServerAPI) HandleRegisterGkillNotification(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.RegisterGkillNotificationRequest{}
	response := &req_res.RegisterGkillNotificationResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse register mi task notification response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterMiTaskNotificationResponse,
				ErrorMessage: "Miタスク通知登録に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get register mi task notification request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterMiTaskNotificationRequest,
			ErrorMessage: "Miタスク通知登録に失敗しました",
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

	gkillNotificationTarget := &gkill_notification.GkillNotificateTarget{
		ID:           sqlite3impl.GenerateNewID(),
		UserID:       userID,
		PublicKey:    request.PublicKey,
		Subscription: gkill_notification.JSONString(request.Subscription),
	}

	_, err = g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.AddGkillNotificationTarget(r.Context(), gkillNotificationTarget)
	if err != nil {
		err = fmt.Errorf("error at add mi notification target : %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddGkillNotificationTargetError,
			ErrorMessage: "Miタスク通知登録に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.UpdateTagSuccessMessage,
		Message:     "通知登録が完了しました",
	})
}

func (g *GkillServerAPI) HandleOpenDirectory(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.OpenDirectoryRequest{}
	response := &req_res.OpenDirectoryResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse open directory response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterOpenDirectoryResponse,
				ErrorMessage: "フォルダを開けませんでした",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse open directory request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenDirectoryRequest,
			ErrorMessage: "フォルダを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderError,
			ErrorMessage: "フォルダを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if !session.IsLocalAppUser {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderNotLocalAccountError,
			ErrorMessage: "フォルダを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "フォルダを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(session.UserID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories. userid = %s device = %s: %w", session.UserID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: "フォルダを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	filename, err := repositories.Reps.GetPath(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get path. id = %s userid = %s device = %s: %w", request.TargetID, session.UserID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: "フォルダを開けませんでした",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "フォルダを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.OpenDirectorySuccessMessage,
		Message:     "フォルダを開きました",
	})
}

func (g *GkillServerAPI) HandleOpenFile(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.OpenFileRequest{}
	response := &req_res.OpenFileResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse open file response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidRegisterOpenFileResponse,
				ErrorMessage: "ファイルを開けませんでした",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse open file request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenFileRequest,
			ErrorMessage: "ファイルを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	session, err := g.GkillDAOManager.ConfigDAOs.LoginSessionDAO.GetLoginSession(r.Context(), request.SessionID)
	if err != nil {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderError,
			ErrorMessage: "ファイルを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	if !session.IsLocalAppUser {
		err = fmt.Errorf("error at get login session session id = %s: %w", request.SessionID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.OpenFolderNotLocalAccountError,
			ErrorMessage: "ファイルを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		err = fmt.Errorf("error at get server config device = %s: %w", device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ファイルを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(session.UserID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories. userid = %s device = %s: %w", session.UserID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepositoriesError,
			ErrorMessage: "ファイルを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	filename, err := repositories.Reps.GetPath(r.Context(), request.TargetID)
	if err != nil {
		err = fmt.Errorf("error at get path. id = %s userid = %s device = %s: %w", request.TargetID, session.UserID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetRepPathError,
			ErrorMessage: "ファイルを開けませんでした",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ファイルを開けませんでした",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.OpenFileSuccessMessage,
		Message:     "ファイルを開きました",
	})
}

func (g *GkillServerAPI) HandleReloadRepositories(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ReloadRepositoriesRequest{}
	response := &req_res.ReloadRepositoriesResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse reload repositories response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidReloadRepositoriesResponse,
				ErrorMessage: "リロードに失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse reload repositories request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidRegisterOpenFileRequest,
			ErrorMessage: "リロードに失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	_, err = g.GkillDAOManager.CloseUserRepositories(userID, device)
	if err != nil {
		panic(err)
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "リロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "リロードに失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.ReloadRepositoriesSuccessMessage,
		Message:     "リロードしました",
	})
}

func (g *GkillServerAPI) ifRedirectResetAdminAccountIsNotFound(w http.ResponseWriter, r *http.Request) bool {
	accounts, err := g.GkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(context.TODO())
	if err != nil {
		err = fmt.Errorf("error at get all account config")
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetAllAccountConfigError,
			ErrorMessage: "アカウント設定情報の取得に失敗しました",
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
	defer func() { runtime.GC() }()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	request := &req_res.URLogBookmarkletRequest{}

	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse urlog bookmarklet request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidAddKmemoRequestDataError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		// response.Errors = append(response.Errors, gkillError)
		_ = gkillError
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		gkill_log.Debug.Println(err.Error())
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	repositories, err := g.GkillDAOManager.GetRepositories(userID, device)
	if err != nil {
		err = fmt.Errorf("error at get repositories user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "URLog追加に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	if existURLog != nil {
		err = fmt.Errorf("exist urlog id = %s", urlog.ID)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AleadyExistURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// applicationConfigを取得
	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil {
		err = fmt.Errorf("error at get applicationConfig user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetApplicationConfigError,
			ErrorMessage: "ApplicationConfig取得に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// serverConfigを取得
	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(r.Context(), device)
	if err != nil {
		err = fmt.Errorf("error at get serverConfig user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetServerConfigError,
			ErrorMessage: "ServerConfig取得に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	urlog.FillURLogField(serverConfig, applicationConfig)

	err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), urlog)
	if err != nil {
		err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AddURLogError,
			ErrorMessage: "URLog追加に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	// defer g.WebPushUpdatedData(r.Context(), userID, device, urlog.ID)

	if len(repositories.URLogReps) == 1 && gkill_options.CacheURLogReps {
		err = repositories.URLogReps[0].AddURLogInfo(r.Context(), urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: "URLog追加に失敗しました",
			}
			_ = gkillError
			// response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
	if err != nil {
		err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}
	_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
		IsDeleted:                              urlog.IsDeleted,
		TargetID:                               urlog.ID,
		DataUpdateTime:                         urlog.UpdateTime,
		LatestDataRepositoryName:               repName,
		LatestDataRepositoryAddressUpdatedTime: time.Now(),
	})
	if err != nil {
		err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetURLogError,
			ErrorMessage: "URLog追加後取得に失敗しました",
		}
		_ = gkillError
		// response.Errors = append(response.Errors, gkillError)
		return
	}

	// 通知する
	// 現在のServerConfigを取得する
	var currentServerConfig *server_config.ServerConfig
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(r.Context())
	if err != nil {
		gkill_log.Debug.Print(err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		gkill_log.Debug.Print(err)
		return
	}

	// 送信対象を取得する
	notificationTargets, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(r.Context(), userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		gkill_log.Debug.Print(err)
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
		gkill_log.Debug.Print(err)
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
			gkill_log.Debug.Println(err.Error())
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(r.Context(), notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				gkill_log.Debug.Println(err.Error())
			}
		}
	}
}

func (g *GkillServerAPI) HandleGetUpdatedDatasByTime(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
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
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidGetUpdatedDatasByTimeResponse,
				ErrorMessage: "最新情報の取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse get updated data by time request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidGetUpdatedDatasByTimeRequest,
			ErrorMessage: "最新情報の取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// アカウントを取得
	account, gkillError, err := g.getAccountFromSessionID(r.Context(), request.SessionID)
	if err != nil {
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, gkillError)
		return
	}

	userID := account.UserID
	device, err := g.GetDevice()
	if err != nil {
		err = fmt.Errorf("error at get device name: %w", err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "最新情報取得に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	limit := gkill_options.CacheClearCountLimit + 1
	updatedInfos, err := repositories.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddressByUpdateTimeAfter(r.Context(), request.LastUpdatedTime, limit)
	if err != nil {
		err = fmt.Errorf("error at get latest data repositories data user id = %s device = %s: %w", userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetLatestDataRepositoryAddressByUpdateTimeAfterError,
			ErrorMessage: "最新情報取得に失敗しました",
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
	serverConfigs, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		/*
			gkillError := &message.GkillError{
				ErrorCode:    message.GetDeviceError,
				ErrorMessage: "内部エラー",
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
		gkill_log.Debug.Println(err.Error())
		/*
			gkillError := &message.GkillError{
				ErrorCode:    message.GetServerConfigError,
				ErrorMessage: "ServerConfig取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
		*/
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	if serverConfig == nil {
		err = fmt.Errorf("error at server config is nil device = %s: %w", device, err)
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Print(err)
		return
	}
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			currentServerConfig = serverConfig
		}
	}
	if currentServerConfig == nil {
		err = fmt.Errorf("current server config is not found. in gkill notificator")
		gkill_log.Debug.Print(err)
		return
	}

	// 送信対象を取得する
	notificationTargets, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.GetGkillNotificationTargets(ctx, userID, currentServerConfig.GkillNotificationPublicKey)
	if err != nil {
		err = fmt.Errorf("get notification target. in gkill notificator.: %w", err)
		gkill_log.Debug.Print(err)
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
		gkill_log.Debug.Print(err)
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
			gkill_log.Debug.Println(err.Error())
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}
		// 登録解除されていたらDBから消す
		if resp.Status == "410 Gone" {
			_, err := g.GkillDAOManager.ConfigDAOs.GkillNotificationTargetDAO.DeleteGkillNotificationTarget(ctx, notificationTarget.ID)
			if err != nil {
				err = fmt.Errorf("error at delete gkill notification target after got 410 Gone: %w", err)
				gkill_log.Debug.Println(err.Error())
			}
		}
	}
}

func (g *GkillServerAPI) HandleCommitTx(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.CommitTxRequest{}
	response := &req_res.CommitTxResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse commit tx response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidCommitTxResponseDataError,
				ErrorMessage: "保存に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse commit tx request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidCommitTxRequestDataError,
			ErrorMessage: "保存に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.RepositoriesGetError,
			ErrorMessage: "kmemo追加に失敗しました",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	txID := request.TXID
	ctx := r.Context()

	kmemos, err := g.GkillDAOManager.TempReps.KmemoTempRep.GetKmemosByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kmemo by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetKmemoError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}
	kcs, err := g.GkillDAOManager.TempReps.KCTempRep.GetKCsByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get kc by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetKCError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	idfKyous, err := g.GkillDAOManager.TempReps.IDFKyouTempRep.GetIDFKyousByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get idfkyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetIDFKyouError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	lantanas, err := g.GkillDAOManager.TempReps.LantanaTempRep.GetLantanasByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get lantana by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetLantanaError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	mis, err := g.GkillDAOManager.TempReps.MiTempRep.GetMisByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get mi by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetMiError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	nlogs, err := g.GkillDAOManager.TempReps.NlogTempRep.GetNlogsByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get nlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetNlogError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	notifications, err := g.GkillDAOManager.TempReps.NotificationTempRep.GetNotificationsByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get notification by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetNotificationError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	rekyous, err := g.GkillDAOManager.TempReps.ReKyouTempRep.GetReKyousByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get rekyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetReKyouError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	tags, err := g.GkillDAOManager.TempReps.TagTempRep.GetTagsByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get tag by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTagError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	texts, err := g.GkillDAOManager.TempReps.TextTempRep.GetTextsByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get text by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTextError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	timeiss, err := g.GkillDAOManager.TempReps.TimeIsTempRep.GetTimeIssByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get timeis by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetTimeIsError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	urlogs, err := g.GkillDAOManager.TempReps.URLogTempRep.GetURLogsByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at get urlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.CommitTxGetURLogError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	for _, idfKyou := range idfKyous {
		err = repositories.WriteIDFKyouRep.AddIDFKyouInfo(r.Context(), idfKyou)
		if err != nil {
			err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddIDFKyouError,
				ErrorMessage: "IDFKyou更新に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		// defer g.WebPushUpdatedData(r.Context(), userID, device, request.IDFKyou.ID)

		repName, err := repositories.WriteIDFKyouRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: "IDFKyou更新後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.IDFKyouReps) == 1 && gkill_options.CacheIDFKyouReps {
			err = repositories.IDFKyouReps[0].AddIDFKyouInfo(r.Context(), idfKyou)
			if err != nil {
				err = fmt.Errorf("error at add idfKyou user id = %s device = %s idfKyou = %#v: %w", userID, device, idfKyou, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddIDFKyouError,
					ErrorMessage: "IDFKyou更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              idfKyou.IsDeleted,
			TargetID:                               idfKyou.ID,
			DataUpdateTime:                         idfKyou.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get idfKyou user id = %s device = %s id = %s: %w", userID, device, idfKyou.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetIDFKyouError,
				ErrorMessage: "IDFKyou更新後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, kc := range kcs {
		err = repositories.WriteKCRep.AddKCInfo(r.Context(), kc)
		if err != nil {
			err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKCError,
				ErrorMessage: "KC追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KCReps) == 1 && gkill_options.CacheKCReps {
			err = repositories.KCReps[0].AddKCInfo(r.Context(), kc)
			if err != nil {
				err = fmt.Errorf("error at add kc user id = %s device = %s kc = %#v: %w", userID, device, kc, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddKCError,
					ErrorMessage: "KC追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteKCRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: "KC追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              kc.IsDeleted,
			TargetID:                               kc.ID,
			DataUpdateTime:                         kc.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get kc user id = %s device = %s id = %s: %w", userID, device, kc.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKCError,
				ErrorMessage: "KC追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, kmemo := range kmemos {
		err = repositories.WriteKmemoRep.AddKmemoInfo(r.Context(), kmemo)
		if err != nil {
			err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddKmemoError,
				ErrorMessage: "Kmemo追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.KmemoReps) == 1 && gkill_options.CacheKmemoReps {
			err = repositories.KmemoReps[0].AddKmemoInfo(r.Context(), kmemo)
			if err != nil {
				err = fmt.Errorf("error at add kmemo user id = %s device = %s kmemo = %#v: %w", userID, device, kmemo, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddKmemoError,
					ErrorMessage: "Kmemo追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteKmemoRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: "Kmemo追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              kmemo.IsDeleted,
			TargetID:                               kmemo.ID,
			DataUpdateTime:                         kmemo.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get kmemo user id = %s device = %s id = %s: %w", userID, device, kmemo.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetKmemoError,
				ErrorMessage: "Kmemo追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, lantana := range lantanas {
		err = repositories.WriteLantanaRep.AddLantanaInfo(r.Context(), lantana)
		if err != nil {
			err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddLantanaError,
				ErrorMessage: "Lantana追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.LantanaReps) == 1 && gkill_options.CacheLantanaReps {
			err = repositories.LantanaReps[0].AddLantanaInfo(r.Context(), lantana)
			if err != nil {
				err = fmt.Errorf("error at add lantana user id = %s device = %s lantana = %#v: %w", userID, device, lantana, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddLantanaError,
					ErrorMessage: "Lantana追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteLantanaRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: "Lantana追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              lantana.IsDeleted,
			TargetID:                               lantana.ID,
			DataUpdateTime:                         lantana.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get lantana user id = %s device = %s id = %s: %w", userID, device, lantana.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetLantanaError,
				ErrorMessage: "Lantana追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, mi := range mis {
		err = repositories.WriteMiRep.AddMiInfo(r.Context(), mi)
		if err != nil {
			err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddMiError,
				ErrorMessage: "Mi追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.MiReps) == 1 && gkill_options.CacheMiReps {
			err = repositories.MiReps[0].AddMiInfo(r.Context(), mi)
			if err != nil {
				err = fmt.Errorf("error at add mi user id = %s device = %s mi = %#v: %w", userID, device, mi, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddMiError,
					ErrorMessage: "Mi追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteMiRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: "Mi追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              mi.IsDeleted,
			TargetID:                               mi.ID,
			DataUpdateTime:                         mi.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get mi user id = %s device = %s id = %s: %w", userID, device, mi.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetMiError,
				ErrorMessage: "Mi追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, nlog := range nlogs {
		err = repositories.WriteNlogRep.AddNlogInfo(r.Context(), nlog)
		if err != nil {
			err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNlogError,
				ErrorMessage: "Nlog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.NlogReps) == 1 && gkill_options.CacheNlogReps {
			err = repositories.NlogReps[0].AddNlogInfo(r.Context(), nlog)
			if err != nil {
				err = fmt.Errorf("error at add nlog user id = %s device = %s nlog = %#v: %w", userID, device, nlog, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddNlogError,
					ErrorMessage: "Nlog追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteNlogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: "Nlog追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              nlog.IsDeleted,
			TargetID:                               nlog.ID,
			DataUpdateTime:                         nlog.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get nlog user id = %s device = %s id = %s: %w", userID, device, nlog.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNlogError,
				ErrorMessage: "Nlog追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, notification := range notifications {
		err = repositories.WriteNotificationRep.AddNotificationInfo(r.Context(), notification)
		if err != nil {
			err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddNotificationError,
				ErrorMessage: "通知追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.NotificationReps) == 1 && gkill_options.CacheNotificationReps {
			err = repositories.NotificationReps[0].AddNotificationInfo(r.Context(), notification)
			if err != nil {
				err = fmt.Errorf("error at add notification user id = %s device = %s notification = %#v: %w", userID, device, notification, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddNotificationError,
					ErrorMessage: "通知更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteNotificationRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNotificationError,
				ErrorMessage: "通知追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              notification.IsDeleted,
			TargetID:                               notification.ID,
			TargetIDInData:                         &notification.TargetID,
			DataUpdateTime:                         notification.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get notification user id = %s device = %s id = %s: %w", userID, device, notification.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetNotificationError,
				ErrorMessage: "通知追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, rekyou := range rekyous {
		err = repositories.WriteReKyouRep.AddReKyouInfo(r.Context(), rekyou)
		if err != nil {
			err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddReKyouError,
				ErrorMessage: "ReKyou追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.ReKyouReps.ReKyouRepositories) == 1 && gkill_options.CacheReKyouReps {
			err = repositories.ReKyouReps.ReKyouRepositories[0].AddReKyouInfo(r.Context(), rekyou)
			if err != nil {
				err = fmt.Errorf("error at add rekyou user id = %s device = %s rekyou = %#v: %w", userID, device, rekyou, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddReKyouError,
					ErrorMessage: "ReKyou更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteReKyouRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: "ReKyou追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              rekyou.IsDeleted,
			TargetID:                               rekyou.ID,
			TargetIDInData:                         &rekyou.TargetID,
			DataUpdateTime:                         rekyou.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get rekyou user id = %s device = %s id = %s: %w", userID, device, rekyou.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetReKyouError,
				ErrorMessage: "ReKyou追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, tag := range tags {
		err = repositories.WriteTagRep.AddTagInfo(r.Context(), tag)
		if err != nil {
			err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTagError,
				ErrorMessage: "タグ追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		// キャッシュに書き込み
		if len(repositories.TagReps) == 1 && gkill_options.CacheTagReps {
			err = repositories.TagReps[0].AddTagInfo(r.Context(), tag)
			if err != nil {
				err = fmt.Errorf("error at add tag user id = %s device = %s tag = %#v: %w", userID, device, tag, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTagError,
					ErrorMessage: "タグ追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteTagRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: "タグ追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              tag.IsDeleted,
			TargetID:                               tag.ID,
			TargetIDInData:                         &tag.TargetID,
			DataUpdateTime:                         tag.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get tag user id = %s device = %s id = %s: %w", userID, device, tag.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTagError,
				ErrorMessage: "タグ追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, text := range texts {
		err = repositories.WriteTextRep.AddTextInfo(r.Context(), text)
		if err != nil {
			err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTextError,
				ErrorMessage: "テキスト追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		if len(repositories.TextReps) == 1 && gkill_options.CacheTextReps {
			err = repositories.TextReps[0].AddTextInfo(r.Context(), text)
			if err != nil {
				err = fmt.Errorf("error at add text user id = %s device = %s text = %#v: %w", userID, device, text, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTextError,
					ErrorMessage: "テキスト追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteTextRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTextError,
				ErrorMessage: "テキスト追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              text.IsDeleted,
			TargetID:                               text.ID,
			TargetIDInData:                         &text.TargetID,
			DataUpdateTime:                         text.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get text user id = %s device = %s id = %s: %w", userID, device, text.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTextError,
				ErrorMessage: "テキスト追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, timeis := range timeiss {
		err = repositories.WriteTimeIsRep.AddTimeIsInfo(r.Context(), timeis)
		if err != nil {
			err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddTimeIsError,
				ErrorMessage: "TimeIs追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.TimeIsReps) == 1 && gkill_options.CacheTimeIsReps {
			err = repositories.TimeIsReps[0].AddTimeIsInfo(r.Context(), timeis)
			if err != nil {
				err = fmt.Errorf("error at add timeis user id = %s device = %s timeis = %#v: %w", userID, device, timeis, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddTimeIsError,
					ErrorMessage: "TimeIs追加に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteTimeIsRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: "TimeIs追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              timeis.IsDeleted,
			TargetID:                               timeis.ID,
			DataUpdateTime:                         timeis.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get timeis user id = %s device = %s id = %s: %w", userID, device, timeis.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetTimeIsError,
				ErrorMessage: "TimeIs追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	for _, urlog := range urlogs {
		err = repositories.WriteURLogRep.AddURLogInfo(r.Context(), urlog)
		if err != nil {
			err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AddURLogError,
				ErrorMessage: "URLog追加に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}

		if len(repositories.URLogReps) == 1 && gkill_options.CacheURLogReps {
			err = repositories.URLogReps[0].AddURLogInfo(r.Context(), urlog)
			if err != nil {
				err = fmt.Errorf("error at add urlog user id = %s device = %s urlog = %#v: %w", userID, device, urlog, err)
				gkill_log.Debug.Println(err.Error())
				gkillError := &message.GkillError{
					ErrorCode:    message.AddURLogError,
					ErrorMessage: "URLog更新に失敗しました",
				}
				response.Errors = append(response.Errors, gkillError)
				return
			}
		}

		repName, err := repositories.WriteURLogRep.GetRepName(r.Context())
		if err != nil {
			err = fmt.Errorf("error at get rep name user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: "URLog追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
		_, err = repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(r.Context(), &account_state.LatestDataRepositoryAddress{
			IsDeleted:                              urlog.IsDeleted,
			TargetID:                               urlog.ID,
			DataUpdateTime:                         urlog.UpdateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		})
		if err != nil {
			err = fmt.Errorf("error at get urlog user id = %s device = %s id = %s: %w", userID, device, urlog.ID, err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.GetURLogError,
				ErrorMessage: "URLog追加後取得に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.CommitTxSuccessMessage,
		Message:     "保存しました",
	})
}
func (g *GkillServerAPI) HandleDiscardTX(w http.ResponseWriter, r *http.Request) {
	defer func() { runtime.GC() }()
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DiscardTxRequest{}
	response := &req_res.DiscardTxResponse{}

	defer r.Body.Close()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse discart tx response to json: %w", err)
			gkill_log.Debug.Println(err.Error())
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidDiscardTxResponseDataError,
				ErrorMessage: "トランザクション破棄に失敗しました",
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse discard tx request to json: %w", err)
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidDiscardTxRequestDataError,
			ErrorMessage: "トランザクション破棄に失敗しました",
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
		gkill_log.Debug.Println(err.Error())
		gkillError := &message.GkillError{
			ErrorCode:    message.GetDeviceError,
			ErrorMessage: "内部エラー",
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	txID := request.TXID
	ctx := r.Context()

	err = g.GkillDAOManager.TempReps.IDFKyouTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete idfKyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteIDFKyouError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.KCTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete kc by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteKCError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.KmemoTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete kmemo by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteKmemoError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.LantanaTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete lantana by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteLantanaError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.MiTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete mi by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteMiError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.NlogTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete nlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteNlogError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.NotificationTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete notification by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteNotificationError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.ReKyouTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete rekyou by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteReKyouError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.TagTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete tag by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTagError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.TextTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete text by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTextError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.TimeIsTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete timeis by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteTimeIsError,
			ErrorMessage: "内部エラー",
		})
		return
	}
	err = g.GkillDAOManager.TempReps.URLogTempRep.DeleteByTXID(ctx, txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at delete urlog by tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		gkill_log.Debug.Println(err.Error())
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    message.CommitTxDeleteURLogError,
			ErrorMessage: "内部エラー",
		})
		return
	}
}

func (g *GkillServerAPI) PrintStartedMessage() {
	device, err := g.GetDevice()
	if err != nil {
		gkill_log.Debug.Println("Error getting device information:", err)
		return
	}

	serverConfig, err := g.GkillDAOManager.ConfigDAOs.ServerConfigDAO.GetServerConfig(context.Background(), device)
	if err != nil {
		gkill_log.Debug.Println("Error getting server configuration:", err)
		return
	}

	port := serverConfig.Address
	protocol := "http"
	if serverConfig.EnableTLS {
		protocol = "https"
	}

	os.Stdout.WriteString("gkill server started.\n")
	os.Stdout.WriteString(fmt.Sprintf("Access your record space at : %s://localhost%s\n", protocol, port))
}
