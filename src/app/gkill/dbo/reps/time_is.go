// ˅
package reps

import "time"

// ˄

type TimeIs struct {
	// ˅

	// ˄

	IsDeleted bool `json:"is_deleted"`

	ID string `json:"id"`

	RepName string `json:"rep_name"`

	DataType string `json:"data_type"`

	CreateTime time.Time `json:"create_time"`

	CreateApp string `json:"create_app"`

	CreateDevice string `json:"create_device"`

	CreateUser string `json:"create_user"`

	UpdateTime time.Time `json:"update_time"`

	UpdateApp string `json:"update_app"`

	UpdateUser string `json:"update_user"`

	UpdateDevice string `json:"update_device"`

	Title string `json:"title"`

	StartTime time.Time `json:"start_time"`

	EndTime *time.Time `json:"end_time"`

	// ˅

	// ˄
}

// ˅

// ˄
