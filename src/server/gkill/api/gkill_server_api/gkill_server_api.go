package gkill_server_api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/mt3hr/gkill/src/server/gkill/usecase"
)

func NewGkillServerAPI() (*GkillServerAPI, error) {
	ctx := context.Background()
	gkillDAOManager, err := dao.NewGkillDAOManager()
	if err != nil {
		err = fmt.Errorf("error at create gkill dao manager: %w", err)
		return nil, err
	}

	// 初回起動の場合、Adminアカウントデータを作成する
	accounts, err := gkillDAOManager.ConfigDAOs.AccountDAO.GetAllAccounts(ctx)
	if err != nil {
		err = fmt.Errorf("error at get all account: %w", err)
		return nil, err
	}
	if len(accounts) == 0 {
		passwordResetToken := GenerateNewID()
		passwordResetTokenExpiration := time.Now().Add(account.PasswordResetTokenTTL)
		adminAccount := &account.Account{
			UserID:                       "admin",
			PasswordHash:                 nil,
			IsAdmin:                      true,
			IsEnable:                     true,
			PasswordResetToken:           &passwordResetToken,
			PasswordResetTokenExpiration: &passwordResetTokenExpiration,
		}
		_, err := gkillDAOManager.ConfigDAOs.AccountDAO.AddAccount(ctx, adminAccount)
		if err != nil {
			err = fmt.Errorf("error at add admin account: %w", err)
			return nil, err
		}
	}

	serverConfigs, err := gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.Background())
	if err != nil {
		err = fmt.Errorf("error at get all server configs: %w", err)
		slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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

		_, err = gkillDAOManager.ConfigDAOs.ServerConfigDAO.AddServerConfig(ctx, serverConfig)
		if err != nil {
			err = fmt.Errorf("error at add init data to server config db: %w", err)
			return nil, err
		}

		serverConfigs, err = gkillDAOManager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(context.Background())
		if err != nil {
			err = fmt.Errorf("error at get all server configs: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
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

	applicationConfigs, err := gkillDAOManager.ConfigDAOs.AppllicationConfigDAO.GetAllApplicationConfigs(ctx)
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

	findFilter := &api.FindFilter{}
	return &GkillServerAPI{
		APIAddress:       NewGKillAPIAddress(),
		GkillDAOManager:  gkillDAOManager,
		FindFilter:       findFilter,
		UsecaseCtx:       usecase.NewUsecaseContext(gkillDAOManager, findFilter),
		RebootServerCh:   make(chan struct{}),
		loginRateLimiter: newLoginRateLimiter(),
		// set_new_passwordは未認証で叩けるので、ログインとは別枠で試行回数を絞る
		passwordResetRateLimiter: newLoginRateLimiter(),
	}, nil
}

type GkillServerAPI struct {
	server *http.Server

	APIAddress *GkillServerAPIAddress

	GkillDAOManager *dao.GkillDAOManager

	FindFilter *api.FindFilter

	UsecaseCtx *usecase.UsecaseContext

	RebootServerCh chan (struct{})

	// このプロセスが動いているデバイス名。
	//
	// 取得元の GetAllServerConfigs は SERVER_CONFIG への相関サブクエリを18本使う重いSQLで、
	// 認証付きAPI1本につき最低3回(filterLocalOnly で2回 + auth ミドルウェアで1回)、
	// /files/ の画像配信では1枚ごとに走っていた。
	//
	// 設定を更新する handle_update_server_configs は自分でサーバを落として
	// GkillServerAPI ごと作り直す(common.go の LaunchGkillServerAPI のループ)ので、
	// この構造体に持たせたキャッシュは設定更新時に自然に捨てられる。無効化は要らない。
	deviceOnce sync.Once
	device     string
	deviceErr  error

	loginRateLimiter *loginRateLimiter

	passwordResetRateLimiter *loginRateLimiter

	closeOnce sync.Once
	closeErr  error
}
