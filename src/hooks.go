package orgboss

import "context"

// HookFunc はフック関数の型
type HookFunc func(ctx context.Context, data interface{}) error

// Hooks はフックポイントを管理する
type Hooks struct {
	BeforeOrganizationCreate HookFunc
	AfterOrganizationCreate  HookFunc
	BeforeOrganizationUpdate HookFunc
	AfterOrganizationUpdate  HookFunc
	BeforeInvite             HookFunc
	AfterInvite              HookFunc
	BeforeUserDelete         HookFunc
	AfterUserDelete          HookFunc
}

// NewHooks は新しいHooksを作成する
func NewHooks() *Hooks {
	return &Hooks{}
}

