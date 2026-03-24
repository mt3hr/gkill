package kftl

import (
	"context"
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

// GenerateAndExecuteRequests parses StatementText, generates KFTLRequests,
// and executes each request against the provided repositories.
func (s *KFTLStatement) GenerateAndExecuteRequests(
	ctx context.Context,
	repos *reps.GkillRepositories,
	applicationConfig *user_config.ApplicationConfig,
	userID, device, appName, localeName string,
) error {
	factory := newKFTLFactory()
	factory.reset()

	txID := sqlite3impl.GenerateNewID()
	baseTime := time.Now()

	lines, err := s.generateKFTLLines(factory, txID, baseTime, repos, applicationConfig, userID, device, appName, localeName)
	if err != nil {
		return err
	}

	requestMap := NewKFTLRequestMap()
	for _, line := range lines {
		if err := line.ApplyThisLineToRequestMap(ctx, requestMap); err != nil {
			return fmt.Errorf("error applying line %q: %w", line.GetStatementLineText(), err)
		}
	}

	for _, req := range requestMap.All() {
		if err := req.DoRequest(ctx); err != nil {
			return fmt.Errorf("error executing request id=%s: %w", req.GetRequestID(), err)
		}
	}
	return nil
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
