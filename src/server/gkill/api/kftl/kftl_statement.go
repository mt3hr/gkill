package kftl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// KFTLStatement is the entry point for parsing and executing KFTL text.
// Mirrors: src/classes/kftl/kftl-statement.ts
type KFTLStatement struct {
	StatementText string
	// FindKyous は打刻終了の対象検索に使う Kyou 検索（任意）。ハンドラが api.FindFilter を閉包で渡す。
	// 設定の playing 検索条件のタグ・非表示タグは Kyou 検索の層（find_filter.go）でしか効かないので、
	// これが無いと `/end` は語の条件だけで対象を探す。
	FindKyous FindKyousFunc
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

// requireNextLineText は「プレフィックス行の次に値の行が要る」文の共通検査。
//
// 値の行が無いまま終わったときの結末が型ごとにばらばらだった。
// `/mood` は**気分値0(最低)のLantanaを黙って書き**、`/num` は空のKCを黙って書き、
// `/expense` `/url` `/mi` `/start` `/timeis` は無言で0件だった。
// どちらも利用者からは「なぜそうなったか」が分からない（2026-08-24 の実利用報告）。
// 書き込みが起きる前のこのフェーズで行別エラーへ倒す（ADR-0502）。
//
// 「次の行」に保存マーカー「！」は含まれない。generateKFTLLines がマーカー行で本文を
// 切り詰めてから NextStatementLineText を組み立てるので、`ーち` の直後が「！」なら
// ここは "" を見る。2026-09-15 まではマーカー行がそのまま「次の行」に入り、Web の
// 「！」で保存する経路だけがこの検査を素通りしてタイトル空のまま DoRequest に届いていた
// （`ーら`+「！」は気分値0を書き、`ーか`+「！」は 500 になっていた。ADR-0508）。
func requireNextLineText(ctx *KFTLStatementLineContext) error {
	if strings.TrimSpace(ctx.NextStatementLineText) != "" {
		return nil
	}
	return newKFTLInputError("KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE",
		fmt.Errorf("prefix %q needs its value on the next line", ctx.ThisStatementLineText))
}

// KFTLExecutionError は実行フェーズ(実際に書きにいく段)で起きた失敗に行番号を添える。
//
// 入力ミスではないので KFTLInputError とは別物で、ハンドラは 500 のまま扱う。
// それでも行番号だけは載せる —— 「メモ帳のテキストの記録に失敗しました」の1文では、
// 何行目で止まったのか(＝ created[] のどこまでが書けたのか)が応答から分からず、
// 後始末の手がかりが無かった(実利用レビュー)。
type KFTLExecutionError struct {
	// LineNumber は1始まりの行番号。0 は「行が分からない」。
	LineNumber int
	LineText   string
	RequestID  string
	Cause      error
}

func (e *KFTLExecutionError) Error() string {
	return fmt.Sprintf("error executing request id=%s (line %d %q): %v", e.RequestID, e.LineNumber, e.LineText, e.Cause)
}

func (e *KFTLExecutionError) Unwrap() error { return e.Cause }

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

// KFTLAnalysis は Analyze の結果。書き込みは一切していない。
type KFTLAnalysis struct {
	// InputErrors は行別の入力エラー（1件も無ければ空）。
	InputErrors []*KFTLInputError
	// Tags は送信すると付くタグ名（重複なし・出現順）。未知タグの確認に使う。
	Tags []string
	// MiBoardNames は Mi / MiReKyou に書かれた板名（空欄は含めない・重複なし・出現順）。
	// 既定板への解決はしない —— 確認ダイアログは利用者が書いたとおりの名前で聞く。
	MiBoardNames []string
	// RecordCount は繰り返しを展開したあとの、書き込みの候補になるリクエスト数。
	RecordCount int
}

// miBoardNameProvider は板名を持つリクエスト（Mi / MiReKyou）が実装する。
// KFTLRequest インタフェースには足さない —— 板名の無い型に空実装を撒くと、
// 型を足したときに「板名を返し忘れた」がコンパイルエラーにならず、確認が黙って抜ける。
type miBoardNameProvider interface {
	MiBoardName() string
}

// Analyze は StatementText を解釈して、書き込みをせずに結果だけ返す。
//
// Web のメモ帳が「おかしな行」のピンク表示・未知タグ／未知板名の確認に使う（ADR-0507）。
// **GenerateAndExecuteRequests と同じ prepareRequests を通す**ので、ここで通った入力が
// 送信で弾かれることも、その逆も起きない。DB は読まない（repos は nil。繰り返しの既存判定は
// repositoriesOf が nil を「既存なし」と扱う）ので、打鍵のたびに呼ばれても軽い。
func (s *KFTLStatement) Analyze(
	ctx context.Context,
	applicationConfig *user_config.ApplicationConfig,
	userID, device, localeName string,
) (*KFTLAnalysis, error) {
	analysis := &KFTLAnalysis{}
	requestMap, err := s.prepareRequests(ctx, nil, applicationConfig, userID, device, "", localeName, sqlite3impl.GenerateNewID(), time.Now())
	if err != nil {
		inputErrors := CollectKFTLInputErrors(err)
		if len(inputErrors) == 0 {
			return nil, err
		}
		analysis.InputErrors = inputErrors
		return analysis, nil
	}
	seenTags := map[string]struct{}{}
	seenBoards := map[string]struct{}{}
	for _, req := range requestMap.All() {
		analysis.RecordCount++
		for _, tag := range req.GetTags() {
			if _, ok := seenTags[tag]; ok {
				continue
			}
			seenTags[tag] = struct{}{}
			analysis.Tags = append(analysis.Tags, tag)
		}
		if provider, ok := req.(miBoardNameProvider); ok {
			boardName := provider.MiBoardName()
			if boardName == "" {
				continue
			}
			if _, ok := seenBoards[boardName]; ok {
				continue
			}
			seenBoards[boardName] = struct{}{}
			analysis.MiBoardNames = append(analysis.MiBoardNames, boardName)
		}
	}
	return analysis, nil
}

// prepareRequests は「行の解釈 → 行をリクエストへ適用（全行評価）→ 繰り返しの展開」までを行う。
// **ここまでは1バイトも書かない。** 書かない入口（Analyze）と書く入口（GenerateAndExecuteRequests）が
// 同じ関数を通ることで、検査の内容が2つの入口でずれない。
func (s *KFTLStatement) prepareRequests(
	ctx context.Context,
	repos *reps.GkillRepositories,
	applicationConfig *user_config.ApplicationConfig,
	userID, device, appName, localeName string,
	txID string,
	baseTime time.Time,
) (*KFTLRequestMap, error) {
	factory := newKFTLFactory()
	factory.reset()

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

	// 内容が空のリクエストと、付け先の無いメタ情報（プロトタイプ）の検査。
	// 行ごとの適用では分からない（kmemo の本文は複数行を束ねて初めて空と分かる）ので、
	// 全行を適用し終えてから、まだ1バイトも書いていないここで見る。
	// 行エラーが1件でもあれば呼ばない —— 適用に失敗した行のぶんだけ map が欠けていて、
	// 「付け先が無い」のような偽のエラーを重ねてしまう。
	if err := validateRequestContents(requestMap); err != nil {
		return nil, err
	}

	// 繰り返し（「？？」）の展開。**書き込みの直前、行の解釈が全部終わってから**やる。
	// ここでやると「？？」をブロックのどこに書いても結果が同じになり、
	// クライアント側の「本文が変わるたびに全行を解釈し直す」経路とも切り離せる。
	if err := expandRepeats(ctx, requestMap, baseTime); err != nil {
		return nil, err
	}
	return requestMap, nil
}

// validateRequestContents は「書く前に分かる、内容の欠けたリクエスト」を行別エラーにする。
//
// 2026-09-15 まで DoRequest が「本文が空なら何も書かずに nil」で済ませていたので、
// `ーち` だけ書いて「！」で保存すると 200「保存しました」でタブが閉じ、何も残らなかった
// （旧 Web の TS は ERR9000xx「内容がない打刻の保存がスキップされました」で送信を止めていた）。
// タグ行だけ・関連時刻行だけの送信（付け先の記録が無いプロトタイプ）も黙って0件だった。
// Analyze と GenerateAndExecuteRequests の両方がここを通るので、Web は打鍵中にピンクになり、
// 送信も同じ理由で止まる（ADR-0508）。
//
// 各型の判定は ValidateContent が持つ。基底に既定実装を置かないので、型を足したら
// 「何を空とみなすか」を書かないとコンパイルが通らない。
func validateRequestContents(requestMap *KFTLRequestMap) error {
	var inputErrs []error
	for _, req := range requestMap.All() {
		err := req.ValidateContent()
		if err == nil {
			continue
		}
		lineNumber, lineText := 0, ""
		if lineCtx := req.GetContext(); lineCtx != nil {
			lineNumber, lineText = lineCtx.LineIndex+1, lineCtx.ThisStatementLineText
		}
		inputErrs = append(inputErrs, withLine(err, lineNumber, lineText))
	}
	if len(inputErrs) != 0 {
		return errors.Join(inputErrs...)
	}
	return nil
}

// GenerateAndExecuteRequests parses StatementText, generates KFTLRequests,
// and executes each request against the provided repositories.
func (s *KFTLStatement) GenerateAndExecuteRequests(
	ctx context.Context,
	repos *reps.GkillRepositories,
	applicationConfig *user_config.ApplicationConfig,
	userID, device, appName, localeName string,
) ([]KFTLCreatedRecord, error) {
	txID := sqlite3impl.GenerateNewID()
	baseTime := time.Now()

	requestMap, err := s.prepareRequests(ctx, repos, applicationConfig, userID, device, appName, localeName, txID, baseTime)
	if err != nil {
		return nil, err
	}

	// ここから先は書き込みが起きる —— ただし各 DoRequest が書くのは txID 付きの temp rep で、
	// 実 rep へは最後の CommitTx が**1つの SQLite トランザクション**で確定する。
	// 途中で失敗したら DiscardTx して何も残さない（2026-09-15 まで実 rep へ直書きしていて、
	// 失敗した行より前の記録が残り、利用者が created[] を見て後始末する設計だった）。
	// 打刻の終了は実 rep から対象を引くので、同じテキスト内で開始した打刻は見つからない（TS 側と同じ）。
	var created []KFTLCreatedRecord
	for _, req := range requestMap.All() {
		err := req.DoRequest(ctx)
		if err != nil {
			discardStagedTx(ctx, repos, txID, userID, device)
			lineNumber, lineText := 0, ""
			if lineCtx := req.GetContext(); lineCtx != nil {
				lineNumber, lineText = lineCtx.LineIndex+1, lineCtx.ThisStatementLineText
			}
			var inputErr *KFTLInputError
			if errors.As(err, &inputErr) {
				return nil, withLine(err, lineNumber, lineText)
			}
			// 行番号を構造として持たせる。文字列へ畳むと、ハンドラが
			// 「何行目で止まったか」を利用者へ返せない。
			return nil, &KFTLExecutionError{
				LineNumber: lineNumber,
				LineText:   lineText,
				RequestID:  req.GetRequestID(),
				Cause:      err,
			}
		}
		created = append(created, req.GetCreatedRecords()...)
	}
	if _, err := repos.CommitTx(ctx, txID, userID, device); err != nil {
		// CommitTx は失敗時に temp rep の行を残す（再 commit できるように）。ここでは再試行しないので捨てる。
		discardStagedTx(ctx, repos, txID, userID, device)
		return nil, &KFTLExecutionError{Cause: fmt.Errorf("error at commit kftl tx id = %s: %w", txID, err)}
	}
	return created, nil
}

// discardStagedTx は失敗した送信の temp rep の行を捨てる。捨て損ねても利用者には返らない
// （本命のエラーが別にある）ので、ここが唯一の記録として Error で残す。
func discardStagedTx(ctx context.Context, repos *reps.GkillRepositories, txID, userID, device string) {
	if err := repos.DiscardTx(ctx, txID, userID, device); err != nil {
		err = fmt.Errorf("error at discard kftl tx id = %s: %w", txID, err)
		slog.Log(ctx, gkill_log.Error, "error at discard kftl tx", "error", fmt.Sprintf("%q", err))
	}
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
	// 保存マーカー「！」の行で本文を切り詰める（1行目は除く。「！」だけのメモは本文）。
	// **NextStatementLineText を組み立てる前に切ること。** ループの末尾で break する形だと、
	// マーカー直前の行の「次の行」が「！」になり、requireNextLineText が「値の行がある」と
	// 誤判定して `ーち`+「！」がタイトル空のまま DoRequest まで届く（2026-09-15 まで Web の
	// 「！」で保存する経路だけがこれを踏んでいた。ADR-0508）。
	// Mirrors: if (i != 0 && line_text == KFTL_SAVE_CHARACTOR) break
	lineTexts := strings.Split(s.StatementText, "\n")
	for i, lineText := range lineTexts {
		if i != 0 && (lineText == splitterSaveCharacter || lineText == splitterSaveCharacterAscii) {
			lineTexts = lineTexts[:i]
			break
		}
	}
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
			FindKyous:                 s.FindKyous,
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
		lines = append(lines, line)
	}

	return lines, nil
}
