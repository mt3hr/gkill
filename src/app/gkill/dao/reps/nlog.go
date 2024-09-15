package reps

import "time"

type Nlog struct {
	IsDeleted bool `json:"is_deleted"`

	ID string `json:"id"`

	RepName string `json:"rep_name"`

	RelatedTime time.Time `json:"related_time"`

	DataType string `json:"data_type"`

	CreateTime time.Time `json:"create_time"`

	CreateApp string `json:"create_app"`

	CreateDevice string `json:"create_device"`

	CreateUser string `json:"create_user"`

	UpdateTime time.Time `json:"update_time"`

	UpdateApp string `json:"update_app"`

	UpdateUser string `json:"update_user"`

	UpdateDevice string `json:"update_device"`

	Shop string `json:"shop"`

	Title string `json:"title"`

	Amount int `json:"amount"`
}
