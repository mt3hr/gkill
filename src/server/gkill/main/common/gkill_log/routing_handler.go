package gkill_log

import (
	"context"
	"log/slog"
)

// routingHandler は1つの Record を、設定に応じて
// - split（レベル別）
// - merged（統合）
// - stdout mirror
// に複製して流す
type routingHandler struct {
	r *Router
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
		mergedH := h.r.newLeafHandler(h.r.merged)
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
			splitH := h.r.newLeafHandler(sink)
			if splitH.Enabled(ctx, rec.Level) {
				if e := splitH.Handle(ctx, rec); e != nil && err == nil {
					err = e
				}
			}
		}
	}

	// ③ stdoutミラー（“有効なレベル”だけ）
	if opts.StdoutMirror {
		stdH := h.r.newLeafHandler(h.r.stdout)
		if stdH.Enabled(ctx, rec.Level) {
			if e := stdH.Handle(ctx, rec); e != nil && err == nil {
				err = e
			}
		}
	}

	return err
}

func (h *routingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Router.Logger().With(...) を使う運用ならここが呼ばれる。
	// 簡易実装として、Record側に属性が乗るのでそのまま返す。
	return h
}

func (h *routingHandler) WithGroup(name string) slog.Handler {
	return h
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
	case l <= Info:
		return Info
	case l <= Warn:
		return Warn
	default:
		return Error
	}
}
