package user_config

type KFTLTemplate struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	Title string `json:"title"`

	Template string `json:"template"`

	ParentFolderID string `json:"parent_folder_id"`

	Seq int `json:"seq"`
}
