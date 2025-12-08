package orgboss

import "orgboss/types"

// 型エイリアスで types パッケージの型を再エクスポート
type Role = types.Role
type InvitationStatus = types.InvitationStatus
type Organization = types.Organization
type User = types.User
type Invitation = types.Invitation

// 定数も再エクスポート
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
