package user_config

import (
	"encoding/json"
	"time"
)

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

	DefaultPage string `json:"default_page"`

	KFTLTemplate *json.RawMessage `json:"kftl_template_struct"`

	TagStruct *json.RawMessage `json:"tag_struct"`

	RepStruct *json.RawMessage `json:"rep_struct"`

	DeviceStruct *json.RawMessage `json:"device_struct"`

	RepTypeStruct *json.RawMessage `json:"rep_type_struct"`

	DnoteJSONData *json.RawMessage `json:"dnote_json_data"`

	RyuuJSONData *json.RawMessage `json:"ryuu_json_data"`

	MiBoardStruct *json.RawMessage `json:"mi_board_struct"`

	AccountIsAdmin bool `json:"account_is_admin"`

	ShowTagsInList bool `json:"show_tags_in_list"`

	SessionIsLocal bool `json:"session_is_local"`

	URLogBookmarkletSession string `json:"urlog_bookmarklet_session"`

	UserIsAdmin bool `json:"user_is_admin"`

	CacheClearCountLimit int64 `json:"cache_clear_count_limit"`

	GlobalIP string `json:"global_ip"`

	PrivateIP string `json:"private_ip"`

	Version string `json:"version"`

	BuildTime time.Time `json:"build_time"`

	CommitHash string `json:"commit_hash"`
}
