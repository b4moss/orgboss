# orgboss

Organization multi-tenant management package with authboss

[![Tests](https://github.com/b4m-oss/orgboss/actions/workflows/test.yml/badge.svg?branch=develop)](https://github.com/b4m-oss/orgboss/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/b4m-oss/orgboss/graph/badge.svg?branch=develop)](https://codecov.io/gh/b4m-oss/orgboss)

## Features

- **Organization and User Management**: Create organizations and users atomically with transaction support
- **Invitation System**: Invite users to organizations via email with token-based authentication and expiry management
- **Role-Based Access Control**: Manager and user roles with different permissions (manager: full access, user: read-only)
- **Customizable Handlers**: Implement custom logic via `RoleChecker`, `DeletionHandler`, and `EmailSender` interfaces
- **Hook System**: Extensible hooks for before/after operations (organization creation, user deletion, etc.)
- **Authboss Integration**: Seamless integration with [Authboss](https://github.com/aarondl/authboss) authentication package
- **Storage Flexibility**: In-memory storage for testing, PostgreSQL storage for production via GORM

## Requirements

- Go 1.25 or higher
- PostgreSQL (for production use)
- SMTP server (for email invitations)

## Installation

```bash
go get github.com/b4m-oss/orgboss
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/b4m-oss/orgboss"
)

func main() {
    ctx := context.Background()
    
    // Create a manager with default configuration
    manager := orgboss.NewManager(nil)
    
    // Create an organization with the first user (automatically assigned manager role)
    org, user, err := manager.CreateOrganizationWithUser(
        ctx,
        "My Organization",
        "manager@example.com",
    )
    if err != nil {
        panic(err)
    }
    
    // The user is automatically assigned the manager role
    // org.ID and user.ID are now available
}
```

## Configuration

Use `Config` to customize orgboss behavior:

```go
import "time"

config := &orgboss.Config{
    InvitationExpiryDuration:        24 * time.Hour,
    DefaultRole:                     orgboss.RoleUser,
    EnableBulkInvite:                true,
    MaxBulkInviteCount:              100,
    InvitationBaseURL:               "http://localhost:8080",
    InvitationRedirectPath:          "/reset-password",
    EnableAutoLoginAfterPasswordReset: true,
}

manager := orgboss.NewManager(config)
```

### Configuration Options

- `InvitationExpiryDuration`: Duration before invitations expire (default: 24 hours)
- `DefaultRole`: Default role for new users (default: `RoleUser`)
- `EnableBulkInvite`: Enable bulk invitation feature (default: `true`)
- `MaxBulkInviteCount`: Maximum number of invitations per bulk operation (default: 100)
- `InvitationBaseURL`: Base URL for invitation links (required for email invitations)
- `InvitationRedirectPath`: Redirect path after invitation acceptance (default: `/reset-password`)
- `EnableAutoLoginAfterPasswordReset`: Enable auto-login after password reset (default: `true`)

## Storage

By default, orgboss uses an in-memory storage implementation suitable for testing. For production use, implement the `Storage` interface and use `NewManagerWithStorage`.

### Using PostgreSQL Storage

```go
import (
    "github.com/b4m-oss/orgboss/internal/database"
    "github.com/b4m-oss/orgboss/internal/storage"
)

// Connect to PostgreSQL
db, err := database.Connect()
if err != nil {
    panic(err)
}

// Run migrations
err = database.Migrate(db)
if err != nil {
    panic(err)
}

// Create PostgreSQL storage
postgresStorage := storage.NewPostgresStorage(db)

// Create manager with PostgreSQL storage
config := orgboss.DefaultConfig()
config.EmailSender = email.NewSMTPEmailSender() // Required for invitations
manager := orgboss.NewManagerWithStorage(config, postgresStorage)
```

## Examples

### Inviting Users

```go
// Invite a single user
invitation, err := manager.InviteUser(ctx, org.ID, "user@example.com")
if err != nil {
    panic(err)
}

// Bulk invite multiple users
emails := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
invitations, err := manager.InviteUsers(ctx, org.ID, emails)
if err != nil {
    panic(err)
}
```

### Accepting Invitations

```go
// Accept an invitation (creates user with random password)
user, err := manager.AcceptInvitation(ctx, invitation.Token)
if err != nil {
    panic(err)
}

// Update password (invitation status becomes accepted)
err = manager.UpdatePassword(ctx, user.ID, org.ID, "newpassword123")
if err != nil {
    panic(err)
}
```

### Permission Checking

```go
// Check if user has permission to perform an action
err := manager.CheckPermission(ctx, userID, orgID, "update")
if err != nil {
    // Permission denied
}

// Validate organization access
err := manager.ValidateOrganizationAccess(ctx, userID, orgID)
if err != nil {
    // Access denied
}
```

## Customization

### Custom RoleChecker

```go
type CustomRoleChecker struct{}

func (c *CustomRoleChecker) HasPermission(role orgboss.Role, action string) bool {
    switch role {
    case orgboss.RoleManager:
        return true
    case orgboss.RoleUser:
        return action == "read" || action == "update"
    default:
        return false
    }
}

config := orgboss.DefaultConfig()
config.RoleChecker = &CustomRoleChecker{}
manager := orgboss.NewManager(config)
```

### Custom DeletionHandler

```go
type CustomDeletionHandler struct {
    storage orgboss.Storage
}

func (h *CustomDeletionHandler) DeleteUser(ctx context.Context, user *orgboss.User) error {
    // Implement custom deletion logic (e.g., email masking, anonymization)
    user.Email = "deleted@example.com"
    return h.storage.UpdateUser(ctx, user)
}

func (h *CustomDeletionHandler) DeleteOrganization(ctx context.Context, org *orgboss.Organization) error {
    // Implement custom organization deletion logic
    return h.storage.DeleteOrganization(ctx, org.ID)
}

func (h *CustomDeletionHandler) SetStorage(storage orgboss.Storage) {
    h.storage = storage
}

config := orgboss.DefaultConfig()
config.DeletionHandler = &CustomDeletionHandler{}
manager := orgboss.NewManager(config)
```

### Custom EmailSender

```go
import "github.com/b4m-oss/orgboss/internal/email"

// Use built-in SMTP email sender
smtpSender := email.NewSMTPEmailSender()
// Configure via environment variables:
// SMTP_HOST=localhost
// SMTP_PORT=1025
// SMTP_FROM=noreply@example.com

config := orgboss.DefaultConfig()
config.EmailSender = smtpSender
config.InvitationBaseURL = "http://localhost:8080"
manager := orgboss.NewManager(config)
```

### Using Hooks

```go
manager := orgboss.NewManager(nil)

// Register hook before organization creation
manager.Hooks().BeforeOrganizationCreate = func(ctx context.Context, data interface{}) error {
    // Custom logic before organization creation
    return nil
}

// Register hook after organization creation
manager.Hooks().AfterOrganizationCreate = func(ctx context.Context, data interface{}) error {
    org := data.(*orgboss.Organization)
    // Custom logic after organization creation
    return nil
}
```

## Authboss Integration

orgboss integrates seamlessly with [Authboss v3](https://github.com/aarondl/authboss) for authentication.

### Setup

```go
import (
    "github.com/aarondl/authboss/v3"
    authbossuser "github.com/b4m-oss/orgboss/internal/authboss"
    "gorm.io/gorm"
)

// Setup Authboss with orgboss integration
ab := &authboss.Authboss{}
err := authbossuser.SetupAuthboss(db, ab)
if err != nil {
    panic(err)
}

// Enable auto-login after password reset
err = authbossuser.SetupAuthbossWithAutoLogin(db, ab, true)
if err != nil {
    panic(err)
}
```

### User Model Extension

orgboss extends Authboss's User model with `organization_id` and `role` fields. The `authbossuser.User` struct implements Authboss's User interface.

## Development

### Prerequisites

- Go 1.24 or higher
- Docker and Docker Compose
- PostgreSQL (via Docker Compose)

### Running Tests

```bash
# Run unit tests
make test

# Run integration tests (requires Docker)
make test-integration

# Run tests with coverage
make test COV=TRUE
```

### Development Environment

```bash
# Start development environment (PostgreSQL, Mailpit)
make up

# Run database migrations
make migrate

# Stop development environment
make down
```

### Available Make Commands

- `make up` - Start development environment
- `make down` - Stop development environment
- `make test` - Run unit tests
- `make test-integration` - Run integration tests
- `make fmt` - Format code
- `make tidy` - Tidy dependencies
- `make migrate` - Run database migrations

## Version

Current version: **0.3.0-.rc1**

## CAUTION: NEVER USE ON PRODUCTION

This module is not stable.

## LICENSE

MIT License - see [LICENSE](LICENSE) file for details.

----

That's all