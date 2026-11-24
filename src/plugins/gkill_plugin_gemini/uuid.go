package main

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
)

// geminiNamespace は KyouID を導出するための名前空間UUID。
//
// この値は永久に変えないこと。変えると全KyouのIDが作り直され、
// ユーザが付けたタグやテキストの紐付けが全部切れる。
var geminiNamespace = [16]byte{
	0x2f, 0x84, 0x1b, 0x6e, 0xc0, 0x37, 0x4d, 0x92,
	0xa5, 0x63, 0x7e, 0x10, 0x9b, 0x44, 0xd8, 0x01,
}

// uuidV5 は名前空間と名前からUUIDv5を作る。
//
// google/uuid を直接使わないのは、このプラグインの依存を
// 「gkillのSDKと sqlite と x/net/html だけ」に保つため(uuid は sqlite 経由の間接依存でしかない)。
// gkill_plugin_vrchat / gkill_plugin_codex と同じ実装。
func uuidV5(namespace [16]byte, name string) string {
	hash := sha1.New()
	_, _ = hash.Write(namespace[:])
	_, _ = hash.Write([]byte(name))
	sum := hash.Sum(nil)

	var uuid [16]byte
	copy(uuid[:], sum[:16])
	uuid[6] = (uuid[6] & 0x0f) | 0x50 // version 5
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant RFC4122

	encoded := make([]byte, 36)
	hex.Encode(encoded[0:8], uuid[0:4])
	encoded[8] = '-'
	hex.Encode(encoded[9:13], uuid[4:6])
	encoded[13] = '-'
	hex.Encode(encoded[14:18], uuid[6:8])
	encoded[18] = '-'
	hex.Encode(encoded[19:23], uuid[8:10])
	encoded[23] = '-'
	hex.Encode(encoded[24:36], uuid[10:16])
	return string(encoded)
}

// kyouIDOf は1発言の Kyou ID を導出する。
//
// マイアクティビティの書き出しには**メッセージ ID が無い**ので、
// 会話 ID・時刻・役割・同じ3つ組の中での連番から決定的に作る。
// 同じ往復が複数の書き出しに入っていても同じ ID になるので、1件に畳まれる。
//
// 連番は「会話 ID と時刻が同じ往復」が複数あるときだけ 0 以外になる（実物ではまず無い）。
// 並べ替えの基準は中身のハッシュなので、書き出しの順に左右されない。
func kyouIDOf(convID string, timeKey string, role string, ordinal int) string {
	return uuidV5(geminiNamespace, convID+"\n"+timeKey+"\n"+role+"\n"+strconv.Itoa(ordinal))
}
