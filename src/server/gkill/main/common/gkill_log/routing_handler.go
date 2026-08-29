package gkill_log

import (
	"context"
	"log/slog"
	"slices"
)

// routingHandler は1つの Record を、設定に応じて
// - split（レベル別）
// - merged（統合）
// - stdout mirror
// に複製して流す
type routingHandler struct {
	r *Router

	// mods は Logger().With(...) / WithGroup(...) で積まれた修飾を、積まれた順に保持する。
	//
	// leaf handler は Record ごとに作り直すので、修飾もそのたびに掛け直す必要がある。
	// **ここを空実装（return h）にすると、静的フィールドが1つも出力されない。**
	// 実際に長らくそうなっていて、資料が「{"app":"gkill"} が付く」と書いているのに
	// 1行も出ていなかった（TestStaticFieldsAreEmitted がその回帰止め）。
	mods []handlerMod
}

// handlerMod は WithAttrs か WithGroup のどちらか一方を表す。
// group が非空なら WithGroup、そうでなければ WithAttrs。
type handlerMod struct {
	attrs []slog.Attr
	group string
}

func newRoutingHandler(r *Router) slog.Handler {
	return &routingHandler{r: r}
}

func (h *routingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// slog側でLevelVarを見てくれるが、ここでも最低限true/falseを返す
	return level >= h.r.level.Level()
}

func (h *routingHandler) Handle(ctx context.Context, rec slog.Record) error {
	// レベル無効なら何もしない
	if rec.Level < h.r.level.Level() {
		return nil
	}
	// None以上なら全停止
	if h.r.level.Level() >= None {
		return nil
	}

	h.r.lock()
	opts := h.r.opts
	h.r.unlock()

	// leaf handlerを都度作ると重いので本当はキャッシュしたいが、
	// まずは分かりやすさ優先で、Router側で作る形にする（キャッシュ化は後で）
	// → ここは後で最適化可能（必要なら言って）
	var err error

	// ④ 統合出力
	if opts.Mode == MergedOnly || opts.Mode == MergedAndSplit {
		mergedH := h.applyMods(h.r.newLeafHandler(h.r.merged))
		if mergedH.Enabled(ctx, rec.Level) {
			if e := mergedH.Handle(ctx, rec); e != nil && err == nil {
				err = e
			}
		}
	}

	// ② 分割出力
	if opts.Mode == SplitOnly || opts.Mode == MergedAndSplit {
		sink := h.r.byLevel[normalizeSplitLevel(rec.Level)]
		if sink != nil {
			splitH := h.applyMods(h.r.newLeafHandler(sink))
			if splitH.Enabled(ctx, rec.Level) {
				if e := splitH.Handle(ctx, rec); e != nil && err == nil {
					err = e
				}
			}
		}
	}

	// ③ stdoutミラー（“有効なレベル”だけ）
	if opts.StdoutMirror {
		stdH := h.applyMods(h.r.newLeafHandler(h.r.stdout))
		if stdH.Enabled(ctx, rec.Level) {
			if e := stdH.Handle(ctx, rec); e != nil && err == nil {
				err = e
			}
		}
	}

	return err
}

// applyMods は leaf handler へ、積まれた順に With/WithGroup を掛け直す。
func (h *routingHandler) applyMods(leaf slog.Handler) slog.Handler {
	for _, mod := range h.mods {
		if mod.group != "" {
			leaf = leaf.WithGroup(mod.group)
			continue
		}
		leaf = leaf.WithAttrs(mod.attrs)
	}
	return leaf
}

func (h *routingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return &routingHandler{
		r:    h.r,
		mods: append(slices.Clip(h.mods), handlerMod{attrs: slices.Clip(attrs)}),
	}
}

func (h *routingHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &routingHandler{
		r:    h.r,
		mods: append(slices.Clip(h.mods), handlerMod{group: name}),
	}
}

// rec.Level は TraceSQL/Trace/Debug/Info/Warn/Error 以外も来得る。
// splitは「最も近いバケット」に寄せる（運用上扱いやすい）。
func normalizeSplitLevel(l slog.Level) slog.Level {
	switch {
	case l <= TraceSQL:
		return TraceSQL
	case l <= Trace:
		return Trace
	case l <= Debug:
		return Debug
	case l <= Access:
		return Access
	case l <= Info:
		return Info
	case l <= Warn:
		return Warn
	default:
		return Error
	}
}
