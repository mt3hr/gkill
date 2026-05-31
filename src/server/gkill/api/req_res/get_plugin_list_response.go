package req_res

import (
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

// PluginInfo はAPIが返すプラグインの情報。
type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	DataType    string `json:"data_type"`
	RepName     string `json:"rep_name"`
	IsAlive     bool   `json:"is_alive"`
}

type GetPluginListResponse struct {
	Messages []*message.GkillMessage `json:"messages"`
	Errors   []*message.GkillError   `json:"errors"`
	Plugins  []PluginInfo            `json:"plugins"`
}
