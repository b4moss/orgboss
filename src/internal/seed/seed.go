package seed

import (
	"context"
	"time"

	"gorm.io/gorm"

	"orgboss/types"
)

// SeedData はテスト用のシードデータを生成する
type SeedData struct {
	Organizations []*types.Organization
	Users         []*types.User
	Invitations   []*types.Invitation
}

// Seed はデータベースにシードデータを投入する
func Seed(ctx context.Context, db *gorm.DB) (*SeedData, error) {
	data := &SeedData{}

	// Organizationsを作成
	org1 := &types.Organization{
		Name:      "テスト組織1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.WithContext(ctx).Create(org1).Error; err != nil {
		return nil, err
	}
	data.Organizations = append(data.Organizations, org1)

	org2 := &types.Organization{
		Name:      "テスト組織2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.WithContext(ctx).Create(org2).Error; err != nil {
		return nil, err
	}
	data.Organizations = append(data.Organizations, org2)

	// Usersを作成
	user1 := &types.User{
		Email:          "manager1@example.com",
		OrganizationID: org1.ID,
		Role:           types.RoleManager,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := db.WithContext(ctx).Create(user1).Error; err != nil {
		return nil, err
	}
	data.Users = append(data.Users, user1)

	user2 := &types.User{
		Email:          "user1@example.com",
		OrganizationID: org1.ID,
		Role:           types.RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := db.WithContext(ctx).Create(user2).Error; err != nil {
		return nil, err
	}
	data.Users = append(data.Users, user2)

	user3 := &types.User{
		Email:          "manager2@example.com",
		OrganizationID: org2.ID,
		Role:           types.RoleManager,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := db.WithContext(ctx).Create(user3).Error; err != nil {
		return nil, err
	}
	data.Users = append(data.Users, user3)

	// Invitationsを作成
	invitation1 := &types.Invitation{
		Email:          "invited1@example.com",
		OrganizationID: org1.ID,
		Token:          "test-token-1",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         types.InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	if err := db.WithContext(ctx).Create(invitation1).Error; err != nil {
		return nil, err
	}
	data.Invitations = append(data.Invitations, invitation1)

	invitation2 := &types.Invitation{
		Email:          "invited2@example.com",
		OrganizationID: org1.ID,
		Token:          "test-token-2-expired",
		ExpiresAt:      time.Now().Add(-1 * time.Hour), // 期限切れ
		Status:         types.InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	if err := db.WithContext(ctx).Create(invitation2).Error; err != nil {
		return nil, err
	}
	data.Invitations = append(data.Invitations, invitation2)

	invitation3 := &types.Invitation{
		Email:          "invited3@example.com",
		OrganizationID: org2.ID,
		Token:          "test-token-3",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         types.InvitationStatusAccepted,
		CreatedAt:      time.Now(),
	}
	if err := db.WithContext(ctx).Create(invitation3).Error; err != nil {
		return nil, err
	}
	data.Invitations = append(data.Invitations, invitation3)

	return data, nil
}

// Cleanup はシードデータを削除する
func Cleanup(ctx context.Context, db *gorm.DB) error {
	// 外部キー制約の順序で削除
	if err := db.WithContext(ctx).Exec("DELETE FROM invitations").Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec("DELETE FROM users").Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec("DELETE FROM organizations").Error; err != nil {
		return err
	}
	return nil
}

