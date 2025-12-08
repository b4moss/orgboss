package storage

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/b4m-oss/orgboss/types"
)

// PostgresStorage is a PostgreSQL storage implementation using GORM
type PostgresStorage struct {
	db *gorm.DB
}

// NewPostgresStorage creates a new PostgresStorage
func NewPostgresStorage(db *gorm.DB) types.Storage {
	return &PostgresStorage{db: db}
}

// CreateOrganization creates an Organization
func (s *PostgresStorage) CreateOrganization(ctx context.Context, org *types.Organization) error {
	return s.db.WithContext(ctx).Create(org).Error
}

// GetOrganization gets an Organization
func (s *PostgresStorage) GetOrganization(ctx context.Context, id uint) (*types.Organization, error) {
	var org types.Organization
	err := s.db.WithContext(ctx).First(&org, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}

// GetOrganizationBySignature gets an Organization by Signature
func (s *PostgresStorage) GetOrganizationBySignature(ctx context.Context, signature string) (*types.Organization, error) {
	var org types.Organization
	err := s.db.WithContext(ctx).Where("signature = ?", signature).First(&org).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrOrganizationNotFound
		}
		return nil, err
	}
	return &org, nil
}

// UpdateOrganization updates an Organization
func (s *PostgresStorage) UpdateOrganization(ctx context.Context, org *types.Organization) error {
	result := s.db.WithContext(ctx).Save(org)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return types.ErrOrganizationNotFound
	}
	return nil
}

// DeleteOrganization deletes an Organization (logical deletion)
func (s *PostgresStorage) DeleteOrganization(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&types.Organization{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return types.ErrOrganizationNotFound
	}
	return nil
}

// ListOrganizations gets all Organizations
func (s *PostgresStorage) ListOrganizations(ctx context.Context) ([]*types.Organization, error) {
	var orgs []*types.Organization
	err := s.db.WithContext(ctx).Find(&orgs).Error
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

// CreateUser creates a User
func (s *PostgresStorage) CreateUser(ctx context.Context, user *types.User) error {
	return s.db.WithContext(ctx).Create(user).Error
}

// GetUser gets a User
func (s *PostgresStorage) GetUser(ctx context.Context, id uint) (*types.User, error) {
	var user types.User
	err := s.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail gets a User by email address
func (s *PostgresStorage) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	var user types.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// GetUsersByOrganizationID gets Users by organization ID
func (s *PostgresStorage) GetUsersByOrganizationID(ctx context.Context, orgID uint) ([]*types.User, error) {
	var users []*types.User
	err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUser updates a User
func (s *PostgresStorage) UpdateUser(ctx context.Context, user *types.User) error {
	result := s.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return types.ErrUserNotFound
	}
	return nil
}

// DeleteUser deletes a User (logical deletion)
func (s *PostgresStorage) DeleteUser(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&types.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return types.ErrUserNotFound
	}
	return nil
}

// CreateInvitation creates an Invitation
func (s *PostgresStorage) CreateInvitation(ctx context.Context, invitation *types.Invitation) error {
	return s.db.WithContext(ctx).Create(invitation).Error
}

// GetInvitationByToken gets an Invitation by token
func (s *PostgresStorage) GetInvitationByToken(ctx context.Context, token string) (*types.Invitation, error) {
	var invitation types.Invitation
	err := s.db.WithContext(ctx).Where("token = ?", token).First(&invitation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

// GetInvitationByID gets an Invitation by ID
func (s *PostgresStorage) GetInvitationByID(ctx context.Context, id uint) (*types.Invitation, error) {
	var invitation types.Invitation
	err := s.db.WithContext(ctx).First(&invitation, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, types.ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

// GetInvitationsByOrganizationID gets Invitations by organization ID
func (s *PostgresStorage) GetInvitationsByOrganizationID(ctx context.Context, orgID uint) ([]*types.Invitation, error) {
	var invitations []*types.Invitation
	err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).Find(&invitations).Error
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

// GetInvitationsByEmail gets Invitations by email address
func (s *PostgresStorage) GetInvitationsByEmail(ctx context.Context, email string) ([]*types.Invitation, error) {
	var invitations []*types.Invitation
	err := s.db.WithContext(ctx).Where("email = ?", email).Find(&invitations).Error
	if err != nil {
		return nil, err
	}
	return invitations, nil
}

// UpdateInvitation updates an Invitation
func (s *PostgresStorage) UpdateInvitation(ctx context.Context, invitation *types.Invitation) error {
	result := s.db.WithContext(ctx).Save(invitation)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return types.ErrInvitationNotFound
	}
	return nil
}

// DeleteInvitation deletes an Invitation
func (s *PostgresStorage) DeleteInvitation(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&types.Invitation{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return types.ErrInvitationNotFound
	}
	return nil
}

// DB returns the internal GORM instance (for transactions)
func (s *PostgresStorage) DB() *gorm.DB {
	return s.db
}

