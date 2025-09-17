package session

//import (
//	"context"
//	"errors"
//	"testing"
//	"time"
//
//	"github.com/stretchr/testify/mock"
//
//	"github.com/talx-hub/gophkeeper/internal/model"
//	"github.com/talx-hub/gophkeeper/pkg/session/mocks"
//)
//
//type storageMockBuilder struct {
//	storage *mocks.MockRefreshTokenStorage
//}
//
//func newStorageMock(t *testing.T) *storageMockBuilder {
//	t.Helper()
//
//	s := mocks.NewMockRefreshTokenStorage(t)
//	t.Cleanup(func() {
//		s.AssertExpectations(t)
//	})
//	return &storageMockBuilder{storage: s}
//}
//
//func (b *storageMockBuilder) Build() *mocks.MockRefreshTokenStorage {
//	return b.storage
//}
//
//func (b *storageMockBuilder) WithSave() *storageMockBuilder {
//	b.storage.EXPECT().
//		Save(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
//		RunAndReturn(func(_ context.Context, tokenID string, userID model.UserID, expiresAt time.Time) error {
//			if userID == "save-fail" {
//				return errors.New("save error")
//			}
//			return nil
//		})
//	return b
//}
//
//func (b *storageMockBuilder) WithValidate() *storageMockBuilder {
//	b.storage.EXPECT().
//		Validate(mock.Anything, mock.Anything).
//		RunAndReturn(func(_ context.Context, tokenID string) error {
//			if tokenID == "validate-fail" {
//				return errors.New("validate error")
//			}
//			return nil
//		})
//	return b
//}
//
//func (b *storageMockBuilder) WithDelete() *storageMockBuilder {
//	b.storage.EXPECT().
//		Delete(mock.Anything, mock.Anything).
//		RunAndReturn(func(_ context.Context, tokenID string) error {
//			if tokenID == "delete-fail" {
//				return errors.New("delete error")
//			}
//			return nil
//		})
//	return b
//}
