package gkill_log

import (
	"context"
	"fmt"
	"log/slog"
)

// TraceSQLEnabled はTRACE_SQLレベルが有効かを返します。
//
// slog.Log は内部でレベル判定をしますが、**引数はGoの評価規則で必ず先に評価されます**。
// 既定のログレベルは none (gkill_log.go の LogLevelFromCmd) なので、
// 呼び出し側で素に fmt.Sprintf("%q", sql) と書くと、捨てるためだけの文字列を毎回作ることになります。
// 1行ごとのループ(キャッシュ再構築のINSERT)では、それが行数ぶん積み上がります。
//
// routingHandler.Enabled は LevelVar の読み取りだけなので、この判定はほぼ無料です。
func TraceSQLEnabled(ctx context.Context) bool {
	return slog.Default().Enabled(ctx, TraceSQL)
}

// LogSQL はTRACE_SQLが有効なときだけSQLを出力します。
// 出力の形は従来の呼び出しと同一です。
func LogSQL(ctx context.Context, sql string) {
	if !TraceSQLEnabled(ctx) {
		return
	}
	slog.Log(ctx, TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql))
}

// LogSQLParams はTRACE_SQLが有効なときだけSQLとバインド引数を出力します。
// キー名("params")も整形の仕方も従来の呼び出しと同一なので、ログの中身は変わりません。
func LogSQLParams(ctx context.Context, sql string, queryArgs []any) {
	if !TraceSQLEnabled(ctx) {
		return
	}
	slog.Log(ctx, TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "params", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
}

// LogSQLQuery はTRACE_SQLが有効なときだけSQLとバインド引数を出力します。
// キー名が "query" である点だけ LogSQLParams と異なります(既存の呼び出しに合わせてあります)。
func LogSQLQuery(ctx context.Context, sql string, queryArgs []any) {
	if !TraceSQLEnabled(ctx) {
		return
	}
	slog.Log(ctx, TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
}

// LogIndexSQL はTRACE_SQLが有効なときだけ索引作成SQLを出力します。
// メッセージが "index sql" である点だけ LogSQL と異なります(既存の呼び出しに合わせてあります)。
func LogIndexSQL(ctx context.Context, sql string) {
	if !TraceSQLEnabled(ctx) {
		return
	}
	slog.Log(ctx, TraceSQL, "index sql", "sql", fmt.Sprintf("%q", sql))
}

// LogSQLQueryArgs はTRACE_SQLが有効なときだけSQLとバインド引数を出力します。
// キー名が "query args" である点だけ LogSQLParams と異なります(既存の呼び出しに合わせてあります)。
func LogSQLQueryArgs(ctx context.Context, sql string, queryArgs []any) {
	if !TraceSQLEnabled(ctx) {
		return
	}
	slog.Log(ctx, TraceSQL, "sql", "sql", fmt.Sprintf("%q", sql), "query args", fmt.Sprintf("%q", fmt.Sprint(queryArgs)))
}
