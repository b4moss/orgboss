package orgboss

import (
	"context"
	"os"
	"testing"

	"github.com/aarondl/authboss/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	authbossuser "orgboss/internal/authboss"
	"orgboss/internal/database"
	"orgboss/internal/seed"
	"orgboss/types"
)

// setupAuthbossTestDB はAuthboss統合テスト用のデータベース接続を返す
func setupAuthbossTestDB(t *testing.T) (*gorm.DB, func()) {
	// 環境変数から接続情報を取得
	host := getEnv("DB_HOST", "orgboss-db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "orgboss")
	password := getEnv("DB_PASSWORD", "orgboss")
	dbname := getEnv("DB_NAME", "orgboss")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "データベースに接続できません。Docker Composeが起動していることを確認してください: make up")

	// マイグレーション実行
	err = database.Migrate(db)
	require.NoError(t, err)

	// テスト開始時にデータをクリーンアップ
	ctx := context.Background()
	if err := seed.Cleanup(ctx, db); err != nil {
		t.Logf("テスト開始時のクリーンアップでエラーが発生しました（無視します）: %v", err)
	}

	skipCleanup := os.Getenv("SKIP_CLEANUP") == "true"
	cleanup := func() {
		ctx := context.Background()
		if !skipCleanup {
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

// TestAuthbossIntegration_UserRegistration はAuthbossを使ったユーザー登録の統合テスト
// 注意: このテストはAuthbossの機能をテストするもので、orgbossのコードは直接テストしません
func TestAuthbossIntegration_UserRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupAuthbossTestDB(t)
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

// TestAuthbossIntegration_UserLogin はAuthbossを使ったユーザーログインの統合テスト
// 注意: このテストはAuthbossの機能をテストするもので、orgbossのコードは直接テストしません
func TestAuthbossIntegration_UserLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupAuthbossTestDB(t)
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

// Authbossのストレージ実装（簡易版）
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
	org := &types.Organization{
		Name: "テスト組織",
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

	// AuthbossのUserインターフェースを実装した構造体に変換
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(user.Email)
	authbossUser.PutPassword(user.Password)

	return authbossUser, nil
}


