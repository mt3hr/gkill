package kftl

import "strings"

// KFTL line prefix/splitter constants.
// These mirror the i18n locale values used in the TypeScript implementation.
const (
	splitterTag                   = "。"
	splitterStartText             = "ーー"
	splitterRelatedTime           = "？"
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
	splitterRelatedTimeAscii           = "?"
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
func (f *kftlFactory) generateNlogConstructor(nextLineText string, req *kftlNlogRequest) StatementLineConstructorFunc {
	return f.generateDefaultConstructor(nextLineText, func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		f.prevLineIsMetaInfo = false
		return newKFTLNlogTitleStatementLine(lineText, ctx, req)
	})
}

// generateDefaultConstructor checks nextLineText against all known patterns
// and returns the appropriate constructor function.
// If no pattern matches, lastFunc is returned.
// Mirrors: KFTLStatementLineConstructorFactory.generate_default_constructor()
func (f *kftlFactory) generateDefaultConstructor(nextLineText string, lastFunc StatementLineConstructorFunc) StatementLineConstructorFunc {
	switch {
	case strings.HasPrefix(nextLineText, splitterTag) || strings.HasPrefix(nextLineText, splitterTagAscii):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLTagStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
	case nextLineText == splitterStartText || nextLineText == splitterStartTextAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLStartTextStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
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
