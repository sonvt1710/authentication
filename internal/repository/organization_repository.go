package repository

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	coreServer "github.com/lee-tech/core/server"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrganizationRepository handles organization, department, and membership persistence.
type OrganizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository constructs a new repository instance.
func NewOrganizationRepository(db *gorm.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// CreateOrganization persists a new organization.
func (r *OrganizationRepository) CreateOrganization(org *models.Organization) error {
	return r.db.Create(org).Error
}

// EnsureOrganization finds or creates an organization with the supplied identifiers.
func (r *OrganizationRepository) EnsureOrganization(name, description, domain string) (*models.Organization, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("organization name is required")
	}

	cleanDomain := strings.TrimSpace(domain)
	cleanName := strings.TrimSpace(name)

	var org models.Organization
	query := r.db.Model(&models.Organization{})
	if cleanDomain != "" {
		if err := query.Where("domain = ?", cleanDomain).First(&org).Error; err == nil {
			return r.updateOrganizationDefaults(&org, description, cleanDomain, true)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if err := r.db.Where("name = ?", cleanName).First(&org).Error; err == nil {
		return r.updateOrganizationDefaults(&org, description, cleanDomain, true)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	org = models.Organization{
		Name:        cleanName,
		Description: strings.TrimSpace(description),
		Domain:      cleanDomain,
		IsActive:    true,
	}
	if err := r.db.Create(&org).Error; err != nil {
		return nil, err
	}

	return &org, nil
}

func (r *OrganizationRepository) updateOrganizationDefaults(org *models.Organization, description, domain string, isActive bool) (*models.Organization, error) {
	updates := map[string]any{
		"is_active": isActive,
	}
	if strings.TrimSpace(description) != "" && org.Description != description {
		updates["description"] = strings.TrimSpace(description)
	}
	if strings.TrimSpace(domain) != "" && org.Domain != strings.TrimSpace(domain) {
		updates["domain"] = strings.TrimSpace(domain)
	}
	if len(updates) > 0 {
		if err := r.db.Model(org).Updates(updates).Error; err != nil {
			return nil, err
		}
		if err := r.db.First(org, "id = ?", org.ID).Error; err != nil {
			return nil, err
		}
	}
	return org, nil
}

// UpdateOrganization updates an existing organization.
func (r *OrganizationRepository) UpdateOrganization(org *models.Organization) error {
	return r.db.Save(org).Error
}

// GetOrganizationByID fetches an organization with optional relationships.
func (r *OrganizationRepository) GetOrganizationByID(id uint64) (*models.Organization, error) {
	var org models.Organization
	err := r.db.
		Preload("Departments").
		Preload("Children").
		First(&org, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &org, nil
}

// ListOrganizations returns all organizations ordered by name.
func (r *OrganizationRepository) ListOrganizations() ([]*models.Organization, error) {
	var orgs []*models.Organization
	if err := r.db.
		Model(&models.Organization{}).
		Order("name ASC").
		Find(&orgs).Error; err != nil {
		return nil, err
	}
	return orgs, nil
}

// CreateDepartment persists a new department.
func (r *OrganizationRepository) CreateDepartment(dept *models.Department) error {
	return r.db.Create(dept).Error
}

// GetDepartmentByID fetches a department with its relationships.
func (r *OrganizationRepository) GetDepartmentByID(id uint64) (*models.Department, error) {
	var dept models.Department
	err := r.db.
		Preload("Children").
		First(&dept, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dept, nil
}

// ListDepartmentsByOrganization returns departments for a given organization.
func (r *OrganizationRepository) ListDepartmentsByOrganization(orgID uint64) ([]*models.Department, error) {
	var departments []*models.Department
	err := r.db.
		Model(&models.Department{}).
		Where("organization_id = ?", orgID).
		Order("name ASC").
		Find(&departments).Error
	return departments, err
}

// ListUserOrganizations returns the organizations a user belongs to together with membership metadata.
func (r *OrganizationRepository) ListUserOrganizations(userID uint64) ([]*models.UserOrganization, error) {
	var memberships []*models.UserOrganization
	err := r.db.
		Preload("Organization").
		Where("user_id = ?", userID).
		Order("is_primary DESC, updated_at DESC").
		Find(&memberships).Error
	return memberships, err
}

// ListUserDepartments returns the departments a user belongs to together with membership metadata.
func (r *OrganizationRepository) ListUserDepartments(userID uint64) ([]*models.UserDepartment, error) {
	var memberships []*models.UserDepartment
	err := r.db.
		Preload("Department").
		Where("user_id = ?", userID).
		Order("is_primary DESC, updated_at DESC").
		Find(&memberships).Error
	return memberships, err
}

// UpsertUserOrganization creates or updates membership between a user and organization.
func (r *OrganizationRepository) UpsertUserOrganization(userID, orgID uint64, role models.OrganizationRole, isPrimary bool) error {
	membership := &models.UserOrganization{
		UserID:         userID,
		OrganizationID: orgID,
		Role:           role,
		IsPrimary:      isPrimary,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "organization_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "is_primary", "updated_at"}),
	}).Create(membership).Error
}

// GetUserOrganization fetches a single membership entry between a user and organization.
func (r *OrganizationRepository) GetUserOrganization(userID, orgID uint64) (*models.UserOrganization, error) {
	var membership models.UserOrganization
	err := r.db.
		Preload("Organization").
		First(&membership, "user_id = ? AND organization_id = ?", userID, orgID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &membership, nil
}

// UpsertUserDepartment creates or updates membership between a user and department.
func (r *OrganizationRepository) UpsertUserDepartment(userID, deptID uint64, role string, isPrimary bool) error {
	membership := &models.UserDepartment{
		UserID:       userID,
		DepartmentID: deptID,
		Role:         role,
		IsPrimary:    isPrimary,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "department_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "is_primary", "updated_at"}),
	}).Create(membership).Error
}

// GetUserDepartment fetches a single membership entry between a user and department.
func (r *OrganizationRepository) GetUserDepartment(userID, deptID uint64) (*models.UserDepartment, error) {
	var membership models.UserDepartment
	err := r.db.
		Preload("Department").
		First(&membership, "user_id = ? AND department_id = ?", userID, deptID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &membership, nil
}

// ClearPrimaryOrganization resets the primary flag for all user organization memberships.
func (r *OrganizationRepository) ClearPrimaryOrganization(userID uint64) error {
	return r.db.Model(&models.UserOrganization{}).
		Where("user_id = ?", userID).
		Update("is_primary", false).Error
}

// ClearPrimaryDepartment resets the primary flag for all user department memberships.
func (r *OrganizationRepository) ClearPrimaryDepartment(userID uint64) error {
	return r.db.Model(&models.UserDepartment{}).
		Where("user_id = ?", userID).
		Update("is_primary", false).Error
}

// SetUserPrimaryOrganization updates the user record with the primary organization.
func (r *OrganizationRepository) SetUserPrimaryOrganization(userID, orgID uint64) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("primary_organization_id", orgID).Error
}

// SetUserPrimaryDepartment updates the user record with the primary department.
func (r *OrganizationRepository) SetUserPrimaryDepartment(userID, deptID uint64) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("primary_department_id", deptID).Error
}

// RemoveUserOrganization removes a membership entry.
func (r *OrganizationRepository) RemoveUserOrganization(userID, orgID uint64) error {
	return r.db.Delete(&models.UserOrganization{}, "user_id = ? AND organization_id = ?", userID, orgID).Error
}

// RemoveUserDepartment removes a department membership.
func (r *OrganizationRepository) RemoveUserDepartment(userID, deptID uint64) error {
	return r.db.Delete(&models.UserDepartment{}, "user_id = ? AND department_id = ?", userID, deptID).Error
}

func init() {
	coreServer.RegisterRepository(constants.ComponentKey.OrganizationRepository, func(app *coreServer.HTTPApp) (interface{}, error) {
		if app.DB == nil {
			return nil, fmt.Errorf("database not initialised")
		}
		return NewOrganizationRepository(app.DB), nil
	})
}
