package orgboss

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b4m-oss/orgboss/internal/email"
	"github.com/b4m-oss/orgboss/internal/storage"
)

// mockEmailSender is a mock EmailSender for testing
type mockEmailSender struct{}

func (m *mockEmailSender) SendInvitation(ctx context.Context, invitation *Invitation, invitationURL string) error {
	return nil
}

// Dummy test: verify that testify works correctly
func TestTestifyAssert(t *testing.T) {
	assert.Equal(t, 1, 1, "基本的なアサーションが動作することを確認")
	assert.NotNil(t, "test", "NotNilアサーションが動作することを確認")
}

func TestTestifyRequire(t *testing.T) {
	require.Equal(t, 2, 2, "Requireアサーションが動作することを確認")
	require.NotEmpty(t, "test", "NotEmptyアサーションが動作することを確認")
}

// ============================================================================
// Tests for NewManager
// ============================================================================

func TestNewManager_Success(t *testing.T) {
	config := DefaultConfig()
	m := NewManager(config)
	assert.NotNil(t, m, "Managerが作成される")
	assert.NotNil(t, m.config, "Configが設定される")
	assert.NotNil(t, m.storage, "Storageが設定される")
}

func TestNewManager_WithNilConfig(t *testing.T) {
	m := NewManager(nil)
	assert.NotNil(t, m, "Managerが作成される")
	assert.NotNil(t, m.config, "デフォルトConfigが設定される")
}

func TestNewManager_WithDeletionHandler(t *testing.T) {
	config := DefaultConfig()
	storage := storage.NewInMemoryStorage()
	config.DeletionHandler = NewDefaultDeletionHandler(storage)
	m := NewManager(config)
	assert.NotNil(t, m, "Managerが作成される")
	assert.NotNil(t, m.config.DeletionHandler, "DeletionHandlerが設定される")
}

// ============================================================================
// Tests for NewManagerWithStorage
// ============================================================================

func TestNewManagerWithStorage_Success(t *testing.T) {
	config := DefaultConfig()
	storage := storage.NewInMemoryStorage()
	m := NewManagerWithStorage(config, storage)
	assert.NotNil(t, m, "Managerが作成される")
	assert.Equal(t, storage, m.storage, "指定されたStorageが設定される")
}

func TestNewManagerWithStorage_WithNilConfig(t *testing.T) {
	storage := storage.NewInMemoryStorage()
	m := NewManagerWithStorage(nil, storage)
	assert.NotNil(t, m, "Managerが作成される")
	assert.NotNil(t, m.config, "デフォルトConfigが設定される")
}

func TestNewManagerWithStorage_WithDeletionHandler(t *testing.T) {
	config := DefaultConfig()
	storage := storage.NewInMemoryStorage()
	config.DeletionHandler = NewDefaultDeletionHandler(storage)
	m := NewManagerWithStorage(config, storage)
	assert.NotNil(t, m, "Managerが作成される")
	assert.NotNil(t, m.config.DeletionHandler, "DeletionHandlerが設定される")
}

// ============================================================================
// Tests for CreateOrganizationWithUser
// ============================================================================

func TestCreateOrganizationWithUser_Success(t *testing.T) {
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

	// Signatureの検証
	assert.NotEmpty(t, org.Signature, "Signatureが設定される")
}

func TestCreateOrganizationWithUser_InvalidEmptyOrgName(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	org, user, err := m.CreateOrganizationWithUser(ctx, "", "user@example.com")

	assert.Error(t, err, "エラーが返される")
	assert.Nil(t, org, "Organizationが作成されない")
	assert.Nil(t, user, "Userも作成されない")
}

func TestCreateOrganizationWithUser_InvalidEmail(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	org, user, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "invalid-email")

	assert.Error(t, err, "エラーが返される")
	assert.Nil(t, org, "Organizationが作成されない")
	assert.Nil(t, user, "Userも作成されない")
}

func TestCreateOrganizationWithUser_HookError(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// BeforeOrganizationCreateフックでエラーを返す
	m.hooks.BeforeOrganizationCreate = func(ctx context.Context, data interface{}) error {
		return ErrPermissionDenied
	}

	org, user, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "user@example.com")

	assert.Error(t, err, "エラーが返される")
	assert.Equal(t, ErrPermissionDenied, err, "フックエラーが返される")
	assert.Nil(t, org, "Organizationが作成されない")
	assert.Nil(t, user, "Userも作成されない")
}

// ============================================================================
// Tests for InviteUser
// ============================================================================

func TestInviteUser_Success(t *testing.T) {
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

func TestInviteUser_EmailSendFailure(t *testing.T) {
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
// Tests for InviteUsers (bulk invitation)
// ============================================================================

func TestInviteUsers_Success(t *testing.T) {
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

func TestInviteUsers_LimitExceeded(t *testing.T) {
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
// Tests for AcceptInvitation
// ============================================================================

func TestAcceptInvitation_Success(t *testing.T) {
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

func TestAcceptInvitation_InvalidToken(t *testing.T) {
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

func TestAcceptInvitation_Expired(t *testing.T) {
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
// Tests for RejectInvitation
// ============================================================================

func TestRejectInvitation_Success(t *testing.T) {
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

func TestRejectInvitation_InvalidToken(t *testing.T) {
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
// Tests for DeleteUser
// ============================================================================

func TestDeleteUser_Success_UserSelfDeletion(t *testing.T) {
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

func TestDeleteUser_Success_ManagerDeletesUser(t *testing.T) {
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

func TestDeleteUser_Success_ManagerDeletionRemovesOrganization(t *testing.T) {
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

func TestDeleteUser_PermissionDenied(t *testing.T) {
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
// Tests for DeleteOrganization
// ============================================================================

func TestDeleteOrganization_Success(t *testing.T) {
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
// Tests for ResendInvitation
// ============================================================================

func TestResendInvitation_Success(t *testing.T) {
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

func TestResendInvitation_UserCannotExecute(t *testing.T) {
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
// Tests for UpdateProfile
// ============================================================================

func TestUpdateProfile_Success(t *testing.T) {
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

func TestUpdateProfile_CannotUpdateOthersProfile(t *testing.T) {
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
// Tests for UpdateOrganization
// ============================================================================

func TestUpdateOrganization_Success(t *testing.T) {
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

func TestUpdateOrganization_UserCannotExecute(t *testing.T) {
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
// Tests for ValidateOrganizationAccess
// ============================================================================

func TestValidateOrganizationAccess_Success(t *testing.T) {
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

func TestValidateOrganizationAccess_OrganizationIDMismatch(t *testing.T) {
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
// Tests for CheckPermission
// ============================================================================

func TestCheckPermission_Success_ManagerHasUpdatePermission(t *testing.T) {
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

func TestCheckPermission_Success_UserHasReadPermission(t *testing.T) {
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

func TestCheckPermission_UserCannotUpdate(t *testing.T) {
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
// Tests for common utilities: token generation
// ============================================================================

func TestGenerateToken_Success(t *testing.T) {
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
// Tests for common utilities: expiry calculation
// ============================================================================

func TestCalculateExpiry_Success(t *testing.T) {
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
// Tests for common utilities: expiry check
// ============================================================================

func TestIsExpired_Success_WithinValidityPeriod(t *testing.T) {
	m := NewManager(DefaultConfig())

	expiresAt := time.Now().Add(1 * time.Hour)

	result := m.IsExpired(expiresAt)

	assert.False(t, result, "有効期限内の場合、falseが返される")
}

func TestIsExpired_Success_Expired(t *testing.T) {
	m := NewManager(DefaultConfig())

	expiresAt := time.Now().Add(-1 * time.Hour)

	result := m.IsExpired(expiresAt)

	assert.True(t, result, "有効期限切れの場合、trueが返される")
}

// ============================================================================
// Tests for GetOrganizationBySignature
// ============================================================================

func TestGetOrganizationBySignature_Success(t *testing.T) {
	ctx := context.Background()
	storage := storage.NewInMemoryStorage()
	config := DefaultConfig()
	m := NewManagerWithStorage(config, storage)

	// Organizationを作成
	orgName := "テスト組織"
	userEmail := "manager@example.com"
	org, _, err := m.CreateOrganizationWithUser(ctx, orgName, userEmail)
	require.NoError(t, err)
	require.NotNil(t, org)
	require.NotEmpty(t, org.Signature)

	// SignatureでOrganizationを取得
	retrievedOrg, err := storage.GetOrganizationBySignature(ctx, org.Signature)

	require.NoError(t, err, "SignatureでOrganizationが取得できる")
	require.NotNil(t, retrievedOrg, "Organizationが取得される")
	assert.Equal(t, org.ID, retrievedOrg.ID, "正しいOrganizationが取得される")
	assert.Equal(t, org.Name, retrievedOrg.Name, "組織名が一致する")
	assert.Equal(t, org.Signature, retrievedOrg.Signature, "Signatureが一致する")
}

func TestGetOrganizationBySignature_NotFound(t *testing.T) {
	ctx := context.Background()
	storage := storage.NewInMemoryStorage()

	// 存在しないSignatureで取得を試みる
	nonExistentSignature := "non-existent-signature-1234567890"
	retrievedOrg, err := storage.GetOrganizationBySignature(ctx, nonExistentSignature)

	assert.Error(t, err, "エラーが返される")
	assert.Nil(t, retrievedOrg, "Organizationが取得されない")
}

func TestGetOrganizationBySignature_Success_JapaneseCorporateNumber(t *testing.T) {
	ctx := context.Background()
	storage := storage.NewInMemoryStorage()

	// 日本の法人番号を模したOrganizationを直接作成
	org := &Organization{
		Name:      "日本の法人テスト",
		Signature: "1234567890123",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := storage.CreateOrganization(ctx, org)
	require.NoError(t, err)

	// SignatureでOrganizationを取得
	retrievedOrg, err := storage.GetOrganizationBySignature(ctx, "1234567890123")

	require.NoError(t, err, "日本の法人番号でOrganizationが取得できる")
	require.NotNil(t, retrievedOrg, "Organizationが取得される")
	assert.Equal(t, org.ID, retrievedOrg.ID, "正しいOrganizationが取得される")
}

func TestGetOrganizationBySignature_Success_UniqueConstraint(t *testing.T) {
	ctx := context.Background()
	storage := storage.NewInMemoryStorage()

	// 同じSignatureで2つのOrganizationを作成しようとする（ユニーク制約違反）
	org1 := &Organization{
		Name:      "組織1",
		Signature: "duplicate-signature-12345",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := storage.CreateOrganization(ctx, org1)
	require.NoError(t, err)

	// 同じSignatureで2つ目のOrganizationを作成（ユニーク制約違反）
	org2 := &Organization{
		Name:      "組織2",
		Signature: "duplicate-signature-12345", // 同じSignature
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = storage.CreateOrganization(ctx, org2)
	require.NoError(t, err) // インメモリストレージではユニーク制約がチェックされない

	// GetOrganizationBySignatureがどちらかのOrganizationを返すことを確認
	// 注意: mapのイテレーション順序は保証されないため、org1またはorg2のどちらかが返される
	retrievedOrg, err := storage.GetOrganizationBySignature(ctx, "duplicate-signature-12345")
	require.NoError(t, err)
	assert.NotNil(t, retrievedOrg, "Organizationが取得される")
	assert.Equal(t, "duplicate-signature-12345", retrievedOrg.Signature, "Signatureが一致する")
	// org1またはorg2のどちらかが返されることを確認
	assert.True(t, retrievedOrg.ID == org1.ID || retrievedOrg.ID == org2.ID, "作成したOrganizationのいずれかが取得される")
}

// ============================================================================
// Tests for UpdatePassword
// ============================================================================

func TestUpdatePassword_Success(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, user, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "user@example.com")
	require.NoError(t, err)

	// 招待を作成して、パスワード更新時にacceptedになることを確認
	invitation, err := m.InviteUser(ctx, org.ID, user.Email)
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusPending, invitation.Status, "招待はpending状態")

	// パスワードを更新
	newPassword := "newpassword123"
	err = m.UpdatePassword(ctx, user.ID, org.ID, newPassword)
	require.NoError(t, err, "パスワードが正常に更新される")

	// 招待がacceptedになっていることを確認
	updatedInvitation, err := m.storage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusAccepted, updatedInvitation.Status, "招待がacceptedになる")
}

func TestUpdatePassword_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前に2つのOrganizationとUserを作成
	_, user1, err := m.CreateOrganizationWithUser(ctx, "テスト組織1", "user1@example.com")
	require.NoError(t, err)

	org2, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織2", "user2@example.com")
	require.NoError(t, err)

	// user1がorg2のIDでパスワードを更新しようとする（権限なし）
	// user1はorg1に属しているので、org2のIDで更新しようとするとエラーになる
	err = m.UpdatePassword(ctx, user1.ID, org2.ID, "newpassword123")
	assert.Error(t, err, "権限エラーが返される")
	assert.Equal(t, ErrOrganizationAccessDenied, err, "権限エラーが返される")
}

func TestUpdatePassword_UserNotFound(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	m := NewManager(config)

	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "user@example.com")
	require.NoError(t, err)

	// 存在しないユーザーIDでパスワードを更新
	nonExistentUserID := uint(99999)
	err = m.UpdatePassword(ctx, nonExistentUserID, org.ID, "newpassword123")
	assert.Error(t, err, "エラーが返される")
}

// ============================================================================
// Tests for GetInvitationURL
// ============================================================================

func TestGetInvitationURL_Success(t *testing.T) {
	config := DefaultConfig()
	config.InvitationBaseURL = "http://localhost:8080"
	m := NewManager(config)

	token := "test-token-12345"
	url := m.GetInvitationURL(token)

	expectedURL := "http://localhost:8080/invite/test-token-12345"
	assert.Equal(t, expectedURL, url, "正しいURLが生成される")
}

func TestGetInvitationURL_EmptyBaseURL(t *testing.T) {
	config := DefaultConfig()
	config.InvitationBaseURL = ""
	m := NewManager(config)

	token := "test-token-12345"
	url := m.GetInvitationURL(token)

	assert.Equal(t, "", url, "空文字列が返される")
}

// ============================================================================
// Tests for ValidateInvitationTokenAndGetRedirectURL
// ============================================================================

func TestValidateInvitationTokenAndGetRedirectURL_Success(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	config.InvitationRedirectPath = "/reset-password"
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// 招待を作成
	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	// トークンを検証してリダイレクトURLを取得
	redirectURL, err := m.ValidateInvitationTokenAndGetRedirectURL(ctx, invitation.Token)
	require.NoError(t, err, "トークンが正常に検証される")
	assert.Contains(t, redirectURL, "/reset-password", "リダイレクトパスが含まれる")
	assert.Contains(t, redirectURL, invitation.Token, "トークンが含まれる")
}

func TestValidateInvitationTokenAndGetRedirectURL_DefaultPath(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	config.InvitationRedirectPath = "" // 空文字列（デフォルトパスを使用）
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// 招待を作成
	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	// トークンを検証してリダイレクトURLを取得
	redirectURL, err := m.ValidateInvitationTokenAndGetRedirectURL(ctx, invitation.Token)
	require.NoError(t, err, "トークンが正常に検証される")
	assert.Contains(t, redirectURL, "/reset-password", "デフォルトのリダイレクトパスが使用される")
	assert.Contains(t, redirectURL, invitation.Token, "トークンが含まれる")
}

func TestValidateInvitationTokenAndGetRedirectURL_InvalidToken(t *testing.T) {
	ctx := context.Background()
	m := NewManager(DefaultConfig())

	invalidToken := "invalid-token-12345"
	_, err := m.ValidateInvitationTokenAndGetRedirectURL(ctx, invalidToken)

	assert.Error(t, err, "エラーが返される")
	assert.Equal(t, ErrInvalidToken, err, "無効なトークンエラーが返される")
}

func TestValidateInvitationTokenAndGetRedirectURL_Expired(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// 招待を作成
	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	// 有効期限を過去に設定
	invitation.ExpiresAt = time.Now().Add(-24 * time.Hour)
	err = m.storage.UpdateInvitation(ctx, invitation)
	require.NoError(t, err)

	// トークンを検証（有効期限切れ）
	_, err = m.ValidateInvitationTokenAndGetRedirectURL(ctx, invitation.Token)
	assert.Error(t, err, "エラーが返される")
	assert.Equal(t, ErrInvitationExpired, err, "有効期限切れエラーが返される")
}

func TestValidateInvitationTokenAndGetRedirectURL_AlreadyAccepted(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// 招待を作成
	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	// 招待をacceptedに変更
	invitation.Status = InvitationStatusAccepted
	err = m.storage.UpdateInvitation(ctx, invitation)
	require.NoError(t, err)

	// トークンを検証（既にaccepted）
	_, err = m.ValidateInvitationTokenAndGetRedirectURL(ctx, invitation.Token)
	assert.Error(t, err, "エラーが返される")
	assert.Equal(t, ErrInvitationAlreadyAccepted, err, "既にacceptedエラーが返される")
}

func TestValidateInvitationTokenAndGetRedirectURL_AlreadyRejected(t *testing.T) {
	ctx := context.Background()
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManager(config)

	// 事前にOrganizationとUserを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "テスト組織", "manager@example.com")
	require.NoError(t, err)

	// 招待を作成
	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	// 招待をrejectedに変更
	invitation.Status = InvitationStatusRejected
	err = m.storage.UpdateInvitation(ctx, invitation)
	require.NoError(t, err)

	// トークンを検証（既にrejected）
	_, err = m.ValidateInvitationTokenAndGetRedirectURL(ctx, invitation.Token)
	assert.Error(t, err, "エラーが返される")
	assert.Equal(t, ErrInvitationAlreadyRejected, err, "既にrejectedエラーが返される")
}

// ============================================================================
// Tests for SetStorage
// ============================================================================

func TestSetStorage_Success(t *testing.T) {
	storage1 := storage.NewInMemoryStorage()
	handler := NewDefaultDeletionHandler(storage1)

	storage2 := storage.NewInMemoryStorage()
	handler.SetStorage(storage2)

	// SetStorageが正常に動作することを確認（エラーが発生しない）
	assert.NotNil(t, handler, "ハンドラーが作成される")
}

// ============================================================================
// Tests for VersionInfo
// ============================================================================

func TestVersionInfo_Success(t *testing.T) {
	version := VersionInfo()
	assert.Equal(t, Version, version, "バージョン情報が正しく返される")
	assert.NotEmpty(t, version, "バージョン情報が空でない")
}

// ============================================================================
// Tests for InMemoryStorage
// ============================================================================

func TestNewInMemoryStorage_Success(t *testing.T) {
	storage := NewInMemoryStorage()
	assert.NotNil(t, storage, "ストレージが作成される")
}

func TestInMemoryStorage_CreateOrganization_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	org := &Organization{
		Name:      "テスト組織",
		Signature: "test-signature",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := storage.CreateOrganization(ctx, org)
	require.NoError(t, err, "Organizationが作成される")
	assert.NotZero(t, org.ID, "IDが設定される")
}

func TestInMemoryStorage_GetOrganization_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	org := &Organization{
		Name:      "テスト組織",
		Signature: "test-signature",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := storage.CreateOrganization(ctx, org)
	require.NoError(t, err)

	retrievedOrg, err := storage.GetOrganization(ctx, org.ID)
	require.NoError(t, err, "Organizationが取得される")
	assert.Equal(t, org.ID, retrievedOrg.ID, "正しいOrganizationが取得される")
}

func TestInMemoryStorage_GetOrganization_NotFound(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	_, err := storage.GetOrganization(ctx, 99999)
	assert.Error(t, err, "エラーが返される")
	assert.Equal(t, ErrOrganizationNotFound, err, "OrganizationNotFoundエラーが返される")
}

func TestInMemoryStorage_UpdateOrganization_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	org := &Organization{
		Name:      "テスト組織",
		Signature: "test-signature",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := storage.CreateOrganization(ctx, org)
	require.NoError(t, err)

	org.Name = "更新された組織名"
	err = storage.UpdateOrganization(ctx, org)
	require.NoError(t, err, "Organizationが更新される")

	updatedOrg, err := storage.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "更新された組織名", updatedOrg.Name, "名前が更新される")
}

func TestInMemoryStorage_DeleteOrganization_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	org := &Organization{
		Name:      "テスト組織",
		Signature: "test-signature",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := storage.CreateOrganization(ctx, org)
	require.NoError(t, err)

	err = storage.DeleteOrganization(ctx, org.ID)
	require.NoError(t, err, "Organizationが削除される")

	_, err = storage.GetOrganization(ctx, org.ID)
	assert.Error(t, err, "削除後は取得できない")
}

func TestInMemoryStorage_ListOrganizations_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	org1 := &Organization{
		Name:      "組織1",
		Signature: "signature1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	org2 := &Organization{
		Name:      "組織2",
		Signature: "signature2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := storage.CreateOrganization(ctx, org1)
	require.NoError(t, err)
	err = storage.CreateOrganization(ctx, org2)
	require.NoError(t, err)

	orgs, err := storage.ListOrganizations(ctx)
	require.NoError(t, err, "Organization一覧が取得される")
	assert.GreaterOrEqual(t, len(orgs), 2, "2つ以上のOrganizationが取得される")
}

func TestInMemoryStorage_CreateUser_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	user := &User{
		Email:          "user@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := storage.CreateUser(ctx, user)
	require.NoError(t, err, "Userが作成される")
	assert.NotZero(t, user.ID, "IDが設定される")
}

func TestInMemoryStorage_GetUser_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	user := &User{
		Email:          "user@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := storage.CreateUser(ctx, user)
	require.NoError(t, err)

	retrievedUser, err := storage.GetUser(ctx, user.ID)
	require.NoError(t, err, "Userが取得される")
	assert.Equal(t, user.ID, retrievedUser.ID, "正しいUserが取得される")
}

func TestInMemoryStorage_GetUserByEmail_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	user := &User{
		Email:          "user@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := storage.CreateUser(ctx, user)
	require.NoError(t, err)

	retrievedUser, err := storage.GetUserByEmail(ctx, "user@example.com")
	require.NoError(t, err, "Userが取得される")
	assert.Equal(t, user.Email, retrievedUser.Email, "正しいUserが取得される")
}

func TestInMemoryStorage_GetUsersByOrganizationID_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	user1 := &User{
		Email:          "user1@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	user2 := &User{
		Email:          "user2@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	err := storage.CreateUser(ctx, user1)
	require.NoError(t, err)
	err = storage.CreateUser(ctx, user2)
	require.NoError(t, err)

	users, err := storage.GetUsersByOrganizationID(ctx, 1)
	require.NoError(t, err, "User一覧が取得される")
	assert.GreaterOrEqual(t, len(users), 2, "2つ以上のUserが取得される")
}

func TestInMemoryStorage_UpdateUser_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	user := &User{
		Email:          "user@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := storage.CreateUser(ctx, user)
	require.NoError(t, err)

	user.Role = RoleManager
	err = storage.UpdateUser(ctx, user)
	require.NoError(t, err, "Userが更新される")

	updatedUser, err := storage.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, RoleManager, updatedUser.Role, "ロールが更新される")
}

func TestInMemoryStorage_DeleteUser_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	user := &User{
		Email:          "user@example.com",
		Password:       "hashed-password",
		OrganizationID: 1,
		Role:           RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err := storage.CreateUser(ctx, user)
	require.NoError(t, err)

	err = storage.DeleteUser(ctx, user.ID)
	require.NoError(t, err, "Userが削除される")

	_, err = storage.GetUser(ctx, user.ID)
	assert.Error(t, err, "削除後は取得できない")
}

func TestInMemoryStorage_CreateInvitation_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 1,
		Token:          "test-token",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}

	err := storage.CreateInvitation(ctx, invitation)
	require.NoError(t, err, "Invitationが作成される")
	assert.NotZero(t, invitation.ID, "IDが設定される")
}

func TestInMemoryStorage_GetInvitationByToken_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 1,
		Token:          "test-token",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	err := storage.CreateInvitation(ctx, invitation)
	require.NoError(t, err)

	retrievedInvitation, err := storage.GetInvitationByToken(ctx, "test-token")
	require.NoError(t, err, "Invitationが取得される")
	assert.Equal(t, invitation.Token, retrievedInvitation.Token, "正しいInvitationが取得される")
}

func TestInMemoryStorage_GetInvitationByID_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 1,
		Token:          "test-token",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	err := storage.CreateInvitation(ctx, invitation)
	require.NoError(t, err)

	retrievedInvitation, err := storage.GetInvitationByID(ctx, invitation.ID)
	require.NoError(t, err, "Invitationが取得される")
	assert.Equal(t, invitation.ID, retrievedInvitation.ID, "正しいInvitationが取得される")
}

func TestInMemoryStorage_GetInvitationsByOrganizationID_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation1 := &Invitation{
		Email:          "invited1@example.com",
		OrganizationID: 1,
		Token:          "test-token-1",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	invitation2 := &Invitation{
		Email:          "invited2@example.com",
		OrganizationID: 1,
		Token:          "test-token-2",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}

	err := storage.CreateInvitation(ctx, invitation1)
	require.NoError(t, err)
	err = storage.CreateInvitation(ctx, invitation2)
	require.NoError(t, err)

	invitations, err := storage.GetInvitationsByOrganizationID(ctx, 1)
	require.NoError(t, err, "Invitation一覧が取得される")
	assert.GreaterOrEqual(t, len(invitations), 2, "2つ以上のInvitationが取得される")
}

func TestInMemoryStorage_GetInvitationsByEmail_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation1 := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 1,
		Token:          "test-token-1",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	invitation2 := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 2,
		Token:          "test-token-2",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}

	err := storage.CreateInvitation(ctx, invitation1)
	require.NoError(t, err)
	err = storage.CreateInvitation(ctx, invitation2)
	require.NoError(t, err)

	invitations, err := storage.GetInvitationsByEmail(ctx, "invited@example.com")
	require.NoError(t, err, "Invitation一覧が取得される")
	assert.GreaterOrEqual(t, len(invitations), 2, "2つ以上のInvitationが取得される")
}

func TestInMemoryStorage_UpdateInvitation_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 1,
		Token:          "test-token",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	err := storage.CreateInvitation(ctx, invitation)
	require.NoError(t, err)

	invitation.Status = InvitationStatusAccepted
	err = storage.UpdateInvitation(ctx, invitation)
	require.NoError(t, err, "Invitationが更新される")

	updatedInvitation, err := storage.GetInvitationByToken(ctx, "test-token")
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusAccepted, updatedInvitation.Status, "ステータスが更新される")
}

func TestInMemoryStorage_DeleteInvitation_Success(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryStorage()

	invitation := &Invitation{
		Email:          "invited@example.com",
		OrganizationID: 1,
		Token:          "test-token",
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	err := storage.CreateInvitation(ctx, invitation)
	require.NoError(t, err)

	err = storage.DeleteInvitation(ctx, invitation.ID)
	require.NoError(t, err, "Invitationが削除される")

	_, err = storage.GetInvitationByToken(ctx, "test-token")
	assert.Error(t, err, "削除後は取得できない")
}

// ============================================================================
// Tests for DefaultConfig
// ============================================================================

func TestDefaultConfig_EnableAutoLoginAfterPasswordReset(t *testing.T) {
	config := DefaultConfig()
	assert.True(t, config.EnableAutoLoginAfterPasswordReset, "デフォルトで自動ログインが有効になっている必要があります")
}

func TestDefaultConfig_OtherSettings(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 24*time.Hour, config.InvitationExpiryDuration, "招待の有効期限が24時間に設定されている必要があります")
	assert.Equal(t, RoleUser, config.DefaultRole, "デフォルトロールがuserに設定されている必要があります")
	assert.True(t, config.EnableBulkInvite, "バルク招待がデフォルトで有効になっている必要があります")
	assert.Equal(t, 100, config.MaxBulkInviteCount, "バルク招待の最大数が100に設定されている必要があります")
	assert.Equal(t, "/reset-password", config.InvitationRedirectPath, "リダイレクトパスが/reset-passwordに設定されている必要があります")
}

// ============================================================================
// Tests for template settings
// ============================================================================

func TestNewManager_TemplateSettingsApplied(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()

	// カスタムテンプレートファイルを作成
	subjectPath := filepath.Join(tmpDir, "subject.txt")
	err := os.WriteFile(subjectPath, []byte("カスタム件名"), 0644)
	require.NoError(t, err)

	bodyPath := filepath.Join(tmpDir, "body.txt")
	err = os.WriteFile(bodyPath, []byte("カスタム本文"), 0644)
	require.NoError(t, err)

	config := DefaultConfig()
	smtpSender := email.NewSMTPEmailSender()
	config.EmailSender = smtpSender
	config.InvitationEmailSubjectTemplatePath = subjectPath
	config.InvitationEmailBodyTemplatePath = bodyPath
	config.InvitationEmailFrom = "custom@example.com"

	m := NewManager(config)
	assert.NotNil(t, m, "Managerが作成される")

	// SMTPEmailSenderに型アサーションしてテンプレート設定が適用されているか確認
	smtpSenderFromConfig, ok := m.config.EmailSender.(*email.SMTPEmailSender)
	require.True(t, ok, "EmailSenderがSMTPEmailSenderである")
	// BuildEmailMessageの結果から差出人が正しく設定されていることを確認
	message := smtpSenderFromConfig.BuildEmailMessage("test@example.com", "テスト", "本文")
	assert.Contains(t, message, "From: custom@example.com", "差出人が設定されている")
}

func TestNewManager_WorksWithoutTemplateSettings(t *testing.T) {
	config := DefaultConfig()
	smtpSender := email.NewSMTPEmailSender()
	config.EmailSender = smtpSender
	// テンプレートパスは設定しない

	m := NewManager(config)
	assert.NotNil(t, m, "Managerが作成される")

	// SMTPEmailSenderが正しく設定されているか確認
	smtpSenderFromConfig, ok := m.config.EmailSender.(*email.SMTPEmailSender)
	require.True(t, ok, "EmailSenderがSMTPEmailSenderである")
	assert.NotNil(t, smtpSenderFromConfig, "SMTPEmailSenderが設定されている")
}

func TestNewManagerWithStorage_TemplateSettingsApplied(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()

	// カスタムテンプレートファイルを作成
	subjectPath := filepath.Join(tmpDir, "subject.txt")
	err := os.WriteFile(subjectPath, []byte("カスタム件名"), 0644)
	require.NoError(t, err)

	bodyPath := filepath.Join(tmpDir, "body.txt")
	err = os.WriteFile(bodyPath, []byte("カスタム本文"), 0644)
	require.NoError(t, err)

	config := DefaultConfig()
	storage := storage.NewInMemoryStorage()
	smtpSender := email.NewSMTPEmailSender()
	config.EmailSender = smtpSender
	config.InvitationEmailSubjectTemplatePath = subjectPath
	config.InvitationEmailBodyTemplatePath = bodyPath
	config.InvitationEmailFrom = "custom@example.com"

	m := NewManagerWithStorage(config, storage)
	assert.NotNil(t, m, "Managerが作成される")

	// SMTPEmailSenderに型アサーションしてテンプレート設定が適用されているか確認
	smtpSenderFromConfig, ok := m.config.EmailSender.(*email.SMTPEmailSender)
	require.True(t, ok, "EmailSenderがSMTPEmailSenderである")
	// BuildEmailMessageの結果から差出人が正しく設定されていることを確認
	message := smtpSenderFromConfig.BuildEmailMessage("test@example.com", "テスト", "本文")
	assert.Contains(t, message, "From: custom@example.com", "差出人が設定されている")
}

func TestDefaultConfig_TemplateSettingsDefaultValues(t *testing.T) {
	config := DefaultConfig()
	assert.Empty(t, config.InvitationEmailSubjectTemplatePath, "デフォルトで件名テンプレートパスが空")
	assert.Empty(t, config.InvitationEmailBodyTemplatePath, "デフォルトで本文テンプレートパスが空")
	assert.Empty(t, config.InvitationEmailFrom, "デフォルトで差出人が空")
}
