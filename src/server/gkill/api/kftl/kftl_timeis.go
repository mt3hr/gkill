package kftl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// ─── KFTLTimeIsRequest (ーち): full TimeIs with start + optional end ──────────

// kftlTimeIsRequest records a TimeIs with title, start_time, and optional end_time.
// Mirrors: kftl-time-is-request.ts
type kftlTimeIsRequest struct {
	KFTLRequestBase
	title     string
	startTime time.Time
	endTime   *time.Time
}

func newKFTLTimeIsRequest(requestID string, ctx *KFTLStatementLineContext) *kftlTimeIsRequest {
	return &kftlTimeIsRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

// ValidateContent はタイトルの無い打刻を入力エラーにする（旧 Web の ERR900016 と同じ文言）。
// 通常は start 行の requireNextLineText が先に止める。ここは保存マーカーの穴（ADR-0508）のような
// 経路でタイトル空のまま届いたときの防御線。
func (r *kftlTimeIsRequest) ValidateContent() error {
	if r.title == "" {
		return newKFTLInputError("KFTL_TIMEIS_BLANK_SKIP_SAVE_MESSAGE_TITLE",
			fmt.Errorf("timeis title is empty: id=%s", r.RequestID))
	}
	return nil
}

func (r *kftlTimeIsRequest) DoRequest(ctx context.Context) error {
	if err := r.ValidateContent(); err != nil {
		return err
	}
	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	if !r.startTime.IsZero() {
		relatedTime = r.startTime
	}
	now := r.CreateTime
	timeis := reps.TimeIs{
		ID:           r.RequestID,
		Title:        r.title,
		StartTime:    relatedTime,
		EndTime:      r.endTime,
		CreateTime:   now,
		CreateApp:    r.Ctx.ApplicationName,
		CreateDevice: r.Ctx.Device,
		CreateUser:   r.Ctx.UserID,
		UpdateTime:   now,
		UpdateApp:    r.Ctx.ApplicationName,
		UpdateDevice: r.Ctx.Device,
		UpdateUser:   r.Ctx.UserID,
	}
	if err := r.Ctx.Repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, timeis, r.Ctx.TXID, r.Ctx.UserID, r.Ctx.Device); err != nil {
		return err
	}
	r.recordCreated("timeis", timeis.ID, r.GetRelatedTime())
	return nil
}

// AnchorTimeForRepeat は打刻の開始時刻を基準にする。
func (r *kftlTimeIsRequest) AnchorTimeForRepeat() (time.Time, bool) {
	return r.startTime, true
}

// CloneForRepeat は開始と終了を同じ日数だけずらす。長さはそのまま保たれる。
func (r *kftlTimeIsRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	c.startTime = shiftTime(r.startTime, dayShift)
	c.endTime = shiftTimePtr(r.endTime, dayShift)
	return &c
}

// kftlStartTimeIsStatementLine handles "ーち".
// Line sequence: ーち → title → start_time → end_time
// Mirrors: kftl-start-time-is-statement-line.ts
type kftlStartTimeIsStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLStartTimeIsStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLTimeIsRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// タイトル・開始日時・終了日時の3行が要る。1行も無ければ無言で0件だった。
	if err := requireNextLineText(l.ctx); err != nil {
		return err
	}
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsStatementLine) GetLabelName() string                  { return "timeIs" }
func (l *kftlStartTimeIsStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTimeIsStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsTitleStatementLine reads the title for a ーち TimeIs.
type kftlTimeIsTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLTimeIsTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsRequest) *kftlTimeIsTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsStartTimeStatementLine(lt, c, req)
	}
	return &kftlTimeIsTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlTimeIsTitleStatementLine) GetLabelName() string                  { return "timeIsTitle" }
func (l *kftlTimeIsTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsStartTimeStatementLine reads start_time for a ーち TimeIs.
type kftlTimeIsStartTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLTimeIsStartTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsRequest) *kftlTimeIsStartTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndTimeStatementLine(lt, c, req)
	}
	return &kftlTimeIsStartTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsStartTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	timeStr := strings.TrimPrefix(l.lineText, splitterRelatedTime)
	timeStr = strings.TrimPrefix(timeStr, splitterRelatedTimeAscii)
	t, err := parseDateTime(timeStr, l.ctx.BaseTime)
	if err != nil {
		return newKFTLInputError("KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE",
			fmt.Errorf("invalid timeis start_time %q: %w", l.lineText, err))
	}
	l.req.startTime = t
	l.req.SetRelatedTime(t)
	return nil
}
func (l *kftlTimeIsStartTimeStatementLine) GetLabelName() string                  { return "timeIsStartTime" }
func (l *kftlTimeIsStartTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsStartTimeStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsEndTimeStatementLine reads end_time for a ーち TimeIs.
type kftlTimeIsEndTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLTimeIsEndTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsRequest) *kftlTimeIsEndTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsEndTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsEndTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	timeStr := strings.TrimPrefix(l.lineText, splitterRelatedTime)
	timeStr = strings.TrimPrefix(timeStr, splitterRelatedTimeAscii)
	t, err := parseDateTime(timeStr, l.ctx.BaseTime)
	if err != nil {
		return newKFTLInputError("KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE",
			fmt.Errorf("invalid timeis end_time %q: %w", l.lineText, err))
	}
	l.req.endTime = &t
	return nil
}
func (l *kftlTimeIsEndTimeStatementLine) GetLabelName() string                  { return "timeIsEndTime" }
func (l *kftlTimeIsEndTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsEndTimeStatementLine) GetStatementLineText() string          { return l.lineText }

// ─── TimeIs Start-only (ーた) ─────────────────────────────────────────────────

// kftlTimeIsStartRequest records only start_time (no end_time).
// Mirrors: kftl-time-is-start-request.ts
type kftlTimeIsStartRequest struct {
	KFTLRequestBase
	title string
}

func newKFTLTimeIsStartRequest(requestID string, ctx *KFTLStatementLineContext) *kftlTimeIsStartRequest {
	return &kftlTimeIsStartRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

// ValidateContent はタイトルの無い打刻開始を入力エラーにする（`ーち` と同じ理由・同じ文言）。
func (r *kftlTimeIsStartRequest) ValidateContent() error {
	if r.title == "" {
		return newKFTLInputError("KFTL_TIMEIS_BLANK_SKIP_SAVE_MESSAGE_TITLE",
			fmt.Errorf("timeis start title is empty: id=%s", r.RequestID))
	}
	return nil
}

func (r *kftlTimeIsStartRequest) DoRequest(ctx context.Context) error {
	if err := r.ValidateContent(); err != nil {
		return err
	}
	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	now := r.CreateTime
	timeis := reps.TimeIs{
		ID:           r.RequestID,
		Title:        r.title,
		StartTime:    relatedTime,
		EndTime:      nil,
		CreateTime:   now,
		CreateApp:    r.Ctx.ApplicationName,
		CreateDevice: r.Ctx.Device,
		CreateUser:   r.Ctx.UserID,
		UpdateTime:   now,
		UpdateApp:    r.Ctx.ApplicationName,
		UpdateDevice: r.Ctx.Device,
		UpdateUser:   r.Ctx.UserID,
	}
	if err := r.Ctx.Repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, timeis, r.Ctx.TXID, r.Ctx.UserID, r.Ctx.Device); err != nil {
		return err
	}
	r.recordCreated("timeis", timeis.ID, r.GetRelatedTime())
	return nil
}

// SetRepeatSpec は打刻開始のみの繰り返しを弾く。
//
// end_time の無い打刻を回数ぶん作ると、**走行中のスタンプがその数だけ残る**。
// 終わりの無いスタンプは以降の全記録を覆うので、掃除するまで検索結果に付き続ける。
func (r *kftlTimeIsStartRequest) SetRepeatSpec(_ *repeatSpec) error {
	return newKFTLInputError("KFTL_REPEAT_TYPE_NOT_SUPPORTED_MESSAGE_TITLE",
		fmt.Errorf("repeat is not supported for a start-only timeis"))
}

func (r *kftlTimeIsStartRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	return &c
}

// kftlStartTimeIsStartStatementLine handles "ーた".
// Mirrors: kftl-start-time-is-start-statement-line.ts
type kftlStartTimeIsStartStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsStartRequest
}

func newKFTLStartTimeIsStartStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsStartStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLTimeIsStartRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsStartTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsStartStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsStartStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// 開始ラベルの行が無いと無言で0件になり、なぜ作られなかったのか分からない。
	if err := requireNextLineText(l.ctx); err != nil {
		return err
	}
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsStartStatementLine) GetLabelName() string                  { return "timeIsStart" }
func (l *kftlStartTimeIsStartStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTimeIsStartStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsStartTitleStatementLine reads the title for a ーた TimeIs.
type kftlTimeIsStartTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsStartRequest
}

func newKFTLTimeIsStartTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsStartRequest) *kftlTimeIsStartTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsStartTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsStartTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlTimeIsStartTitleStatementLine) GetLabelName() string                  { return "timeIsStartTitle" }
func (l *kftlTimeIsStartTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsStartTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// ─── 終了対象の検索条件 ────────────────────────────────────────────────────────

// configOf はリクエストの文脈から ApplicationConfig を取る（テストのように無ければ nil）。
func configOf(base *KFTLRequestBase) *user_config.ApplicationConfig {
	if base.Ctx == nil {
		return nil
	}
	return base.Ctx.ApplicationConfig
}

// playingTimeIsQueryFromConfig は「いま走っている打刻」を探す検索条件を作る。
//
// 設定の playing 検索条件（`playing_timeis_json_data` の `playing_timeis_find_kyou_query`）が
// あれば、TS の `generate_playing_timeis_query` が写すのと同じ欄
// （words / words_and / not_words / tags / tags_and）と、タグ構造の `is_force_hide` から
// 組んだ非表示タグを写す。2026-09-15 まで Web だけが設定条件を適用し、Wear / MCP 経由の
// `/end` は全 rep から探していたので、条件で絞っている利用者は経路ごとに終わる打刻が
// 違っていた（ADR-0507）。rep 名では絞らない（TS も `reps = null`）。
// 設定が無い・壊れているときは従来どおり「実行中の打刻すべて」。
func playingTimeIsQueryFromConfig(applicationConfig *user_config.ApplicationConfig, playingNow time.Time) *find.FindQuery {
	query := &find.FindQuery{
		PlayingTime:    &playingNow,
		OnlyLatestData: true,
		RepTypes:       []string{"timeis"},
	}
	if applicationConfig == nil || applicationConfig.PlayingTimeIsJSONData == nil {
		return query
	}
	var wrapper struct {
		PlayingTimeIsFindKyouQuery json.RawMessage `json:"playing_timeis_find_kyou_query"`
	}
	if err := json.Unmarshal(*applicationConfig.PlayingTimeIsJSONData, &wrapper); err != nil {
		return query
	}
	if len(wrapper.PlayingTimeIsFindKyouQuery) == 0 || string(wrapper.PlayingTimeIsFindKyouQuery) == "null" {
		return query
	}
	// 保存された条件は世代がまばら（use_* を持つ旧形式が残りうる）ので、null 意味論へ揃えてから読む
	migrated, _, err := find.MigrateLegacyFindQueryJSON(wrapper.PlayingTimeIsFindKyouQuery)
	if err != nil {
		return query
	}
	var saved find.FindQuery
	if err := json.Unmarshal(migrated, &saved); err != nil {
		return query
	}
	query.Words = saved.Words
	query.WordsAnd = saved.WordsAnd
	query.NotWords = saved.NotWords
	query.Tags = saved.Tags
	query.TagsAnd = saved.TagsAnd
	query.HideTags = forceHideTagNames(applicationConfig.TagStruct)
	return query
}

// findPlayingTimeIsEntries は「いま走っている打刻」を、設定の playing 検索条件で絞って返す。
//
// 語の条件は rep の SQL が見るが、**タグ・非表示タグは Kyou 検索の層（api.FindFilter）でしか効かない**。
// そこで、条件にタグが含まれるときは閉包 FindKyous（ハンドラが渡す）で同じ条件の Kyou を引き、
// その ID 集合と rep の結果を突き合わせる。閉包が無い（テスト・直叩き）ときは語だけで絞る。
func findPlayingTimeIsEntries(ctx context.Context, base *KFTLRequestBase) ([]reps.TimeIs, error) {
	query := playingTimeIsQueryFromConfig(configOf(base), time.Now())
	playingEntries, err := base.Ctx.Repositories.TimeIsReps.FindTimeIs(ctx, query)
	if err != nil {
		return nil, err
	}
	needTagFilter := query.Tags != nil || len(query.HideTags) != 0
	if !needTagFilter || base.Ctx.FindKyous == nil {
		return playingEntries, nil
	}
	kyous, err := base.Ctx.FindKyous(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error at find kyous for playing timeis: %w", err)
	}
	matched := make(map[string]struct{}, len(kyous))
	for _, kyou := range kyous {
		matched[kyou.ID] = struct{}{}
	}
	filtered := playingEntries[:0]
	for _, entry := range playingEntries {
		if _, ok := matched[entry.ID]; ok {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

// forceHideTagNames はタグ構造のうち `is_force_hide` のタグ名を集める（TS の `apply_hide_tags` と同じ）。
func forceHideTagNames(tagStruct *json.RawMessage) []string {
	if tagStruct == nil {
		return nil
	}
	type node struct {
		TagName     string  `json:"tag_name"`
		IsForceHide bool    `json:"is_force_hide"`
		Children    []*node `json:"children"`
	}
	var root node
	if err := json.Unmarshal(*tagStruct, &root); err != nil {
		return nil
	}
	var names []string
	var walk func(n *node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		if n.IsForceHide && n.TagName != "" {
			names = append(names, n.TagName)
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(&root)
	return names
}

// ─── TimeIs End by title (ーえ / ーいえ) ──────────────────────────────────────

// kftlTimeIsEndByTitleRequest finds a playing TimeIs by title and sets its end_time.
// Mirrors: kftl-time-is-end-by-title-request.ts
type kftlTimeIsEndByTitleRequest struct {
	KFTLRequestBase
	title                   string
	errorWhenTargetNotExist bool
}

func newKFTLTimeIsEndByTitleRequest(requestID string, ctx *KFTLStatementLineContext, errorWhenNotExist bool) *kftlTimeIsEndByTitleRequest {
	return &kftlTimeIsEndByTitleRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
		errorWhenTargetNotExist: errorWhenNotExist,
	}
}

// ValidateContent はタイトル行の無い打刻終了を入力エラーにする。
// 2026-09-15 まで DoRequest だけが見ていて、`ーえ` 単独は Analyze（ピンク）に出ず送信で初めて 400 になっていた。
func (r *kftlTimeIsEndByTitleRequest) ValidateContent() error {
	if r.title == "" {
		return newKFTLInputError("KFTL_TIMEIS_END_REQUIRE_END_TITLE_MESSAGE_TITLE",
			fmt.Errorf("timeis end: title is empty"))
	}
	return nil
}

func (r *kftlTimeIsEndByTitleRequest) DoRequest(ctx context.Context) error {
	if err := r.ValidateContent(); err != nil {
		return err
	}
	endTime := r.GetRelatedTime()

	playingEntries, err := findPlayingTimeIsEntries(ctx, &r.KFTLRequestBase)
	if err != nil {
		return fmt.Errorf("error finding playing timeis: %w", err)
	}

	var target *reps.TimeIs
	for i := range playingEntries {
		if playingEntries[i].Title == r.title {
			t := playingEntries[i]
			target = &t
			break
		}
	}

	if target == nil {
		if r.errorWhenTargetNotExist {
			return newKFTLInputError("KFTL_TIMEIS_END_TARGET_NOT_FOUND_MESSAGE_TITLE",
				fmt.Errorf("no playing timeis with title=%q", r.title))
		}
		return nil
	}

	// タグ・テキストの書き込みは「終了対象が見つかってから」。以前はこの判定より前にあり、
	// 終了できなかったときでもタグだけが実在しないIDを指したまま残っていた。
	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
		return err
	}

	now := r.CreateTime
	updated := *target
	updated.EndTime = &endTime
	updated.UpdateTime = now
	updated.UpdateApp = r.Ctx.ApplicationName
	updated.UpdateDevice = r.Ctx.Device
	updated.UpdateUser = r.Ctx.UserID
	if err := r.Ctx.Repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, updated, r.Ctx.TXID, r.Ctx.UserID, r.Ctx.Device); err != nil {
		return err
	}
	r.recordUpdated("timeis", target.ID, endTime)
	return nil
}

// SetRepeatSpec は打刻終了の繰り返しを弾く。
//
// 終了は PlayingTime に**現在時刻**を渡して「いま走っている1件」を探す。
// 繰り返しても同じ1件を狙うだけで、2回目以降は既に閉じていてヒットしない。
func (r *kftlTimeIsEndByTitleRequest) SetRepeatSpec(_ *repeatSpec) error {
	return newKFTLInputError("KFTL_REPEAT_TYPE_NOT_SUPPORTED_MESSAGE_TITLE",
		fmt.Errorf("repeat is not supported for a timeis end"))
}

func (r *kftlTimeIsEndByTitleRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	return &c
}

// kftlStartTimeIsEndStatementLine handles "ーえ" (error if not found).
// Mirrors: kftl-start-time-is-end-statement-line.ts
type kftlStartTimeIsEndStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTitleRequest
}

func newKFTLStartTimeIsEndStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTitleRequest(targetID, ctx, true)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndStatementLine) GetLabelName() string                  { return "timeIsEnd" }
func (l *kftlStartTimeIsEndStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTimeIsEndStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsEndTitleStatementLine reads the title to find the playing TimeIs.
type kftlTimeIsEndTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTitleRequest
}

func newKFTLTimeIsEndTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsEndByTitleRequest) *kftlTimeIsEndTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsEndTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsEndTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlTimeIsEndTitleStatementLine) GetLabelName() string                  { return "timeIsEndTitle" }
func (l *kftlTimeIsEndTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsEndTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlStartTimeIsEndIfExistStatementLine handles "ーいえ" (no error if not found).
// Mirrors: kftl-start-time-is-end-if-exist-statement-line.ts
type kftlStartTimeIsEndIfExistStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTitleRequest
}

func newKFTLStartTimeIsEndIfExistStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndIfExistStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTitleRequest(targetID, ctx, false)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndIfExistStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndIfExistStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndIfExistStatementLine) GetLabelName() string {
	return "timeIsEndIfExist"
}
func (l *kftlStartTimeIsEndIfExistStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlStartTimeIsEndIfExistStatementLine) GetStatementLineText() string { return l.lineText }

// ─── TimeIs End by tag (ーたえ / ーいたえ) ─────────────────────────────────────

// kftlTimeIsEndByTagRequest finds a playing TimeIs by tags and sets its end_time.
// Mirrors: kftl-time-is-end-by-tag-request.ts
type kftlTimeIsEndByTagRequest struct {
	KFTLRequestBase
	searchTags              []string
	errorWhenTargetNotExist bool
}

func newKFTLTimeIsEndByTagRequest(requestID string, ctx *KFTLStatementLineContext, errorWhenNotExist bool) *kftlTimeIsEndByTagRequest {
	return &kftlTimeIsEndByTagRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
		errorWhenTargetNotExist: errorWhenNotExist,
	}
}

// Override AddTag so searchTags also gets the value.
func (r *kftlTimeIsEndByTagRequest) AddTag(tag string) {
	r.KFTLRequestBase.AddTag(tag)
	r.searchTags = append(r.searchTags, tag)
}

// ValidateContent はタグの行を書かなかった打刻終了を入力エラーにする。
// searchTags が空だと DoRequest の照合はどの打刻とも一致せず、
// 「終了対象の打刻が存在しませんでした」という**原因と違う**エラーになっていた
// （if-exist 版では無言で0件）。打ち間違いとして名指しする。
// 2026-09-15 まで DoRequest だけが見ていて、Analyze（ピンク）には出なかった。
func (r *kftlTimeIsEndByTagRequest) ValidateContent() error {
	if len(r.searchTags) == 0 {
		return newKFTLInputError("KFTL_TIMEIS_END_REQUIRE_END_TAG_MESSAGE_TITLE",
			fmt.Errorf("end-by-tag needs at least one tag on the next line"))
	}
	return nil
}

func (r *kftlTimeIsEndByTagRequest) DoRequest(ctx context.Context) error {
	if err := r.ValidateContent(); err != nil {
		return err
	}
	endTime := r.GetRelatedTime()

	playingEntries, err := findPlayingTimeIsEntries(ctx, &r.KFTLRequestBase)
	if err != nil {
		return fmt.Errorf("error finding playing timeis for tag-end: %w", err)
	}

	var target *reps.TimeIs
outer:
	for i := range playingEntries {
		entryTags, err := r.Ctx.Repositories.GetTagsByTargetID(ctx, playingEntries[i].ID)
		if err != nil {
			continue
		}
		for _, et := range entryTags {
			for _, wantTag := range r.searchTags {
				if et.Tag == wantTag {
					t := playingEntries[i]
					target = &t
					break outer
				}
			}
		}
	}

	if target == nil {
		if r.errorWhenTargetNotExist {
			return newKFTLInputError("KFTL_TIMEIS_END_TARGET_NOT_FOUND_MESSAGE_TITLE",
				fmt.Errorf("no playing timeis with tags=%v", r.searchTags))
		}
		return nil
	}

	// タイトル指定版と同じ理由で、終了対象が見つかってから書く
	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
		return err
	}

	now := r.CreateTime
	updated := *target
	updated.EndTime = &endTime
	updated.UpdateTime = now
	updated.UpdateApp = r.Ctx.ApplicationName
	updated.UpdateDevice = r.Ctx.Device
	updated.UpdateUser = r.Ctx.UserID
	if err := r.Ctx.Repositories.TempReps.TimeIsTempRep.AddTimeIsInfo(ctx, updated, r.Ctx.TXID, r.Ctx.UserID, r.Ctx.Device); err != nil {
		return err
	}
	r.recordUpdated("timeis", target.ID, endTime)
	return nil
}

// SetRepeatSpec はタグ指定の打刻終了の繰り返しを弾く（理由はタイトル指定と同じ）。
func (r *kftlTimeIsEndByTagRequest) SetRepeatSpec(_ *repeatSpec) error {
	return newKFTLInputError("KFTL_REPEAT_TYPE_NOT_SUPPORTED_MESSAGE_TITLE",
		fmt.Errorf("repeat is not supported for a timeis end"))
}

func (r *kftlTimeIsEndByTagRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	c.searchTags = append([]string(nil), r.searchTags...)
	return &c
}

// kftlStartTimeIsEndByTagStatementLine handles "ーたえ" (error if not found).
type kftlStartTimeIsEndByTagStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTagRequest
}

func newKFTLStartTimeIsEndByTagStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndByTagStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTagRequest(targetID, ctx, true)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndByTagTagStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndByTagStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndByTagStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndByTagStatementLine) GetLabelName() string { return "timeIsEndByTag" }
func (l *kftlStartTimeIsEndByTagStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlStartTimeIsEndByTagStatementLine) GetStatementLineText() string { return l.lineText }

// kftlStartTimeIsEndByTagIfExistStatementLine handles "ーいたえ" (no error if not found).
type kftlStartTimeIsEndByTagIfExistStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTagRequest
}

func newKFTLStartTimeIsEndByTagIfExistStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndByTagIfExistStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTagRequest(targetID, ctx, false)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndByTagTagStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndByTagIfExistStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndByTagIfExistStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndByTagIfExistStatementLine) GetLabelName() string {
	return "timeIsEndByTagIfExist"
}
func (l *kftlStartTimeIsEndByTagIfExistStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlStartTimeIsEndByTagIfExistStatementLine) GetStatementLineText() string {
	return l.lineText
}

// kftlTimeIsEndByTagTagStatementLine reads the tag name (plain text, no 。 prefix)
// for ーたえ / ーいたえ. Multiple tags can be separated by 、.
type kftlTimeIsEndByTagTagStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTagRequest
}

func newKFTLTimeIsEndByTagTagStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsEndByTagRequest) *kftlTimeIsEndByTagTagStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsEndByTagTagStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsEndByTagTagStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	for _, tag := range strings.FieldsFunc(l.lineText, func(r rune) bool { return r == '、' || r == ',' }) {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			l.req.AddTag(tag)
		}
	}
	return nil
}
func (l *kftlTimeIsEndByTagTagStatementLine) GetLabelName() string { return "timeIsEndByTagTag" }
func (l *kftlTimeIsEndByTagTagStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlTimeIsEndByTagTagStatementLine) GetStatementLineText() string { return l.lineText }
