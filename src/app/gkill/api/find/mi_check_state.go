package find

import "encoding/json"

type MiCheckState string

var (
	All      MiCheckState = "all"
	Checked  MiCheckState = "checked"
	UncCheck MiCheckState = "uncheck"
)

func (m *MiCheckState) UnmarshalJSON(b []byte) error {
	var checkStateStr string = ""
	err := json.Unmarshal(b, &checkStateStr)
	if err != nil {
		return err
	}
	*m = MiCheckState(checkStateStr)
	return nil
}

func (m MiCheckState) MarshalJSON() ([]byte, error) {
	var checkStateStr string = string(m)
	return json.Marshal([]byte(checkStateStr))
}
