// Package orgboss provides organization and multi-tenant management functionality
// that integrates with the Authboss authentication package.
//
// Overview
//
// orgboss manages organizations and users in a multi-tenant architecture.
// Each organization can have multiple users with different roles (manager, user).
// The first user created with an organization is automatically assigned the manager role.
//
// Key Features
//
//   - Organization and User Creation: Create organizations and users atomically
//   - Invitation System: Invite users to organizations via email with token-based authentication
//   - Role-Based Access Control: Manager and user roles with different permissions
//   - Customizable Handlers: RoleChecker, DeletionHandler, and EmailSender interfaces
//   - Hook System: Extensible hooks for before/after operations
//
// Basic Usage
//
//   manager := orgboss.NewManager(nil)
//   org, user, err := manager.CreateOrganizationWithUser(ctx, "My Organization", "user@example.com")
//
// Configuration
//
// Use Config to customize behavior:
//
//   config := &orgboss.Config{
//       InvitationExpiryDuration: 24 * time.Hour,
//       DefaultRole:              orgboss.RoleUser,
//       EnableBulkInvite:         true,
//       MaxBulkInviteCount:       100,
//   }
//   manager := orgboss.NewManager(config)
//
// Storage
//
// By default, orgboss uses an in-memory storage implementation. For production use,
// implement the Storage interface and use NewManagerWithStorage.
//
// Customization
//
// Implement the following interfaces to customize behavior:
//
//   - RoleChecker: Custom role permission logic
//   - DeletionHandler: Custom deletion behavior (logical deletion, email masking, etc.)
//   - EmailSender: Custom email sending implementation
//
// Hooks
//
// Register hooks to execute custom logic at various points:
//
//   manager.Hooks().BeforeOrganizationCreate = func(ctx context.Context, org *orgboss.Organization) error {
//       // Custom logic before organization creation
//       return nil
//   }
//
// See the documentation for each type and function for more details.
package orgboss
