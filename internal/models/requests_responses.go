package models

import (
	coreServer "github.com/lee-tech/core/server"
)

// OrganizationMembershipInfo exposes basic organization membership details.
type OrganizationMembershipInfo struct {
	OrganizationID   uint64 `json:"organization_id"`
	OrganizationName string `json:"organization_name,omitempty"`
	Role             string `json:"role,omitempty"`
	IsPrimary        bool   `json:"is_primary"`
}

// DepartmentMembershipInfo exposes basic department membership details.
type DepartmentMembershipInfo struct {
	DepartmentID   uint64 `json:"department_id"`
	DepartmentName string `json:"department_name,omitempty"`
	Role           string `json:"role,omitempty"`
	IsPrimary      bool   `json:"is_primary"`
}

// UserInfo represents public user information
type UserInfo struct {
	ID                    uint64                       `json:"id"`
	Email                 string                       `json:"email"`
	Username              string                       `json:"username"`
	FirstName             string                       `json:"first_name"`
	LastName              string                       `json:"last_name"`
	PrimaryOrganizationID *uint64                      `json:"primary_organization_id,omitempty"`
	PrimaryDepartmentID   *uint64                      `json:"primary_department_id,omitempty"`
	IsSuperAdmin          bool                         `json:"is_super_admin"`
	MFAEnabled            bool                         `json:"mfa_enabled"`
	Organizations         []OrganizationMembershipInfo `json:"organizations,omitempty"`
	Departments           []DepartmentMembershipInfo   `json:"departments,omitempty"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username       string `json:"username" validate:"required"`
	Password       string `json:"password" validate:"required"`
	OrganizationID uint64 `json:"organization_id" validate:"required"`
	DepartmentID   uint64 `json:"department_id,omitempty" validate:"omitempty"` // CEO seems doesn't need department_id.
	RoleID         uint64 `json:"role_id,omitempty" validate:"omitempty"`       // You can Select Role with departments, at least role_id or department_id is required.
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	AccessToken        string        `json:"access_token"`
	RefreshToken       string        `json:"refresh_token"`
	ExpiresIn          int           `json:"expires_in"`
	TokenType          string        `json:"token_type"`
	User               *UserInfo     `json:"user"`
	LoggedOrganization *Organization `json:"logged_organization,omitempty"`
	LoggedDepartment   *Department   `json:"logged_department,omitempty"`
}

// CreateOrganizationInput captures the data required to create a new organization.
type CreateOrganizationInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Domain      string  `json:"domain"`
	ParentID    *uint64 `json:"parent_id,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// CreateDepartmentInput captures the data required to create a new department.
type CreateDepartmentInput struct {
	OrganizationID uint64          `json:"organization_id"`
	ParentID       *uint64         `json:"parent_id,omitempty"`
	Code           *DepartmentCode `json:"code,omitempty"`
	Name           string          `json:"name"`
	Kind           DepartmentKind  `json:"kind"`
	Description    string          `json:"description"`
	Function       string          `json:"function"`
	IsActive       *bool           `json:"is_active,omitempty"`
}

// AssignUserOrganizationInput represents a request to associate a user with an organization.
type AssignUserOrganizationInput struct {
	UserID         uint64           `json:"user_id"`
	OrganizationID uint64           `json:"organization_id"`
	Role           OrganizationRole `json:"role"`
	IsPrimary      bool             `json:"is_primary"`
}

// AssignUserDepartmentInput represents a request to associate a user with a department.
type AssignUserDepartmentInput struct {
	UserID       *uint64 `json:"user_id"`
	DepartmentID *uint64 `json:"department_id"`
	Role         string  `json:"role"`
	IsPrimary    bool    `json:"is_primary"`
}

func init() {
	coreServer.RegisterSchemaType("login-request", LoginRequest{})
	coreServer.RegisterSchemaType("login-response", LoginResponse{})
}
