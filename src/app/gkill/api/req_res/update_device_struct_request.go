package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateDeviceStructRequest struct {
	SessionID string `json:"session_id"`

	DeviceStruct []*user_config.DeviceStruct `json:"device_struct"`
}
