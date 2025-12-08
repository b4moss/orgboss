package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"orgboss/types"
)

// Connect はデータベースに接続し、GORMインスタンスを返す
func Connect() (*gorm.DB, error) {
	// 環境変数から接続情報を取得
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "orgboss")
	password := getEnv("DB_PASSWORD", "orgboss")
	dbname := getEnv("DB_NAME", "orgboss")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// DATABASE_URLが設定されている場合は優先
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// Migrate はデータベーススキーマをマイグレーションする
func Migrate(db *gorm.DB) error {
	// 既存のusersテーブルにpasswordカラムがない場合の対応
	// passwordカラムが存在するか確認
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'password'").Scan(&count).Error; err != nil {
		// エラーは無視（テーブルが存在しない場合など）
	}
	
	if count == 0 {
		// passwordカラムが存在しない場合は追加（nullableで、既存データ対応）
		if err := db.Exec("ALTER TABLE users ADD COLUMN password TEXT DEFAULT ''").Error; err != nil {
			// エラーは無視（既に存在する場合など）
		}
		// 既存のnull値を空文字列に更新
		if err := db.Exec("UPDATE users SET password = '' WHERE password IS NULL").Error; err != nil {
			// エラーは無視
		}
	}

	// organizationsテーブルにsignatureカラムがない場合の対応
	var signatureCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = 'signature'").Scan(&signatureCount).Error; err != nil {
		// エラーは無視（テーブルが存在しない場合など）
	}
	
	if signatureCount == 0 {
		// signatureカラムが存在しない場合は追加（まずnullableで追加）
		if err := db.Exec("ALTER TABLE organizations ADD COLUMN signature TEXT").Error; err != nil {
			// エラーは無視（既に存在する場合など）
		}
	}
	
	// 既存のレコードでsignatureが空（NULLまたは空文字列）の場合にランダムな値を設定
	// PostgreSQLのgen_random_uuid()とmd5()を使ってランダムな文字列を生成（24文字に切り詰め）
	if err := db.Exec("UPDATE organizations SET signature = LEFT(md5(gen_random_uuid()::text || id::text || random()::text), 24) WHERE signature IS NULL OR signature = ''").Error; err != nil {
		// エラーは無視
	}
	
	// NOT NULL制約を追加（既に存在する場合はエラーになるが無視）
	if err := db.Exec("ALTER TABLE organizations ALTER COLUMN signature SET NOT NULL").Error; err != nil {
		// エラーは無視（既にNOT NULLの場合など）
	}
	
	// ユニークインデックスを追加
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_signature ON organizations(signature)").Error; err != nil {
		// エラーは無視（既に存在する場合など）
	}

	// signature_typeカラムが存在する場合は削除（不要になったため）
	var signatureTypeCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = 'signature_type'").Scan(&signatureTypeCount).Error; err != nil {
		// エラーは無視（テーブルが存在しない場合など）
	}
	
	if signatureTypeCount > 0 {
		// signature_typeカラムのインデックスを削除
		if err := db.Exec("DROP INDEX IF EXISTS idx_organizations_signature_type").Error; err != nil {
			// エラーは無視
		}
		// signature_typeカラムを削除
		if err := db.Exec("ALTER TABLE organizations DROP COLUMN IF EXISTS signature_type").Error; err != nil {
			// エラーは無視
		}
	}

	// AutoMigrateを実行
	return db.AutoMigrate(
		&types.Organization{},
		&types.User{},
		&types.Invitation{},
	)
}

// getEnv は環境変数を取得し、デフォルト値を返す
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

