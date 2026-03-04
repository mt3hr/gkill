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
// Mirrors: KFTLStatementLineConstructorFactory.generate_nlog_constructor()
func (f *kftlFactory) generateNlogConstructor(req *kftlNlogRequest) StatementLineConstructorFunc {
	return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		f.prevLineIsMetaInfo = false
		return newKFTLNlogTitleStatementLine(lineText, ctx, req)
	}
}

// generateDefaultConstructor checks nextLineText against all known patterns
// and returns the appropriate constructor function.
// If no pattern matches, lastFunc is returned.
// Mirrors: KFTLStatementLineConstructorFactory.generate_default_constructor()
func (f *kftlFactory) generateDefaultConstructor(nextLineText string, lastFunc StatementLineConstructorFunc) StatementLineConstructorFunc {
	switch {
	case strings.HasPrefix(nextLineText, splitterTag):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLTagStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
	case nextLineText == splitterStartText:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLStartTextStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
	case strings.HasPrefix(nextLineText, splitterRelatedTime):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLRelatedTimeStatementLine(lineText, ctx, f.prevLineIsMetaInfo)
		}
	case nextLineText == splitterSplitNextSecond:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = true
			return newKFTLSplitAndNextSecondStatementLine(lineText, ctx)
		}
	case nextLineText == splitterSplit:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = true
			return newKFTLSplitStatementLine(lineText, ctx)
		}
	case nextLineText == splitterKC:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartKCStatementLine(lineText, ctx)
		}
	case nextLineText == splitterMi:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartMiStatementLine(lineText, ctx)
		}
	case nextLineText == splitterLantana:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartLantanaStatementLine(lineText, ctx)
		}
	case nextLineText == splitterNlog:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartNlogStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsStart:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsStartStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEnd:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIs:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEndIfExist:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndIfExistStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEndByTag:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndByTagStatementLine(lineText, ctx)
		}
	case nextLineText == splitterTimeIsEndByTagIfExist:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartTimeIsEndByTagIfExistStatementLine(lineText, ctx)
		}
	case nextLineText == splitterURLog:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			f.prevLineIsMetaInfo = false
			return newKFTLStartURLogStatementLine(lineText, ctx)
		}
	default:
		return lastFunc
	}
}
