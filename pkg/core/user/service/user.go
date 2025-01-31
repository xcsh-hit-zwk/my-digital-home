package service

type UserService interface {
	IsUsernameExists(username string) (bool, error)
	IsEmailExists(email string) (bool, error)
	CreateUser(username, email, hashedPwd string) error
	GetPasswordHash(username string) (string, uint, error) // 返回哈希和用户ID
	UpdatePassword(userID uint, newPwdHash string) error
}
