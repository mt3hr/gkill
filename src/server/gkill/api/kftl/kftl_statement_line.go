package kftl

import "context"

// KFTLStatementLine is the interface for each line in a KFTL statement.
// Mirrors: src/classes/kftl/kftl-statement-line.ts
type KFTLStatementLine interface {
	ApplyThisLineToRequestMap(ctx context.Context, requestMap *KFTLRequestMap) error
	GetLabelName() string
	GetContext() *KFTLStatementLineContext
	GetStatementLineText() string
}
