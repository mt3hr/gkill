package reps

// Kyou の JSON エンコードのコスト測定と、**手書き MarshalJSON にしてはいけない**根拠。
//
// 検索応答は Kyou を数十万件並べたものになる。encoding/json の反射エンコーダは
// time.Time の MarshalJSON を1件ごとに3回(RelatedTime / CreateTime / UpdateTime)呼び、
// それぞれが新しいバイトスライスを返すので **1件につき3確保**。
// 確保回数だけ見ると手書きに変える価値がありそうに見えるが、実際は遅くなる。
//
// 実測(20万件, go test -run '^$' -bench BenchmarkEncodeKyousJSON -benchmem, 2026-08-18)
//
//	実装                          ns/op          B/op         allocs/op
//	----------------------------  -------------  -----------  ---------
//	反射エンコーダ(現状)            433,521,767   118,284,266    600,064
//	Kyou に MarshalJSON を手書き    683,057,100   169,512,288    200,020   ← 1.6倍遅い
//
// **なぜ遅くなるか**: encoding/json は MarshalJSON の戻り値をそのまま出力へ流さず、
// `compact()` に通す(JSONとしての妥当性検査とHTMLエスケープを兼ねる)。
// つまり手書きにすると「自分で組み立てる」+「バイト列を全部走査し直す」の二重になる。
// 確保回数は3分の1になるが、走査が1回増えるぶん時間では負ける。
//
// 出力の同一性も曲者だった(手書き版を書いてみて分かったこと):
//   - Go 1.26 の反射エンコーダは 0x08 / 0x0C を \u0008 ではなく \b / \f で出す
//   - U+2028 / U+2029 は **HTMLエスケープの有無にかかわらず**常にエスケープされるが、
//     compact() は escapeHTML のときしかエスケープしない
//   - 不正なUTF-8は U+FFFD の**文字**ではなく6文字のエスケープ表記で出る
//
// ここを速くしたいなら、応答全体を encoding/json を通さずに直接書き出す
// (compact を避ける)しかない。req_res の応答封筒ごと変える話になるので別起案。

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"
)

func BenchmarkEncodeKyousJSON(b *testing.B) {
	const kyouCount = 200_000
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	kyous := make([]Kyou, 0, kyouCount)
	for i := range kyouCount {
		relatedTime := base.Add(time.Duration(i) * time.Minute)
		kyous = append(kyous, Kyou{
			ID:           fmt.Sprintf("kyou-%08d", i),
			RepName:      "rep-1",
			DataType:     "kmemo",
			RelatedTime:  relatedTime,
			CreateTime:   relatedTime,
			UpdateTime:   relatedTime,
			CreateApp:    "gkill",
			CreateDevice: "device",
			CreateUser:   "user",
			UpdateApp:    "gkill",
			UpdateDevice: "device",
			UpdateUser:   "user",
		})
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := json.NewEncoder(io.Discard).Encode(kyous); err != nil {
			b.Fatal(err)
		}
	}
}
