package types

import "context"

// Storage はデータストレージのインターフェース
type Storage interface {
	// Organization操作
	CreateOrganization(ctx context.Context, org *Organization) error
	GetOrganization(ctx context.Context, id uint) (*Organization, error)
	UpdateOrganization(ctx context.Context, org *Organization) error
	DeleteOrganization(ctx context.Context, id uint) error
	ListOrganizations(ctx context.Context) ([]*Organization, error)

	// User操作
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id uint) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUsersByOrganizationID(ctx context.Context, orgID uint) ([]*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uint) error

	// Invitation操作
	CreateInvitation(ctx context.Context, invitation *Invitation) error
	GetInvitationByToken(ctx context.Context, token string) (*Invitation, error)
	GetInvitationByID(ctx context.Context, id uint) (*Invitation, error)
	GetInvitationsByOrganizationID(ctx context.Context, orgID uint) ([]*Invitation, error)
	UpdateInvitation(ctx context.Context, invitation *Invitation) error
	DeleteInvitation(ctx context.Context, id uint) error
}

// RoleChecker はロール判定ロジックを提供するインターフェース
type RoleChecker interface {
	HasPermission(role Role, action string) bool
}

// DeletionHandler は削除処理を提供するインターフェース
type DeletionHandler interface {
	DeleteUser(ctx context.Context, user *User) error
	DeleteOrganization(ctx context.Context, org *Organization) error
	// SetStorage はストレージを設定する（オプション、実装されていない場合は無視される）
	SetStorage(storage Storage)
}

// EmailSender はメール送信を提供するインターフェース
type EmailSender interface {
	SendInvitation(ctx context.Context, invitation *Invitation) error
}

