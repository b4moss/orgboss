package orgboss

import (
	"errors"

	"github.com/b4m-oss/orgboss/types"
)

// Re-export errors from the types package
var (
	ErrOrganizationNotFound     = types.ErrOrganizationNotFound
	ErrUserNotFound             = types.ErrUserNotFound
	ErrInvitationNotFound       = types.ErrInvitationNotFound
	ErrInvalidInput             = types.ErrInvalidInput
	ErrEmailAlreadyExists       = types.ErrEmailAlreadyExists
)

// Additional error definitions (not in the types package)
var (
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvitationExpired         = errors.New("invitation expired")
	ErrInvitationAlreadyAccepted = errors.New("invitation already accepted")
	ErrInvitationAlreadyRejected = errors.New("invitation already rejected")
	ErrPermissionDenied          = errors.New("permission denied")
	ErrOrganizationAccessDenied  = errors.New("organization access denied")
	ErrBulkInviteLimitExceeded   = errors.New("bulk invite limit exceeded")
	ErrEmailSendFailed           = errors.New("email send failed")
)
