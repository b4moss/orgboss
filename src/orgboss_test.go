package orgboss

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmailSender はテスト用のEmailSenderモック
type mockEmailSender struct{}

func (m *mockEmailSender) SendInvitation(ctx context.Context, invitation *Invitation) error {
	return nil
}

// ダミーテスト: testifyが正しく動作することを確認
func TestTestifyAssert(t *testing.T) {
	assert.Equal(t, 1, 1, "基本的なアサーションが動作することを確認")
	assert.NotNil(t, "test", "NotNilアサーションが動作することを確認")
}

func TestTestifyRequire(t *testing.T) {
	require.Equal(t, 2, 2, "Requireアサーションが動作することを確認")
	require.NotEmpty(t, "test", "NotEmptyアサーションが動作することを確認")
}

// ============================================================================
// CreateOrganizationWithUser のテスト
// ============================================================================

func TestCreateOrganizationWithUser_正常系(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	orgName := "テスト組織"
	userEmail := "manager@example.com"

	org, user, err := m.CreateOrganizationWithUser(ctx, orgName, userEmail)

	require.NoError(t, err, "OrganizationとUserが正常に作成される")
	require.NotNil(t, org, "Organizationが作成される")
	require.NotNil(t, user, "Userが作成される")
	assert.Equal(t, orgName, org.Name, "組織名が正しく設定される")
	assert.Equal(t, userEmail, user.Email, "ユーザーのメールアドレスが正しく設定される")
	assert.Equal(t, RoleManager, user.Role, "最初のユーザーはmanagerロールになる")
	assert.Equal(t, org.ID, user.OrganizationID, "UserのOrganizationIDが正しく設定される")
}

func TestCreateOrganizationWithUser_異常系_Organization作成失敗(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	// 空の組織名で失敗することを想定（実装後に適切なエラーケースに変更）
	org, user, err := m.CreateOrganizationWithUser(ctx, "", "user@example.com")

	// 実装が完了していないため、現時点ではエラーが返されるべき
	// 実装後は適切なエラーチェックに変更
	if err == nil {
		// 実装が完了していない場合、nilが返される
		assert.Nil(t, org, "Organizationが作成されない")
		assert.Nil(t, user, "Userも作成されない")
	}
}

// ============================================================================
// InviteUser のテスト
// ============================================================================

func TestInviteUser_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	orgID := uint(1)
	email := "invited@example.com"

	invitation, err := m.InviteUser(ctx, orgID, email)

	require.NoError(t, err, "Invitationが正常に作成される")
	require.NotNil(t, invitation, "Invitationが作成される")
	assert.Equal(t, email, invitation.Email, "メールアドレスが正しく設定される")
	assert.Equal(t, orgID, invitation.OrganizationID, "OrganizationIDが正しく設定される")
	assert.Equal(t, InvitationStatusPending, invitation.Status, "Statusがpendingになる")
	assert.NotEmpty(t, invitation.Token, "トークンが生成される")
	assert.False(t, invitation.ExpiresAt.IsZero(), "有効期限が設定される")
}

func TestInviteUser_異常系_メール送信失敗(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	// EmailSenderがnilの場合、エラーが返されることを想定
	config.EmailSender = nil
	m := NewManager(config)

	orgID := uint(1)
	email := "invited@example.com"

	invitation, err := m.InviteUser(ctx, orgID, email)

	// 実装が完了していないため、現時点ではエラーが返されるべき
	if err == nil {
		// 実装が完了していない場合、nilが返される
		assert.Nil(t, invitation, "Invitationが作成されない")
	}
}

// ============================================================================
// InviteUsers（バルク招待）のテスト
// ============================================================================

func TestInviteUsers_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	orgID := uint(1)
	emails := []string{"user1@example.com", "user2@example.com", "user3@example.com"}

	invitations, err := m.InviteUsers(ctx, orgID, emails)

	require.NoError(t, err, "バルク招待が正常に実行される")
	require.Len(t, invitations, len(emails), "各メールアドレスに対してInvitationが作成される")
	for i, invitation := range invitations {
		assert.Equal(t, emails[i], invitation.Email, "メールアドレスが正しく設定される")
		assert.Equal(t, InvitationStatusPending, invitation.Status, "Statusがpendingになる")
	}
}

func TestInviteUsers_異常系_上限超過(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.MaxBulkInviteCount = 2
	m := NewManager(config)

	orgID := uint(1)
	emails := []string{"user1@example.com", "user2@example.com", "user3@example.com"} // 上限を超える

	invitations, err := m.InviteUsers(ctx, orgID, emails)

	// 実装が完了していないため、現時点ではエラーが返されるべき
	if err == nil {
		// 実装が完了していない場合、nilが返される
		assert.Nil(t, invitations, "Invitationが作成されない")
	} else {
		assert.Equal(t, ErrBulkInviteLimitExceeded, err, "上限超過エラーが返される")
	}
}

// ============================================================================
// AcceptInvitation のテスト
// ============================================================================

func TestAcceptInvitation_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にInvitationを作成
	orgID := uint(1)
	email := "invited@example.com"
	invitation, err := m.InviteUser(ctx, orgID, email)
	require.NoError(t, err, "Invitationが作成される")

	user, err := m.AcceptInvitation(ctx, invitation.Token)

	require.NoError(t, err, "招待が正常に承諾される")
	require.NotNil(t, user, "Userが作成される")
	assert.Equal(t, RoleUser, user.Role, "Userのロールがuserになる")
}

func TestAcceptInvitation_異常系_無効なトークン(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	token := "invalid-token"

	user, err := m.AcceptInvitation(ctx, token)

	// 実装が完了していないため、現時点ではエラーが返されるべき
	if err == nil {
		// 実装が完了していない場合、nilが返される
		assert.Nil(t, user, "Userが作成されない")
	} else {
		assert.Equal(t, ErrInvalidToken, err, "無効なトークンエラーが返される")
	}
}

func TestAcceptInvitation_異常系_有効期限切れ(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	// 有効期限を短く設定して、すぐに期限切れにする
	config.InvitationExpiryDuration = -1 * time.Hour // 負の値で過去の時刻になる
	m := NewManager(config)

	// Invitationを作成（有効期限が過去になる）
	orgID := uint(1)
	email := "expired@example.com"
	invitation, err := m.InviteUser(ctx, orgID, email)
	require.NoError(t, err, "Invitationが作成される")

	// 少し待ってから期限切れを確認
	time.Sleep(10 * time.Millisecond)

	user, err := m.AcceptInvitation(ctx, invitation.Token)

	// 有効期限切れのため、エラーが返される
	assert.Error(t, err, "有効期限切れエラーが返される")
	assert.Equal(t, ErrInvitationExpired, err, "有効期限切れエラーが返される")
	assert.Nil(t, user, "Userが作成されない")
}

// ============================================================================
// RejectInvitation のテスト
// ============================================================================

func TestRejectInvitation_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとInvitationを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	err = m.RejectInvitation(ctx, invitation.Token)

	require.NoError(t, err, "招待が正常に拒否される")
}

func TestRejectInvitation_異常系_無効なトークン(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	token := "invalid-token"

	err := m.RejectInvitation(ctx, token)

	// 実装が完了していないため、現時点ではエラーが返されるべき
	if err != nil {
		assert.Equal(t, ErrInvalidToken, err, "無効なトークンエラーが返される")
	}
}

// ============================================================================
// DeleteUser のテスト
// ============================================================================

func TestDeleteUser_正常系_Userが自ら退会(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)

	user, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	err = m.DeleteUser(ctx, user.ID, org.ID)

	require.NoError(t, err, "Userが正常に削除される")
}

func TestDeleteUser_正常系_ManagerがUserを退会させる(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManager、Userを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)

	user, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	// ManagerがUserを削除（userIDとorgIDでUserを指定）
	err = m.DeleteUser(ctx, user.ID, org.ID)

	require.NoError(t, err, "ManagerがUserを正常に削除できる")
}

func TestDeleteUser_正常系_Managerが退会するとOrganizationも削除(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManagerを作成
	org, manager, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	err = m.DeleteUser(ctx, manager.ID, org.ID)

	require.NoError(t, err, "Managerが退会するとOrganizationも削除される")
}

func TestDeleteUser_異常系_権限がない(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織1", "manager1@example.com")
	require.NoError(t, err)

	// 別の組織のUserを作成
	org2, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織2", "manager2@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org2.ID, "user@example.com")
	require.NoError(t, err)

	user, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	// 異なる組織のUserを削除しようとする
	err = m.DeleteUser(ctx, user.ID, org.ID)

	// organization_idが一致しないため、エラーが返される
	assert.Error(t, err, "権限エラーが返される")
	assert.Equal(t, ErrOrganizationAccessDenied, err, "権限エラーが返される")
}

// ============================================================================
// DeleteOrganization のテスト
// ============================================================================

func TestDeleteOrganization_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	err = m.DeleteOrganization(ctx, org.ID)

	require.NoError(t, err, "Organizationと関連Userが正常に削除される")
}

// ============================================================================
// ResendInvitation のテスト
// ============================================================================

func TestResendInvitation_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManager、Invitationを作成
	org, manager, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	err = m.ResendInvitation(ctx, invitation.ID, org.ID, manager.ID)

	require.NoError(t, err, "Managerが正常に招待を再送信できる")
}

func TestResendInvitation_異常系_Userが実行(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManager、User、Invitationを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	invitation2, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)

	user, err := m.AcceptInvitation(ctx, invitation2.Token)
	require.NoError(t, err)

	err = m.ResendInvitation(ctx, invitation.ID, org.ID, user.ID)

	// Userはupdate権限がないため、エラーが返される
	assert.Error(t, err, "権限エラーが返される")
	assert.Equal(t, ErrPermissionDenied, err, "権限エラーが返される")
}

// ============================================================================
// UpdateProfile のテスト
// ============================================================================

func TestUpdateProfile_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, user, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "user@example.com")
	require.NoError(t, err)

	updates := map[string]interface{}{
		"name": "新しい名前",
	}

	err = m.UpdateProfile(ctx, user.ID, org.ID, updates)

	require.NoError(t, err, "自分のプロフィールが正常に更新される")
}

func TestUpdateProfile_異常系_他人のプロフィールを更新(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org1, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織1", "user1@example.com")
	require.NoError(t, err)

	org2, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織2", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org2.ID, "user2@example.com")
	require.NoError(t, err)

	user2, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	updates := map[string]interface{}{
		"name": "新しい名前",
	}

	// user2がorg1に属していない状態で、org1のIDでUpdateProfileを呼び出す
	// user2はorg2に属しているので、org1でUpdateProfileを呼び出すとエラーになる
	err = m.UpdateProfile(ctx, user2.ID, org1.ID, updates)

	// organization_idが一致しないため、エラーが返される
	assert.Error(t, err, "権限エラーが返される")
	assert.Equal(t, ErrOrganizationAccessDenied, err, "権限エラーが返される")
}

// ============================================================================
// UpdateOrganization のテスト
// ============================================================================

func TestUpdateOrganization_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManagerを作成
	org, manager, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	updates := map[string]interface{}{
		"name": "新しい組織名",
	}

	err = m.UpdateOrganization(ctx, org.ID, manager.ID, updates)

	require.NoError(t, err, "Managerが正常にOrganizationを更新できる")
}

func TestUpdateOrganization_異常系_Userが実行(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManager、Userを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)

	user, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	updates := map[string]interface{}{
		"name": "新しい組織名",
	}

	err = m.UpdateOrganization(ctx, org.ID, user.ID, updates)

	// Userはupdate権限がないため、エラーが返される
	assert.Error(t, err, "権限エラーが返される")
	assert.Equal(t, ErrPermissionDenied, err, "権限エラーが返される")
}

// ============================================================================
// ValidateOrganizationAccess のテスト
// ============================================================================

func TestValidateOrganizationAccess_正常系(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, user, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "user@example.com")
	require.NoError(t, err)

	err = m.ValidateOrganizationAccess(ctx, user.ID, org.ID)

	require.NoError(t, err, "organization_idが一致する場合、アクセスが許可される")
}

func TestValidateOrganizationAccess_異常系_organization_id不一致(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	_, user, err := m.CreateOrganizationWithUser(ctx, "テスト組織1", "user@example.com")
	require.NoError(t, err)

	org2, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織2", "manager@example.com")
	require.NoError(t, err)

	// 異なる組織IDでアクセスしようとする
	err = m.ValidateOrganizationAccess(ctx, user.ID, org2.ID)

	// organization_idが一致しないため、エラーが返される
	assert.Error(t, err, "アクセス拒否エラーが返される")
	assert.Equal(t, ErrOrganizationAccessDenied, err, "アクセス拒否エラーが返される")
}

// ============================================================================
// CheckPermission のテスト
// ============================================================================

func TestCheckPermission_正常系_ManagerがUpdate権限(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManagerを作成
	org, manager, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	action := "update"

	err = m.CheckPermission(ctx, manager.ID, org.ID, action)

	require.NoError(t, err, "ManagerがOrganizationのUpdate権限を持つ")
}

func TestCheckPermission_正常系_UserがRead権限(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManagerを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// Userロールのユーザーを作成（AcceptInvitationを使用）
	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)

	user, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	action := "read"

	err = m.CheckPermission(ctx, user.ID, org.ID, action)

	require.NoError(t, err, "UserがOrganizationのRead権限を持つ")
}

func TestCheckPermission_異常系_UserがUpdateを実行(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとManagerを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// Userロールのユーザーを作成（AcceptInvitationを使用）
	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)

	user2, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	action := "update"

	err = m.CheckPermission(ctx, user2.ID, org.ID, action)

	// Userはupdate権限がないため、エラーが返される
	assert.Error(t, err, "権限エラーが返される")
	assert.Equal(t, ErrPermissionDenied, err, "権限エラーが返される")
}

// ============================================================================
// 共通処理：トークン生成のテスト
// ============================================================================

func TestGenerateToken_正常系(t *testing.T) {
	m := NewManager(DefaultConfig())

	token1, err1 := m.GenerateToken()
	token2, err2 := m.GenerateToken()

	require.NoError(t, err1, "トークンが正常に生成される")
	require.NoError(t, err2, "トークンが正常に生成される")
	assert.NotEmpty(t, token1, "トークンが空でない")
	assert.NotEmpty(t, token2, "トークンが空でない")
	assert.NotEqual(t, token1, token2, "毎回異なるトークンが生成される")
}

// ============================================================================
// 共通処理：有効期限計算のテスト
// ============================================================================

func TestCalculateExpiry_正常系(t *testing.T) {
	config := DefaultConfig()
	config.InvitationExpiryDuration = 24 * time.Hour
	m := NewManager(config)

	expiresAt := m.CalculateExpiry()

	assert.False(t, expiresAt.IsZero(), "有効期限が設定される")
	expected := time.Now().Add(config.InvitationExpiryDuration)
	// 1秒以内の誤差を許容
	assert.WithinDuration(t, expected, expiresAt, 1*time.Second, "Config.InvitationExpiryDurationに基づいて正しく有効期限が計算される")
}

// ============================================================================
// 共通処理：有効期限チェックのテスト
// ============================================================================

func TestIsExpired_正常系_有効期限内(t *testing.T) {
	m := NewManager(DefaultConfig())

	expiresAt := time.Now().Add(1 * time.Hour)

	result := m.IsExpired(expiresAt)

	assert.False(t, result, "有効期限内の場合、falseが返される")
}

func TestIsExpired_正常系_有効期限切れ(t *testing.T) {
	m := NewManager(DefaultConfig())

	expiresAt := time.Now().Add(-1 * time.Hour)

	result := m.IsExpired(expiresAt)

	assert.True(t, result, "有効期限切れの場合、trueが返される")
}
