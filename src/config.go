package orgboss

import "time"

// Config はorgbossの設定を表す
type Config struct {
	InvitationExpiryDuration time.Duration
	DefaultRole              Role
	EnableBulkInvite         bool
	MaxBulkInviteCount       int
	RoleChecker              RoleChecker
	DeletionHandler          DeletionHandler
	EmailSender              EmailSender
}

// DefaultConfig はデフォルト設定を返す
// 注意: RoleCheckerとDeletionHandlerはNewManagerで設定される
func DefaultConfig() *Config {
	return &Config{
		InvitationExpiryDuration: 24 * time.Hour,
		DefaultRole:              RoleUser,
		EnableBulkInvite:         true,
		MaxBulkInviteCount:       100,
		RoleChecker:              nil, // NewManagerで設定される
		DeletionHandler:          nil, // NewManagerで設定される
		EmailSender:              nil, // 実装が必要
	}
}

