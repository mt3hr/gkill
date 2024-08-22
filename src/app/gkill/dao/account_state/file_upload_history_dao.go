// ˅
package account_state

import "context"

// ˄

type FileUploadHistoryDAO interface {
	GetAllFileUploadHistories(ctx context.Context) ([]*FileUploadHistory, error)

	GetFileUploadHistories(ctx context.Context, userID string, device string) ([]*FileUploadHistory, error)

	AddFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) (bool, error)

	UpdateFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) (bool, error)

	DeleteFileUploadHistory(ctx context.Context, id string) (bool, error)

	// ˅

	// ˄
}

// ˅

// ˄
