// ˅
package dbo

import (
	"github.com/mt3hr/gkill/src/app/gkill/dbo/account"
	"github.com/mt3hr/gkill/src/app/gkill/dbo/account_state"
	"github.com/mt3hr/gkill/src/app/gkill/dbo/mi_share_info"
	"github.com/mt3hr/gkill/src/app/gkill/dbo/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dbo/user_config"
)

// ˄

type ConfigDAOs struct {
	// ˅

	// ˄

	AccountDAO account.AccountDAO

	LoginSessionDAO account_state.LoginSessionDAO

	FileUploadHistoryDAO account_state.FileUploadHistoryDAO

	MiShareInfoDAO mi_share_info.MiShareInfoDAO

	ServerConfigDAO server_config.ServerConfigDAO

	AppllicationConfigDAO user_config.ApplicationConfigDAO

	RepositoryDAO user_config.RepositoryDAO

	KFTLTemplateDAO user_config.KFTLTemplateDAO

	TagStructDAO user_config.TagStructDAO

	RepStructDAO user_config.RepStructDAO

	DeviceStructDAO user_config.DeviceStructDAO

	RepTypeStructDAO user_config.RepTypeStructDAO

	// ˅

	// ˄
}

// ˅

// ˄