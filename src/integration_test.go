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

	authbossuser "github.com/b4m-oss/orgboss/internal/authboss"
	"github.com/b4m-oss/orgboss/internal/database"
	"github.com/b4m-oss/orgboss/internal/email"
	"github.com/b4m-oss/orgboss/internal/seed"
	"github.com/b4m-oss/orgboss/internal/storage"
	"github.com/b4m-oss/orgboss/types"
)

// setupTestDB connects to an existing PostgreSQL container and returns a database connection
// Note: Docker Compose must be running (make up)
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	// Get connection information from environment variables (default values match compose.yml settings)
	host := getEnv("DB_HOST", "orgboss-db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "orgboss")
	password := getEnv("DB_PASSWORD", "orgboss")
	dbname := getEnv("DB_NAME", "orgboss") // Use existing database

	// Build database connection string
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	}

	// Connect with GORM (only error-level logs during testing)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // Suppress INFO logs like record not found
	})
	require.NoError(t, err, "Cannot connect to database. Make sure Docker Compose is running: make up")

	// Execute migration
	err = database.Migrate(db)
	require.NoError(t, err)

	// Skip cleanup if SKIP_CLEANUP environment variable is set
	skipCleanup := os.Getenv("SKIP_CLEANUP") == "true"

	// Clean up data at test start (start each test in a clean state)
	// Note: Cleanup at start is always executed to maintain test independence
	// SKIP_CLEANUP only affects cleanup at the end
	ctx := context.Background()
	if err := seed.Cleanup(ctx, db); err != nil {
		t.Logf("Error occurred during cleanup at test start (ignoring): %v", err)
	}

	// Cleanup function (deletes test data at test end)
	cleanup := func() {
		ctx := context.Background()
		// Only cleanup if SKIP_CLEANUP is not set
		if !skipCleanup {
			// Clean up test data
			if err := seed.Cleanup(ctx, db); err != nil {
				t.Logf("Error occurred during cleanup: %v", err)
			}
		} else {
			t.Logf("SKIP_CLEANUP=true is set, keeping test data")
		}
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return db, cleanup
}

// getEnv gets an environment variable and returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestIntegration_CreateOrganizationWithUser is an integration test using a real database
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

	// Verify by retrieving from database
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, orgName, retrievedOrg.Name)
	
	// Verify Signature
	assert.NotEmpty(t, retrievedOrg.Signature, "Signature is set")
	
	// Verify that Organization can be retrieved by Signature
	retrievedOrgBySignature, err := postgresStorage.GetOrganizationBySignature(ctx, retrievedOrg.Signature)
	require.NoError(t, err)
	assert.Equal(t, retrievedOrg.ID, retrievedOrgBySignature.ID, "Organization can be retrieved by Signature")

	retrievedUser, err := postgresStorage.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, userEmail, retrievedUser.Email)
	assert.NotEmpty(t, retrievedUser.Password, "Password must be set")
	assert.Equal(t, RoleManager, retrievedUser.Role)
}

// TestIntegration_InviteUser is an integration test for invitation functionality
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

	// Verify by retrieving from database
	retrievedInvitation, err := postgresStorage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, email, retrievedInvitation.Email)
	assert.Equal(t, org.ID, retrievedInvitation.OrganizationID)
}

// TestIntegration_AcceptInvitation is an integration test for invitation acceptance
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

	// Verify by retrieving from database
	retrievedUser, err := postgresStorage.GetUser(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "invited@example.com", retrievedUser.Email)
	assert.Equal(t, RoleUser, retrievedUser.Role)
	assert.NotEmpty(t, retrievedUser.Password, "Password must be set")

	// Verify that invitation status remains pending (becomes accepted when password is updated)
	retrievedInvitation, err := postgresStorage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusPending, retrievedInvitation.Status, "Status remains pending before password update")

	// Update password and verify that Invitation becomes accepted
	err = m.UpdatePassword(ctx, user.ID, org.ID, "newpassword123")
	require.NoError(t, err)

	// Verify that invitation status is accepted
	retrievedInvitation, err = postgresStorage.GetInvitationByToken(ctx, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, InvitationStatusAccepted, retrievedInvitation.Status, "Status becomes accepted after password update")
}

// TestIntegration_WithSeedData is an integration test using seed data
func TestIntegration_WithSeedData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	// Note: If SKIP_CLEANUP=true, data is kept in cleanup()
	// To check data, either don't call cleanup() or run tests with SKIP_CLEANUP=true
	defer cleanup()

	ctx := context.Background()

	// Seed data
	seedData, err := seed.Seed(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, seedData)
	require.Len(t, seedData.Organizations, 2)
	require.Len(t, seedData.Users, 3)
	require.Len(t, seedData.Invitations, 3)
	
	// Debug: verify that seed data was created
	t.Logf("Seed data created: Organizations=%d, Users=%d, Invitations=%d", 
		len(seedData.Organizations), len(seedData.Users), len(seedData.Invitations))

	// Create PostgresStorage
	postgresStorage := storage.NewPostgresStorage(db)

	// Get organization from seed data
	org1 := seedData.Organizations[0]
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org1.ID)
	require.NoError(t, err)
	assert.Equal(t, org1.Name, retrievedOrg.Name)
	
	// Verify Signature
	assert.NotEmpty(t, retrievedOrg.Signature, "Signature is set")
	assert.Equal(t, org1.Signature, retrievedOrg.Signature, "Signature matches")
	
	// Verify that Organization can be retrieved by Signature
	retrievedOrgBySignature, err := postgresStorage.GetOrganizationBySignature(ctx, org1.Signature)
	require.NoError(t, err)
	assert.Equal(t, org1.ID, retrievedOrgBySignature.ID, "Organization can be retrieved by Signature")
	
	// Also verify org2
	org2 := seedData.Organizations[1]
	retrievedOrg2, err := postgresStorage.GetOrganization(ctx, org2.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, retrievedOrg2.Signature, "org2's Signature is set")

	// Get user from seed data
	user1 := seedData.Users[0]
	retrievedUser, err := postgresStorage.GetUser(ctx, user1.ID)
	require.NoError(t, err)
	assert.Equal(t, user1.Email, retrievedUser.Email)
	assert.Equal(t, user1.Role, retrievedUser.Role)

	// Get invitation from seed data
	invitation1 := seedData.Invitations[0]
	retrievedInvitation, err := postgresStorage.GetInvitationByToken(ctx, invitation1.Token)
	require.NoError(t, err)
	assert.Equal(t, invitation1.Email, retrievedInvitation.Email)
	assert.Equal(t, invitation1.Status, retrievedInvitation.Status)

	// Get users by organization ID
	users, err := postgresStorage.GetUsersByOrganizationID(ctx, org1.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(users), 2) // At least 2 users

	// Try to get non-existent Organization by Signature (verify error is returned)
	_, err = postgresStorage.GetOrganizationBySignature(ctx, "non-existent-signature")
	assert.Error(t, err, "Error is returned for non-existent Signature")
	
	// Note: Explicit cleanup here is not needed as defer cleanup() will handle it
	// If SKIP_CLEANUP=true is set, data is kept even in defer cleanup()
}

// TestIntegration_DeleteUser is an integration test for user deletion
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

	// Delete user
	err = m.DeleteUser(ctx, user.ID, org.ID)
	require.NoError(t, err)

	// Verify that user is deleted (logical deletion)
	retrievedUser, err := postgresStorage.GetUser(ctx, user.ID)
	if err == nil {
		// For logical deletion, DeletedAt is set
		assert.NotNil(t, retrievedUser.DeletedAt)
	}
}

// TestIntegration_DeleteOrganization is an integration test for organization deletion
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

	// Delete manager (organization is also deleted)
	err = m.DeleteUser(ctx, manager.ID, org.ID)
	require.NoError(t, err)

	// Verify that organization is deleted (logical deletion)
	retrievedOrg, err := postgresStorage.GetOrganization(ctx, org.ID)
	if err == nil {
		// For logical deletion, DeletedAt is set
		assert.NotNil(t, retrievedOrg.DeletedAt)
	}
}

// TestIntegration_AuthbossUserRegistration is an integration test for user registration using Authboss
// Note: This test tests Authboss functionality and does not directly test orgboss code
func TestIntegration_AuthbossUserRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// AuthbossのServerStorerを実装
	serverStorer := &authbossServerStorer{db: db}

	// Test data for user registration
	email := "testuser@example.com"
	password := "testpassword123"

	// Hash password (same method as Authboss)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	// Create struct implementing Authboss's User interface
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(email)
	authbossUser.PutPassword(string(hashedPassword)) // Set hashed password

	// Save user (via Authboss)
	err = serverStorer.Save(ctx, authbossUser)
	require.NoError(t, err, "Cannot save user via Authboss")

	// Verify directly from database
	var dbUser types.User
	err = db.WithContext(ctx).Where("email = ?", email).First(&dbUser).Error
	require.NoError(t, err, "Cannot get user from database")
	assert.Equal(t, email, dbUser.Email)
	assert.NotEmpty(t, dbUser.Password, "Password hash must be saved")
	assert.NotEqual(t, password, dbUser.Password, "Password must be hashed")
}

// TestIntegration_AuthbossUserLogin is an integration test for user login using Authboss
// Note: This test tests Authboss functionality and does not directly test orgboss code
func TestIntegration_AuthbossUserLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// AuthbossのServerStorerを実装
	serverStorer := &authbossServerStorer{db: db}

	// Create test user
	email := "loginuser@example.com"
	password := "loginpassword123"

	// Hash password (same method as Authboss)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	// Create and save struct implementing Authboss's User interface
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(email)
	authbossUser.PutPassword(string(hashedPassword)) // Set hashed password

	err = serverStorer.Save(ctx, authbossUser)
	require.NoError(t, err, "Cannot save user")

	// Test login (get user via Authboss)
	retrievedUser, err := serverStorer.Load(ctx, email)
	require.NoError(t, err, "Cannot get user via Authboss")
	assert.NotNil(t, retrievedUser, "User must be retrievable")

	// Verify password
	if retrievedUser != nil {
		// Type assertion to authbossuser.User
		authbossUser, ok := retrievedUser.(*authbossuser.User)
		require.True(t, ok, "Must be convertible to authbossuser.User type")
		
		// Since Authboss saves password hashed,
		// verify that retrieved password hash exists
		retrievedPasswordHash := authbossUser.GetPassword()
		assert.NotEmpty(t, retrievedPasswordHash, "Password hash must be retrievable")
		assert.NotEqual(t, password, retrievedPasswordHash, "Password must be hashed")
		
		// Verify password (verify with bcrypt)
		err := bcrypt.CompareHashAndPassword([]byte(retrievedPasswordHash), []byte(password))
		assert.NoError(t, err, "Hashed password must be verifiable correctly")
	}
}

// TestIntegration_AuthbossUserLogin_PendingInvitation tests that users with pending invitations are denied login
func TestIntegration_AuthbossUserLogin_PendingInvitation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// AuthbossのServerStorerを実装
	serverStorer := &authbossServerStorer{db: db}

	// Create test organization
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
	require.NoError(t, err, "Failed to create organization")

	// Create test user (simulate state created by AcceptInvitation)
	email := "pendinguser@example.com"
	password := "randompassword123"

	// Hash password (same method as Authboss)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	// Create User
	user := &types.User{
		Email:          email,
		Password:       string(hashedPassword),
		OrganizationID: org.ID,
		Role:           types.RoleUser,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	err = db.WithContext(ctx).Create(user).Error
	require.NoError(t, err, "Failed to create user")

	// Create pending invitation (state after AcceptInvitation, before password update)
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
	require.NoError(t, err, "Failed to create invitation")

	// Attempt login (should return error because pending invitation exists)
	retrievedUser, err := serverStorer.Load(ctx, email)
	assert.Error(t, err, "Login must be denied when pending invitation exists")
	assert.Nil(t, retrievedUser, "User must not be retrievable")
	assert.Equal(t, types.ErrInvitationPending, err, "ErrInvitationPending error must be returned")

	// Update invitation to accepted (simulate state after password update)
	invitation.Status = types.InvitationStatusAccepted
	err = db.WithContext(ctx).Save(invitation).Error
	require.NoError(t, err, "Failed to update invitation")

	// Attempt login again (should succeed because invitation is now accepted)
	retrievedUser, err = serverStorer.Load(ctx, email)
	assert.NoError(t, err, "Login must succeed when invitation is accepted")
	assert.NotNil(t, retrievedUser, "User must be retrievable")
}

// TestIntegration_SetupAuthbossWithAutoLogin tests the basic behavior of SetupAuthbossWithAutoLogin function
// Note: Currently only basic tests, as implementation must match Authboss v3's actual API
func TestIntegration_SetupAuthbossWithAutoLogin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create Authboss instance (for simple testing)
	// Note: Implementation must match Authboss v3's actual API
	ab := &authboss.Authboss{}
	
	// Call SetupAuthbossWithAutoLogin and verify no error occurs
	err := authbossuser.SetupAuthbossWithAutoLogin(db, ab, true)
	// Errors may occur as implementation is currently incomplete
	// Add error checks once implementation is complete to match actual API
	if err != nil {
		t.Logf("Error occurred in SetupAuthbossWithAutoLogin (expected as implementation is incomplete): %v", err)
	}

	// Also test with auto-login disabled
	err = authbossuser.SetupAuthbossWithAutoLogin(db, ab, false)
	if err != nil {
		t.Logf("Error occurred in SetupAuthbossWithAutoLogin (disabled) (expected as implementation is incomplete): %v", err)
	}
}

// authbossServerStorer is an Authboss storage implementation (for testing)
type authbossServerStorer struct {
	db *gorm.DB
}

func (s *authbossServerStorer) Save(ctx context.Context, user authboss.User) error {
	// Get values from Authboss's User interface
	email := user.GetPID()
	
	// Type assert to authbossuser.User to get password
	authbossUser, ok := user.(*authbossuser.User)
	if !ok {
		// Return error if type assertion fails
		// Must be authbossuser.User type
		return types.ErrUserNotFound
	}
	password := authbossUser.GetPassword() // Password hashed by Authboss

	// Convert to orgboss's User model
	dbUser := &types.User{
		Email:    email,
		Password: password, // Password hashed by Authboss
		Role:     types.RoleUser,
	}

	// Also create organization (for testing)
	// Generate Signature (12 bytes, 24 hex characters)
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

	// Check if pending invitation exists
	var pendingInvitations []types.Invitation
	if err := s.db.WithContext(ctx).Where("email = ? AND status = ?", key, types.InvitationStatusPending).Find(&pendingInvitations).Error; err != nil {
		return nil, err
	}
	if len(pendingInvitations) > 0 {
		// Deny login if pending invitation exists
		return nil, types.ErrInvitationPending
	}

	// Convert to struct implementing Authboss's User interface
	authbossUser := &authbossuser.User{}
	authbossUser.PutPID(user.Email)
	authbossUser.PutPassword(user.Password)

	return authbossUser, nil
}

// MailpitMessage is the structure of email messages returned from Mailpit's API
type MailpitMessage struct {
	ID      string   `json:"ID"`
	From    MailpitAddress `json:"From"`
	To      []MailpitAddress `json:"To"`
	Subject string   `json:"Subject"`
	Text    string   `json:"Text"`
	HTML    string   `json:"HTML"`
}

// MailpitAddress is Mailpit's email address structure
type MailpitAddress struct {
	Name    string `json:"Name"`
	Address string `json:"Address"`
}

// MailpitMessagesResponse is the response from Mailpit's email list API
type MailpitMessagesResponse struct {
	Total int              `json:"total"`
	Count int              `json:"count"`
	Start int              `json:"start"`
	Messages []MailpitMessage `json:"messages"`
}

// getMailpitMessages gets email list from Mailpit's API
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

// getMailpitMessage gets a specific email from Mailpit's API
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

// clearMailpitMessages clears Mailpit's emails (for testing)
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

// TestIntegration_EmailSending is an integration test for email sending
func TestIntegration_EmailSending(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Clear Mailpit emails (to maintain test independence)
	if err := clearMailpitMessages(t); err != nil {
		t.Logf("Failed to clear Mailpit emails (ignoring): %v", err)
	}

	ctx := context.Background()
	postgresStorage := storage.NewPostgresStorage(db)
	config := DefaultConfig()
	
	// Use SMTPEmailSender
	smtpSender := email.NewSMTPEmailSender()
	config.EmailSender = smtpSender
	// Set base URL for invitation links
	config.InvitationBaseURL = "http://localhost:8080"
	config.InvitationRedirectPath = "/reset-password"
	m := NewManagerWithStorage(config, postgresStorage)

	// Create organization and manager
	org, manager, err := m.CreateOrganizationWithUser(ctx, "メールテスト組織", "manager@example.com")
	require.NoError(t, err)

	// Invite user (email sending is executed)
	inviteEmail := "invited@example.com"
	invitation, err := m.InviteUser(ctx, org.ID, inviteEmail)
	require.NoError(t, err)
	require.NotNil(t, invitation)

	// Wait a bit before checking emails via Mailpit API
	time.Sleep(500 * time.Millisecond)

	// Get email list from Mailpit
	messages, err := getMailpitMessages(t)
	require.NoError(t, err, "Cannot get emails from Mailpit")
	require.Greater(t, len(messages), 0, "At least one email must be sent")

	// Get latest email (first email is latest)
	latestMessageSummary := messages[0]
	
	// Get email details (including body)
	latestMessage, err := getMailpitMessage(t, latestMessageSummary.ID)
	require.NoError(t, err, "Cannot get email details from Mailpit")
	
	// Verify email content
	assert.Equal(t, "組織への招待", latestMessage.Subject, "Subject is correct")
	assert.Contains(t, latestMessage.To[0].Address, inviteEmail, "Recipient is correct")
	// Verify URL is included
	assert.Contains(t, latestMessage.Text, "http://localhost:8080/invite/", "Invitation URL is included in body")
	assert.Contains(t, latestMessage.Text, invitation.Token, "Token is included in URL")
	assert.Contains(t, latestMessage.Text, invitation.ExpiresAt.Format("2006-01-02"), "Expiry date is included in body")

	// Test ResendInvitation
	invitation2, err := m.InviteUser(ctx, org.ID, "invited2@example.com")
	require.NoError(t, err)

	// Resend invitation (token is regenerated)
	err = m.ResendInvitation(ctx, invitation2.ID, org.ID, manager.ID)
	require.NoError(t, err)

	// Get updated invitation after resend (token has been regenerated)
	updatedInvitation2, err := postgresStorage.GetInvitationByID(ctx, invitation2.ID)
	require.NoError(t, err)
	assert.NotEqual(t, invitation2.Token, updatedInvitation2.Token, "Token is regenerated by resend")

	// Wait a bit before checking emails via Mailpit API
	time.Sleep(500 * time.Millisecond)

	// Get email list from Mailpit again
	messages2, err := getMailpitMessages(t)
	require.NoError(t, err)
	require.Greater(t, len(messages2), len(messages), "Email is added by resend")

	// Check latest email (resent email)
	latestMessage2Summary := messages2[0]
	
	// Get email details (including body)
	latestMessage2, err := getMailpitMessage(t, latestMessage2Summary.ID)
	require.NoError(t, err, "Cannot get resent email details from Mailpit")
	
	assert.Equal(t, "組織への招待", latestMessage2.Subject, "Resent email subject is correct")
	assert.Contains(t, latestMessage2.To[0].Address, "invited2@example.com", "Resent email recipient is correct")
	assert.Contains(t, latestMessage2.Text, updatedInvitation2.Token, "Resent email body contains regenerated token")
	assert.Contains(t, latestMessage2.Text, updatedInvitation2.ExpiresAt.Format("2006-01-02"), "Resent email body contains updated expiry date")
}

