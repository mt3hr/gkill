package kftl

import (
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// StatementLineConstructorFunc is a function that creates a KFTLStatementLine.
type StatementLineConstructorFunc func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine

// KFTLStatementLineContext holds shared state between consecutive statement lines.
// Mirrors: src/classes/kftl/kftl-statement-line-context.ts
type KFTLStatementLineContext struct {
	TXID string

	ThisStatementLineText     string
	ThisStatementLineTargetID string
	ThisIsPrototype           bool

	// LineIndex は元テキストでの0始まりの行位置。エラーへ「何行目か」を添えるためだけに持つ。
	// TS 側には不正行の位置を返す get_invalid_line_indexs があるのに Go 側には無く、
	// 失敗の原因行が呼び出し側へ一切伝わっていなかった（2026-08-24 の再監査）。
	LineIndex int

	NextStatementLineText     string
	NextStatementLineTargetID *string // nil means generate a new UUID for the next line
	NextIsPrototype           bool
	// NextStatementLineConstructor overrides the factory for the next line.
	// nil means use the factory's generateKmemoConstructor.
	NextStatementLineConstructor StatementLineConstructorFunc

	KFTLStatementLines []KFTLStatementLine
	AddSecond          int

	// factory is used by statement lines that need to call generateKmemoConstructor etc.
	factory *kftlFactory

	// BaseTime is the time when GenerateAndExecuteRequests was called.
	// Used as the basis for create_time and related_time (+ add_second offset).
	BaseTime time.Time

	// Go-side: repositories and config (TS version has GkillAPI)
	Repositories      *reps.GkillRepositories
	UserID            string
	Device            string
	ApplicationName   string
	LocaleName        string
	ApplicationConfig *user_config.ApplicationConfig
}

// GetPrevLine returns the last line in KFTLStatementLines, or nil if empty.
// Mirrors: KFTLStatementLine.get_prev_line() (requires length >= 1)
func (c *KFTLStatementLineContext) GetPrevLine() KFTLStatementLine {
	if len(c.KFTLStatementLines) >= 1 {
		return c.KFTLStatementLines[len(c.KFTLStatementLines)-1]
	}
	return nil
}

// nowFromCtx returns the request create time: BaseTime + AddSecond.
func nowFromCtx(ctx *KFTLStatementLineContext) time.Time {
	return ctx.BaseTime.Add(time.Duration(ctx.AddSecond) * time.Second)
}
