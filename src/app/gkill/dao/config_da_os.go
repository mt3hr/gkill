package dao

import (
	"github.com/mt3hr/gkill/src/app/gkill/dao/account"
	"github.com/mt3hr/gkill/src/app/gkill/dao/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dao/gkill_notification"
	"github.com/mt3hr/gkill/src/app/gkill/dao/mi_share_info"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dao/user_config"
)

type ConfigDAOs struct {
	AccountDAO account.AccountDAO

	LoginSessionDAO account_state.LoginSessionDAO

	FileUploadHistoryDAO account_state.FileUploadHistoryDAO

	MiShareInfoDAO mi_share_info.MiShareInfoDAO

	ServerConfigDAO server_config.ServerConfigDAO

	AppllicationConfigDAO user_config.ApplicationConfigDAO

	RepositoryDAO user_config.RepositoryDAO

	KFTLTemplateDAO user_config.KFTLTemplateDAO

	DnoteDataDAO user_config.DnoteDataDAO

	TagStructDAO user_config.TagStructDAO

	RepStructDAO user_config.RepStructDAO

	DeviceStructDAO user_config.DeviceStructDAO

	RepTypeStructDAO user_config.RepTypeStructDAO

	GkillNotificationTargetDAO gkill_notification.GkillNotificateTargetDAO
}
