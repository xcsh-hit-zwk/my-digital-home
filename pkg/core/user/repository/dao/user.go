package dao

// 原生整接口定义
type UserRepository interface {
	IsUsernameExists(username string) (bool, error)
	IsEmailExists(email string) (bool, error)
	CreateUser(username, email, hashedPwd string) error
	GetPasswordHash(username string) (string, uint, error)
	UpdatePassword(userID uint, newPwdHash string) error
}
