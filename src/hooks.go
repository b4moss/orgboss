package orgboss

import "context"

// HookFunc is the type for hook functions
type HookFunc func(ctx context.Context, data interface{}) error

// Hooks manages hook points
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

// NewHooks creates a new Hooks instance
func NewHooks() *Hooks {
	return &Hooks{}
}

