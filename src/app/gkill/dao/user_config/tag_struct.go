package user_config

type TagStruct struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	TagName string `json:"tag_name"`

	ParentFolderID string `json:"parent_folder_id"`

	Seq int `json:"seq"`

	CheckWhenInited bool `json:"check_when_inited"`

	IsForceHide bool `json:"is_force_hide"`

	IsDir bool `json:"is_dir"`

	IsOpenDefault bool `json:"is_open_default"`
}
