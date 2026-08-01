package reps

import "time"

// MiReKyou は既存Kyouをタスク化した情報です。
// ReKyou由来のTargetIDと、Mi由来のスケジュール項目を併せ持ちます。
// タイトルは持たず、表示時はTargetIDの指すKyouをそのまま描画します。
type MiReKyou struct {
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

	TargetID string `json:"target_id"`

	IsChecked bool `json:"is_checked"`

	BoardName string `json:"board_name"`

	LimitTime *time.Time `json:"limit_time"`

	EstimateStartTime *time.Time `json:"estimate_start_time"`

	EstimateEndTime *time.Time `json:"estimate_end_time"`
}

// ToMi はMi検索パイプライン(filterMiForMi/sortResultKyous)へ流し込むためにMiへ変換します。
// MiReKyouはタイトルを持たないためTitleは空文字になります。
func (m MiReKyou) ToMi() Mi {
	return Mi{
		IsDeleted:         m.IsDeleted,
		ID:                m.ID,
		RepName:           m.RepName,
		DataType:          m.DataType,
		CreateTime:        m.CreateTime,
		CreateApp:         m.CreateApp,
		CreateDevice:      m.CreateDevice,
		CreateUser:        m.CreateUser,
		UpdateTime:        m.UpdateTime,
		UpdateApp:         m.UpdateApp,
		UpdateUser:        m.UpdateUser,
		UpdateDevice:      m.UpdateDevice,
		Title:             "",
		IsChecked:         m.IsChecked,
		BoardName:         m.BoardName,
		LimitTime:         m.LimitTime,
		EstimateStartTime: m.EstimateStartTime,
		EstimateEndTime:   m.EstimateEndTime,
	}
}
