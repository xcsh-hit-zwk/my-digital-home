package dao

import (
	"errors"
	"fmt"
	"gorm.io/gorm/clause"
	"my-digital-home/pkg/core/user/model"
	"my-digital-home/pkg/core/user/repository/dao"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrDuplicateEntry   = errors.New("duplicate user entry")
	ErrDatabaseInternal = errors.New("database internal error")
)

type GormUserRepository struct {
	db *gorm.DB
}

// User查询方法实现（优化版本）
func (r *GormUserRepository) QueryByID(id int64) (model.User, error) {
	var user model.User
	err := r.db.Select("id", "username", "email", "created_at", "updated_at", "version").
		Where("id = ? AND is_active = ?", id, true).
		First(&user).
		Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return model.User{}, ErrUserNotFound
	case err != nil:
		return model.User{}, fmt.Errorf("%w: user query failed", wrapGormError(err))
	default:
		return user, nil
	}
}

var DefaultUserRepo dao.UserRepository

func NewUserRepository(db *gorm.DB) {
	DefaultUserRepo = &GormUserRepository{
		db: db.Model(&model.User{}),
	}
}

// Check username existence with active status
func (r *GormUserRepository) IsUsernameExists(username string) (bool, error) {
	var count int64
	err := r.db.Where("username = ? AND is_active = ?", username, true).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("%w: failed to check username", wrapGormError(err))
	}
	return count > 0, nil
}

// Check email existence with active status
func (r *GormUserRepository) IsEmailExists(email string) (bool, error) {
	var count int64
	err := r.db.Where("email = ? AND is_active = ?", email, true).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("%w: failed to check email", wrapGormError(err))
	}
	return count > 0, nil
}

// Create new user with transaction
func (r *GormUserRepository) CreateUser(user model.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			if isDuplicateError(err) {
				return ErrDuplicateEntry
			}
			return fmt.Errorf("%w: user creation failed", wrapGormError(err))
		}
		return nil
	})
}

// Get user credentials with Optimistic Lock check
func (r *GormUserRepository) GetPasswordHash(username string) (string, int64, error) {
	var user model.User
	err := r.db.Select("password_hash", "id", "version").
		Where("username = ? AND is_active = ?", username, true).
		First(&user).Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return "", 0, ErrUserNotFound
	case err != nil:
		return "", 0, fmt.Errorf("%w: password lookup failed", wrapGormError(err))
	default:
		return user.PasswordHash, user.ID, nil
	}
}

// Update password with version control
func (r *GormUserRepository) UpdatePassword(userID uint, newPwdHash string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND is_active = ?", userID, true).
			First(&user).Error; err != nil {
			return wrapGormError(err)
		}

		result := tx.Model(&model.User{}).
			Where("id = ? AND version = ?", userID, user.Version).
			Updates(map[string]interface{}{
				"password_hash": newPwdHash,
				"version":       user.Version + 1,
				"updated_at":    time.Now(),
			})

		if result.Error != nil {
			return fmt.Errorf("%w: password update failed", wrapGormError(result.Error))
		}

		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}
		return nil
	})
}

// Error handling utils
func isDuplicateError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return true
	}
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

func wrapGormError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserNotFound
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062:
			return ErrDuplicateEntry
		case 1048, 1044, 1146: // Common MySQL operation errors
			return ErrDatabaseInternal
		}
	}

	if errors.Is(err, gorm.ErrInvalidDB) ||
		errors.Is(err, gorm.ErrInvalidTransaction) ||
		errors.Is(err, gorm.ErrUnsupportedRelation) {
		return ErrDatabaseInternal
	}

	return err // Return original error if no specific mapping
}
