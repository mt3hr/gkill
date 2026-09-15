package message

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"net/url"
	"syscall"

	"modernc.org/sqlite"
)

// reason（何が起きたか）は Go の error を分類して決める。
//
// 利用者向け文言は操作単位（「メモ追加に失敗しました」）で、原因は Debug ログにしか
// 残らなかった。「書き込み先が未設定」「USB ディスクが外れた」「DB がロック中」「検索が
// 時間切れ」は打つ手が違うので、原因の種類だけを機械語のトークンで応答へ載せる。
//
// **分類は errors.Is / errors.As だけで行い、文字列照合はしない。** ドライバや OS の
// 文言が変わった回にだけ静かに分類が外れる（ADR-0208 と同じ罠）。
// 分類できない error は空（reason 省略）で、消費者は kind のヒントへ落ちる。

// reason の語彙。順序は資料（documents/reverse/error-handling-and-security.md）の表と同じ。
const (
	// ReasonWriteRepMissing はその種別の書き込み先 rep が未設定（設定→保存先で直せる）。
	ReasonWriteRepMissing = "write_rep_missing"
	// ReasonStorageUnavailable は保存先のファイルを開けない（外付けディスク・USB・ネットワークドライブが外れている）。
	ReasonStorageUnavailable = "storage_unavailable"
	// ReasonStorageCorrupted は保存先の DB が壊れている可能性（SQLITE_CORRUPT / NOTADB）。
	ReasonStorageCorrupted = "storage_corrupted"
	// ReasonStorageReadonly は保存先に書き込めない（読み取り専用・権限不足）。
	ReasonStorageReadonly = "storage_readonly"
	// ReasonStorageFull はディスクの空きが無い。
	ReasonStorageFull = "storage_full"
	// ReasonStorageIOError はディスク I/O の失敗（SQLITE_IOERR。ハードウェア・ファイルシステム側）。
	ReasonStorageIOError = "storage_io_error"
	// ReasonDBBusy は他の処理（キャッシュ更新・取り込み）と競合した。少し待って再試行。
	ReasonDBBusy = "db_busy"
	// ReasonTimeout は時間内に終わらなかった。期間や条件を絞って再試行。
	ReasonTimeout = "timeout"
	// ReasonCanceled は呼び出し側が中断した（Web の AbortController 等）。表示しなくてよい。
	ReasonCanceled = "canceled"
	// ReasonPluginBusy はプラグインのスロットが塞がっていて応答しなかった。
	ReasonPluginBusy = "plugin_busy"
	// ReasonPluginError はプラグインが自分のエラーを返した。
	ReasonPluginError = "plugin_error"
	// ReasonExternalFetchFailed は外部サイト（URL）への接続に失敗した。
	ReasonExternalFetchFailed = "external_fetch_failed"
	// ReasonLocalOnlyAccess はサーバが「ローカルアクセスのみ許可」でローカル以外から来た。
	ReasonLocalOnlyAccess = "local_only_access"
	// ReasonTLSFilesMissing はサーバ設定の TLS 証明書・秘密鍵ファイルが無い。
	ReasonTLSFilesMissing = "tls_files_missing"
)

// reasonTokens は reason の語彙の一覧（資料の件数検査と、テストの網羅確認に使う）。
var reasonTokens = []string{
	ReasonWriteRepMissing,
	ReasonStorageUnavailable,
	ReasonStorageCorrupted,
	ReasonStorageReadonly,
	ReasonStorageFull,
	ReasonStorageIOError,
	ReasonDBBusy,
	ReasonTimeout,
	ReasonCanceled,
	ReasonPluginBusy,
	ReasonPluginError,
	ReasonExternalFetchFailed,
	ReasonLocalOnlyAccess,
	ReasonTLSFilesMissing,
}

// Reasoner は自分の reason を名乗る error。
//
// dao 側の番兵（reps.ErrPluginBusy 等）や型付きエラー（reps.CommitTxWriteRepMissingError）が
// 実装する。message パッケージは dao を import できない（逆向きの依存）ので、
// 具体的な型ではなくこのインタフェースで受ける。
type Reasoner interface {
	error
	ErrorReason() string
}

// ReasonError は reason 付きの error を作る最小の実装。
// 番兵（var ErrX = &message.ReasonError{...} の形）と %w 包みの両方で使える。
type ReasonError struct {
	Msg    string
	Reason string
}

// Error は Msg を返します。
func (e *ReasonError) Error() string { return e.Msg }

// ErrorReason は Reason を返します。
func (e *ReasonError) ErrorReason() string { return e.Reason }

// errorCodeReason は Cause が無くてもコードだけで理由が決まるもの。
//
// ミドルウェアで打ち切る経路（filterLocalOnly）や設定不備のように、error 値を伴わずに
// GkillError を立てる箇所のため。Cause から分類できたときはそちらを優先する。
var errorCodeReason = map[string]string{
	LocalOnlyAccessDeniedError: ReasonLocalOnlyAccess, // ERR000414
	NotFoundTLSCertFileError:   ReasonTLSFilesMissing, // ERR000346
	NotFoundTLSKeyFileError:    ReasonTLSFilesMissing, // ERR000347
	WriteRepMissingError:       ReasonWriteRepMissing, // ERR000422
}

// SQLite の主結果コード。拡張コード（SQLITE_IOERR_READ 等）は下位 8 bit に主コードを持つ。
// 値は SQLite の ABI として固定されている（https://www.sqlite.org/rescode.html）。
// modernc.org/sqlite/lib を import しないのは、定数だけのために巨大なパッケージを message へ引き込まないため。
const (
	sqliteResultBusy     = 5
	sqliteResultLocked   = 6
	sqliteResultReadonly = 8
	sqliteResultIOErr    = 10
	sqliteResultCorrupt  = 11
	sqliteResultFull     = 13
	sqliteResultCantOpen = 14
	sqliteResultNotADB   = 26
)

// ReasonOf は Go の error から reason を分類します。分類できなければ空文字。
//
// 判定の順序: Reasoner（dao の番兵・型付きエラー）→ context → SQLite の結果コード →
// OS のファイル系エラー → ネットワーク。最初に当たったものを返します。
// **文字列は見ません。** 新しい分類を足すときも errors.Is / errors.As で判定できる形にすること。
func ReasonOf(err error) string {
	if err == nil {
		return ""
	}

	var reasoner Reasoner
	if errors.As(err, &reasoner) {
		return reasoner.ErrorReason()
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ReasonTimeout
	}
	if errors.Is(err, context.Canceled) {
		return ReasonCanceled
	}

	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() & 0xff {
		case sqliteResultBusy, sqliteResultLocked:
			return ReasonDBBusy
		case sqliteResultCantOpen:
			return ReasonStorageUnavailable
		case sqliteResultCorrupt, sqliteResultNotADB:
			return ReasonStorageCorrupted
		case sqliteResultReadonly:
			return ReasonStorageReadonly
		case sqliteResultFull:
			return ReasonStorageFull
		case sqliteResultIOErr:
			return ReasonStorageIOError
		}
	}

	if errors.Is(err, fs.ErrNotExist) {
		return ReasonStorageUnavailable
	}
	if errors.Is(err, fs.ErrPermission) {
		return ReasonStorageReadonly
	}
	if errors.Is(err, syscall.ENOSPC) {
		return ReasonStorageFull
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return ReasonExternalFetchFailed
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ReasonTimeout
		}
		return ReasonExternalFetchFailed
	}

	return ""
}

// reasonForError は GkillError 1件ぶんの reason を決める（Cause 優先、無ければコードの表）。
func reasonForError(errorCode string, cause error) string {
	if reason := ReasonOf(cause); reason != "" {
		return reason
	}
	return errorCodeReason[errorCode]
}
