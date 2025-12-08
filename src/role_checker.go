package orgboss

// DefaultRoleChecker is the default RoleChecker implementation
type DefaultRoleChecker struct{}

// NewDefaultRoleChecker creates a new DefaultRoleChecker
func NewDefaultRoleChecker() *DefaultRoleChecker {
	return &DefaultRoleChecker{}
}

// HasPermission checks permissions based on role and action
func (r *DefaultRoleChecker) HasPermission(role Role, action string) bool {
	switch role {
	case RoleManager:
		// manager can execute all actions
		return true
	case RoleUser:
		// user can only execute read
		return action == "read"
	default:
		return false
	}
}
