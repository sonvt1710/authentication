package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	"github.com/lee-tech/authentication/internal/repository"
	coreServer "github.com/lee-tech/core/server"
)

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrDepartmentNotFound   = errors.New("department not found")
	ErrUserNotFound         = errors.New("user not found")
)

// OrganizationService coordinates tenant hierarchy and membership management.
type OrganizationService struct {
	orgRepo  *repository.OrganizationRepository
	userRepo *repository.UserRepository
}

// NewOrganizationService constructs the service.
func NewOrganizationService(orgRepo *repository.OrganizationRepository, userRepo *repository.UserRepository) *OrganizationService {
	return &OrganizationService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
	}
}

// CreateOrganization provisions a new organization record.
func (s *OrganizationService) CreateOrganization(input *models.CreateOrganizationInput) (*models.Organization, error) {
	if input == nil {
		return nil, fmt.Errorf("input required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("organization name is required")
	}

	var parent *models.Organization
	var err error
	if input.ParentID != nil {
		parent, err = s.orgRepo.GetOrganizationByID(*input.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrOrganizationNotFound
		}
	}

	org := &models.Organization{
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Domain:      strings.TrimSpace(strings.ToLower(input.Domain)),
		ParentID:    input.ParentID,
		IsActive:    true,
	}
	if input.IsActive != nil {
		org.IsActive = *input.IsActive
	}

	if err := s.orgRepo.CreateOrganization(org); err != nil {
		return nil, err
	}

	if parent != nil {
		org.Parent = parent
	}

	return org, nil
}

// ListOrganizations returns all organizations.
func (s *OrganizationService) ListOrganizations() ([]*models.Organization, error) {
	return s.orgRepo.ListOrganizations()
}

// CreateDepartment provisions a new department under an organization.
func (s *OrganizationService) CreateDepartment(input *models.CreateDepartmentInput) (*models.Department, error) {
	if input == nil {
		return nil, fmt.Errorf("input required")
	}
	if input.OrganizationID == 0 {
		return nil, fmt.Errorf("organization_id is required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, fmt.Errorf("department name is required")
	}

	org, err := s.orgRepo.GetOrganizationByID(input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrganizationNotFound
	}

	var parentDept *models.Department
	if input.ParentID != nil {
		parentDept, err = s.orgRepo.GetDepartmentByID(*input.ParentID)
		if err != nil {
			return nil, err
		}
		if parentDept == nil {
			return nil, ErrDepartmentNotFound
		}
		if parentDept.OrganizationID != input.OrganizationID {
			return nil, fmt.Errorf("parent department belongs to another organization")
		}
	}

	kind := input.Kind
	if kind == "" {
		kind = models.DepartmentKindDepartment
	}

	dept := &models.Department{
		OrganizationID: input.OrganizationID,
		ParentID:       input.ParentID,
		Name:           name,
		Kind:           kind,
		Description:    strings.TrimSpace(input.Description),
		Function:       strings.TrimSpace(input.Function),
		IsActive:       true,
	}
	if input.Code != nil {
		code := strings.TrimSpace(string(*input.Code))
		if code != "" {
			c := models.DepartmentCode(code)
			dept.Code = &c
		}
	}
	if input.IsActive != nil {
		dept.IsActive = *input.IsActive
	}

	if err := s.orgRepo.CreateDepartment(dept); err != nil {
		return nil, err
	}

	if parentDept != nil {
		dept.Parent = parentDept
	}

	return dept, nil
}

// ListDepartments returns departments for an organization.
func (s *OrganizationService) ListDepartments(orgID *uint64) ([]*models.Department, error) {
	if orgID == nil {
		return nil, fmt.Errorf("organization_id is required")
	}
	return s.orgRepo.ListDepartmentsByOrganization(*orgID)
}

// AssignUserToOrganization associates a user with an organization and optionally marks it as primary.
func (s *OrganizationService) AssignUserToOrganization(input *models.AssignUserOrganizationInput) (*models.UserOrganization, error) {
	if input == nil {
		return nil, fmt.Errorf("input required")
	}
	if input.UserID == 0 {
		return nil, fmt.Errorf("user_id is required")
	}
	if input.OrganizationID == 0 {
		return nil, fmt.Errorf("organization_id is required")
	}

	user, err := s.userRepo.GetByID(input.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	org, err := s.orgRepo.GetOrganizationByID(input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrganizationNotFound
	}

	if input.IsPrimary {
		if err := s.orgRepo.ClearPrimaryOrganization(input.UserID); err != nil {
			return nil, err
		}
	}

	if err := s.orgRepo.UpsertUserOrganization(input.UserID, input.OrganizationID, input.Role, input.IsPrimary); err != nil {
		return nil, err
	}

	if input.IsPrimary {
		if err := s.orgRepo.SetUserPrimaryOrganization(input.UserID, input.OrganizationID); err != nil {
			return nil, err
		}
	}

	membership, err := s.orgRepo.GetUserOrganization(input.UserID, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// AssignUserToDepartment associates a user with a department and optionally marks it as primary.
func (s *OrganizationService) AssignUserToDepartment(input *models.AssignUserDepartmentInput) (*models.UserDepartment, error) {
	if input == nil {
		return nil, fmt.Errorf("input required")
	}
	if input.UserID == nil {
		return nil, fmt.Errorf("user_id is required")
	}
	if input.DepartmentID == nil {
		return nil, fmt.Errorf("department_id is required")
	}

	user, err := s.userRepo.GetByID(*input.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	dept, err := s.orgRepo.GetDepartmentByID(*input.DepartmentID)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, ErrDepartmentNotFound
	}

	if input.IsPrimary {
		if err := s.orgRepo.ClearPrimaryDepartment(*input.UserID); err != nil {
			return nil, err
		}
	}

	if err := s.orgRepo.UpsertUserDepartment(*input.UserID, *input.DepartmentID, input.Role, input.IsPrimary); err != nil {
		return nil, err
	}

	if input.IsPrimary {
		if err := s.orgRepo.SetUserPrimaryDepartment(*input.UserID, *input.DepartmentID); err != nil {
			return nil, err
		}
	}

	membership, err := s.orgRepo.GetUserDepartment(*input.UserID, *input.DepartmentID)
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// ListUserOrganizations returns the organizations associated with a user.
func (s *OrganizationService) ListUserOrganizations(userID *uint64) ([]*models.UserOrganization, error) {
	if userID == nil {
		return nil, fmt.Errorf("user_id is required")
	}
	return s.orgRepo.ListUserOrganizations(*userID)
}

// ListUserDepartments returns the departments associated with a user.
func (s *OrganizationService) ListUserDepartments(userID *uint64) ([]*models.UserDepartment, error) {
	if userID == nil {
		return nil, fmt.Errorf("user_id is required")
	}
	return s.orgRepo.ListUserDepartments(*userID)
}

// RemoveUserOrganization removes a user's membership from an organization.
func (s *OrganizationService) RemoveUserOrganization(userID, orgID *uint64) error {
	if userID == nil || orgID == nil {
		return fmt.Errorf("user_id and organization_id are required")
	}
	if err := s.orgRepo.RemoveUserOrganization(*userID, *orgID); err != nil {
		return err
	}
	return nil
}

// RemoveUserDepartment removes a user's membership from a department.
func (s *OrganizationService) RemoveUserDepartment(userID, deptID *uint64) error {
	if userID == nil || deptID == nil {
		return fmt.Errorf("user_id and department_id are required")
	}
	return s.orgRepo.RemoveUserDepartment(*userID, *deptID)
}

func init() {
	coreServer.RegisterService(constants.ComponentKey.OrganizationService, func(app *coreServer.HTTPApp) (interface{}, error) {
		orgRepoComponent, ok := app.GetComponent(constants.ComponentKey.OrganizationRepository)
		if !ok {
			return nil, fmt.Errorf("component %s not found", constants.ComponentKey.OrganizationRepository)
		}
		orgRepo, ok := orgRepoComponent.(*repository.OrganizationRepository)
		if !ok {
			return nil, fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.OrganizationRepository, orgRepoComponent)
		}

		userRepoComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationUserRepo)
		if !ok {
			return nil, fmt.Errorf("component %s not found", constants.ComponentKey.AuthenticationUserRepo)
		}
		userRepo, ok := userRepoComponent.(*repository.UserRepository)
		if !ok {
			return nil, fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.AuthenticationUserRepo, userRepoComponent)
		}

		return NewOrganizationService(orgRepo, userRepo), nil
	})
}
