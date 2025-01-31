package dao

import (
	"my-digital-home/pkg/core/user/model"
)

type UserRepository interface {
	QueryByID(id int64) (model.User, error)
	IsUsernameExists(username string) (bool, error)
	IsEmailExists(email string) (bool, error)
	CreateUser(user model.User) error
	GetPasswordHash(username string) (string, int64, error) // 返回哈希和用户ID
	UpdatePassword(userID uint, newPwdHash string) error
}
