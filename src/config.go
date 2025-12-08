package orgboss

import "time"

// Config はorgbossの設定を表す
type Config struct {
	InvitationExpiryDuration        time.Duration
	DefaultRole                     Role
	EnableBulkInvite                bool
	MaxBulkInviteCount              int
	RoleChecker                     RoleChecker
	DeletionHandler                 DeletionHandler
	EmailSender                     EmailSender
	InvitationBaseURL               string // 招待URLのベースURL（例: "http://localhost:8080"）
	InvitationRedirectPath           string // リダイレクト先のパス（例: "/reset-password"）
	EnableAutoLoginAfterPasswordReset bool // パスワードリセット後の自動ログインを有効にする（デフォルト: true）
	InvitationEmailSubjectTemplatePath string // 件名テンプレートファイルのパス（オプション、空の場合はデフォルトテンプレートを使用）
	InvitationEmailBodyTemplatePath   string // 本文テンプレートファイルのパス（オプション、空の場合はデフォルトテンプレートを使用）
	InvitationEmailFrom              string // 差出人（オプション、既存のSMTP_FROMをオーバーライド）
}

// DefaultConfig はデフォルト設定を返す
// 注意: RoleCheckerとDeletionHandlerはNewManagerで設定される
func DefaultConfig() *Config {
	return &Config{
		InvitationExpiryDuration:        24 * time.Hour,
		DefaultRole:                     RoleUser,
		EnableBulkInvite:                true,
		MaxBulkInviteCount:              100,
		RoleChecker:                     nil, // NewManagerで設定される
		DeletionHandler:                 nil, // NewManagerで設定される
		EmailSender:                     nil, // 実装が必要
		InvitationBaseURL:               "",  // デフォルトは空文字列（設定が必要）
		InvitationRedirectPath:          "/reset-password", // デフォルトのリダイレクト先
		EnableAutoLoginAfterPasswordReset: true, // デフォルトで自動ログインを有効にする
	}
}

