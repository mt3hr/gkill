// ˅
package reps

import "time"

// ˄

type Mi struct {
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

	IsChecked bool `json:"is_checked"`

	CheckedTime *time.Time `json:"checked_time"`

	BoardName string `json:"board_name"`

	LimitTime *time.Time `json:"limit_time"`

	EstimateStartTime *time.Time `json:"estimate_start_time"`

	EstimateEndTime *time.Time `json:"estimate_end_time"`

	// ˅

	// ˄
}

// ˅

// ˄
