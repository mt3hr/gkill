package kftl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// KFTLStatement is the entry point for parsing and executing KFTL text.
// Mirrors: src/classes/kftl/kftl-statement.ts
type KFTLStatement struct {
	StatementText string
}

// KFTLInputError は「利用者の書いたテキストが悪い」失敗を表す。
//
// 2026-08-24 まで、気分値が範囲外という打ち間違いも、DBの書き込み失敗も、
// 同じ ERR000351(HTTP 500) + 定型文1本に畳まれていた。応答からは
// 何行目の何が悪いのか一切分からず、しかも利用者の入力ミスがサーバ障害と
// 同じステータスで返っていた。サーバ障害と区別して 400 を返し、
// 行番号と原因を応答へ載せるためにこの型が要る。
type KFTLInputError struct {
	// LineNumber は1始まりの行番号。0 は「まだ行が分からない」。
	// 行を知っているのは実行ループの側なので、生成時点では 0 のことがある。
	LineNumber int
	LineText   string
	// MessageID は利用者向けの多言語メッセージID。空なら Cause の文面を使う。
	MessageID string
	Cause     error
}

func (e *KFTLInputError) Error() string {
	if e.LineNumber > 0 {
		return fmt.Sprintf("kftl input error at line %d %q: %v", e.LineNumber, e.LineText, e.Cause)
	}
	return fmt.Sprintf("kftl input error: %v", e.Cause)
}

func (e *KFTLInputError) Unwrap() error { return e.Cause }

// newKFTLInputError は多言語メッセージIDつきの入力エラーを作る。
func newKFTLInputError(messageID string, cause error) *KFTLInputError {
	return &KFTLInputError{MessageID: messageID, Cause: cause}
}

// withLine は行番号と行テキストを添えた入力エラーにする。
// 既に入力エラーなら行の情報だけを埋め、そうでなければ包む。
func withLine(err error, lineNumber int, lineText string) *KFTLInputError {
	var inputErr *KFTLInputError
	if errors.As(err, &inputErr) {
		return &KFTLInputError{
			LineNumber: lineNumber,
			LineText:   lineText,
			MessageID:  inputErr.MessageID,
			Cause:      inputErr.Cause,
		}
	}
	return &KFTLInputError{LineNumber: lineNumber, LineText: lineText, Cause: err}
}

// CollectKFTLInputErrors は errors.Join で束ねた入力エラーを平らに取り出す。
// 1件も無ければ nil を返す（＝サーバ障害として扱ってよい）。
//
// 束ねたエラーへ errors.As を直に当てると最初の1件しか拾えないので、
// 先に Unwrap() []error を辿ってから判定する。
func CollectKFTLInputErrors(err error) []*KFTLInputError {
	var collected []*KFTLInputError
	var walk func(error)
	walk = func(e error) {
		if e == nil {
			return
		}
		// 自分自身が入力エラーならそこで確定。errors.As を使うと連鎖の先まで
		// 潜ってしまい、包んだ側と包まれた側を二重に拾う。
		if inputErr, ok := e.(*KFTLInputError); ok {
			collected = append(collected, inputErr)
			return
		}
		// errors.Join で束ねたもの。
		if joined, ok := e.(interface{ Unwrap() []error }); ok {
			for _, child := range joined.Unwrap() {
				walk(child)
			}
			return
		}
		// fmt.Errorf("%w") で1段包んだもの。**ここを辿らないと束が見えない。**
		// 呼び出し側（ハンドラ）は文脈を足すために束ねたエラーをもう一度包むので、
		// 単段の Unwrap を辿らないと errors.As が最初の1件しか拾わず、
		// 「全行ぶん集める」が黙って1件に戻る（2026-08-24 のデプロイ後の実測で発覚）。
		if wrapped, ok := e.(interface{ Unwrap() error }); ok {
			walk(wrapped.Unwrap())
		}
	}
	walk(err)
	return collected
}

// GenerateAndExecuteRequests parses StatementText, generates KFTLRequests,
// and executes each request against the provided repositories.
func (s *KFTLStatement) GenerateAndExecuteRequests(
	ctx context.Context,
	repos *reps.GkillRepositories,
	applicationConfig *user_config.ApplicationConfig,
	userID, device, appName, localeName string,
) ([]KFTLCreatedRecord, error) {
	factory := newKFTLFactory()
	factory.reset()

	txID := sqlite3impl.GenerateNewID()
	baseTime := time.Now()

	lines, err := s.generateKFTLLines(factory, txID, baseTime, repos, applicationConfig, userID, device, appName, localeName)
	if err != nil {
		return nil, err
	}

	requestMap := NewKFTLRequestMap()
	// このフェーズはまだ1バイトも書いていないので、最初の1件で止めずに全行を評価する。
	// 利用者は1往復で全部直せる（TS 側は元から複数集める設計で、Go だけが遅れていた）。
	var inputErrs []error
	for i, line := range lines {
		if err := line.ApplyThisLineToRequestMap(ctx, requestMap); err != nil {
			inputErrs = append(inputErrs, withLine(err, i+1, line.GetStatementLineText()))
		}
	}
	if len(inputErrs) != 0 {
		return nil, errors.Join(inputErrs...)
	}

	// ここから先は書き込みが起きる。1件でも失敗したら止めるが、
	// **既に書けたぶんはロールバックされない**（commit_tx はDBトランザクションではない）。
	// 途中で失敗しても、そこまでに書けたぶんは呼び出し側へ返す。
	// 残ってしまったものが分からないと利用者は後始末ができない。
	var created []KFTLCreatedRecord
	for _, req := range requestMap.All() {
		err := req.DoRequest(ctx)
		created = append(created, req.GetCreatedRecords()...)
		if err != nil {
			lineNumber, lineText := 0, ""
			if lineCtx := req.GetContext(); lineCtx != nil {
				lineNumber, lineText = lineCtx.LineIndex+1, lineCtx.ThisStatementLineText
			}
			var inputErr *KFTLInputError
			if errors.As(err, &inputErr) {
				return created, withLine(err, lineNumber, lineText)
			}
			return created, fmt.Errorf("error executing request id=%s (line %d %q): %w", req.GetRequestID(), lineNumber, lineText, err)
		}
	}
	return created, nil
}

// generateKFTLLines splits the statement text into lines and constructs
// the corresponding KFTLStatementLine objects.
// Mirrors: KFTLStatement.generate_kftl_lines() in TS.
func (s *KFTLStatement) generateKFTLLines(
	factory *kftlFactory,
	txID string,
	baseTime time.Time,
	repos *reps.GkillRepositories,
	applicationConfig *user_config.ApplicationConfig,
	userID, device, appName, localeName string,
) ([]KFTLStatementLine, error) {
	lineTexts := strings.Split(s.StatementText, "\n")
	var lines []KFTLStatementLine
	var prevCtx *KFTLStatementLineContext
	prevAddSecond := 0

	for i, lineText := range lineTexts {
		nextLineText := ""
		if i < len(lineTexts)-1 {
			nextLineText = lineTexts[i+1]
		}

		// Determine target ID for this line
		var targetID string
		if prevCtx != nil && prevCtx.NextStatementLineTargetID != nil {
			targetID = *prevCtx.NextStatementLineTargetID
		} else {
			targetID = sqlite3impl.GenerateNewID()
		}

		// Prototype flag mirrors TS:
		// prototype_flag = (prev_context != null && prev_context.is_this_prototype() != null) ? prev_context.is_next_prototype() : true
		prototypeFlag := true
		if prevCtx != nil {
			prototypeFlag = prevCtx.NextIsPrototype
		}

		lineCtx := &KFTLStatementLineContext{
			TXID:                      txID,
			LineIndex:                 i,
			ThisStatementLineText:     lineText,
			ThisStatementLineTargetID: targetID,
			ThisIsPrototype:           prototypeFlag,
			NextStatementLineText:     nextLineText,
			NextIsPrototype:           false,
			KFTLStatementLines:        lines, // current slice (read-only from line constructors)
			AddSecond:                 prevAddSecond,
			factory:                   factory,
			BaseTime:                  baseTime,
			Repositories:              repos,
			UserID:                    userID,
			Device:                    device,
			ApplicationName:           appName,
			LocaleName:                localeName,
			ApplicationConfig:         applicationConfig,
		}

		// Determine line constructor: use prev line's NextStatementLineConstructor if set,
		// otherwise fall back to factory's generateKmemoConstructor.
		// Mirrors: KFTLStatement.generate_kftl_line()
		var line KFTLStatementLine
		if prevCtx != nil && prevCtx.NextStatementLineConstructor != nil {
			line = prevCtx.NextStatementLineConstructor(lineText, lineCtx)
		} else {
			line = factory.generateKmemoConstructor(lineText)(lineText, lineCtx)
		}

		// Track add_second increments from SplitAndNextSecond lines
		if _, ok := line.(*kftlSplitAndNextSecondStatementLine); ok {
			prevAddSecond++
		}

		prevCtx = lineCtx

		// Stop at save character (except on first line)
		// Mirrors: if (i != 0 && line_text == KFTL_SAVE_CHARACTOR) break
		if i != 0 && (lineText == splitterSaveCharacter || lineText == splitterSaveCharacterAscii) {
			break
		}

		lines = append(lines, line)
	}

	return lines, nil
}
