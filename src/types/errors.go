package types

import "errors"

var (
	ErrOrganizationNotFound     = errors.New("organization not found")
	ErrUserNotFound             = errors.New("user not found")
	ErrInvitationNotFound       = errors.New("invitation not found")
	ErrInvalidInput             = errors.New("invalid input")
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrOrganizationAccessDenied = errors.New("organization access denied")
)

