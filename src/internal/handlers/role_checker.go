package handlers

import "github.com/b4m-oss/orgboss"

// DefaultRoleChecker is the default RoleChecker implementation
type DefaultRoleChecker struct{}

// NewDefaultRoleChecker creates a new DefaultRoleChecker
func NewDefaultRoleChecker() *DefaultRoleChecker {
	return &DefaultRoleChecker{}
}

// HasPermission checks permissions based on role and action
func (r *DefaultRoleChecker) HasPermission(role orgboss.Role, action string) bool {
	switch role {
	case orgboss.RoleManager:
		// manager can execute all actions
		return true
	case orgboss.RoleUser:
		// user can only execute read
		return action == "read"
	default:
		return false
	}
}
