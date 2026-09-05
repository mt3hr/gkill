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

// DoRequest is a no-op for prototypes — they never write any data.
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
