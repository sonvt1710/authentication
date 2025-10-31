package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/lee-tech/authentication/config"
	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	"github.com/lee-tech/authentication/internal/repository"
	coreServer "github.com/lee-tech/core/server"
	"github.com/lee-tech/core/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountLocked      = errors.New("account is locked due to too many failed attempts")
	ErrAccountInactive    = errors.New("account is not active")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
)

// AuthenticationService handles authentication business logic
type AuthenticationService struct {
	userRepo *repository.UserRepository
	orgRepo  *repository.OrganizationRepository
	config   *config.AuthConfig
}

// BootstrapAdminInput describes the desired bootstrap configuration for the root administrator.
type BootstrapAdminInput struct {
	OrganizationName        string
	OrganizationDescription string
	OrganizationDomain      string
	AdminEmail              string
	AdminUsername           string
	AdminPassword           string
	AdminFirstName          string
	AdminLastName           string
	ForcePasswordReset      bool
}

// NewAuthService creates a new auth service
func NewAuthenticationService(userRepo *repository.UserRepository, orgRepo *repository.OrganizationRepository, config *config.AuthConfig) *AuthenticationService {
	return &AuthenticationService{
		userRepo: userRepo,
		orgRepo:  orgRepo,
		config:   config,
	}
}

// BootstrapDefaultAdmin ensures the default organization and super-admin account exist.
func (s *AuthenticationService) BootstrapDefaultAdmin() (*models.Organization, *models.User, error) {
	input := &BootstrapAdminInput{
		OrganizationName:        s.config.BootstrapOrganizationName,
		OrganizationDescription: s.config.BootstrapOrganizationDescription,
		OrganizationDomain:      s.config.BootstrapOrganizationDomain,
		AdminEmail:              s.config.BootstrapAdminEmail,
		AdminUsername:           s.config.BootstrapAdminUsername,
		AdminPassword:           s.config.BootstrapAdminPassword,
		AdminFirstName:          s.config.BootstrapAdminFirstName,
		AdminLastName:           s.config.BootstrapAdminLastName,
	}
	return s.BootstrapAdmin(input)
}

// BootstrapAdmin performs bootstrap/rotation based on the provided input.
func (s *AuthenticationService) BootstrapAdmin(input *BootstrapAdminInput) (*models.Organization, *models.User, error) {
	if s == nil || s.userRepo == nil || s.orgRepo == nil || s.config == nil {
		return nil, nil, fmt.Errorf("authentication service not initialised for bootstrap")
	}

	if input == nil {
		return nil, nil, fmt.Errorf("bootstrap input is required")
	}

	org, err := s.orgRepo.EnsureOrganization(
		input.OrganizationName,
		input.OrganizationDescription,
		input.OrganizationDomain,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("ensure organization: %w", err)
	}

	email := strings.TrimSpace(input.AdminEmail)
	if email == "" {
		return nil, nil, fmt.Errorf("bootstrap admin email is required")
	}

	username := strings.TrimSpace(input.AdminUsername)
	if username == "" {
		username = email
	}

	password := input.AdminPassword
	minPasswordLength := s.config.PasswordMinLength
	if minPasswordLength <= 0 {
		minPasswordLength = 8
	}
	if len(password) < minPasswordLength {
		return nil, nil, fmt.Errorf("bootstrap admin password must be at least %d characters", minPasswordLength)
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, nil, fmt.Errorf("lookup admin user: %w", err)
	}

	if user == nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BCryptCost)
		if err != nil {
			return nil, nil, fmt.Errorf("hash password: %w", err)
		}

		firstName := strings.TrimSpace(input.AdminFirstName)
		if firstName == "" {
			firstName = "System"
		}
		lastName := strings.TrimSpace(input.AdminLastName)
		if lastName == "" {
			lastName = "Administrator"
		}

		user = &models.User{
			Email:                 email,
			Username:              username,
			Password:              string(hashedPassword),
			FirstName:             firstName,
			LastName:              lastName,
			IsActive:              true,
			IsVerified:            true,
			IsSuperAdmin:          true,
			PrimaryOrganizationID: &org.ID,
		}
		if err := s.userRepo.Create(user); err != nil {
			return nil, nil, fmt.Errorf("create admin user: %w", err)
		}
	} else {
		firstName := strings.TrimSpace(input.AdminFirstName)
		if firstName == "" {
			firstName = user.FirstName
		}
		lastName := strings.TrimSpace(input.AdminLastName)
		if lastName == "" {
			lastName = user.LastName
		}

		// Update user profile if necessary.
		user.Username = username
		user.FirstName = firstName
		user.LastName = lastName
		user.IsActive = true
		user.IsVerified = true
		user.IsSuperAdmin = true
		user.PrimaryOrganizationID = &org.ID

		needPasswordUpdate := input.ForcePasswordReset
		if !needPasswordUpdate {
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
				needPasswordUpdate = true
			}
		}
		if needPasswordUpdate {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BCryptCost)
			if err != nil {
				return nil, nil, fmt.Errorf("hash password: %w", err)
			}
			user.Password = string(hashedPassword)
		}

		if err := s.userRepo.Update(user); err != nil {
			return nil, nil, fmt.Errorf("update admin user: %w", err)
		}
	}

	if err := s.orgRepo.UpsertUserOrganization(user.ID, org.ID, models.OrganizationRoleSystemAdmin, true); err != nil {
		return nil, nil, fmt.Errorf("assign admin organization membership: %w", err)
	}
	if err := s.orgRepo.SetUserPrimaryOrganization(user.ID, org.ID); err != nil {
		return nil, nil, fmt.Errorf("set admin primary organization: %w", err)
	}

	return org, user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthenticationService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	// Find user by email or username
	user, err := s.userRepo.GetByEmailOrUsername(req.Username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check if account is locked
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// Check if account is active
	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// Increment login attempts
		s.userRepo.IncrementLoginAttempts(user.ID)

		// Check if we need to lock the account
		if user.LoginAttempts+1 >= s.config.MaxLoginAttempts {
			lockUntil := time.Now().Add(s.config.LockoutDuration)
			s.userRepo.LockAccount(user.ID, lockUntil)
		}

		return nil, ErrInvalidCredentials
	}

	orgMemberships, deptMemberships, err := s.collectMemberships(&user.ID)
	if err != nil {
		return nil, err
	}

	var loggedOrganization *models.Organization

	for _, member := range orgMemberships {
		if member.OrganizationID == uint64(req.OrganizationID) {
			// Role validation
			if role := string(member.Role); role != "" && role != string(models.OrganizationRoleSystemAdmin) {
				return nil, fmt.Errorf("user does not have the required role in the organization")
			}

			org, err := s.orgRepo.GetOrganizationByID(member.OrganizationID)
			if err != nil {
				return nil, fmt.Errorf("failed to get organization: %w", err)
			}

			loggedOrganization = org
			break
		}
	}

	var loggedDepartment *models.Department
	for _, member := range deptMemberships {
		if member.DepartmentID == uint64(req.DepartmentID) {
			dept, err := s.orgRepo.GetDepartmentByID(member.DepartmentID)
			if err != nil {
				return nil, fmt.Errorf("failed to get department: %w", err)
			}
			loggedDepartment = dept
			break
		}
	}

	if loggedOrganization == nil {
		return nil, fmt.Errorf("organization not found or user not a member")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user, orgMemberships, deptMemberships)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Update last login and reset login attempts
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	return &models.LoginResponse{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		ExpiresIn:          int(s.config.TokenExpiration.Seconds()),
		TokenType:          "Bearer",
		User:               s.composeUserInfo(user, orgMemberships, deptMemberships),
		LoggedOrganization: loggedOrganization,
		LoggedDepartment:   loggedDepartment,
	}, nil
}

// Register creates a new user account
func (s *AuthenticationService) Register(req *models.RegisterRequest) (*models.User, error) {
	// Check if email already exists
	exists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("email already registered")
	}

	// Check if username already exists
	exists, err = s.userRepo.ExistsByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("username already taken")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.config.BCryptCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.User{
		Email:                 req.Email,
		Username:              req.Username,
		Password:              string(hashedPassword),
		FirstName:             req.FirstName,
		LastName:              req.LastName,
		PrimaryOrganizationID: req.PrimaryOrganizationID,
		IsActive:              true,
		IsVerified:            false, // Will need email verification
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// RefreshToken validates a refresh token and returns new tokens
func (s *AuthenticationService) RefreshToken(refreshToken string) (*models.LoginResponse, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Check token type
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	// Get user ID from claims
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := utils.ParseUint64(userIDStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Get user from database
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	if user == nil || !user.IsActive {
		return nil, ErrInvalidToken
	}

	orgMemberships, deptMemberships, err := s.collectMemberships(&user.ID)
	if err != nil {
		return nil, err
	}

	// Generate new tokens
	newAccessToken, err := s.generateAccessToken(user, orgMemberships, deptMemberships)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.config.TokenExpiration.Seconds()),
		TokenType:    "Bearer",
		User:         s.composeUserInfo(user, orgMemberships, deptMemberships),
	}, nil
}

// generateAccessToken generates a JWT access token enriched with membership context.
func (s *AuthenticationService) generateAccessToken(user *models.User, orgMemberships []*models.UserOrganization, deptMemberships []*models.UserDepartment) (string, error) {
	now := time.Now()
	expiresAt := now.Add(s.config.TokenExpiration)

	claims := jwt.MapClaims{
		"iss":      s.config.Config.ServiceName,
		"sub":      user.ID,
		"aud":      []string{s.config.Config.ServiceName},
		"exp":      expiresAt.Unix(),
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"jti":      uuid.NewString(),
		"type":     "access",
		"user_id":  user.ID,
		"email":    user.Email,
		"username": user.Username,
	}

	// Add organization ID if present
	if user.PrimaryOrganizationID != nil {
		claims["org_id"] = user.PrimaryOrganizationID
	}

	// Add super admin flag
	if user.IsSuperAdmin {
		claims["is_super_admin"] = true
	}

	if len(orgMemberships) > 0 {
		orgClaims := make([]map[string]any, 0, len(orgMemberships))
		roles := make([]string, 0, len(orgMemberships))
		for _, membership := range orgMemberships {
			if membership == nil {
				continue
			}
			claim := map[string]any{
				"id":         membership.OrganizationID,
				"is_primary": membership.IsPrimary,
			}
			if membership.Organization != nil {
				claim["name"] = membership.Organization.Name
			}
			if membership.Role != "" {
				claim["role"] = string(membership.Role)
				roles = append(roles, string(membership.Role))
			}
			orgClaims = append(orgClaims, claim)
		}
		claims["organizations"] = orgClaims
		if len(roles) > 0 {
			claims["roles"] = uniqueStrings(roles)
		}
	}

	if len(deptMemberships) > 0 {
		deptClaims := make([]map[string]any, 0, len(deptMemberships))
		for _, membership := range deptMemberships {
			if membership == nil {
				continue
			}
			claim := map[string]any{
				"id":         membership.DepartmentID,
				"is_primary": membership.IsPrimary,
			}
			if membership.Department != nil {
				claim["name"] = membership.Department.Name
			}
			if membership.Role != "" {
				claim["role"] = membership.Role
			}
			deptClaims = append(deptClaims, claim)
		}
		claims["departments"] = deptClaims
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Config.JWTSecret))
}

// generateRefreshToken generates a JWT refresh token
func (s *AuthenticationService) generateRefreshToken(user *models.User) (string, error) {
	now := time.Now()
	expiresAt := now.Add(s.config.RefreshExpiration)

	claims := jwt.MapClaims{
		"iss":     s.config.Config.ServiceName,
		"sub":     user.ID,
		"aud":     []string{s.config.Config.ServiceName},
		"exp":     expiresAt.Unix(),
		"iat":     now.Unix(),
		"nbf":     now.Unix(),
		"jti":     uuid.NewString(),
		"type":    "refresh",
		"user_id": user.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Config.JWTSecret))
}

// ValidateToken validates an access token and returns the user ID
func (s *AuthenticationService) ValidateToken(tokenString string) (*uint64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.Config.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Check token type
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "access" {
		return nil, ErrInvalidToken
	}

	// Get user ID from claims
	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userId, err := utils.ParseUint64(userIDStr)
	return &userId, err
}

func (s *AuthenticationService) collectMemberships(userID *uint64) ([]*models.UserOrganization, []*models.UserDepartment, error) {
	if userID == nil || s.orgRepo == nil {
		return nil, nil, nil
	}

	orgs, err := s.orgRepo.ListUserOrganizations(*userID)
	if err != nil {
		return nil, nil, err
	}

	depts, err := s.orgRepo.ListUserDepartments(*userID)
	if err != nil {
		return nil, nil, err
	}

	return orgs, depts, nil
}

func (s *AuthenticationService) composeUserInfo(user *models.User, orgs []*models.UserOrganization, depts []*models.UserDepartment) *models.UserInfo {
	if user == nil {
		return nil
	}
	info := user.ToUserInfo()

	if len(orgs) > 0 {
		memberships := make([]models.OrganizationMembershipInfo, 0, len(orgs))
		for _, membership := range orgs {
			if membership == nil {
				continue
			}
			item := models.OrganizationMembershipInfo{
				OrganizationID: membership.OrganizationID,
				Role:           string(membership.Role),
				IsPrimary:      membership.IsPrimary,
			}
			if membership.Organization != nil {
				item.OrganizationName = membership.Organization.Name
			}
			memberships = append(memberships, item)
		}
		info.Organizations = memberships
	}

	if len(depts) > 0 {
		memberships := make([]models.DepartmentMembershipInfo, 0, len(depts))
		for _, membership := range depts {
			if membership == nil {
				continue
			}
			item := models.DepartmentMembershipInfo{
				DepartmentID: membership.DepartmentID,
				Role:         membership.Role,
				IsPrimary:    membership.IsPrimary,
			}
			if membership.Department != nil {
				item.DepartmentName = membership.Department.Name
			}
			memberships = append(memberships, item)
		}
		info.Departments = memberships
	}

	return info
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, val := range values {
		trimmed := strings.TrimSpace(val)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

// JWTSecret exposes the signing secret used for validating tokens.
func (s *AuthenticationService) JWTSecret() string {
	return s.config.Config.JWTSecret
}

// GetUserByID retrieves a user by UUID.
func (s *AuthenticationService) GetUserByID(id uint64) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

// GetUserInfoByID retrieves a user info projection enriched with membership details.
func (s *AuthenticationService) GetUserInfoByID(id uint64) (*models.UserInfo, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}

	orgs, depts, err := s.collectMemberships(&user.ID)
	if err != nil {
		return nil, err
	}

	return s.composeUserInfo(user, orgs, depts), nil
}

// ListUsers retrieves a paginated list of users with membership context.
func (s *AuthenticationService) ListUsers(offset, limit int) ([]*models.UserInfo, int64, error) {
	users, total, err := s.userRepo.List(offset, limit)
	if err != nil {
		return nil, 0, err
	}

	infos := make([]*models.UserInfo, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		orgs, depts, err := s.collectMemberships(&user.ID)
		if err != nil {
			return nil, 0, err
		}
		infos = append(infos, s.composeUserInfo(user, orgs, depts))
	}

	return infos, total, nil
}

func init() {
	coreServer.RegisterService(constants.ComponentKey.AuthenticationService, func(app *coreServer.HTTPApp) (interface{}, error) {
		repoComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationUserRepo)
		if !ok {
			return nil, fmt.Errorf("component %s not found", constants.ComponentKey.AuthenticationUserRepo)
		}

		userRepo, ok := repoComponent.(*repository.UserRepository)
		if !ok {
			return nil, fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.AuthenticationUserRepo, repoComponent)
		}

		orgRepoComponent, ok := app.GetComponent(constants.ComponentKey.OrganizationRepository)
		if !ok {
			return nil, fmt.Errorf("component %s not found", constants.ComponentKey.OrganizationRepository)
		}

		orgRepo, ok := orgRepoComponent.(*repository.OrganizationRepository)
		if !ok {
			return nil, fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.OrganizationRepository, orgRepoComponent)
		}

		cfgComponent, ok := app.GetComponent(constants.ComponentKey.AuthenticationConfig)
		if !ok {
			return nil, fmt.Errorf("component %s not found", constants.ComponentKey.AuthenticationConfig)
		}

		authCfg, ok := cfgComponent.(*config.AuthConfig)
		if !ok {
			return nil, fmt.Errorf("component %s has unexpected type %T", constants.ComponentKey.AuthenticationConfig, cfgComponent)
		}

		return NewAuthenticationService(userRepo, orgRepo, authCfg), nil
	})
}
