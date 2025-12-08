package orgboss

import (
	"context"
	"time"
)

// DefaultDeletionHandler はデフォルトのDeletionHandler実装（論理削除）
type DefaultDeletionHandler struct {
	storage Storage
}

// NewDefaultDeletionHandler は新しいDefaultDeletionHandlerを作成する
func NewDefaultDeletionHandler(storage Storage) *DefaultDeletionHandler {
	return &DefaultDeletionHandler{
		storage: storage,
	}
}

// SetStorage はストレージを設定する（テスト用など）
func (h *DefaultDeletionHandler) SetStorage(storage Storage) {
	h.storage = storage
}

// DeleteUser はUserを論理削除する
func (h *DefaultDeletionHandler) DeleteUser(ctx context.Context, user *User) error {
	now := time.Now()
	user.DeletedAt = &now
	return h.storage.UpdateUser(ctx, user)
}

// DeleteOrganization はOrganizationを論理削除する
func (h *DefaultDeletionHandler) DeleteOrganization(ctx context.Context, org *Organization) error {
	now := time.Now()
	org.DeletedAt = &now
	return h.storage.UpdateOrganization(ctx, org)
}
