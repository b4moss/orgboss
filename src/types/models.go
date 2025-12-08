package types

import "time"

// Role represents a user's role
type Role string

const (
	RoleManager Role = "manager"
	RoleUser    Role = "user"
)

// InvitationStatus represents the status of an invitation
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusRejected InvitationStatus = "rejected"
	InvitationStatusExpired  InvitationStatus = "expired"
)

// Organization represents an organization
type Organization struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Signature string    `gorm:"uniqueIndex;not null"` // Unique identifier for the organization (corporate number or random string). Has unique constraint.
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// User represents a user (extends Authboss's User model)
type User struct {
	ID             uint   `gorm:"primaryKey"`
	Email          string `gorm:"uniqueIndex;not null"`
	Password       string `gorm:"default:''"` // Password hashed by Authboss (nullable for existing data compatibility)
	OrganizationID uint   `gorm:"not null;index"`
	Role           Role   `gorm:"not null;default:'user'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"index"`
}

// Invitation represents an invitation
type Invitation struct {
	ID             uint            `gorm:"primaryKey"`
	Email          string          `gorm:"not null;index"`
	OrganizationID uint            `gorm:"not null;index"`
	Token          string          `gorm:"uniqueIndex;not null"`
	ExpiresAt      time.Time       `gorm:"not null"`
	Status         InvitationStatus `gorm:"not null;default:'pending'"`
	CreatedAt      time.Time
}

