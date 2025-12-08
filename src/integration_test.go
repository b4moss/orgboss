package orgboss

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"orgboss/internal/database"
	"orgboss/internal/seed"
	"orgboss/internal/storage"
)

// setupTestDB は既存のPostgreSQLコンテナに接続し、データベース接続を返す
// 注意: Docker Composeが起動している必要があります（make up）
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	// 環境変数から接続情報を取得（デフォルト値はcompose.ymlの設定に合わせる）
	host := getEnv("DB_HOST", "orgboss-db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "orgboss")
	password := getEnv("DB_PASSWORD", "orgboss")
	dbname := getEnv("DB_NAME", "orgboss") // 既存のデータベースを使用

	// データベース接続文字列を構築
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	}

	// GORMで接続
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "データベースに接続できません。Docker Composeが起動していることを確認してください: make up")

	// マイグレーション実行
	err = database.Migrate(db)
	require.NoError(t, err)

	// テスト開始時にデータをクリーンアップ（各テストをクリーンな状態で開始）
	ctx := context.Background()
	if err := seed.Cleanup(ctx, db); err != nil {
		t.Logf("テスト開始時のクリーンアップでエラーが発生しました（無視します）: %v", err)
	}

	// クリーンアップ関数（テスト終了時にテストデータを削除）
	// SKIP_CLEANUP環境変数が設定されている場合はクリーンアップをスキップ
	skipCleanup := os.Getenv("SKIP_CLEANUP") == "true"
	cleanup := func() {
		ctx := context.Background()
		// SKIP_CLEANUPが設定されていない場合のみクリーンアップ
		if !skipCleanup {
			// テストデータをクリーンアップ
			if err := seed.Cleanup(ctx, db); err != nil {
				t.Logf("クリーンアップ中にエラーが発生しました: %v", err)
			}
		} else {
			t.Logf("SKIP_CLEANUP=true が設定されているため、テストデータを保持します")
		}
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

// getEnv は環境変数を取得し、デフォルト値を返す
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestIntegration_CreateOrganizationWithUser は実際のデータベースを使った統合テスト
func TestIntegration_CreateOrganizationWithUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManagerWithStorage(config, postgresStorage)

	orgName := "統合テスト組織"
	userEmail := "integration@example.com"

	org, user, err := m.CreateOrganizationWithUser(ctx, orgName, userEmail)

	require.NoError(t, err)
	require.NotNil(t, org)
	require.NotNil(t, user)
	assert.Equal(t, orgName, org.Name)
	assert.Equal(t, userEmail, user.Email)
	assert.Equal(t, RoleManager, user.Role)
	assert.Equal(t, org.ID, user.OrganizationID)

	// データベースから取得して確認
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, orgName, retrievedOrg.Name)

	retrievedUser, err := postgresStorage.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, userEmail, retrievedUser.Email)
	assert.NotEmpty(t, retrievedUser.Password, "パスワードが設定されている必要があります")
	assert.Equal(t, RoleManager, retrievedUser.Role)
}

// TestIntegration_InviteUser は招待機能の統合テスト
func TestIntegration_InviteUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManagerWithStorage(config, postgresStorage)

	// 組織とマネージャーを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "統合テスト組織", "manager@example.com")
	require.NoError(t, err)

	// ユーザーを招待
	email := "invited@example.com"
	invitation, err := m.InviteUser(ctx, org.ID, email)

	require.NoError(t, err)
	require.NotNil(t, invitation)
	assert.Equal(t, email, invitation.Email)
	assert.Equal(t, org.ID, invitation.OrganizationID)
	assert.Equal(t, InvitationStatusPending, invitation.Status)
	assert.NotEmpty(t, invitation.Token)

	// データベースから取得して確認
	retrievedInvitation, err := postgresStorage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, email, retrievedInvitation.Email)
	assert.Equal(t, org.ID, retrievedInvitation.OrganizationID)
}

// TestIntegration_AcceptInvitation は招待承諾の統合テスト
func TestIntegration_AcceptInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManagerWithStorage(config, postgresStorage)

	// 組織とマネージャーを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "統合テスト組織", "manager@example.com")
	require.NoError(t, err)

	// ユーザーを招待
	invitation, err := m.InviteUser(ctx, org.ID, "invited@example.com")
	require.NoError(t, err)

	// 招待を承諾
	user, err := m.AcceptInvitation(ctx, invitation.Token)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "invited@example.com", user.Email)
	assert.Equal(t, RoleUser, user.Role)
	assert.Equal(t, org.ID, user.OrganizationID)

	// データベースから取得して確認
	retrievedUser, err := postgresStorage.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "invited@example.com", retrievedUser.Email)
	assert.Equal(t, RoleUser, retrievedUser.Role)
	assert.NotEmpty(t, retrievedUser.Password, "パスワードが設定されている必要があります")

	// 招待のステータスがpendingのままであることを確認（パスワード更新時にacceptedになる）
	retrievedInvitation, err := postgresStorage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusPending, retrievedInvitation.Status, "パスワード更新前はpendingのまま")

	// パスワードを更新して、Invitationがacceptedになることを確認
	err = m.UpdatePassword(ctx, user.ID, org.ID, "newpassword123")
	require.NoError(t, err)

	// 招待のステータスがacceptedになっていることを確認
	retrievedInvitation, err = postgresStorage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusAccepted, retrievedInvitation.Status, "パスワード更新後はacceptedになる")
}

// TestIntegration_WithSeedData はシードデータを使った統合テスト
func TestIntegration_WithSeedData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// シードデータを投入
	seedData, err := seed.Seed(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, seedData)
	require.Len(t, seedData.Organizations, 2)
	require.Len(t, seedData.Users, 3)
	require.Len(t, seedData.Invitations, 3)

	// PostgresStorageを作成
	postgresStorage := storage.NewPostgresStorage(db)

	// シードデータの組織を取得
	org1 := seedData.Organizations[0]
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org1.ID)
	require.NoError(t, err)
	assert.Equal(t, org1.Name, retrievedOrg.Name)

	// シードデータのユーザーを取得
	user1 := seedData.Users[0]
	retrievedUser, err := postgresStorage.GetUser(ctx, user1.ID)
	require.NoError(t, err)
	assert.Equal(t, user1.Email, retrievedUser.Email)
	assert.Equal(t, user1.Role, retrievedUser.Role)

	// シードデータの招待を取得
	invitation1 := seedData.Invitations[0]
	retrievedInvitation, err := postgresStorage.GetInvitationByToken(ctx, invitation1.Token)
	require.NoError(t, err)
	assert.Equal(t, invitation1.Email, retrievedInvitation.Email)
	assert.Equal(t, invitation1.Status, retrievedInvitation.Status)

	// 組織IDでユーザーを取得
	users, err := postgresStorage.GetUsersByOrganizationID(ctx, org1.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2) // 少なくとも2人のユーザーがいる

	// クリーンアップ
	err = seed.Cleanup(ctx, db)
	require.NoError(t, err)
}

// TestIntegration_DeleteUser はユーザー削除の統合テスト
func TestIntegration_DeleteUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManagerWithStorage(config, postgresStorage)

	// 組織とマネージャーを作成
	org, _, err := m.CreateOrganizationWithUser(ctx, "統合テスト組織", "manager@example.com")
	require.NoError(t, err)

	// ユーザーを招待して承諾
	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)
	user, err := m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	// ユーザーを削除
	err = m.DeleteUser(ctx, user.ID, org.ID)
	require.NoError(t, err)

	// ユーザーが削除されていることを確認（論理削除）
	retrievedUser, err := postgresStorage.GetUser(ctx, user.ID)
	if err == nil {
		// 論理削除の場合、DeletedAtが設定されている
		assert.NotNil(t, retrievedUser.DeletedAt)
	}
}

// TestIntegration_DeleteOrganization は組織削除の統合テスト
func TestIntegration_DeleteOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	config.EmailSender = &mockEmailSender{}
	m := NewManagerWithStorage(config, postgresStorage)

	// 組織とマネージャーを作成
	org, manager, err := m.CreateOrganizationWithUser(ctx, "統合テスト組織", "manager@example.com")
	require.NoError(t, err)

	// ユーザーを招待して承諾
	invitation, err := m.InviteUser(ctx, org.ID, "user@example.com")
	require.NoError(t, err)
	_, err = m.AcceptInvitation(ctx, invitation.Token)
	require.NoError(t, err)

	// マネージャーを削除（組織も削除される）
	err = m.DeleteUser(ctx, manager.ID, org.ID)
	require.NoError(t, err)

	// 組織が削除されていることを確認（論理削除）
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org.ID)
	if err == nil {
		// 論理削除の場合、DeletedAtが設定されている
		assert.NotNil(t, retrievedOrg.DeletedAt)
	}
}

