package handlers

import (
	"context"
	"time"

	"orgboss"
)

// DefaultDeletionHandler はデフォルトのDeletionHandler実装（論理削除）
type DefaultDeletionHandler struct {
	storage orgboss.Storage
}

// NewDefaultDeletionHandler は新しいDefaultDeletionHandlerを作成する
func NewDefaultDeletionHandler(storage orgboss.Storage) *DefaultDeletionHandler {
	return &DefaultDeletionHandler{
		storage: storage,
	}
}

// SetStorage はストレージを設定する（テスト用など）
func (h *DefaultDeletionHandler) SetStorage(storage orgboss.Storage) {
	h.storage = storage
}

// DeleteUser はUserを論理削除する
func (h *DefaultDeletionHandler) DeleteUser(ctx context.Context, user *orgboss.User) error {
	now := time.Now()
	user.DeletedAt = &now
	return h.storage.UpdateUser(ctx, user)
}

// DeleteOrganization はOrganizationを論理削除する
func (h *DefaultDeletionHandler) DeleteOrganization(ctx context.Context, org *orgboss.Organization) error {
	now := time.Now()
	org.DeletedAt = &now
	return h.storage.UpdateOrganization(ctx, org)
}
