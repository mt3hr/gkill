package req_res

import "github.com/mt3hr/gkill/src/server/gkill/api/message"

type ZipEntry struct {
	Path string `json:"path"`

	IsDir bool `json:"is_dir"`

	Size int64 `json:"size"`

	IsImage bool `json:"is_image"`

	FileURL string `json:"file_url"`
}

type BrowseZipContentsResponse struct {
	Messages []*message.GkillMessage `json:"messages"`

	Errors []*message.GkillError `json:"errors"`

	Entries []*ZipEntry `json:"entries"`
}
