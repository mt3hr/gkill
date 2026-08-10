package find

import "encoding/json"

type MiCheckState string

var (
	All      MiCheckState = "all"
	Checked  MiCheckState = "checked"
	UncCheck MiCheckState = "uncheck"
)

func (m *MiCheckState) UnmarshalJSON(b []byte) error {
	var checkStateStr string
	err := json.Unmarshal(b, &checkStateStr)
	if err != nil {
		return err
	}
	*m = MiCheckState(checkStateStr)
	return nil
}

func (m MiCheckState) MarshalJSON() ([]byte, error) {
	// []byteでMarshalするとBase64文字列になってしまうため、stringのままMarshalする。
	// UnmarshalJSON側は文字列を期待しており、Base64だと往復で壊れる
	return json.Marshal(string(m))
}
