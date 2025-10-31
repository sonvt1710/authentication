package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/lee-tech/authentication/internal/constants"
	"github.com/lee-tech/authentication/internal/models"
	coreServer "github.com/lee-tech/core/server"
	"gorm.io/gorm"
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *gorm.DB
}

func (r *UserRepository) baseQuery() *gorm.DB {
	return r.db.
		Preload("PrimaryOrganization").
		Preload("PrimaryDepartment")
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uint64) (*models.User, error) {
	var user models.User
	err := r.baseQuery().First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.baseQuery().First(&user, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.baseQuery().First(&user, "username = ?", username).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmailOrUsername retrieves a user by email or username
func (r *UserRepository) GetByEmailOrUsername(identifier string) (*models.User, error) {
	var user models.User
	err := r.baseQuery().Where("email = ? OR username = ?", identifier, identifier).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update updates a user in the database
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// UpdateLastLogin updates the last login timestamp for a user
func (r *UserRepository) UpdateLastLogin(userID uint64) error {
	now := time.Now()
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login":     now,
			"login_attempts": 0,
		}).Error
}

// IncrementLoginAttempts increments the login attempts counter
func (r *UserRepository) IncrementLoginAttempts(userID uint64) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("login_attempts", gorm.Expr("login_attempts + ?", 1)).
		Error
}

// LockAccount locks a user account until the specified time
func (r *UserRepository) LockAccount(userID uint64, until time.Time) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("locked_until", until).
		Error
}

// UnlockAccount unlocks a user account
func (r *UserRepository) UnlockAccount(userID uint64) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"locked_until":   nil,
			"login_attempts": 0,
		}).Error
}

// Delete soft deletes a user
func (r *UserRepository) Delete(userID uint64) error {
	return r.db.Delete(&models.User{}, "id = ?", userID).Error
}

// List retrieves users with pagination
func (r *UserRepository) List(offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// Get total count
	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	if err := r.baseQuery().Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// ExistsByUsername checks if a user with the given username exists
func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func init() {
	coreServer.RegisterRepository(constants.ComponentKey.AuthenticationUserRepo, func(app *coreServer.HTTPApp) (interface{}, error) {
		if app.DB == nil {
			return nil, fmt.Errorf("database not initialised")
		}
		return NewUserRepository(app.DB), nil
	})
}
