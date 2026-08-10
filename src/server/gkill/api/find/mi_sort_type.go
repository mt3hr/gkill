package find

import "encoding/json"

type MiSortType string

const (
	CreateTime        MiSortType = "create_time"
	EstimateStartTime MiSortType = "estimate_start_time"
	EstimateEndTime   MiSortType = "estimate_end_time"
	LimitTime         MiSortType = "limit_time"
)

func (m *MiSortType) UnmarshalJSON(b []byte) error {
	var sortTypeStr string
	err := json.Unmarshal(b, &sortTypeStr)
	if err != nil {
		return err
	}
	*m = MiSortType(sortTypeStr)
	return nil
}

func (m MiSortType) MarshalJSON() ([]byte, error) {
	// []byteでMarshalするとBase64文字列になってしまうため、stringのままMarshalする。
	// UnmarshalJSON側は文字列を期待しており、Base64だと往復で壊れる
	return json.Marshal(string(m))
}
