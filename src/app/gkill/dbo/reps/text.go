// ˅
package reps

import "time"

// ˄

type Text struct {
	// ˅

	// ˄

	IsDeleted bool `json:"is_deleted"`

	ID string `json:"id"`

	TargetID string `json:"target_id"`

	RelatedTime time.Time `json:"related_time"`

	CreateTime time.Time `json:"create_time"`

	CreateApp string `json:"create_app"`

	CreateDevice string `json:"create_device"`

	CreateUser string `json:"create_user"`

	UpdateTime time.Time `json:"update_time"`

	UpdateApp string `json:"update_app"`

	UpdateDevice string `json:"update_device"`

	UpdateUser string `json:"update_user"`

	Text string `json:"text"`

	// ˅

	// ˄
}

// ˅

// ˄
