// ˅
package account_state

import "context"

// ˄

type fileUploadHistoryDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (f *fileUploadHistoryDAOSQLite3Impl) GetAllFileUploadHistories(ctx context.Context) ([]*FileUploadHistory, error) {
	panic("notImplements")
}

func (f *fileUploadHistoryDAOSQLite3Impl) GGetFileUploadHistories(ctx context.Context, userID string, device string) ([]*FileUploadHistory, error) {
	panic("notImplements")
}

func (f *fileUploadHistoryDAOSQLite3Impl) GAddFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) (bool, error) {
	panic("notImplements")
}

func (f *fileUploadHistoryDAOSQLite3Impl) GUpdateFileUploadHistory(ctx context.Context, fileUploadHistory *FileUploadHistory) (bool, error) {
	panic("notImplements")
}

func (f *fileUploadHistoryDAOSQLite3Impl) GDeleteFileUploadHistory(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

// ˄
