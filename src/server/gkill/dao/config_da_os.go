package dao

import (
	"github.com/mt3hr/gkill/src/app/gkill/dao/account"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/gkill_notification"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dao/share_kyou_info"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
)

type ConfigDAOs struct {
	AccountDAO account.AccountDAO

	LoginSessionDAO account_state.LoginSessionDAO

	FileUploadHistoryDAO account_state.FileUploadHistoryDAO

	ShareKyouInfoDAO share_kyou_info.ShareKyouInfoDAO

	ServerConfigDAO server_config.ServerConfigDAO

	AppllicationConfigDAO user_config.ApplicationConfigDAO

	RepositoryDAO user_config.RepositoryDAO

	GkillNotificationTargetDAO gkill_notification.GkillNotificateTargetDAO
}
