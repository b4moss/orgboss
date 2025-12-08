package types

import "time"

// Role はユーザーのロールを表す
type Role string

const (
	RoleManager Role = "manager"
	RoleUser    Role = "user"
)

// InvitationStatus は招待のステータスを表す
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusRejected InvitationStatus = "rejected"
	InvitationStatusExpired  InvitationStatus = "expired"
)

// Organization は組織を表す
type Organization struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Signature string    `gorm:"uniqueIndex;not null"` // 組織の一意識別子（法人番号またはランダム文字列）。ユニーク制約あり。
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

// User はユーザーを表す（AuthbossのUserモデルを拡張）
type User struct {
	ID             uint   `gorm:"primaryKey"`
	Email          string `gorm:"uniqueIndex;not null"`
	Password       string `gorm:"default:''"` // Authbossでハッシュ化されたパスワード（既存データ対応のためnullable）
	OrganizationID uint   `gorm:"not null;index"`
	Role           Role   `gorm:"not null;default:'user'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time `gorm:"index"`
}

// Invitation は招待を表す
type Invitation struct {
	ID             uint            `gorm:"primaryKey"`
	Email          string          `gorm:"not null;index"`
	OrganizationID uint            `gorm:"not null;index"`
	Token          string          `gorm:"uniqueIndex;not null"`
	ExpiresAt      time.Time       `gorm:"not null"`
	Status         InvitationStatus `gorm:"not null;default:'pending'"`
	CreatedAt      time.Time
}

