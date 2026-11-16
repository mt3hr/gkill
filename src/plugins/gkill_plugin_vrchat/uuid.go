package main

import (
	"crypto/sha1"
	"encoding/hex"
)

// vrchatNamespace は KyouID を導出するための名前空間UUID。
//
// この値は永久に変えないこと。変えると全KyouのIDが作り直され、
// ユーザが付けたタグやテキストの紐付けが全部切れる。
var vrchatNamespace = [16]byte{
	0x6b, 0x1e, 0x3d, 0xa4, 0x52, 0x0f, 0x4c, 0x7e,
	0x91, 0x2a, 0xc8, 0x47, 0x3b, 0xe5, 0x60, 0x9d,
}

// uuidV5 は名前空間と名前からUUIDv5を作る。
//
// google/uuid を直接使わないのは、このプラグインの依存を
// 「gkillのSDKと sqlite だけ」に保つため(uuid は sqlite 経由の間接依存でしかない)。
// gkill_plugin_codex と同じ実装。
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
