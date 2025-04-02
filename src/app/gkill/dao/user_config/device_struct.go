package user_config

type DeviceStruct struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	DeviceName string `json:"device_name"`

	Device string `json:"device"`

	ParentFolderID string `json:"parent_folder_id"`

	Seq int `json:"seq"`

	CheckWhenInited bool `json:"check_when_inited"`

	IsDir bool `json:"is_dir"`

	IsOpenDefault bool `json:"is_open_default"`
}
