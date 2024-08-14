// ˅
package account_state

import "context"

// ˄

type FileUploadHistoryDAO interface {
	GetAllFileUploadHistories(ctx context.Context) []*FileUploadHistory

	GetFileUploadHistories(ctx context.Context, userID string, device string) []*FileUploadHistory

	AddFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) bool

	UpdateFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) bool

	DeleteFileUploadHistory(ctx context.Context, id string) bool

	// ˅

	// ˄
}

// ˅

// ˄
