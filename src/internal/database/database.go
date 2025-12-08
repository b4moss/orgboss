package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/b4m-oss/orgboss/types"
)

// Connect connects to the database and returns a GORM instance
func Connect() (*gorm.DB, error) {
	// Get connection information from environment variables
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "orgboss")
	password := getEnv("DB_PASSWORD", "orgboss")
	dbname := getEnv("DB_NAME", "orgboss")
	sslmode := getEnv("DB_SSLMODE", "disable")

	// DATABASE_URL takes precedence if set
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

// Migrate migrates the database schema
func Migrate(db *gorm.DB) error {
	// Handle case where password column doesn't exist in existing users table
	// Check if password column exists
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'password'").Scan(&count).Error; err != nil {
		// Ignore errors (e.g., table doesn't exist)
	}
	
	if count == 0 {
		// Add password column if it doesn't exist (nullable for existing data compatibility)
		if err := db.Exec("ALTER TABLE users ADD COLUMN password TEXT DEFAULT ''").Error; err != nil {
			// Ignore errors (e.g., already exists)
		}
		// Update existing null values to empty string
		if err := db.Exec("UPDATE users SET password = '' WHERE password IS NULL").Error; err != nil {
			// Ignore errors
		}
	}

	// Handle case where signature column doesn't exist in organizations table
	var signatureCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = 'signature'").Scan(&signatureCount).Error; err != nil {
		// Ignore errors (e.g., table doesn't exist)
	}
	
	if signatureCount == 0 {
		// Add signature column if it doesn't exist (first add as nullable)
		if err := db.Exec("ALTER TABLE organizations ADD COLUMN signature TEXT").Error; err != nil {
			// Ignore errors (e.g., already exists)
		}
	}
	
	// Set random values for existing records where signature is empty (NULL or empty string)
	// Generate random string using PostgreSQL's gen_random_uuid() and md5() (truncated to 24 characters)
	if err := db.Exec("UPDATE organizations SET signature = LEFT(md5(gen_random_uuid()::text || id::text || random()::text), 24) WHERE signature IS NULL OR signature = ''").Error; err != nil {
		// Ignore errors
	}
	
	// Add NOT NULL constraint (ignore error if already exists)
	if err := db.Exec("ALTER TABLE organizations ALTER COLUMN signature SET NOT NULL").Error; err != nil {
		// Ignore errors (e.g., already NOT NULL)
	}
	
	// Add unique index
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_organizations_signature ON organizations(signature)").Error; err != nil {
		// Ignore errors (e.g., already exists)
	}

	// Remove signature_type column if it exists (no longer needed)
	var signatureTypeCount int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'organizations' AND column_name = 'signature_type'").Scan(&signatureTypeCount).Error; err != nil {
		// Ignore errors (e.g., table doesn't exist)
	}
	
	if signatureTypeCount > 0 {
		// Drop signature_type column index
		if err := db.Exec("DROP INDEX IF EXISTS idx_organizations_signature_type").Error; err != nil {
			// Ignore errors
		}
		// Drop signature_type column
		if err := db.Exec("ALTER TABLE organizations DROP COLUMN IF EXISTS signature_type").Error; err != nil {
			// Ignore errors
		}
	}

	// Execute AutoMigrate
	return db.AutoMigrate(
		&types.Organization{},
		&types.User{},
		&types.Invitation{},
	)
}

// getEnv gets an environment variable and returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

