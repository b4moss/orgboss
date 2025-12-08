package storage

import (
	"context"

	"orgboss/types"
)

// NewInMemoryStorage は新しいInMemoryStorageを作成する
func NewInMemoryStorage() types.Storage {
	return &InMemoryStorage{
		organizations:   make(map[uint]*types.Organization),
		users:           make(map[uint]*types.User),
		invitations:     make(map[string]*types.Invitation),
		invitationsByID: make(map[uint]*types.Invitation),
		nextOrgID:       1,
		nextUserID:      1,
		nextInvID:       1,
	}
}

// InMemoryStorage はインメモリストレージの実装
type InMemoryStorage struct {
	organizations   map[uint]*types.Organization
	users           map[uint]*types.User
	invitations     map[string]*types.Invitation // token -> Invitation
	invitationsByID map[uint]*types.Invitation   // id -> Invitation
	nextOrgID       uint
	nextUserID      uint
	nextInvID       uint
}

// CreateOrganization はOrganizationを作成する
func (s *InMemoryStorage) CreateOrganization(ctx context.Context, org *types.Organization) error {
	if org.ID == 0 {
		org.ID = s.nextOrgID
		s.nextOrgID++
	}
	s.organizations[org.ID] = org
	return nil
}

// GetOrganization はOrganizationを取得する
func (s *InMemoryStorage) GetOrganization(ctx context.Context, id uint) (*types.Organization, error) {
	org, exists := s.organizations[id]
	if !exists {
		return nil, types.ErrOrganizationNotFound
	}
	return org, nil
}

// GetOrganizationBySignature はSignatureでOrganizationを取得する
func (s *InMemoryStorage) GetOrganizationBySignature(ctx context.Context, signature string) (*types.Organization, error) {
	for _, org := range s.organizations {
		if org.Signature == signature {
			return org, nil
		}
	}
	return nil, types.ErrOrganizationNotFound
}

// UpdateOrganization はOrganizationを更新する
func (s *InMemoryStorage) UpdateOrganization(ctx context.Context, org *types.Organization) error {
	if _, exists := s.organizations[org.ID]; !exists {
		return types.ErrOrganizationNotFound
	}
	s.organizations[org.ID] = org
	return nil
}

// DeleteOrganization はOrganizationを削除する
func (s *InMemoryStorage) DeleteOrganization(ctx context.Context, id uint) error {
	if _, exists := s.organizations[id]; !exists {
		return types.ErrOrganizationNotFound
	}
	delete(s.organizations, id)
	return nil
}

// ListOrganizations は全てのOrganizationを取得する
func (s *InMemoryStorage) ListOrganizations(ctx context.Context) ([]*types.Organization, error) {
	orgs := make([]*types.Organization, 0, len(s.organizations))
	for _, org := range s.organizations {
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// CreateUser はUserを作成する
func (s *InMemoryStorage) CreateUser(ctx context.Context, user *types.User) error {
	if user.ID == 0 {
		user.ID = s.nextUserID
		s.nextUserID++
	}
	s.users[user.ID] = user
	return nil
}

// GetUser はUserを取得する
func (s *InMemoryStorage) GetUser(ctx context.Context, id uint) (*types.User, error) {
	user, exists := s.users[id]
	if !exists {
		return nil, types.ErrUserNotFound
	}
	return user, nil
}

// GetUserByEmail はメールアドレスでUserを取得する
func (s *InMemoryStorage) GetUserByEmail(ctx context.Context, email string) (*types.User, error) {
	for _, user := range s.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, types.ErrUserNotFound
}

// GetUsersByOrganizationID は組織IDでUserを取得する
func (s *InMemoryStorage) GetUsersByOrganizationID(ctx context.Context, orgID uint) ([]*types.User, error) {
	users := make([]*types.User, 0)
	for _, user := range s.users {
		if user.OrganizationID == orgID {
			users = append(users, user)
		}
	}
	return users, nil
}

// UpdateUser はUserを更新する
func (s *InMemoryStorage) UpdateUser(ctx context.Context, user *types.User) error {
	if _, exists := s.users[user.ID]; !exists {
		return types.ErrUserNotFound
	}
	s.users[user.ID] = user
	return nil
}

// DeleteUser はUserを削除する
func (s *InMemoryStorage) DeleteUser(ctx context.Context, id uint) error {
	if _, exists := s.users[id]; !exists {
		return types.ErrUserNotFound
	}
	delete(s.users, id)
	return nil
}

// CreateInvitation はInvitationを作成する
func (s *InMemoryStorage) CreateInvitation(ctx context.Context, invitation *types.Invitation) error {
	if invitation.ID == 0 {
		invitation.ID = s.nextInvID
		s.nextInvID++
	}
	s.invitations[invitation.Token] = invitation
	s.invitationsByID[invitation.ID] = invitation
	return nil
}

// GetInvitationByToken はトークンでInvitationを取得する
func (s *InMemoryStorage) GetInvitationByToken(ctx context.Context, token string) (*types.Invitation, error) {
	invitation, exists := s.invitations[token]
	if !exists {
		return nil, types.ErrInvitationNotFound
	}
	return invitation, nil
}

// GetInvitationByID はIDでInvitationを取得する
func (s *InMemoryStorage) GetInvitationByID(ctx context.Context, id uint) (*types.Invitation, error) {
	invitation, exists := s.invitationsByID[id]
	if !exists {
		return nil, types.ErrInvitationNotFound
	}
	return invitation, nil
}

// GetInvitationsByOrganizationID は組織IDでInvitationを取得する
func (s *InMemoryStorage) GetInvitationsByOrganizationID(ctx context.Context, orgID uint) ([]*types.Invitation, error) {
	invitations := make([]*types.Invitation, 0)
	for _, invitation := range s.invitations {
		if invitation.OrganizationID == orgID {
			invitations = append(invitations, invitation)
		}
	}
	return invitations, nil
}

// GetInvitationsByEmail はメールアドレスでInvitationを取得する
func (s *InMemoryStorage) GetInvitationsByEmail(ctx context.Context, email string) ([]*types.Invitation, error) {
	invitations := make([]*types.Invitation, 0)
	for _, invitation := range s.invitations {
		if invitation.Email == email {
			invitations = append(invitations, invitation)
		}
	}
	return invitations, nil
}

// UpdateInvitation はInvitationを更新する
func (s *InMemoryStorage) UpdateInvitation(ctx context.Context, invitation *types.Invitation) error {
	if _, exists := s.invitationsByID[invitation.ID]; !exists {
		return types.ErrInvitationNotFound
	}
	s.invitations[invitation.Token] = invitation
	s.invitationsByID[invitation.ID] = invitation
	return nil
}

// DeleteInvitation はInvitationを削除する
func (s *InMemoryStorage) DeleteInvitation(ctx context.Context, id uint) error {
	invitation, exists := s.invitationsByID[id]
	if !exists {
		return types.ErrInvitationNotFound
	}
	delete(s.invitations, invitation.Token)
	delete(s.invitationsByID, id)
	return nil
}
