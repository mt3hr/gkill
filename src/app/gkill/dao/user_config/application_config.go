package user_config

import "encoding/json"

type ApplicationConfig struct {
	UserID string `json:"user_id"`

	Device string `json:"device"`

	EnableBrowserCache bool `json:"enable_browser_cache"`

	GoogleMapAPIKey string `json:"google_map_api_key"`

	RykvImageListColumnNumber int `json:"rykv_image_list_column_number"`

	RykvHotReload bool `json:"rykv_hot_reload"`

	MiDefaultBoard string `json:"mi_default_board"`

	RykvDefaultPeriod json.Number `json:"rykv_default_period"`

	MiDefaultPeriod json.Number `json:"mi_default_period"`

	KFTLTemplate []*KFTLTemplate `json:"kftl_template_struct"`

	TagStruct []*TagStruct `json:"tag_struct"`

	RepStruct []*RepStruct `json:"rep_struct"`

	DeviceStruct []*DeviceStruct `json:"device_struct"`

	RepTypeStruct []*RepTypeStruct `json:"rep_type_struct"`

	AccountIsAdmin bool `json:"account_is_admin"`

	SessionIsLocal bool `json:"session_is_local"`
}
