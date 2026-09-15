package message

import "encoding/json"

// MessageLevelInfo / MessageLevelWarning は GkillMessage.Level の値。
//
// info は「成功した・完了した」の知らせで、Web は数秒で消す。
// warning は「成功はしたが対処が要る」（読み込めなかった rep がある、プラグインの検索が
// 落ちている）で、Web は閉じるまで残す。errors に載せると検索結果ごと捨てられるので、
// 対処が要っても messages に載せるしかなく、その区別のために level がある。
const (
	MessageLevelInfo    = "info"
	MessageLevelWarning = "warning"
)

// GkillMessage はAPIレスポンスの messages 配列の1要素。
type GkillMessage struct {
	MessageCode string `json:"message_code"`

	Message string `json:"message"`

	// Level は MessageLevelInfo（既定。空でも info として出る）か MessageLevelWarning。
	Level string `json:"level"`
}

// MarshalJSON は Level が空のとき "info" を補います。
// 消費者に「空なら info」の規則を配らないため。
func (m GkillMessage) MarshalJSON() ([]byte, error) {
	type gkillMessageJSON GkillMessage
	out := gkillMessageJSON(m)
	if out.Level == "" {
		out.Level = MessageLevelInfo
	}
	return json.Marshal(out)
}

// GkillMessages はレスポンス構造体の Messages フィールドの型。nil でも JSON では `[]` になる（GkillErrors と同じ理由）。
type GkillMessages []*GkillMessage

// MarshalJSON は nil を `[]` として出します。
func (msgs GkillMessages) MarshalJSON() ([]byte, error) {
	if msgs == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]*GkillMessage(msgs))
}
