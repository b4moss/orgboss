package handlers

import (
	"context"
	"time"

	"github.com/b4m-oss/orgboss"
)

// DefaultDeletionHandler is the default DeletionHandler implementation (logical deletion)
type DefaultDeletionHandler struct {
	storage orgboss.Storage
}

// NewDefaultDeletionHandler creates a new DefaultDeletionHandler
func NewDefaultDeletionHandler(storage orgboss.Storage) *DefaultDeletionHandler {
	return &DefaultDeletionHandler{
		storage: storage,
	}
}

// SetStorage sets the storage (for testing, etc.)
func (h *DefaultDeletionHandler) SetStorage(storage orgboss.Storage) {
	h.storage = storage
}

// DeleteUser logically deletes a User
func (h *DefaultDeletionHandler) DeleteUser(ctx context.Context, user *orgboss.User) error {
	now := time.Now()
	user.DeletedAt = &now
	return h.storage.UpdateUser(ctx, user)
}

// DeleteOrganization logically deletes an Organization
func (h *DefaultDeletionHandler) DeleteOrganization(ctx context.Context, org *orgboss.Organization) error {
	now := time.Now()
	org.DeletedAt = &now
	return h.storage.UpdateOrganization(ctx, org)
}
