package reps

import "time"

// NormalizeGPSLogPeriod は GPSLogRepository.GetGPSLogs の期間指定を正規化します。
//
// 契約はインターフェースのdocコメントどおりで、
//   - 両方nilならそのまま (nil, nil) を返します（全件の意味）
//   - 片方だけnilなら残る一方と同じ時刻にします
//   - 逆順で渡されたら入れ替えます
//
// 返る値は引数とは別の実体なので、呼び出し元のポインタは書き換わりません。
// GPXディレクトリ実装・プラグインアダプタ・プラグイン本体の3箇所で
// 同じ判定を書き写してドリフトさせないために切り出してあります。
func NormalizeGPSLogPeriod(startTime *time.Time, endTime *time.Time) (*time.Time, *time.Time) {
	if startTime == nil && endTime == nil {
		return nil, nil
	}
	if startTime == nil {
		startTime = endTime
	}
	if endTime == nil {
		endTime = startTime
	}
	start, end := *startTime, *endTime
	if start.After(end) {
		start, end = end, start
	}
	return &start, &end
}
