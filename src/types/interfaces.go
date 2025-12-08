package types

import "context"

// Storage is the interface for data storage
type Storage interface {
	// Organization operations
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, id uint) (*Organization, error)
	GetOrganizationBySignature(ctx context.Context, signature string) (*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id uint) error
	ListOrganizations(ctx context.Context) ([]*Organization, error)

	// User operations
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id uint) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUsersByOrganizationID(ctx context.Context, orgID uint) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uint) error

	// Invitation operations
	CreateInvitation(ctx context.Context, invitation *Invitation) error
	GetInvitationByToken(ctx context.Context, token string) (*Invitation, error)
	GetInvitationByID(ctx context.Context, id uint) (*Invitation, error)
	GetInvitationsByOrganizationID(ctx context.Context, orgID uint) ([]*Invitation, error)
	GetInvitationsByEmail(ctx context.Context, email string) ([]*Invitation, error)
	UpdateInvitation(ctx context.Context, invitation *Invitation) error
	DeleteInvitation(ctx context.Context, id uint) error
}

// RoleChecker is the interface that provides role checking logic
type RoleChecker interface {
	HasPermission(role Role, action string) bool
}

// DeletionHandler is the interface that provides deletion processing
type DeletionHandler interface {
	DeleteUser(ctx context.Context, user *User) error
	DeleteOrganization(ctx context.Context, org *Organization) error
	// SetStorage sets the storage (optional, ignored if not implemented)
	SetStorage(storage Storage)
}

// EmailSender is the interface that provides email sending
type EmailSender interface {
	// SendInvitation sends an invitation email
	// If invitationURL is not empty, include the URL in the email body
	SendInvitation(ctx context.Context, invitation *Invitation, invitationURL string) error
}

