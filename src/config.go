package orgboss

import "time"

// Config represents the configuration for orgboss
type Config struct {
	InvitationExpiryDuration        time.Duration
	DefaultRole                     Role
	EnableBulkInvite                bool
	MaxBulkInviteCount              int
	RoleChecker                     RoleChecker
	DeletionHandler                 DeletionHandler
	EmailSender                     EmailSender
	InvitationBaseURL               string // Base URL for invitation links (e.g., "http://localhost:8080")
	InvitationRedirectPath           string // Redirect path after invitation acceptance (e.g., "/reset-password")
	EnableAutoLoginAfterPasswordReset bool // Enable auto-login after password reset (default: true)
	InvitationEmailSubjectTemplatePath string // Path to subject template file (optional, uses default template if empty)
	InvitationEmailBodyTemplatePath   string // Path to body template file (optional, uses default template if empty)
	InvitationEmailFrom              string // From address (optional, overrides existing SMTP_FROM)
}

// DefaultConfig returns the default configuration
// Note: RoleChecker and DeletionHandler are set in NewManager
func DefaultConfig() *Config {
	return &Config{
		InvitationExpiryDuration:        24 * time.Hour,
		DefaultRole:                     RoleUser,
		EnableBulkInvite:                true,
		MaxBulkInviteCount:              100,
		RoleChecker:                     nil, // Set in NewManager
		DeletionHandler:                 nil, // Set in NewManager
		EmailSender:                     nil, // Implementation required
		InvitationBaseURL:               "",  // Default is empty string (must be configured)
		InvitationRedirectPath:          "/reset-password", // Default redirect path
		EnableAutoLoginAfterPasswordReset: true, // Enable auto-login by default
	}
}

