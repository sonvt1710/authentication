package models

import (
	"time"

	coreServer "github.com/lee-tech/core/server"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID           uint64 `gorm:"type:bigint;primaryKey" json:"id"`
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	Username     string `gorm:"uniqueIndex;not null" json:"username"`
	Password     string `gorm:"not null" json:"-"` // Never expose password in JSON
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	IsActive     bool   `gorm:"default:true" json:"is_active"`
	IsVerified   bool   `gorm:"default:false" json:"is_verified"`
	IsSuperAdmin bool   `gorm:"default:false" json:"is_super_admin"`

	// Primary organization relationship (for default context)
	PrimaryOrganizationID *uint64       `gorm:"type:bigint;index" json:"primary_organization_id,omitempty"`
	PrimaryOrganization   *Organization `json:"primary_organization,omitempty"`

	// Primary department relationship (for default context)
	PrimaryDepartmentID *uint64     `gorm:"type:bigint;index" json:"primary_department_id,omitempty"`
	PrimaryDepartment   *Department `json:"primary_department,omitempty"`

	// Memberships (many-to-many)
	Organizations []*Organization `gorm:"many2many:user_organizations;joinForeignKey:UserID;joinReferences:OrganizationID;constraint:OnDelete:CASCADE" json:"organizations,omitempty"`
	Departments   []*Department   `gorm:"many2many:user_departments;joinForeignKey:UserID;joinReferences:DepartmentID;constraint:OnDelete:CASCADE" json:"departments,omitempty"`

	// Security fields
	LastLogin           *time.Time `json:"last_login,omitempty"`
	LoginAttempts       int        `gorm:"default:0" json:"-"`
	LockedUntil         *time.Time `json:"-"`
	PasswordResetToken  *string    `json:"-"`
	PasswordResetExpiry *time.Time `json:"-"`
	VerificationToken   *string    `json:"-"`

	// MFA fields
	MFAEnabled bool    `gorm:"default:false" json:"mfa_enabled"`
	MFASecret  *string `json:"-"`

	// Timestamps
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// ToUserInfo converts User to UserInfo
func (u *User) ToUserInfo() *UserInfo {
	return &UserInfo{
		ID:                    u.ID,
		Email:                 u.Email,
		Username:              u.Username,
		FirstName:             u.FirstName,
		LastName:              u.LastName,
		PrimaryOrganizationID: u.PrimaryOrganizationID,
		PrimaryDepartmentID:   u.PrimaryDepartmentID,
		IsSuperAdmin:          u.IsSuperAdmin,
		MFAEnabled:            u.MFAEnabled,
	}
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RegisterRequest represents user registration data
type RegisterRequest struct {
	Email                 string  `json:"email" validate:"required,email"`
	Username              string  `json:"username" validate:"required,min=3,max=50"`
	Password              string  `json:"password" validate:"required,min=8"`
	FirstName             string  `json:"first_name" validate:"required"`
	LastName              string  `json:"last_name" validate:"required"`
	PrimaryOrganizationID *uint64 `json:"primary_organization_id,omitempty"`
}

func init() {
	coreServer.RegisterMigration(func() interface{} { return &User{} })
}
