package errors

import (
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	dao "my-digital-home/pkg/core/user/repository/dao/impl"
)

// region 错误处理工具函数

// wrapGormError 将底层数据库错误转变为业务可识别错误
// 参数说明：
//   - rawErr: 原始GORM错误
//
// 返回值：
//   - error: 标准化错误类型
func WrapGormError(rawErr error) error {
	if rawErr == nil {
		return nil
	}

	// 处理预定义的GORM错误
	switch {
	case errors.Is(rawErr, gorm.ErrRecordNotFound):
		return ErrUserNotFound
	case errors.Is(rawErr, gorm.ErrDuplicatedKey):
		return ErrDuplicateEntry
	}

	// 处理MySQL驱动错误
	var mysqlErr *mysql.MySQLError
	if errors.As(rawErr, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062: // 唯一性约束冲突
			return ErrDuplicateEntry
		case 1045, 1049, 1146: // 数据库连接、表不存在等错误
			return fmt.Errorf("%w: %s", dao.ErrDatabaseInternal, mysqlErr.Message)
		}
	}

	// 兜底处理：附加原始错误信息
	return fmt.Errorf("%w: %v", dao.ErrDatabaseInternal, rawErr)
}

// isDuplicateError 判断是否为重复记录错误
func IsDuplicateError(err error) bool {
	return errors.Is(err, ErrDuplicateEntry) || errors.Is(err, gorm.ErrDuplicatedKey)
}
