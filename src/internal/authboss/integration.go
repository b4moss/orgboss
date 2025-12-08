package authboss

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/aarondl/authboss/v3"
	"gorm.io/gorm"

	"github.com/b4m-oss/orgboss/types"
)

// User is a struct that extends Authboss's User model
// Implements Authboss's standard User interface and adds organization_id and role
type User struct {
	// Standard Authboss fields
	ID       int64  `db:"id"`
	Email    string `db:"email"`
	Password string `db:"password"`

	// orgboss extension fields
	OrganizationID uint        `gorm:"not null;index" db:"organization_id"`
	Role           types.Role   `gorm:"not null;default:'user'" db:"role"`
	CreatedAt      sql.NullTime `db:"created_at"`
	UpdatedAt      sql.NullTime `db:"updated_at"`
	DeletedAt      sql.NullTime `gorm:"index" db:"deleted_at"`
}

// PutPID implements Authboss's User interface
func (u *User) PutPID(pid string) {
	u.Email = pid
}

// PutPassword implements Authboss's User interface
func (u *User) PutPassword(password string) {
	u.Password = password
}

// PutEmail implements Authboss's User interface
func (u *User) PutEmail(email string) {
	u.Email = email
}

// PutConfirmed implements Authboss's User interface (unused)
func (u *User) PutConfirmed(confirmed bool) {
	// Unused in orgboss
}

// PutLocked implements Authboss's User interface (unused)
func (u *User) PutLocked(locked bool) {
	// Unused in orgboss
}

// PutAttemptCount implements Authboss's User interface (unused)
func (u *User) PutAttemptCount(attempts int) {
	// Unused in orgboss
}

// PutLastAttempt implements Authboss's User interface (unused)
func (u *User) PutLastAttempt(lastAttempt *sql.NullTime) {
	// Unused in orgboss
}

// PutExpired implements Authboss's User interface (unused)
func (u *User) PutExpired(expired *sql.NullTime) {
	// Unused in orgboss
}

// GetPID implements Authboss's User interface
func (u *User) GetPID() string {
	return u.Email
}

// GetPassword implements Authboss's User interface
func (u *User) GetPassword() string {
	return u.Password
}

// GetEmail implements Authboss's User interface
// Note: The official Authboss v3 User interface may not have a GetEmail() method
// Email is treated as PID (Principal ID) and retrieved via GetPID()
// This method is kept for orgboss's internal implementation
func (u *User) GetEmail() string {
	return u.Email
}

// GetConfirmed implements Authboss's User interface
func (u *User) GetConfirmed() bool {
	return true // Always true in orgboss
}

// GetLocked implements Authboss's User interface
func (u *User) GetLocked() bool {
	return false // Always false in orgboss
}

// GetAttemptCount implements Authboss's User interface
func (u *User) GetAttemptCount() int {
	return 0 // Unused in orgboss
}

// GetLastAttempt implements Authboss's User interface
func (u *User) GetLastAttempt() *sql.NullTime {
	return nil // Unused in orgboss
}

// GetExpired implements Authboss's User interface
func (u *User) GetExpired() *sql.NullTime {
	return nil // Unused in orgboss
}

// SetupAuthboss configures Authboss and integrates it with orgboss
// Note: Implementation must match Authboss's actual API
// Currently implemented as a placeholder
// Official repository: https://github.com/aarondl/authboss
func SetupAuthboss(db *gorm.DB, ab *authboss.Authboss) error {
	// Configure Authboss storage
	// Here, a simple implementation using GORM is set up
	// In actual implementation, Authboss's storage interface must be implemented

	// Set up BeforeRegister hook to link organization creation
	// Note: Implementation must match Authboss v3's actual API
	// Refer to official documentation to confirm the correct hook setup method
	// Currently commented out
	// ab.Config.Core.BeforeRegister = func(ctx context.Context, r *authboss.RegisterValues) error {
	// 	// Call orgboss's CreateOrganizationWithUser here
	// 	// In actual implementation, need to get orgboss.Manager and call it
	// 	return nil
	// }

	return nil
}

// SetupAuthbossWithAutoLogin configures Authboss and enables redirect functionality after password reset
// If enableAutoLogin is true, enables settings to redirect to login page after password reset completion
// Official repository: https://github.com/aarondl/authboss
//
// Note: Actual redirect processing must be implemented on the password reset HTTP handler side.
// Use the RedirectToLoginAfterPasswordReset helper function to implement redirect processing.
// Reference: https://github.com/aarondl/authboss/blob/v3.5.3/authboss.go#L76
func SetupAuthbossWithAutoLogin(db *gorm.DB, ab *authboss.Authboss, enableAutoLogin bool) error {
	// Execute basic setup
	if err := SetupAuthboss(db, ab); err != nil {
		return err
	}

	// Settings for when auto-login (redirect) is enabled
	// Note: In Authboss v3, redirect processing after password reset must be implemented at the HTTP handler level.
	// Use the RedirectToLoginAfterPasswordReset helper function to implement it.
	if enableAutoLogin {
		// Setup is complete. Actual redirect processing should call RedirectToLoginAfterPasswordReset
		// on the password reset HTTP handler side.
	}

	return nil
}

// RedirectToLoginAfterPasswordReset redirects to the login page after password reset completion
// This function is called from the password reset HTTP handler
//
// Usage example:
//   func passwordResetHandler(w http.ResponseWriter, r *http.Request) {
//       // Password reset processing...
//       if err := authbossuser.RedirectToLoginAfterPasswordReset(r.Context(), w, r, ab); err != nil {
//           // Error handling
//       }
//   }
func RedirectToLoginAfterPasswordReset(ctx context.Context, w http.ResponseWriter, r *http.Request, ab *authboss.Authboss) error {
	// Get redirect path to login page
	loginPath := "/login"
	if ab.Config.Paths.Mount != "" {
		loginPath = ab.Config.Paths.Mount + loginPath
	}

	// Set redirect options
	ro := authboss.RedirectOptions{
		Code:         http.StatusSeeOther,
		RedirectPath: loginPath,
		Success:      "パスワードリセットが完了しました。ログインしてください。",
	}

	// Execute redirect
	if ab.Config.Core.Redirector != nil {
		if err := ab.Config.Core.Redirector.Redirect(w, r, ro); err != nil {
			return err
		}
	} else {
		// Use standard HTTP redirect if Redirector is not configured
		http.Redirect(w, r, loginPath, http.StatusSeeOther)
	}

	return nil
}

// ValidateOrganizationAccess validates organization_id in post-authentication middleware
func ValidateOrganizationAccess(ctx context.Context, user authboss.User, orgID uint, db *gorm.DB) error {
	// Get Email from Authboss's User (using GetPID(), Email is used as PID)
	email := user.GetPID()
	if email == "" {
		return types.ErrUserNotFound
	}

	// Get User from database
	var orgUser types.User
	if err := db.WithContext(ctx).Where("email = ?", email).First(&orgUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return types.ErrUserNotFound
		}
		return err
	}

	// Verify organization_id
	if orgUser.OrganizationID != orgID {
		return types.ErrOrganizationAccessDenied
	}

	return nil
}

