package orgboss

import "github.com/b4m-oss/orgboss/types"

// Re-export types from the types package using type aliases
type Role = types.Role
type InvitationStatus = types.InvitationStatus
type Organization = types.Organization
type User = types.User
type Invitation = types.Invitation

// Re-export constants
const (
	RoleManager = types.RoleManager
	RoleUser    = types.RoleUser
)

const (
	InvitationStatusPending  = types.InvitationStatusPending
	InvitationStatusAccepted = types.InvitationStatusAccepted
	InvitationStatusRejected = types.InvitationStatusRejected
	InvitationStatusExpired  = types.InvitationStatusExpired
)
