package kftl

import (
	"context"
	"fmt"
)

// KFTLPrototypeRequest is a placeholder that holds tags/texts/relatedTime to be
// inherited by the next real request set with the same requestID.
// Its DoRequest is a no-op.
// Mirrors: src/classes/kftl/kftl_prototype/kftl-prototype-request.ts
type KFTLPrototypeRequest struct {
	KFTLRequestBase
}

// newKFTLPrototypeRequest creates a new prototype request for the given target.
func newKFTLPrototypeRequest(requestID string, ctx *KFTLStatementLineContext) *KFTLPrototypeRequest {
	return &KFTLPrototypeRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

// ValidateContent は「全行を適用し終えてもプロトタイプのまま」を入力エラーにする。
//
// プロトタイプはタグ・テキスト・関連時刻の置き場所でしかなく、次に書かれた記録の Set が
// 中身を引き継いで置き換える。置き換わらずに残るのは `。タグ` だけ・`？時刻` だけ・`ーー` だけ・
// `、` の後ろにタグ行だけ、のように**付け先の記録が無い**ときで、2026-09-15 まで
// 200「保存しました」で黙って0件だった（旧 Web も同じ）。中身の有無は問わない ——
// `ーー` だけ（テキスト0行）も「付け先が無い」に変わりはない。
// 唯一の例外は `？時刻` の直後の `ーん` で、こちらは関連時刻を取り込んだあと
// KFTLRequestMap.Delete で外すのでここへ来ない（ADR-0508）。
func (r *KFTLPrototypeRequest) ValidateContent() error {
	return newKFTLInputError("KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE",
		fmt.Errorf("tags/texts/related time have no record to attach to: id=%s", r.RequestID))
}

// DoRequest is a no-op for prototypes — they never write any data.
// prepareRequests の validateRequestContents が先に弾くので、ここへは届かない。
func (r *KFTLPrototypeRequest) DoRequest(_ context.Context) error {
	return nil
}

// SetRepeatSpec は「繰り返しの対象になるレコードが無い」ことを表す。
//
// プロトタイプはタグ・テキスト・関連時刻の置き場所でしかない。
// 「？？」を本文より前やタグ行だけの位置に書くと付け先がこれしか無く、
// 受け入れると何も作らずに黙って終わる。
func (r *KFTLPrototypeRequest) SetRepeatSpec(_ *repeatSpec) error {
	return newKFTLInputError("KFTL_REPEAT_NO_TARGET_MESSAGE_TITLE",
		fmt.Errorf("no record to repeat at this position"))
}

func (r *KFTLPrototypeRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	return &c
}
