package orgboss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	authbossuser "orgboss/internal/authboss"
	"orgboss/internal/database"
	"orgboss/internal/email"
	"orgboss/internal/seed"
	"orgboss/internal/storage"
	"orgboss/types"
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

	// GORMで接続（テスト時はエラーレベルのログのみ出力）
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // record not foundなどのINFOログを抑制
	})
	require.NoError(t, err, "データベースに接続できません。Docker Composeが起動していることを確認してください: make up")

	// マイグレーション実行
	err = database.Migrate(db)
	require.NoError(t, err)

	// SKIP_CLEANUP環境変数が設定されている場合はクリーンアップをスキップ
	skipCleanup := os.Getenv("SKIP_CLEANUP") == "true"

	// テスト開始時にデータをクリーンアップ（各テストをクリーンな状態で開始）
	// 注意: テストの独立性を保つため、開始時のクリーンアップは常に実行する
	// SKIP_CLEANUPは終了時のクリーンアップのみに影響する
	ctx := context.Background()
	if err := seed.Cleanup(ctx, db); err != nil {
		t.Logf("テスト開始時のクリーンアップでエラーが発生しました（無視します）: %v", err)
	}

	// クリーンアップ関数（テスト終了時にテストデータを削除）
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
	
	// Signatureの検証
	assert.NotEmpty(t, retrievedOrg.Signature, "Signatureが設定されている")
	
	// SignatureでOrganizationを取得できることを確認
	retrievedOrgBySignature, err := postgresStorage.GetOrganizationBySignature(ctx, retrievedOrg.Signature)
	require.NoError(t, err)
	assert.Equal(t, retrievedOrg.ID, retrievedOrgBySignature.ID, "SignatureでOrganizationが取得できる")

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
	// 注意: SKIP_CLEANUP=trueの場合、cleanup()でデータが保持される
	// データを確認したい場合は、cleanup()を呼び出さないようにするか、
	// またはSKIP_CLEANUP=trueでテストを実行する
	defer cleanup()

	ctx := context.Background()

	// シードデータを投入
	seedData, err := seed.Seed(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, seedData)
	require.Len(t, seedData.Organizations, 2)
	require.Len(t, seedData.Users, 3)
	require.Len(t, seedData.Invitations, 3)
	
	// デバッグ用: シードデータが作成されたことを確認
	t.Logf("シードデータ作成: Organizations=%d, Users=%d, Invitations=%d", 
		len(seedData.Organizations), len(seedData.Users), len(seedData.Invitations))

	// PostgresStorageを作成
	postgresStorage := storage.NewPostgresStorage(db)

	// シードデータの組織を取得
	org1 := seedData.Organizations[0]
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org1.ID)
	require.NoError(t, err)
	assert.Equal(t, org1.Name, retrievedOrg.Name)
	
	// Signatureの検証
	assert.NotEmpty(t, retrievedOrg.Signature, "Signatureが設定されている")
	assert.Equal(t, org1.Signature, retrievedOrg.Signature, "Signatureが一致する")
	
	// SignatureでOrganizationを取得できることを確認
	retrievedOrgBySignature, err := postgresStorage.GetOrganizationBySignature(ctx, org1.Signature)
	require.NoError(t, err)
	assert.Equal(t, org1.ID, retrievedOrgBySignature.ID, "SignatureでOrganizationが取得できる")
	
	// org2も確認
	org2 := seedData.Organizations[1]
	retrievedOrg2, err := postgresStorage.GetOrganization(ctx, org2.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, retrievedOrg2.Signature, "org2のSignatureが設定されている")

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

	// Signatureで存在しないOrganizationを取得しようとする（エラーになることを確認）
	_, err = postgresStorage.GetOrganizationBySignature(ctx, "non-existent-signature")
	assert.Error(t, err, "存在しないSignatureではエラーが返される")
	
	// 注意: defer cleanup()でクリーンアップされるため、ここでの明示的なクリーンアップは不要
	// SKIP_CLEANUP=trueが設定されている場合は、defer cleanup()でもデータが保持される
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

// TestIntegration_AuthbossUserRegistration はAuthbossを使ったユーザー登録の統合テスト
// 注意: このテストはAuthbossの機能をテストするもので、orgbossのコードは直接テストしません
func TestIntegration_AuthbossUserRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// AuthbossのServerStorerを実装
	serverStorer := &authbossServerStorer{db: db}

	// ユーザー登録のテストデータ
	email := "testuser@example.com"
	password := "testpassword123"

	// パスワードをハッシュ化（Authbossと同じ方法）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "パスワードのハッシュ化に失敗しました")

	// AuthbossのUserインターフェースを実装した構造体を作成
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(email)
	authbossUser.PutPassword(string(hashedPassword)) // ハッシュ化されたパスワードを設定

	// ユーザーを保存（Authboss経由）
	err = serverStorer.Save(ctx, authbossUser)
	require.NoError(t, err, "Authboss経由でユーザーを保存できません")

	// データベースから直接確認
	var dbUser types.User
	err = db.WithContext(ctx).Where("email = ?", email).First(&dbUser).Error
	require.NoError(t, err, "データベースからユーザーを取得できません")
	assert.Equal(t, email, dbUser.Email)
	assert.NotEmpty(t, dbUser.Password, "パスワードハッシュが保存されている必要があります")
	assert.NotEqual(t, password, dbUser.Password, "パスワードはハッシュ化されている必要があります")
}

// TestIntegration_AuthbossUserLogin はAuthbossを使ったユーザーログインの統合テスト
// 注意: このテストはAuthbossの機能をテストするもので、orgbossのコードは直接テストしません
func TestIntegration_AuthbossUserLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// AuthbossのServerStorerを実装
	serverStorer := &authbossServerStorer{db: db}

	// テスト用のユーザーを作成
	email := "loginuser@example.com"
	password := "loginpassword123"

	// パスワードをハッシュ化（Authbossと同じ方法）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "パスワードのハッシュ化に失敗しました")

	// AuthbossのUserインターフェースを実装した構造体を作成して保存
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(email)
	authbossUser.PutPassword(string(hashedPassword)) // ハッシュ化されたパスワードを設定

	err = serverStorer.Save(ctx, authbossUser)
	require.NoError(t, err, "ユーザーを保存できません")

	// ログインのテスト（Authboss経由でユーザーを取得）
	retrievedUser, err := serverStorer.Load(ctx, email)
	require.NoError(t, err, "Authboss経由でユーザーを取得できません")
	assert.NotNil(t, retrievedUser, "ユーザーが取得できる必要があります")

	// パスワードの検証
	if retrievedUser != nil {
		// authbossuser.Userに型アサーション
		authbossUser, ok := retrievedUser.(*authbossuser.User)
		require.True(t, ok, "authbossuser.User型に変換できる必要があります")
		
		// Authbossはパスワードをハッシュ化して保存するため、
		// 取得したパスワードハッシュが存在することを確認
		retrievedPasswordHash := authbossUser.GetPassword()
		assert.NotEmpty(t, retrievedPasswordHash, "パスワードハッシュが取得できる必要があります")
		assert.NotEqual(t, password, retrievedPasswordHash, "パスワードはハッシュ化されている必要があります")
		
		// パスワードの検証（bcryptで検証）
		err := bcrypt.CompareHashAndPassword([]byte(retrievedPasswordHash), []byte(password))
		assert.NoError(t, err, "ハッシュ化されたパスワードが正しく検証できる必要があります")
	}
}

// TestIntegration_AuthbossUserLogin_PendingInvitation はpendingのinvitationのユーザーがログインを拒否されることをテストする
func TestIntegration_AuthbossUserLogin_PendingInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// AuthbossのServerStorerを実装
	serverStorer := &authbossServerStorer{db: db}

	// テスト用の組織を作成
	signatureBytes := make([]byte, 12)
	_, err := rand.Read(signatureBytes)
	require.NoError(t, err)
	signature := hex.EncodeToString(signatureBytes)

	org := &types.Organization{
		Name:      "テスト組織",
		Signature: signature,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = db.WithContext(ctx).Create(org).Error
	require.NoError(t, err, "組織の作成に失敗しました")

	// テスト用のユーザーを作成（AcceptInvitationで作成された状態をシミュレート）
	email := "pendinguser@example.com"
	password := "randompassword123"

	// パスワードをハッシュ化（Authbossと同じ方法）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "パスワードのハッシュ化に失敗しました")

	// Userを作成
	user := &types.User{
		Email:          email,
		Password:       string(hashedPassword),
		OrganizationID: org.ID,
		Role:           types.RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err = db.WithContext(ctx).Create(user).Error
	require.NoError(t, err, "ユーザーの作成に失敗しました")

	// pendingのinvitationを作成（AcceptInvitation後、パスワード更新前の状態）
	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	require.NoError(t, err)
	token := hex.EncodeToString(tokenBytes)

	invitation := &types.Invitation{
		Email:          email,
		OrganizationID: org.ID,
		Token:          token,
		ExpiresAt:      time.Now().Add(24 * time.Hour),
		Status:         types.InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	err = db.WithContext(ctx).Create(invitation).Error
	require.NoError(t, err, "invitationの作成に失敗しました")

	// ログインを試みる（pendingのinvitationがあるため、エラーが返されるはず）
	retrievedUser, err := serverStorer.Load(ctx, email)
	assert.Error(t, err, "pendingのinvitationがある場合、ログインは拒否される必要があります")
	assert.Nil(t, retrievedUser, "ユーザーは取得できない必要があります")
	assert.Equal(t, types.ErrInvitationPending, err, "ErrInvitationPendingエラーが返される必要があります")

	// invitationをacceptedに更新（パスワード更新後の状態をシミュレート）
	invitation.Status = types.InvitationStatusAccepted
	err = db.WithContext(ctx).Save(invitation).Error
	require.NoError(t, err, "invitationの更新に失敗しました")

	// 再度ログインを試みる（acceptedになったため、ログインできるはず）
	retrievedUser, err = serverStorer.Load(ctx, email)
	assert.NoError(t, err, "acceptedのinvitationがある場合、ログインできる必要があります")
	assert.NotNil(t, retrievedUser, "ユーザーが取得できる必要があります")
}

// authbossServerStorer はAuthbossのストレージ実装（テスト用）
type authbossServerStorer struct {
	db *gorm.DB
}

func (s *authbossServerStorer) Save(ctx context.Context, user authboss.User) error {
	// AuthbossのUserインターフェースから値を取得
	email := user.GetPID()
	
	// authbossuser.Userに型アサーションしてパスワードを取得
	authbossUser, ok := user.(*authbossuser.User)
	if !ok {
		// 型アサーションに失敗した場合はエラー
		// authbossuser.User型である必要があります
		return types.ErrUserNotFound
	}
	password := authbossUser.GetPassword() // Authbossがハッシュ化したパスワード

	// orgbossのUserモデルに変換
	dbUser := &types.User{
		Email:    email,
		Password: password, // Authbossがハッシュ化したパスワード
		Role:     types.RoleUser,
	}

	// 組織も作成（テスト用）
	// Signatureを生成（12バイト、24文字の16進数）
	signatureBytes := make([]byte, 12)
	if _, err := rand.Read(signatureBytes); err != nil {
		return err
	}
	signature := hex.EncodeToString(signatureBytes)
	
	org := &types.Organization{
		Name:      "テスト組織",
		Signature: signature,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(org).Error; err != nil {
		return err
	}
	dbUser.OrganizationID = org.ID

	return s.db.WithContext(ctx).Create(dbUser).Error
}

func (s *authbossServerStorer) Load(ctx context.Context, key string) (authboss.User, error) {
	var user types.User
	if err := s.db.WithContext(ctx).Where("email = ?", key).First(&user).Error; err != nil {
		return nil, err
	}

	// pendingのinvitationがあるかチェック
	var pendingInvitations []types.Invitation
	if err := s.db.WithContext(ctx).Where("email = ? AND status = ?", key, types.InvitationStatusPending).Find(&pendingInvitations).Error; err != nil {
		return nil, err
	}
	if len(pendingInvitations) > 0 {
		// pendingのinvitationがある場合はログインを拒否
		return nil, types.ErrInvitationPending
	}

	// AuthbossのUserインターフェースを実装した構造体に変換
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(user.Email)
	authbossUser.PutPassword(user.Password)

	return authbossUser, nil
}

// MailpitMessage はMailpitのAPIから返されるメールメッセージの構造
type MailpitMessage struct {
	ID      string   `json:"ID"`
	From    MailpitAddress `json:"From"`
	To      []MailpitAddress `json:"To"`
	Subject string   `json:"Subject"`
	Text    string   `json:"Text"`
	HTML    string   `json:"HTML"`
}

// MailpitAddress はMailpitのメールアドレス構造
type MailpitAddress struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

// MailpitMessagesResponse はMailpitのメール一覧APIのレスポンス
type MailpitMessagesResponse struct {
	Total int              `json:"total"`
	Count int              `json:"count"`
	Start int              `json:"start"`
	Messages []MailpitMessage `json:"messages"`
}

// getMailpitMessages はMailpitのAPIからメール一覧を取得する
func getMailpitMessages(t *testing.T) ([]MailpitMessage, error) {
	mailpitURL := getEnv("MAILPIT_URL", "http://mailpit:8025")
	url := fmt.Sprintf("%s/api/v1/messages", mailpitURL)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from Mailpit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Mailpit API returned status %d: %s", resp.StatusCode, string(body))
	}

	var messagesResp MailpitMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&messagesResp); err != nil {
		return nil, fmt.Errorf("failed to decode Mailpit response: %w", err)
	}

	return messagesResp.Messages, nil
}

// getMailpitMessage はMailpitのAPIから特定のメールを取得する
func getMailpitMessage(t *testing.T, messageID string) (*MailpitMessage, error) {
	mailpitURL := getEnv("MAILPIT_URL", "http://mailpit:8025")
	url := fmt.Sprintf("%s/api/v1/message/%s", mailpitURL, messageID)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get message from Mailpit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Mailpit API returned status %d: %s", resp.StatusCode, string(body))
	}

	var message MailpitMessage
	if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
		return nil, fmt.Errorf("failed to decode Mailpit response: %w", err)
	}

	return &message, nil
}

// clearMailpitMessages はMailpitのメールをクリアする（テスト用）
func clearMailpitMessages(t *testing.T) error {
	mailpitURL := getEnv("MAILPIT_URL", "http://mailpit:8025")
	url := fmt.Sprintf("%s/api/v1/messages", mailpitURL)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete messages from Mailpit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Mailpit API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// TestIntegration_EmailSending はメール送信の統合テスト
func TestIntegration_EmailSending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Mailpitのメールをクリア（テストの独立性を保つため）
	if err := clearMailpitMessages(t); err != nil {
		t.Logf("Mailpitのメールクリアに失敗しました（無視します）: %v", err)
	}

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	
	// SMTPEmailSenderを使用
	smtpSender := email.NewSMTPEmailSender()
	config.EmailSender = smtpSender
	m := NewManagerWithStorage(config, postgresStorage)

	// 組織とマネージャーを作成
	org, manager, err := m.CreateOrganizationWithUser(ctx, "メールテスト組織", "manager@example.com")
	require.NoError(t, err)

	// ユーザーを招待（メール送信が実行される）
	inviteEmail := "invited@example.com"
	invitation, err := m.InviteUser(ctx, org.ID, inviteEmail)
	require.NoError(t, err)
	require.NotNil(t, invitation)

	// 少し待ってからMailpitのAPIでメールを確認
	time.Sleep(500 * time.Millisecond)

	// Mailpitからメール一覧を取得
	messages, err := getMailpitMessages(t)
	require.NoError(t, err, "Mailpitからメールを取得できません")
	require.Greater(t, len(messages), 0, "少なくとも1通のメールが送信されている必要があります")

	// 最新のメールを取得（最初のメールが最新）
	latestMessageSummary := messages[0]
	
	// メール詳細を取得（本文を含む）
	latestMessage, err := getMailpitMessage(t, latestMessageSummary.ID)
	require.NoError(t, err, "Mailpitからメール詳細を取得できません")
	
	// メールの内容を検証
	assert.Equal(t, "組織への招待", latestMessage.Subject, "件名が正しい")
	assert.Contains(t, latestMessage.To[0].Address, inviteEmail, "宛先が正しい")
	assert.Contains(t, latestMessage.Text, invitation.Token, "本文にトークンが含まれている")
	assert.Contains(t, latestMessage.Text, invitation.ExpiresAt.Format("2006-01-02"), "本文に有効期限が含まれている")

	// ResendInvitationのテスト
	invitation2, err := m.InviteUser(ctx, org.ID, "invited2@example.com")
	require.NoError(t, err)

	// 招待を再送信（トークンが再生成される）
	err = m.ResendInvitation(ctx, invitation2.ID, org.ID, manager.ID)
	require.NoError(t, err)

	// 再送信後に更新された招待を取得（トークンが再生成されているため）
	updatedInvitation2, err := postgresStorage.GetInvitationByID(ctx, invitation2.ID)
	require.NoError(t, err)
	assert.NotEqual(t, invitation2.Token, updatedInvitation2.Token, "再送信によりトークンが再生成されている")

	// 少し待ってからMailpitのAPIでメールを確認
	time.Sleep(500 * time.Millisecond)

	// Mailpitからメール一覧を再取得
	messages2, err := getMailpitMessages(t)
	require.NoError(t, err)
	require.Greater(t, len(messages2), len(messages), "再送信によりメールが追加されている")

	// 最新のメール（再送信されたメール）を確認
	latestMessage2Summary := messages2[0]
	
	// メール詳細を取得（本文を含む）
	latestMessage2, err := getMailpitMessage(t, latestMessage2Summary.ID)
	require.NoError(t, err, "Mailpitから再送信メール詳細を取得できません")
	
	assert.Equal(t, "組織への招待", latestMessage2.Subject, "再送信メールの件名が正しい")
	assert.Contains(t, latestMessage2.To[0].Address, "invited2@example.com", "再送信メールの宛先が正しい")
	assert.Contains(t, latestMessage2.Text, updatedInvitation2.Token, "再送信メールの本文に再生成されたトークンが含まれている")
	assert.Contains(t, latestMessage2.Text, updatedInvitation2.ExpiresAt.Format("2006-01-02"), "再送信メールの本文に更新された有効期限が含まれている")
}

