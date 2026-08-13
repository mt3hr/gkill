// 集計結果の数値を表示用文字列にする。
// 平均は割り算なので素の toString() だと 71.28604651162791 のような桁で出るし、
// 浮動小数の合計も 12345.700000000001 になる。小数2桁で丸めて、整数なら小数点を付けない。
//
// 推移グラフの数値は dnote-trend/aggregated-value-to-number.ts が別経路で作っており
// ここを通らないので、丸めてもグラフの形は変わらない。
export default function format_aggregated_number(value: number): string {
    if (!Number.isFinite(value)) {
        return "0"
    }
    return (Math.round(value * 100) / 100).toString()
}
