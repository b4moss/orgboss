package orgboss

// DefaultRoleChecker はデフォルトのRoleChecker実装
type DefaultRoleChecker struct{}

// NewDefaultRoleChecker は新しいDefaultRoleCheckerを作成する
func NewDefaultRoleChecker() *DefaultRoleChecker {
	return &DefaultRoleChecker{}
}

// HasPermission はロールとアクションに基づいて権限をチェックする
func (r *DefaultRoleChecker) HasPermission(role Role, action string) bool {
	switch role {
	case RoleManager:
		// managerは全てのアクションを実行可能
		return true
	case RoleUser:
		// userはreadのみ実行可能
		return action == "read"
	default:
		return false
	}
}
