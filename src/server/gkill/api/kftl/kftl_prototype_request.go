package kftl

import "context"

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
