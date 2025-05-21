package user_config

import "encoding/json"

type ApplicationConfig struct {
	UserID string `json:"user_id"`

	Device string `json:"device"`

	UseDarkTheme bool `json:"use_dark_theme"`

	GoogleMapAPIKey string `json:"google_map_api_key"`

	RykvImageListColumnNumber int `json:"rykv_image_list_column_number"`

	RykvHotReload bool `json:"rykv_hot_reload"`

	MiDefaultBoard string `json:"mi_default_board"`

	RykvDefaultPeriod json.Number `json:"rykv_default_period"`

	MiDefaultPeriod json.Number `json:"mi_default_period"`

	IsShowShareFooter bool `json:"is_show_share_footer"`

	KFTLTemplate []*KFTLTemplate `json:"kftl_template_struct"`

	TagStruct []*TagStruct `json:"tag_struct"`

	RepStruct []*RepStruct `json:"rep_struct"`

	DeviceStruct []*DeviceStruct `json:"device_struct"`

	RepTypeStruct []*RepTypeStruct `json:"rep_type_struct"`

	DnoteJSONData json.RawMessage `json:"dnote_json_data"`

	RyuuJSONData json.RawMessage `json:"ryuu_json_data"`

	AccountIsAdmin bool `json:"account_is_admin"`

	SessionIsLocal bool `json:"session_is_local"`
}
