// Package kftl implements the KFTL text format parser (Go port of the TypeScript implementation in src/client/classes/kftl).
package kftl

import "strings"

// KFTL line prefix/splitter constants.
// These mirror the i18n locale values used in the TypeScript implementation.
const (
	splitterTag                   = "。"
	splitterStartText             = "ーー"
	splitterMiReKyou              = "～～"
	splitterRelatedTime           = "？"
	splitterRepeat                = "？？"
	splitterSplit                 = "、"
	splitterSplitNextSecond       = "、、"
	splitterKC                    = "ーか"
	splitterMi                    = "ーみ"
	splitterLantana               = "ーら"
	splitterNlog                  = "ーん"
	splitterTimeIsStart           = "ーた"
	splitterTimeIsEnd             = "ーえ"
	splitterTimeIs                = "ーち"
	splitterTimeIsEndIfExist      = "ーいえ"
	splitterTimeIsEndByTag        = "ーたえ"
	splitterTimeIsEndByTagIfExist = "ーいたえ"
	splitterURLog                 = "ーう"
	splitterSaveCharacter         = "！"
)

// ASCII alternatives for non-Japanese locales.
const (
	splitterTagAscii                   = "#"
	splitterStartTextAscii             = "--"
	splitterMiReKyouAscii              = "~~"
	splitterRelatedTimeAscii           = "?"
	splitterRepeatAscii                = "??"
	splitterSplitAscii                 = ","
	splitterSplitNextSecondAscii       = ",,"
	splitterKCAscii                    = "/num"
	splitterMiAscii                    = "/mi"
	splitterLantanaAscii               = "/mood"
	splitterNlogAscii                  = "/expense"
	splitterTimeIsStartAscii           = "/start"
	splitterTimeIsEndAscii             = "/end"
	splitterTimeIsAscii                = "/timeis"
	splitterTimeIsEndIfExistAscii      = "/end?"
	splitterTimeIsEndByTagAscii        = "/endt"
	splitterTimeIsEndByTagIfExistAscii = "/endt?"
	splitterURLogAscii                 = "/url"
	splitterSaveCharacterAscii         = "!"
)

// argTakingPrefixes は「行に単独で書き、値は次の行に書く」プレフィックスの一覧。
//
// タグ(。/#)と関連時刻(？/?)は**前方一致**で受理する設計なので、ここには入れない
// （入れると「# 見出し」や「? を含む英文」が行エラーになり、既存の書き方が壊れる）。
// 区切り(、/,)と打ち切り(！/!)も引数を取らないので対象外。
var argTakingPrefixes = []string{
	splitterKC, splitterMi, splitterLantana, splitterNlog,
	splitterTimeIsStart, splitterTimeIsEnd, splitterTimeIs,
	splitterTimeIsEndIfExist, splitterTimeIsEndByTag, splitterTimeIsEndByTagIfExist,
	splitterURLog, splitterStartText, splitterMiReKyou, splitterRepeat,
	splitterKCAscii, splitterMiAscii, splitterLantanaAscii, splitterNlogAscii,
	splitterTimeIsStartAscii, splitterTimeIsEndAscii, splitterTimeIsAscii,
	splitterTimeIsEndIfExistAscii, splitterTimeIsEndByTagAscii, splitterTimeIsEndByTagIfExistAscii,
	splitterURLogAscii, splitterStartTextAscii, splitterMiReKyouAscii, splitterRepeatAscii,
}

// prefixWrittenWithArgument は「既知のプレフィックス＋同じ行に引数」を書いた行かを返す。
// 該当すればそのプレフィックスを返す。
//
// プレフィックスの判定は完全一致（generateDefaultConstructor）なので、
// `/mood 8` は「/mood ではない行」として本文へ落ち、**気分記録のつもりが
// 本文「/mood 8」のメモ1件になる**。エラーも警告も出ないので気づけない
// （2026-08-24 の実利用報告）。書き込みの前にここで捕まえる。
//
// 長いプレフィックスから先に見る。`/end?` は `/end` で始まるので、
// 短い側から見ると `/end?` が「/end に引数 ? を付けた行」に化ける。
func prefixWrittenWithArgument(lineText string) (string, bool) {
	trimmed := normalizeWaveDash(strings.TrimRight(lineText, " \t"))
	longest := ""
	for _, prefix := range argTakingPrefixes {
		if trimmed == prefix {
			// 単独で書かれている＝正しい書き方。ここでは咎めない
			// （値の行が無いことは requireNextLineText が別に見る）。
			return "", false
		}
		if len(prefix) <= len(longest) {
			continue
		}
		rest, found := strings.CutPrefix(trimmed, prefix)
		if !found || rest == "" {
			continue
		}
		// 直後が空白のときだけ「引数を同じ行に書いた」とみなす。
		// 空白が無ければ別の語（例: `/mine`）なので本文のまま通す。
		if strings.TrimLeft(rest, " \t") != rest {
			longest = prefix
		}
	}
	if longest == "" {
		return "", false
	}
	return longest, true
}

// normalizeWaveDash rewrites the wave dash (U+301C) to the fullwidth tilde (U+FF5E).
//
// 「～」はWindowsのIMEがU+FF5E、macOS/iOSのIMEがU+301Cを出す。見た目が同じで
// 打った端末によって別の文字になるので、揃えずに比較するとiOSからだけ
// リポストタスクの記法が効かない。定数側はU+FF5E。
// Mirrors: normalize_wave_dash in src/client/classes/kftl/kftl-prefixes.ts
func normalizeWaveDash(lineText string) string {
	return strings.ReplaceAll(lineText, "〜", "～")
}

// isMiReKyouSplitter reports whether the line opens or closes a MiReKyou block.
func isMiReKyouSplitter(lineText string) bool {
	normalized := normalizeWaveDash(lineText)
	return normalized == splitterMiReKyou || normalized == splitterMiReKyouAscii
}

// isRepeatSplitter reports whether the line opens or closes a repeat block.
// 完全一致でしか受けない。「？？金3」のように同じ行へ引数を書いたものは
// argTakingPrefixes 側で行エラーになる。
func isRepeatSplitter(lineText string) bool {
	return lineText == splitterRepeat || lineText == splitterRepeatAscii
}

// isRepeatWrittenWithArgument は「？？ 金 3」のように繰り返しの記号と同じ行へ引数を書いた行か。
// 規則そのものは prefixWrittenWithArgument が持つ（直後が空白のときだけ）。
// クライアント側 is_repeat_prefix_with_argument と対。
func isRepeatWrittenWithArgument(lineText string) bool {
	prefix, ok := prefixWrittenWithArgument(lineText)
	return ok && (prefix == splitterRepeat || prefix == splitterRepeatAscii)
}

// kftlFactory tracks the prev_line_is_meta_info state across lines.
// Mirrors: KFTLStatementLineConstructorFactory in TS (singleton with state).
// In Go we use a per-statement instance to avoid global state.
type kftlFactory struct {
	prevLineIsMetaInfo bool
}

func newKFTLFactory() *kftlFactory {
	return &kftlFactory{}
}

// reset initialises the factory for a new statement.
// Mirrors: KFTLStatementLineConstructorFactory.reset()
func (f *kftlFactory) reset() {
	f.prevLineIsMetaInfo = true
}

// generateKmemoConstructor returns a constructor that produces a Kmemo line
// (or a meta-line if nextLineText matches a pattern).
// Mirrors: KFTLStatementLineConstructorFactory.generate_kmemo_constructor()
func (f *kftlFactory) generateKmemoConstructor(nextLineText string) StatementLineConstructorFunc {
	return f.generateDefaultConstructor(nextLineText, func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		f.prevLineIsMetaInfo = false
		return newKFTLKmemoStatementLine(lineText, ctx)
	})
}

// generateNoneConstructor returns a constructor that produces a None line
// (or a meta-line if nextLineText matches a pattern).
// Mirrors: KFTLStatementLineConstructorFactory.generate_none_constructor()
func (f *kftlFactory) generateNoneConstructor(nextLineText string) StatementLineConstructorFunc {
	return f.generateDefaultConstructor(nextLineText, func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		f.prevLineIsMetaInfo = true
		return newKFTLNoneStatementLine(lineText, ctx)
	})
}

// generateNlogConstructor returns a constructor for Nlog title continuation.
// It delegates to generateDefaultConstructor so that separators and other
// prefixes are recognised; only when none match does it fall through to
// creating another nlog title line.
// Mirrors: KFTLStatementLineConstructorFactory.generate_nlog_constructor()
func (f *kftlFactory) generateNlogConstructor(nextLineText string, block *kftlNlogBlock) StatementLineConstructorFunc {
	return f.generateDefaultConstructor(nextLineText, func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		f.prevLineIsMetaInfo = false
		return newKFTLNlogTitleStatementLine(lineText, ctx, block)
	})
}

// resumeConstructorFunc はブロックの中に書かれたメタ情報行(タグ・テキスト)から
// ブロックへ復帰するための「次の行」の決め方。
//
// 素のメタ情報行は次の行を Kmemo か None にするので、ブロックの途中に書くと
// そこでブロックが切れる。ブロック側がこれを渡すと、メタ情報行のあとも
// ブロックの中に留まれる。nil なら今までどおり Kmemo / None へ抜ける。
// Mirrors: KFTLBlockReentryProvider (kftl-statement-line.ts)
type resumeConstructorFunc func(nextLineText string) StatementLineConstructorFunc

// afterMetaInfoConstructor はメタ情報行の「次の行」の決め方を1箇所に集約する。
// Mirrors: kftl-tag-statement-line.ts / kftl-end-text-statement-line.ts の分岐
func afterMetaInfoConstructor(f *kftlFactory, nextLineText string, prevLineIsMetaInfo bool, resume resumeConstructorFunc) StatementLineConstructorFunc {
	if resume != nil {
		return resume(nextLineText)
	}
	if prevLineIsMetaInfo {
		return f.generateKmemoConstructor(nextLineText)
	}
	return f.generateNoneConstructor(nextLineText)
}

// generateDefaultConstructor checks nextLineText against all known patterns
// and returns the appropriate constructor function.
// If no pattern matches, lastFunc is returned.
// Mirrors: KFTLStatementLineConstructorFactory.generate_default_constructor()
func (f *kftlFactory) generateDefaultConstructor(nextLineText string, lastFunc StatementLineConstructorFunc) StatementLineConstructorFunc {
	switch {
	case strings.HasPrefix(nextLineText, splitterTag) || strings.HasPrefix(nextLineText, splitterTagAscii):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLTagStatementLine(lineText, ctx, f.prevLineIsMetaInfo, nil)
		}
	case nextLineText == splitterStartText || nextLineText == splitterStartTextAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLStartTextStatementLine(lineText, ctx, f.prevLineIsMetaInfo, nil)
		}
	case isMiReKyouSplitter(nextLineText):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			// リポストタスクはKyou本体ではなく付随情報なので、タグやテキストと同じく
			// prevLineIsMetaInfo は書き換えずに渡すだけにする
			return newKFTLStartMiReKyouStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
	case isRepeatSplitter(nextLineText):
		// **「？」の前方一致より前に置くこと。** 後ろだと「？？」が関連時刻行に食われて
		// ここへ永久に到達しない（「？」を1つ剥がした残りがパースに失敗するだけになる）
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLStartRepeatStatementLine(lineText, ctx, f.prevLineIsMetaInfo, nil, nil)
		}
	case isRepeatWrittenWithArgument(nextLineText):
		// 「？？ 金 3」。既定の行（Kmemo）へ落として prefixWrittenWithArgument に拾わせる ――
		// 「？」へ流すと「関連時刻がパースできません」という無関係な文言になる。
		//
		// **カーブアウトはここまで。** 「??なんだこれ」のように直後が空白でない行は
		// 従来どおり下の「？」の前方一致へ流す（`？` 始まりの行は日時として解釈する既存仕様）
		return lastFunc
	case strings.HasPrefix(nextLineText, splitterRelatedTime) || strings.HasPrefix(nextLineText, splitterRelatedTimeAscii):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLRelatedTimeStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
	case nextLineText == splitterSplitNextSecond || nextLineText == splitterSplitNextSecondAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = true
			return newKFTLSplitAndNextSecondStatementLine(lineText, ctx)
		}
	case nextLineText == splitterSplit || nextLineText == splitterSplitAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = true
			return newKFTLSplitStatementLine(lineText, ctx)
		}
	case nextLineText == splitterKC || nextLineText == splitterKCAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartKCStatementLine(lineText, ctx)
		}
	case nextLineText == splitterMi || nextLineText == splitterMiAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartMiStatementLine(lineText, ctx)
		}
	case nextLineText == splitterLantana || nextLineText == splitterLantanaAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartLantanaStatementLine(lineText, ctx)
		}
	case nextLineText == splitterNlog || nextLineText == splitterNlogAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartNlogStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEndByTagIfExist || nextLineText == splitterTimeIsEndByTagIfExistAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndByTagIfExistStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEndByTag || nextLineText == splitterTimeIsEndByTagAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndByTagStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEndIfExist || nextLineText == splitterTimeIsEndIfExistAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndIfExistStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsStart || nextLineText == splitterTimeIsStartAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsStartStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEnd || nextLineText == splitterTimeIsEndAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIs || nextLineText == splitterTimeIsAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsStatementLine(lineText, ctx)
		}
	case nextLineText == splitterURLog || nextLineText == splitterURLogAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartURLogStatementLine(lineText, ctx)
		}
	default:
		return lastFunc
	}
}
