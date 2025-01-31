// pkg/common/errors/user_errors.go

/*
  - 使用实例
    // 错误示例:
    if err.(*hzte.Error).Meta != nil { // 可能 panic
    // ...
    }

    // 正确方式:
    if hzteErr, ok := err.(*hzte.Error); ok {
    // 安全访问 Meta
    }
*/
package errors

import (
	"errors"
	hzte "github.com/cloudwego/hertz/pkg/common/errors" // 假设第三方错误库路径
)

// 定义原始错误
var (
	rawErrUserNotFound   = errors.New("user not found")
	rawErrDuplicateEntry = errors.New("username/email already exists")
)

// 包装成 Hertz 错误类型
var (
	ErrUserNotFound   = hzte.New(rawErrUserNotFound, hzte.ErrorTypePublic, nil)
	ErrDuplicateEntry = hzte.New(rawErrDuplicateEntry, hzte.ErrorTypePublic, nil)
)

// 可选添加带有元数据的构造方法
func NewUserNotFound(meta interface{}) *hzte.Error {
	return hzte.New(rawErrUserNotFound, hzte.ErrorTypePublic, meta)
}

func NewDuplicateEntry(meta interface{}) *hzte.Error {
	return hzte.New(rawErrDuplicateEntry, hzte.ErrorTypePublic, meta)
}
