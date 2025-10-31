package models

import (
	"time"

	coreServer "github.com/lee-tech/core/server"
	"gorm.io/gorm"
)

// Organization represents a tenant or company within the system.
type Organization struct {
	ID          uint64 `json:"id" gorm:"primaryKey;autoIncrement;type:bigint"`
	Name        string `gorm:"size:255;not null" json:"name"`
	Description string `gorm:"size:1024" json:"description"`
	Domain      string `gorm:"size:255;uniqueIndex" json:"domain"`
	IsActive    bool   `gorm:"default:true" json:"is_active"`

	ParentID *uint64        `gorm:"type:bigint;index" json:"parent_id,omitempty"`
	Parent   *Organization  `gorm:"constraint:OnDelete:SET NULL" json:"parent,omitempty"`
	Children []Organization `gorm:"foreignKey:ParentID" json:"children,omitempty"`

	Departments []Department `gorm:"constraint:OnDelete:CASCADE" json:"departments,omitempty"`
	Users       []User       `gorm:"many2many:user_organizations;joinForeignKey:OrganizationID;joinReferences:UserID;constraint:OnDelete:CASCADE" json:"users,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Department represents a sub-division within an organization.
type Department struct {
	ID             uint64          `json:"id" gorm:"primaryKey;autoIncrement;type:bigint"`
	OrganizationID uint64          `gorm:"type:bigint;index" json:"organization_id"`
	Code           *DepartmentCode `gorm:"size:64" json:"code,omitempty"`
	Name           string          `gorm:"size:255;not null" json:"name"`
	Kind           DepartmentKind  `gorm:"size:32;default:'DEPARTMENT'" json:"kind"`
	Description    string          `gorm:"size:1024" json:"description"`
	Function       string          `gorm:"size:1024" json:"function"`
	IsActive       bool            `gorm:"default:true" json:"is_active"`
	ParentID       *uint64         `gorm:"type:bigint;index" json:"parent_id,omitempty"`
	Organization   *Organization   `gorm:"constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	Parent         *Department     `gorm:"constraint:OnDelete:SET NULL" json:"parent,omitempty"`
	Children       []Department    `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Users          []User          `gorm:"many2many:user_departments;joinForeignKey:DepartmentID;joinReferences:UserID;constraint:OnDelete:CASCADE" json:"users,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate ensures Kind are present on insert.
func (d *Department) BeforeCreate(tx *gorm.DB) error {
	if d.Kind == "" {
		d.Kind = DepartmentKindDepartment
	}
	return nil
}

func init() {
	coreServer.RegisterMigration(func() interface{} { return &Organization{} })
	coreServer.RegisterMigration(func() interface{} { return &Department{} })
}
