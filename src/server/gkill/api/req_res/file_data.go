package req_res

import "time"

type FileData struct {
	FileName string `json:"file_name"`

	DataBase64 string `json:"data_base64"`

	LastModified time.Time `json:"last_modified"`
}
