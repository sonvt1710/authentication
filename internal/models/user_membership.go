package models

import (
	"time"

	coreServer "github.com/lee-tech/core/server"
	"gorm.io/gorm"
)

// UserOrganization represents the association between a user and an organization.
type UserOrganization struct {
	UserID         uint64           `gorm:"type:bigint;primaryKey" json:"user_id"`
	OrganizationID uint64           `gorm:"type:bigint;primaryKey" json:"organization_id"`
	Role           OrganizationRole `gorm:"size:128" json:"role"`
	IsPrimary      bool             `gorm:"default:false" json:"is_primary"`
	User           *User            `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Organization   *Organization    `gorm:"foreignKey:OrganizationID;references:ID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// UserDepartment represents the association between a user and a department.
type UserDepartment struct {
	UserID       uint64      `gorm:"type:bigint;primaryKey" json:"user_id"`
	DepartmentID uint64      `gorm:"type:bigint;primaryKey" json:"department_id"`
	Role         string      `gorm:"size:128" json:"role"`
	IsPrimary    bool        `gorm:"default:false" json:"is_primary"`
	User         *User       `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Department   *Department `gorm:"foreignKey:DepartmentID;references:ID;constraint:OnDelete:CASCADE" json:"department,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func init() {
	coreServer.RegisterMigration(func() interface{} { return &UserOrganization{} })
	coreServer.RegisterMigration(func() interface{} { return &UserDepartment{} })
}
