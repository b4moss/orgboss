package seed

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/b4m-oss/orgboss/types"
)

// SeedData generates test seed data
type SeedData struct {
	Organizations []*types.Organization
	Users         []*types.User
	Invitations   []*types.Invitation
}

// Seed seeds the database with seed data
func Seed(ctx context.Context, db *gorm.DB) (*SeedData, error) {
	data := &SeedData{}

	// Create Organizations
	// org1 simulates a Japanese corporate number (13-digit number)
	org1 := &types.Organization{
		Name:      "テスト組織1",
		Signature: "1234567890123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.WithContext(ctx).Create(org1).Error; err != nil {
		return nil, err
	}
	data.Organizations = append(data.Organizations, org1)

	// org2 is a random string (simulating voluntary organizations or overseas organizations)
	org2 := &types.Organization{
		Name:      "テスト組織2",
		Signature: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.WithContext(ctx).Create(org2).Error; err != nil {
		return nil, err
	}
	data.Organizations = append(data.Organizations, org2)

	// Create Users
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

	// Create Invitations
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
		ExpiresAt:      time.Now().Add(-1 * time.Hour), // Expired
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

// Cleanup deletes seed data
func Cleanup(ctx context.Context, db *gorm.DB) error {
	// Delete in order of foreign key constraints
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

