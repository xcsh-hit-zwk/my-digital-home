package dao

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrDuplicateEntry   = errors.New("duplicate user entry")
	ErrDatabaseInternal = errors.New("database internal error")
)

// 用户表结构建议 DDL
/*
CREATE TABLE users (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(127) NOT NULL COLLATE utf8mb4_bin,
    email VARCHAR(255) NOT NULL COLLATE utf8mb4_bin,
    password_hash CHAR(60) NOT NULL, -- 适应bcrypt哈希长度
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP(3),
    updated_at TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE INDEX idx_username (username),
    UNIQUE INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
*/

// User 领域模型
type User struct {
	ID           uint      `db:"id"`
	Username     string    `db:"username"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	IsActive     bool      `db:"is_active"`
	Version      int       `db:"version"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// MySQLUserRepository 具体实现
type MySQLUserRepository struct {
	db *sql.DB
}

func NewMySQLUserRepository(db *sql.DB) *MySQLUserRepository {
	return &MySQLUserRepository{db: db}
}

func (r *MySQLUserRepository) IsUsernameExists(username string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE username = ? AND is_active = TRUE)`
	var exists bool
	err := r.db.QueryRow(query, username).Scan(&exists)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("%w: check username failed", wrapMySQLError(err))
	}
	return exists, nil
}

func (r *MySQLUserRepository) IsEmailExists(email string) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM users WHERE email = ? AND is_active = TRUE)`
	var exists bool
	err := r.db.QueryRow(query, email).Scan(&exists)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("%w: check email failed", wrapMySQLError(err))
	}
	return exists, nil
}

func (r *MySQLUserRepository) CreateUser(username, email, hashedPwd string) error {
	const query = `
		INSERT INTO users (username, email, password_hash)
		VALUES (?, ?, ?)
	`

	_, err := r.db.Exec(query, username, email, hashedPwd)
	if err != nil {
		if isDuplicateKeyError(err) {
			return ErrDuplicateEntry
		}
		return fmt.Errorf("%w: create user failed", wrapMySQLError(err))
	}
	return nil
}

func (r *MySQLUserRepository) GetPasswordHash(username string) (string, uint, error) {
	const query = `
		SELECT password_hash, id
		FROM users 
		WHERE username = ? AND is_active = TRUE
		LIMIT 1
	`

	var (
		hash   string
		userID uint
	)
	err := r.db.QueryRow(query, username).Scan(&hash, &userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, ErrUserNotFound
		}
		return "", 0, fmt.Errorf("%w: get password hash failed", wrapMySQLError(err))
	}
	return hash, userID, nil
}

func (r *MySQLUserRepository) UpdatePassword(userID uint, newPwdHash string) error {
	const query = `
		UPDATE users 
		SET 
			password_hash = ?,
			version = version + 1,
			updated_at = CURRENT_TIMESTAMP(3)
		WHERE 
			id = ? AND is_active = TRUE
	`

	result, err := r.db.Exec(query, newPwdHash, userID)
	if err != nil {
		return fmt.Errorf("%w: update password failed", wrapMySQLError(err))
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// 私有工具函数
func isDuplicateKeyError(err error) bool {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == 1062
	}
	return false
}

func wrapMySQLError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		switch mysqlErr.Number {
		case 1045: // 访问被拒绝
		case 1146: // 表不存在
		case 1213: // 死锁
			return fmt.Errorf("%w: %v", ErrDatabaseInternal, mysqlErr)
		}
	}
	return err
}
