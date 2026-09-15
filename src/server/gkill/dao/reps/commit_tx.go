// 編集前に読む: .claude/skills/gkill-go-backend/SKILL.md（この領域の不変条件の正本）
package reps

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"log/slog"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// commit_tx を1つの SQLite トランザクションにした理由と却下案（saga・物理削除・永続 journal）:
// documents/adr/0219-commit-tx-is-one-sqlite-transaction.md

// CommittedRecord は CommitTx が実 rep へ確定した1件。
type CommittedRecord struct {
	DataType string
	ID       string
	// Updated は確定前から最新版アドレス表に載っていた ID（＝既存記録の新しい版）であることを表す。
	// 表に載らない記録（プラグイン・git）は tx で書けないので、tx 経由ではこれで十分に判別できる。
	Updated bool
}

// CommitTxStageReadError は temp rep から未確定データを読み出す段の失敗。まだ何も書いていない。
type CommitTxStageReadError struct {
	DataType string
	Err      error
}

func (e *CommitTxStageReadError) Error() string {
	return fmt.Sprintf("commit tx: read staged %s: %v", e.DataType, e.Err)
}

func (e *CommitTxStageReadError) Unwrap() error { return e.Err }

// CommitTxWriteRepMissingError は未確定データがあるのに、その種別の書き込み rep が未設定のときのエラー。
// まだ何も書いていない。
type CommitTxWriteRepMissingError struct {
	DataType string
}

func (e *CommitTxWriteRepMissingError) Error() string {
	return fmt.Sprintf("commit tx: write repository for %s is not configured", e.DataType)
}

// ErrorReason は応答の reason（write_rep_missing）。message.Reasoner の実装。
func (e *CommitTxWriteRepMissingError) ErrorReason() string { return message.ReasonWriteRepMissing }

// DiscardTxError は temp rep の1種別の破棄失敗。DiscardTx は止まらず全種別を試し、errors.Join で束ねて返す。
type DiscardTxError struct {
	DataType string
	Err      error
}

func (e *DiscardTxError) Error() string {
	return fmt.Sprintf("discard tx: delete staged %s: %v", e.DataType, e.Err)
}

func (e *DiscardTxError) Unwrap() error { return e.Err }

// stagedTx は temp rep から読み出した1 tx ぶんの未確定データ。
type stagedTx struct {
	idfKyous      []IDFKyou
	kcs           []KC
	kmemos        []Kmemo
	lantanas      []Lantana
	mis           []Mi
	nlogs         []Nlog
	notifications []Notification
	rekyous       []ReKyou
	mirekyous     []MiReKyou
	tags          []Tag
	texts         []Text
	timeiss       []TimeIs
	urlogs        []URLog
}

// commitTxWriteRep は書き込み rep のうち CommitTx が使う2メソッド。
// Tag / Text / Notification の rep は Kyou を出さないので Repository を満たさない。
type commitTxWriteRep interface {
	GetPath(ctx context.Context, id string) (string, error)
	GetRepName(ctx context.Context) (string, error)
}

// commitTxWriteTarget は tx に行がある1種別ぶんの書き込み先。
type commitTxWriteTarget struct {
	dataType string
	rep      commitTxWriteRep
	path     string
}

// CommitTx は txID で temp rep に積まれた全種別を、実 rep へ**1つの SQLite トランザクションで**追記する。
//
// 行がある種別の書き込み rep のファイルを1接続に ATTACH し（先頭を main、残りを w1..wN）、
// BEGIN IMMEDIATE → 13種別ぶんの INSERT → COMMIT を1接続で行う。途中で失敗したら ROLLBACK し、
// **実 rep には何も残らない**（SQLite の複数ファイル原子コミット。rep ファイルは journal_mode=DELETE
// なので super-journal が効き、プロセス断でも半端は残らない）。失敗時は temp rep の行を残す
// （呼び出し側が DiscardTx するか、再度 CommitTx できる）。
//
// 成功したら各行を書き込み rep 名でキャッシュへライトスルーし、最新版アドレス表を更新し、
// temp rep の行を消す（commit は tx を消費する）。これらは派生情報なので失敗はログのみ。
//
// ATTACH できる数は SQLite のコンパイル時上限（modernc は 10）に従う。超えると ATTACH 自体が失敗して
// 何も書かずに返る。
//
// 返す CommittedRecord は確定した全行（成功時のみ。失敗時は nil）。
func (g *GkillRepositories) CommitTx(ctx context.Context, txID string, userID string, device string) ([]CommittedRecord, error) {
	staged, err := g.readStagedTx(ctx, txID, userID, device)
	if err != nil {
		return nil, err
	}

	targets, err := g.commitTxWriteTargets(ctx, staged)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		// 積まれたものが無い。書くものも消すものも無いので成功扱い。
		return nil, nil
	}

	// IDF は「レコードの置き場」と「ファイルの置き場」が別。
	// insertIDFKyouRow は idfKyou.RepName を TARGET_REP_NAME（＝ファイルの置き場）として実DBへ永続化するので、
	// temp rep の合成名 "IDF_TEMP" を入れたまま渡すとファイルの所在が実データごと壊れる
	// （キャッシュではないので UpdateCache でも直らない）。一時リポジトリが元の TARGET_REP_NAME を
	// 持っているので、それを戻してから書く。後段の afterCommit で入れ直す RepName は**キャッシュ用**で意味が違う。
	idfOwnRepName := ""
	if len(staged.idfKyous) != 0 {
		idfOwnRepName, err = g.WriteIDFKyouRep.GetRepName(ctx)
		if err != nil {
			return nil, fmt.Errorf("commit tx: get idf write rep name tx id = %s: %w", txID, err)
		}
		for i := range staged.idfKyous {
			staged.idfKyous[i].RepName = staged.idfKyous[i].TargetRepName
		}
	}

	err = g.writeStagedTxAtomically(ctx, txID, staged, targets, idfOwnRepName)
	if err != nil {
		return nil, err
	}

	committed := g.afterCommitTx(ctx, staged)

	// commit は tx を消費する。残すと同じ txID の再 commit で丸ごと二重登録になる。
	if err := g.DiscardTx(ctx, txID, userID, device); err != nil {
		slog.Log(ctx, gkill_log.Error, "error at discard committed tx rows", "tx_id", fmt.Sprintf("%q", txID), "error", fmt.Sprintf("%q", err))
	}
	return committed, nil
}

// DiscardTx は txID で temp rep に積まれた全種別の行を消す。
//
// 1種別が失敗しても止めずに全種別を試し、失敗を errors.Join で束ねて返す（止めると残りの種別の
// 未確定データが残り続ける）。束の各要素は *DiscardTxError。
func (g *GkillRepositories) DiscardTx(ctx context.Context, txID string, userID string, device string) error {
	t := g.TempReps
	steps := []struct {
		dataType string
		del      func(context.Context, string, string, string) error
	}{
		{"idf_kyou", t.IDFKyouTempRep.DeleteByTXID},
		{"kc", t.KCTempRep.DeleteByTXID},
		{"kmemo", t.KmemoTempRep.DeleteByTXID},
		{"lantana", t.LantanaTempRep.DeleteByTXID},
		{"mi", t.MiTempRep.DeleteByTXID},
		{"nlog", t.NlogTempRep.DeleteByTXID},
		{"notification", t.NotificationTempRep.DeleteByTXID},
		{"rekyou", t.ReKyouTempRep.DeleteByTXID},
		{"mirekyou", t.MiReKyouTempRep.DeleteByTXID},
		{"tag", t.TagTempRep.DeleteByTXID},
		{"text", t.TextTempRep.DeleteByTXID},
		{"timeis", t.TimeIsTempRep.DeleteByTXID},
		{"urlog", t.URLogTempRep.DeleteByTXID},
	}
	var errs []error
	for _, step := range steps {
		if err := step.del(ctx, txID, userID, device); err != nil {
			errs = append(errs, &DiscardTxError{DataType: step.dataType, Err: err})
		}
	}
	return errors.Join(errs...)
}

// readStagedTx は temp rep から全種別の未確定データを読む。まだ何も書かない。
func (g *GkillRepositories) readStagedTx(ctx context.Context, txID string, userID string, device string) (*stagedTx, error) {
	t := g.TempReps
	staged := &stagedTx{}
	var err error
	if staged.idfKyous, err = t.IDFKyouTempRep.GetIDFKyousByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "idf_kyou", Err: err}
	}
	if staged.kcs, err = t.KCTempRep.GetKCsByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "kc", Err: err}
	}
	if staged.kmemos, err = t.KmemoTempRep.GetKmemosByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "kmemo", Err: err}
	}
	if staged.lantanas, err = t.LantanaTempRep.GetLantanasByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "lantana", Err: err}
	}
	if staged.mis, err = t.MiTempRep.GetMisByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "mi", Err: err}
	}
	if staged.nlogs, err = t.NlogTempRep.GetNlogsByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "nlog", Err: err}
	}
	if staged.notifications, err = t.NotificationTempRep.GetNotificationsByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "notification", Err: err}
	}
	if staged.rekyous, err = t.ReKyouTempRep.GetReKyousByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "rekyou", Err: err}
	}
	if staged.mirekyous, err = t.MiReKyouTempRep.GetMiReKyousByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "mirekyou", Err: err}
	}
	if staged.tags, err = t.TagTempRep.GetTagsByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "tag", Err: err}
	}
	if staged.texts, err = t.TextTempRep.GetTextsByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "text", Err: err}
	}
	if staged.timeiss, err = t.TimeIsTempRep.GetTimeIssByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "timeis", Err: err}
	}
	if staged.urlogs, err = t.URLogTempRep.GetURLogsByTXID(ctx, txID, userID, device); err != nil {
		return nil, &CommitTxStageReadError{DataType: "urlog", Err: err}
	}
	return staged, nil
}

// commitTxWriteTargets は行がある種別の書き込み rep とそのファイルを、確定する順（固定）で返す。
// 書き込み rep が nil の種別があれば CommitTxWriteRepMissingError（まだ何も書いていない）。
func (g *GkillRepositories) commitTxWriteTargets(ctx context.Context, staged *stagedTx) ([]commitTxWriteTarget, error) {
	candidates := []struct {
		dataType string
		count    int
		rep      commitTxWriteRep
	}{
		{"idf_kyou", len(staged.idfKyous), g.WriteIDFKyouRep},
		{"kc", len(staged.kcs), g.WriteKCRep},
		{"kmemo", len(staged.kmemos), g.WriteKmemoRep},
		{"lantana", len(staged.lantanas), g.WriteLantanaRep},
		{"mi", len(staged.mis), g.WriteMiRep},
		{"nlog", len(staged.nlogs), g.WriteNlogRep},
		{"notification", len(staged.notifications), g.WriteNotificationRep},
		{"rekyou", len(staged.rekyous), g.WriteReKyouRep},
		{"mirekyou", len(staged.mirekyous), g.WriteMiReKyouRep},
		{"tag", len(staged.tags), g.WriteTagRep},
		{"text", len(staged.texts), g.WriteTextRep},
		{"timeis", len(staged.timeiss), g.WriteTimeIsRep},
		{"urlog", len(staged.urlogs), g.WriteURLogRep},
	}
	targets := make([]commitTxWriteTarget, 0, len(candidates))
	for _, c := range candidates {
		if c.count == 0 {
			continue
		}
		if c.rep == nil {
			return nil, &CommitTxWriteRepMissingError{DataType: c.dataType}
		}
		path, err := c.rep.GetPath(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("commit tx: get path of write %s rep: %w", c.dataType, err)
		}
		if path == "" {
			return nil, fmt.Errorf("commit tx: write %s rep has no database file path", c.dataType)
		}
		targets = append(targets, commitTxWriteTarget{dataType: c.dataType, rep: c.rep, path: path})
	}
	return targets, nil
}

// writeStagedTxAtomically は書き込み rep のファイルを1接続に ATTACH し、1トランザクションで全行を INSERT する。
//
// 非修飾のテーブル名（KMEMO / TAG / ...）は main → ATTACH 順に探索されるが、13 種別のテーブル名は
// 重複しない（commit_tx_test.go がソース走査で固定）ので、各 rep の insertXxxRow の SQL がそのまま
// 目当てのファイルに落ちる。
func (g *GkillRepositories) writeStagedTxAtomically(ctx context.Context, txID string, staged *stagedTx, targets []commitTxWriteTarget, idfOwnRepName string) (err error) {
	db, err := sqlite3impl.OpenSQLiteDBConnectionForTx(ctx, targets[0].path)
	if err != nil {
		return fmt.Errorf("commit tx: open %s: %w", targets[0].dataType, err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.Log(context.Background(), gkill_log.Warn, "error at defer close database", "target", "commit_tx", "error", fmt.Sprintf("%q", closeErr))
		}
	}()

	// ATTACH は接続単位の状態なので、プールから1本取り出してそれだけを使う
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("commit tx: get connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Log(context.Background(), gkill_log.Warn, "error at defer close database", "target", "commit_tx connection", "error", fmt.Sprintf("%q", closeErr))
		}
	}()

	for i, target := range targets[1:] {
		schema := fmt.Sprintf("w%d", i+1)
		// rep 自身が開くときと同じ "file:" + パスで ATTACH する（URI 処理は接続単位で有効になっているため、
		// 素のパスで渡すと '?' や '#' を含むパスで意味が変わる）。
		if _, err := conn.ExecContext(ctx, "ATTACH DATABASE ? AS "+schema, "file:"+target.path); err != nil {
			return fmt.Errorf("commit tx: attach %s (%s): %w", target.dataType, schema, err)
		}
		// main は DSN の _pragma で DELETE + FULL になっているが、ATTACH した側には効かないので明示する。
		// 複数ファイルの原子コミット（super-journal）は WAL では効かず、耐久性は synchronous=FULL で買う
		// （documents/adr/0204-keep-journal-mode-delete.md / 0215-data-db-synchronous-full.md）。
		if _, err := conn.ExecContext(ctx, "PRAGMA "+schema+".journal_mode=DELETE"); err != nil {
			return fmt.Errorf("commit tx: set journal_mode of %s: %w", target.dataType, err)
		}
		if _, err := conn.ExecContext(ctx, "PRAGMA "+schema+".synchronous=FULL"); err != nil {
			return fmt.Errorf("commit tx: set synchronous of %s: %w", target.dataType, err)
		}
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("commit tx: begin tx id = %s: %w", txID, err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				slog.Log(context.Background(), gkill_log.Error, "error at rollback commit tx", "tx_id", fmt.Sprintf("%q", txID), "error", fmt.Sprintf("%q", rollbackErr))
			}
		}
	}()

	for _, x := range staged.idfKyous {
		if err = insertIDFKyouRow(ctx, tx, x, idfOwnRepName); err != nil {
			return fmt.Errorf("commit tx: insert idf_kyou %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.kcs {
		if err = insertKCRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert kc %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.kmemos {
		if err = insertKmemoRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert kmemo %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.lantanas {
		if err = insertLantanaRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert lantana %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.mis {
		if err = insertMiRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert mi %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.nlogs {
		if err = insertNlogRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert nlog %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.notifications {
		if err = insertNotificationRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert notification %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.rekyous {
		if err = insertReKyouRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert rekyou %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.mirekyous {
		if err = insertMiReKyouRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert mirekyou %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.tags {
		if err = insertTagRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert tag %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.texts {
		if err = insertTextRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert text %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.timeiss {
		if err = insertTimeIsRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert timeis %s: %w", x.ID, err)
		}
	}
	for _, x := range staged.urlogs {
		if err = insertURLogRow(ctx, tx, x); err != nil {
			return fmt.Errorf("commit tx: insert urlog %s: %w", x.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: commit tx id = %s: %w", txID, err)
	}
	return nil
}

// afterCommitTx は確定した全行をキャッシュへライトスルーし、最新版アドレス表を更新する。
//
// **temp rep の合成 rep 名（"KmemoTemp" 等）をキャッシュへ持ち込まない。** GetXxxByTXID は
// `? AS REP_NAME` に temp rep の名前を差し込んで返すので、そのまま write-through するとキャッシュ表に
// 実在しない rep 名が入る。find_filter.go の filterKyousByRepName は非空で指定 rep に無い名前を落とすため、
// **tx で追加した記録が一覧から消える**（GUI は常に reps を送るので必ずこの経路）。
// 書き込み rep 名を入れ直してから渡す。失敗は派生情報の不整合なのでログのみ（次の UpdateCache で直る）。
func (g *GkillRepositories) afterCommitTx(ctx context.Context, staged *stagedTx) []CommittedRecord {
	var committed []CommittedRecord

	committed = append(committed, commitTxAfterRows(ctx, g, "idf_kyou", g.WriteIDFKyouRep, staged.idfKyous,
		func(x *IDFKyou, repName string) { x.RepName = repName },
		g.WriteThroughIDFKyouCache,
		func(x IDFKyou) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "kc", g.WriteKCRep, staged.kcs,
		func(x *KC, repName string) { x.RepName = repName },
		g.WriteThroughKCCache,
		func(x KC) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "kmemo", g.WriteKmemoRep, staged.kmemos,
		func(x *Kmemo, repName string) { x.RepName = repName },
		g.WriteThroughKmemoCache,
		func(x Kmemo) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "lantana", g.WriteLantanaRep, staged.lantanas,
		func(x *Lantana, repName string) { x.RepName = repName },
		g.WriteThroughLantanaCache,
		func(x Lantana) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "mi", g.WriteMiRep, staged.mis,
		func(x *Mi, repName string) { x.RepName = repName },
		g.WriteThroughMiCache,
		func(x Mi) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "nlog", g.WriteNlogRep, staged.nlogs,
		func(x *Nlog, repName string) { x.RepName = repName },
		g.WriteThroughNlogCache,
		func(x Nlog) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "notification", g.WriteNotificationRep, staged.notifications,
		func(x *Notification, repName string) { x.RepName = repName },
		g.WriteThroughNotificationCache,
		func(x Notification) (string, bool, time.Time, *string) {
			return x.ID, x.IsDeleted, x.UpdateTime, &x.TargetID
		})...)
	committed = append(committed, commitTxAfterRows(ctx, g, "rekyou", g.WriteReKyouRep, staged.rekyous,
		func(x *ReKyou, repName string) { x.RepName = repName },
		g.WriteThroughReKyouCache,
		func(x ReKyou) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, &x.TargetID })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "mirekyou", g.WriteMiReKyouRep, staged.mirekyous,
		func(x *MiReKyou, repName string) { x.RepName = repName },
		g.WriteThroughMiReKyouCache,
		func(x MiReKyou) (string, bool, time.Time, *string) {
			return x.ID, x.IsDeleted, x.UpdateTime, &x.TargetID
		})...)
	committed = append(committed, commitTxAfterRows(ctx, g, "tag", g.WriteTagRep, staged.tags,
		func(x *Tag, repName string) { x.RepName = repName },
		g.WriteThroughTagCache,
		func(x Tag) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, &x.TargetID })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "text", g.WriteTextRep, staged.texts,
		func(x *Text, repName string) { x.RepName = repName },
		g.WriteThroughTextCache,
		func(x Text) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, &x.TargetID })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "timeis", g.WriteTimeIsRep, staged.timeiss,
		func(x *TimeIs, repName string) { x.RepName = repName },
		g.WriteThroughTimeIsCache,
		func(x TimeIs) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)
	committed = append(committed, commitTxAfterRows(ctx, g, "urlog", g.WriteURLogRep, staged.urlogs,
		func(x *URLog, repName string) { x.RepName = repName },
		g.WriteThroughURLogCache,
		func(x URLog) (string, bool, time.Time, *string) { return x.ID, x.IsDeleted, x.UpdateTime, nil })...)

	return committed
}

// commitTxAfterRows は1種別ぶんの確定後処理（ライトスルー・最新版アドレス表・CommittedRecord 化）。
// 行が無ければ何もしない（書き込み rep が nil でも触らない）。
func commitTxAfterRows[T any](
	ctx context.Context,
	g *GkillRepositories,
	dataType string,
	writeRep commitTxWriteRep,
	rows []T,
	setRepName func(*T, string),
	writeThrough func(context.Context, T) error,
	address func(T) (id string, isDeleted bool, updateTime time.Time, targetIDInData *string),
) []CommittedRecord {
	if len(rows) == 0 {
		return nil
	}
	repName, err := writeRep.GetRepName(ctx)
	if err != nil {
		// 取れなければ空にする —— 空は filterKyousByRepName が残すので安全側
		slog.Log(ctx, gkill_log.Error, "error at get write rep name after commit tx", "data_type", dataType, "error", fmt.Sprintf("%q", err))
		repName = ""
	}
	committed := make([]CommittedRecord, 0, len(rows))
	for i := range rows {
		setRepName(&rows[i], repName)
		if err := writeThrough(ctx, rows[i]); err != nil {
			slog.Log(ctx, gkill_log.Error, "error at write through cache after commit tx", "data_type", dataType, "error", fmt.Sprintf("%q", err))
		}
		id, isDeleted, updateTime, targetIDInData := address(rows[i])
		_, updated := g.GetLatestDataRepositoryAddress(id)
		latestDataRepositoryAddress := gkill_cache.LatestDataRepositoryAddress{
			IsDeleted:                              isDeleted,
			TargetID:                               id,
			TargetIDInData:                         targetIDInData,
			DataUpdateTime:                         updateTime,
			LatestDataRepositoryName:               repName,
			LatestDataRepositoryAddressUpdatedTime: time.Now(),
		}
		g.SetLatestDataRepositoryAddress(id, latestDataRepositoryAddress)
		if _, err := g.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, latestDataRepositoryAddress); err != nil {
			slog.Log(ctx, gkill_log.Error, "error at add or update latest data repository address after commit tx", "data_type", dataType, "id", fmt.Sprintf("%q", id), "error", fmt.Sprintf("%q", err))
		}
		committed = append(committed, CommittedRecord{DataType: dataType, ID: id, Updated: updated})
	}
	return committed
}
