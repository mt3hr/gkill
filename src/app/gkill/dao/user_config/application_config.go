package user_config

type ApplicationConfig struct {
	UserID string `json:"user_id"`

	Device string `json:"device"`

	EnableBrowserCache bool `json:"enable_browser_cache"`

	GoogleMapAPIKey string `json:"google_map_api_key"`

	RykvImageListColumnNumber int `json:"rykv_image_list_column_number"`

	RykvHotReload bool `json:"rykv_hot_reload"`

	MiDefaultBoard string `json:"mi_default_board"`

	KFTLTemplate []*KFTLTemplate `json:"kftl_template"`

	TagStrcuct []*TagStruct `json:"tag_strcuct"`

	RepStruct []*RepStruct `json:"rep_struct"`

	DeviceStruct []*DeviceStruct `json:"device_struct"`

	RepTypeStruct []*RepTypeStruct `json:"rep_type_struct"`
}
