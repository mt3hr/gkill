package gkill_log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"
)

type SplitMode int

const (
	// ② レベルごとにファイルに振り分けのみ
	SplitOnly SplitMode = iota
	// ④ 統合ファイルのみに出す（②は止める）
	MergedOnly
	// ④ 統合ファイル + ② の両方に出す
	MergedAndSplit
)

type Options struct {
	JSON         bool // JSON/Text切替
	AddSource    bool
	MinLevel     slog.Level // 有効レベルの下限（Noneで全停止）
	Mode         SplitMode  // 上の3モード
	StdoutMirror bool       // ③ 有効なレベルをstdoutにも出す
	TimeFormat   string     // 出力のtime整形（任意）
	StaticFields []any
}

// ルータの公開API
type Router struct {
	level slog.LevelVar

	// 分割出力（レベル別）
	byLevel map[slog.Level]*FileSink

	// 統合出力
	merged *FileSink

	// stdoutミラー
	stdout *FileSink

	// handler
	logger *slog.Logger

	// オプションは動的に変えたいのでatomicにしてもいいが、まずはmutexで十分
	optMu chan struct{} // 軽量ロック（バッファ1）
	opts  Options
}

func NewRouter(opts Options) *Router {
	r := &Router{
		byLevel: make(map[slog.Level]*FileSink),
		merged:  NewFileSink(os.Stdout), // 初期はstdout（後でSetFileできる）
		stdout:  NewFileSink(os.Stdout),
		optMu:   make(chan struct{}, 1),
	}
	r.optMu <- struct{}{}

	// デフォルト時刻フォーマット
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.RFC3339Nano
	}
	r.opts = opts
	r.level.Set(opts.MinLevel)

	// レベル別sinkを用意（初期は捨て先にしておく）
	// 実際のパスは SetSplitFile で指定してもらう想定
	r.byLevel[TraceSQL] = NewFileSink(os.Stdout)
	r.byLevel[Trace] = NewFileSink(os.Stdout)
	r.byLevel[Debug] = NewFileSink(os.Stdout)
	r.byLevel[Info] = NewFileSink(os.Stdout)
	r.byLevel[Warn] = NewFileSink(os.Stdout)
	r.byLevel[Error] = NewFileSink(os.Stdout)

	h := newRoutingHandler(r)
	r.logger = slog.New(h)
	if len(opts.StaticFields) > 0 {
		r.logger = r.logger.With(opts.StaticFields...)
	}
	return r
}

func (r *Router) Logger() *slog.Logger { return r.logger }

// 実行中にレベルを変更
func (r *Router) SetMinLevel(lv slog.Level) { r.level.Set(lv) }

// ③ stdoutミラーON/OFF
func (r *Router) SetStdoutMirror(enabled bool) {
	r.lock()
	defer r.unlock()
	r.opts.StdoutMirror = enabled
}

// ④ 統合モード切替（SplitOnly / MergedOnly / MergedAndSplit）
func (r *Router) SetMode(mode SplitMode) {
	r.lock()
	defer r.unlock()
	r.opts.Mode = mode
}

// ④ 統合ファイルのパス設定
func (r *Router) SetMergedFile(path string) error {
	return r.merged.SetFile(path)
}

// ② レベルごとのファイル設定
func (r *Router) SetSplitFile(level slog.Level, path string) error {
	s, ok := r.byLevel[level]
	if !ok {
		return fmt.Errorf("unknown level for split: %v", level)
	}
	return s.SetFile(path)
}

// stdoutは常にstdoutで良いが、必要なら差し替えも可
func (r *Router) SetStdoutWriter() {
	r.stdout.sw.Set(os.Stdout)
}

// 内部ロック
func (r *Router) lock()   { <-r.optMu }
func (r *Router) unlock() { r.optMu <- struct{}{} }

// HandlerOptions を生成（JSON/Text等）
func (r *Router) handlerOptions() *slog.HandlerOptions {
	// レベル名をあなた定義に置換
	return &slog.HandlerOptions{
		AddSource: r.opts.AddSource,
		Level:     &r.level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.LevelKey {
				if lv, ok := a.Value.Any().(slog.Level); ok {
					a.Value = slog.StringValue(LevelName(lv))
				}
			}
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(r.opts.TimeFormat))
				}
			}
			return a
		},
	}
}

func (r *Router) newLeafHandler(w interface{ Writer() io.Writer }) slog.Handler {
	opts := r.handlerOptions()
	if r.opts.JSON {
		return slog.NewJSONHandler(w.Writer(), opts)
	}
	return slog.NewTextHandler(w.Writer(), opts)
}
