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

