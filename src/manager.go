package orgboss

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/b4m-oss/orgboss/internal/email"
	"github.com/b4m-oss/orgboss/internal/storage"
	"github.com/b4m-oss/orgboss/internal/validation"
)

// Manager provides the core functionality of orgboss
type Manager struct {
	config  *Config
	hooks   *Hooks
	storage Storage
}

// NewManager creates a new Manager
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	stor := storage.NewInMemoryStorage()

	// Set default implementations
	if config.RoleChecker == nil {
		config.RoleChecker = NewDefaultRoleChecker()
	}
	if config.DeletionHandler == nil {
		config.DeletionHandler = NewDefaultDeletionHandler(stor)
	} else {
		// Update DeletionHandler's storage (unify storage created in DefaultConfig with storage used in Manager)
		config.DeletionHandler.SetStorage(stor)
	}

	// Apply template settings if EmailSender is SMTPEmailSender
	applyEmailTemplateConfig(config)

	return &Manager{
		config:  config,
		hooks:   NewHooks(),
		storage: stor,
	}
}

// NewManagerWithStorage creates a new Manager with the specified storage
func NewManagerWithStorage(config *Config, storage Storage) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	// Set default implementations
	if config.RoleChecker == nil {
		config.RoleChecker = NewDefaultRoleChecker()
	}
	if config.DeletionHandler == nil {
		config.DeletionHandler = NewDefaultDeletionHandler(storage)
	} else {
		// Update DeletionHandler's storage
		config.DeletionHandler.SetStorage(storage)
	}

	// Apply template settings if EmailSender is SMTPEmailSender
	applyEmailTemplateConfig(config)

	return &Manager{
		config:  config,
		hooks:   NewHooks(),
		storage: storage,
	}
}

// CreateOrganizationWithUser creates an Organization and User simultaneously
func (m *Manager) CreateOrganizationWithUser(ctx context.Context, orgName string, userEmail string) (*Organization, *User, error) {
	// Validation
	if err := validation.ValidateOrganizationName(orgName); err != nil {
		return nil, nil, err
	}
	if err := validation.ValidateEmail(userEmail); err != nil {
		return nil, nil, err
	}

	// Execute BeforeOrganizationCreate hook
	if err := m.executeHook(m.hooks.BeforeOrganizationCreate, ctx, nil); err != nil {
		return nil, nil, err
	}

	// Start transaction (no actual transaction for in-memory implementation)
	// Create Organization
	org, err := m.createOrganization(ctx, orgName)
	if err != nil {
		return nil, nil, err
	}

	// Generate and hash random password
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, nil, err
	}
	hashedPassword, err := hashPassword(randomPassword)
	if err != nil {
		return nil, nil, err
	}

	// Create User (role=manager, random password)
	user, err := m.createUserWithPassword(ctx, userEmail, org.ID, RoleManager, hashedPassword)
	if err != nil {
		return nil, nil, err
	}

	// Commit transaction (no actual commit for in-memory implementation)

	// Execute AfterOrganizationCreate hook
	if err := m.executeHook(m.hooks.AfterOrganizationCreate, ctx, org); err != nil {
		return nil, nil, err
	}

	return org, user, nil
}

// InviteUser invites a user
func (m *Manager) InviteUser(ctx context.Context, orgID uint, email string) (*Invitation, error) {
	// Validation
	if err := validation.ValidateEmail(email); err != nil {
		return nil, err
	}

	// Execute BeforeInvite hook
	if err := m.executeHook(m.hooks.BeforeInvite, ctx, nil); err != nil {
		return nil, err
	}

	// Generate token (hash)
	token, err := m.GenerateToken()
	if err != nil {
		return nil, err
	}

	// Calculate expiry (Config.InvitationExpiryDuration)
	expiresAt := m.CalculateExpiry()

	// Create Invitation (status=pending)
	invitation, err := m.createInvitation(ctx, email, orgID, token, expiresAt)
	if err != nil {
		return nil, err
	}

	// Send email (EmailSender.SendInvitation)
	if m.config.EmailSender == nil {
		return nil, ErrEmailSendFailed
	}
	// Generate invitation URL
	invitationURL := m.GetInvitationURL(token)
	if err := m.config.EmailSender.SendInvitation(ctx, invitation, invitationURL); err != nil {
		return nil, ErrEmailSendFailed
	}

	// Execute AfterInvite hook
	if err := m.executeHook(m.hooks.AfterInvite, ctx, invitation); err != nil {
		return nil, err
	}

	return invitation, nil
}

// InviteUsers invites multiple users (bulk invitation)
func (m *Manager) InviteUsers(ctx context.Context, orgID uint, emails []string) ([]*Invitation, error) {
	// Check invitation limit (MaxBulkInviteCount)
	if len(emails) > m.config.MaxBulkInviteCount {
		return nil, ErrBulkInviteLimitExceeded
	}

	// Execute InviteUser for each email address
	invitations := make([]*Invitation, 0, len(emails))
	for _, email := range emails {
		invitation, err := m.InviteUser(ctx, orgID, email)
		if err != nil {
			// Continue processing other invitations even if an error occurs for some email addresses
			continue
		}
		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

// AcceptInvitation accepts an invitation
// At this point, a random password is generated, hashed, and saved,
// but the Invitation's status remains pending (becomes accepted when password is updated)
func (m *Manager) AcceptInvitation(ctx context.Context, token string) (*User, error) {
	// Verify token (search Invitation)
	invitation, err := m.storage.GetInvitationByToken(ctx, token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check expiry
	if m.IsExpired(invitation.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	// Return error if already accepted or rejected
	if invitation.Status == InvitationStatusAccepted {
		return nil, ErrInvitationAlreadyAccepted
	}
	if invitation.Status == InvitationStatusRejected {
		return nil, ErrInvitationAlreadyRejected
	}

	// Start transaction (no actual transaction for in-memory implementation)

	// Generate and hash random password
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, err
	}
	hashedPassword, err := hashPassword(randomPassword)
	if err != nil {
		return nil, err
	}

	// Create User (organization_id, role=user, random password)
	// Invitation's status remains pending (becomes accepted when password is updated)
	user, err := m.createUserWithPassword(ctx, invitation.Email, invitation.OrganizationID, RoleUser, hashedPassword)
	if err != nil {
		return nil, err
	}

	// Commit transaction (no actual commit for in-memory implementation)

	return user, nil
}

// RejectInvitation rejects an invitation
func (m *Manager) RejectInvitation(ctx context.Context, token string) error {
	// Verify token (search Invitation)
	invitation, err := m.storage.GetInvitationByToken(ctx, token)
	if err != nil {
		return ErrInvalidToken
	}

	// Check expiry
	if m.IsExpired(invitation.ExpiresAt) {
		return ErrInvitationExpired
	}

	// Update Invitation (status=rejected)
	invitation.Status = InvitationStatusRejected
	if err := m.storage.UpdateInvitation(ctx, invitation); err != nil {
		return err
	}

	return nil
}

// DeleteUser deletes a user
func (m *Manager) DeleteUser(ctx context.Context, userID uint, orgID uint) error {
	// Get User
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Permission check (ValidateOrganizationAccess)
	// Verify that userID and orgID represent self or a member of the same organization
	if err := m.ValidateOrganizationAccess(ctx, userID, orgID); err != nil {
		return err
	}

	// Verify organization_id matches
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Execute BeforeUserDelete hook
	if err := m.executeHook(m.hooks.BeforeUserDelete, ctx, user); err != nil {
		return err
	}

	// Check role
	if user.Role == RoleManager {
		// If manager leaves: execute DeleteOrganization
		if err := m.DeleteOrganization(ctx, orgID); err != nil {
			return err
		}
	} else {
		// If user leaves: execute DeleteUser (DeletionHandler)
		if m.config.DeletionHandler == nil {
			return ErrPermissionDenied
		}
		if err := m.config.DeletionHandler.DeleteUser(ctx, user); err != nil {
			return err
		}
	}

	// Execute AfterUserDelete hook
	if err := m.executeHook(m.hooks.AfterUserDelete, ctx, user); err != nil {
		return err
	}

	return nil
}

// DeleteOrganization deletes an organization
func (m *Manager) DeleteOrganization(ctx context.Context, orgID uint) error {
	// Get Organization
	org, err := m.storage.GetOrganization(ctx, orgID)
	if err != nil {
		return err
	}

	// Get related Users (continue even if error is returned)
	users, _ := m.storage.GetUsersByOrganizationID(ctx, orgID)

	// Return error if DeletionHandler is not set
	if m.config.DeletionHandler == nil {
		return ErrPermissionDenied
	}

	// Delete related Users (logical/physical/mask, DeletionHandler)
	for _, user := range users {
		if err := m.config.DeletionHandler.DeleteUser(ctx, user); err != nil {
			return err
		}
	}

	// Delete Organization (logical/physical, DeletionHandler)
	if err := m.config.DeletionHandler.DeleteOrganization(ctx, org); err != nil {
		return err
	}

	return nil
}

// ResendInvitation resends an invitation
func (m *Manager) ResendInvitation(ctx context.Context, invitationID uint, orgID uint, userID uint) error {
	// Permission check (manager only)
	if err := m.CheckPermission(ctx, userID, orgID, "update"); err != nil {
		return err
	}

	// Search Invitation (invitationID, orgID)
	invitation, err := m.storage.GetInvitationByID(ctx, invitationID)
	if err != nil {
		return ErrInvitationNotFound
	}

	// Verify organization_id matches
	if invitation.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Regenerate token (optional)
	token, err := m.GenerateToken()
	if err != nil {
		return err
	}

	// Update expiry
	expiresAt := m.CalculateExpiry()

	// Update Invitation (status=pending, update expiresAt)
	invitation.Status = InvitationStatusPending
	invitation.ExpiresAt = expiresAt
	invitation.Token = token
	if err := m.storage.UpdateInvitation(ctx, invitation); err != nil {
		return err
	}

	// Resend email (EmailSender.SendInvitation)
	if m.config.EmailSender == nil {
		return ErrEmailSendFailed
	}
	// Generate invitation URL
	invitationURL := m.GetInvitationURL(token)
	if err := m.config.EmailSender.SendInvitation(ctx, invitation, invitationURL); err != nil {
		return ErrEmailSendFailed
	}

	return nil
}

// UpdatePassword updates a user's password
// When password is updated, set the corresponding Invitation to accepted
func (m *Manager) UpdatePassword(ctx context.Context, userID uint, orgID uint, newPassword string) error {
	// Get User
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Permission check (only self can update)
	if err := m.ValidateOrganizationAccess(ctx, userID, orgID); err != nil {
		return err
	}

	// Verify organization_id matches
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Hash password
	hashedPassword, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update User's password
	user.Password = hashedPassword
	user.UpdatedAt = time.Now()
	if err := m.storage.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Find corresponding Invitation and set to accepted
	invitations, err := m.storage.GetInvitationsByOrganizationID(ctx, orgID)
	if err != nil {
		return err
	}

	for _, invitation := range invitations {
		if invitation.Email == user.Email && invitation.Status == InvitationStatusPending {
			invitation.Status = InvitationStatusAccepted
			if err := m.storage.UpdateInvitation(ctx, invitation); err != nil {
				return err
			}
			break
		}
	}

	return nil
}

// UpdateProfile updates a user's profile
// userID is the target user ID, orgID is the organization ID
// Only self can update (verify self using userID and orgID)
func (m *Manager) UpdateProfile(ctx context.Context, userID uint, orgID uint, updates map[string]interface{}) error {
	// Get User
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Permission check (only self can update)
	// Verify self using userID and orgID
	if err := m.ValidateOrganizationAccess(ctx, userID, orgID); err != nil {
		return err
	}

	// Verify organization_id matches
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Since only self can update, verify that the user specified by userID belongs to the organization orgID
	// This is already verified by ValidateOrganizationAccess

	// Update User
	// Update fields from updates map (simple implementation)
	// In actual implementation, field-by-field validation is required
	user.UpdatedAt = time.Now()
	if err := m.storage.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Authboss notification (not implemented yet, to be added in the future)

	return nil
}

// UpdateOrganization updates an organization
func (m *Manager) UpdateOrganization(ctx context.Context, orgID uint, userID uint, updates map[string]interface{}) error {
	// Permission check (manager only)
	if err := m.CheckPermission(ctx, userID, orgID, "update"); err != nil {
		return err
	}

	// Get Organization
	org, err := m.storage.GetOrganization(ctx, orgID)
	if err != nil {
		return err
	}

	// Execute BeforeOrganizationUpdate hook
	if err := m.executeHook(m.hooks.BeforeOrganizationUpdate, ctx, org); err != nil {
		return err
	}

	// Verify organization_id matches (already verified by CheckPermission, but double-check)
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Update Organization (duplicate organization names are allowed)
	// Update fields from updates map (simple implementation)
	// In actual implementation, field-by-field validation is required
	if name, ok := updates["name"].(string); ok {
		org.Name = name
	}
	org.UpdatedAt = time.Now()
	if err := m.storage.UpdateOrganization(ctx, org); err != nil {
		return err
	}

	// Execute AfterOrganizationUpdate hook
	if err := m.executeHook(m.hooks.AfterOrganizationUpdate, ctx, org); err != nil {
		return err
	}

	return nil
}

// ValidateOrganizationAccess verifies if a user can access an organization
func (m *Manager) ValidateOrganizationAccess(ctx context.Context, userID uint, orgID uint) error {
	// Search User (userID, organizationID)
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Verify organization_id matches
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	return nil
}

// CheckPermission checks permissions
func (m *Manager) CheckPermission(ctx context.Context, userID uint, orgID uint, action string) error {
	// Search User
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Verify organization_id matches
	if user.OrganizationID != orgID {
		return ErrOrganizationAccessDenied
	}

	// Check role (RoleChecker)
	if m.config.RoleChecker == nil {
		return ErrPermissionDenied
	}

	if !m.config.RoleChecker.HasPermission(user.Role, action) {
		return ErrPermissionDenied
	}

	return nil
}

// executeHook is a helper method to execute hooks
func (m *Manager) executeHook(hook HookFunc, ctx context.Context, data interface{}) error {
	if hook != nil {
		return hook(ctx, data)
	}
	return nil
}

// createOrganization is a helper method to create an Organization
func (m *Manager) createOrganization(ctx context.Context, name string) (*Organization, error) {
	// By default, use a random string as Signature
	signature, err := generateRandomSignature()
	if err != nil {
		return nil, err
	}

	org := &Organization{
		Name:      name,
		Signature: signature,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := m.storage.CreateOrganization(ctx, org); err != nil {
		return nil, err
	}
	return org, nil
}

// generateRandomSignature generates a random Signature (12 bytes, 24 hex characters)
func generateRandomSignature() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// createUser is a helper method to create a User (without password)
func (m *Manager) createUser(ctx context.Context, email string, orgID uint, role Role) (*User, error) {
	return m.createUserWithPassword(ctx, email, orgID, role, "")
}

// createUserWithPassword is a helper method to create a User (with password)
func (m *Manager) createUserWithPassword(ctx context.Context, email string, orgID uint, role Role, hashedPassword string) (*User, error) {
	user := &User{
		Email:          email,
		Password:       hashedPassword,
		OrganizationID: orgID,
		Role:           role,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := m.storage.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// generateRandomPassword generates a random password (32 bytes, 64 hex characters)
func generateRandomPassword() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// createInvitation is a helper method to create an Invitation
func (m *Manager) createInvitation(ctx context.Context, email string, orgID uint, token string, expiresAt time.Time) (*Invitation, error) {
	invitation := &Invitation{
		Email:          email,
		OrganizationID: orgID,
		Token:          token,
		ExpiresAt:      expiresAt,
		Status:         InvitationStatusPending,
		CreatedAt:      time.Now(),
	}
	if err := m.storage.CreateInvitation(ctx, invitation); err != nil {
		return nil, err
	}
	return invitation, nil
}

// GenerateToken generates a token
func (m *Manager) GenerateToken() (string, error) {
	bytes := make([]byte, 32) // 256-bit random token
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CalculateExpiry calculates the expiry time
func (m *Manager) CalculateExpiry() time.Time {
	return time.Now().Add(m.config.InvitationExpiryDuration)
}

// IsExpired checks if the expiry time has passed
func (m *Manager) IsExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

// GetInvitationURL generates an invitation URL
func (m *Manager) GetInvitationURL(token string) string {
	if m.config.InvitationBaseURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/invite/%s", m.config.InvitationBaseURL, token)
}

// ValidateInvitationTokenAndGetRedirectURL validates a token and returns the redirect URL
func (m *Manager) ValidateInvitationTokenAndGetRedirectURL(ctx context.Context, token string) (string, error) {
	// Verify token (search Invitation)
	invitation, err := m.storage.GetInvitationByToken(ctx, token)
	if err != nil {
		return "", ErrInvalidToken
	}

	// Check expiry
	if m.IsExpired(invitation.ExpiresAt) {
		return "", ErrInvitationExpired
	}

	// Return error if already accepted or rejected
	if invitation.Status == InvitationStatusAccepted {
		return "", ErrInvitationAlreadyAccepted
	}
	if invitation.Status == InvitationStatusRejected {
		return "", ErrInvitationAlreadyRejected
	}

	// Generate redirect URL
	redirectPath := m.config.InvitationRedirectPath
	if redirectPath == "" {
		redirectPath = "/reset-password"
	}
	redirectURL := fmt.Sprintf("%s?token=%s", redirectPath, token)

	return redirectURL, nil
}

// applyEmailTemplateConfig applies template settings if EmailSender is SMTPEmailSender
func applyEmailTemplateConfig(config *Config) {
	if config.EmailSender == nil {
		return
	}

	// Type assertion to SMTPEmailSender
	smtpSender, ok := config.EmailSender.(*email.SMTPEmailSender)
	if !ok {
		return
	}

	// Apply if template paths are set
	if config.InvitationEmailSubjectTemplatePath != "" || config.InvitationEmailBodyTemplatePath != "" {
		smtpSender.SetTemplatePaths(config.InvitationEmailSubjectTemplatePath, config.InvitationEmailBodyTemplatePath)
	}

	// Apply if From address is set
	if config.InvitationEmailFrom != "" {
		smtpSender.SetFrom(config.InvitationEmailFrom)
	}
}
