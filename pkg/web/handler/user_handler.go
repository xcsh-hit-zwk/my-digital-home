// ----------- pkg/web/handler/user_handler.go -----------
package handler

import (
	"context"
	"errors"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"my-digital-home/pkg/common/config"
	errors2 "my-digital-home/pkg/common/errors"
	dao_model "my-digital-home/pkg/core/user/model"
	"my-digital-home/pkg/core/user/repository/dao"
	dao2 "my-digital-home/pkg/core/user/repository/dao/impl"
	"my-digital-home/pkg/web/model"
	"regexp"
	"time"
	"unicode"
)

type UserHandler struct {
	UserRepo  dao.UserRepository // 使用具体接口
	JWTSecret string
}

var (
	DefaultUserHandler *UserHandler
)

func NewUserHandler(cfg *config.Config) UserHandler {
	if DefaultUserHandler == nil {
		DefaultUserHandler = &UserHandler{
			UserRepo:  dao2.DefaultUserRepo, /* 注入实际的仓储实现 */
			JWTSecret: cfg.Middleware.JWT.Secret,
		}
	}

	return *DefaultUserHandler
}

// 密码规则：同时包含数字、字母和特殊字符，最少8位
var passwordRegex = regexp.MustCompile(`^(?=.*[0-9])(?=.*[a-zA-Z])(?=.*[\W_]).{8,}$`)

// 注册接口优化
func (h *UserHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req model.RegisterReq
	if err := c.BindAndValidate(&req); err != nil {
		respondError(c, 400, "参数校验失败: "+err.Error())
		return
	}

	// 密码合规性检查（复用公共方法）
	if err := validatePasswordStrength(req.Password); err != nil {
		respondError(c, 400, err.Error())
		return
	}

	// 检查用户名唯一性（活跃用户）
	exists, err := h.UserRepo.IsUsernameExists(req.Username)
	if err != nil {
		respondError(c, 500, errors2.WrapGormError(err).Error())
		return
	}
	if exists {
		respondError(c, 409, "用户名已存在")
		return
	}

	// 检查邮箱唯一性（活跃用户）
	exists, err = h.UserRepo.IsEmailExists(req.Email)
	if err != nil {
		respondError(c, 500, errors2.WrapGormError(err).Error())
		return
	}
	if exists {
		respondError(c, 409, "邮箱已被注册")
		return
	}

	// 密码加密
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, 500, "密码加密失败")
		return
	}

	// 创建用户实体
	user := dao_model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPwd),
		IsActive:     true,
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// 调用DAO层方法时传递完整实体
	if err := h.UserRepo.CreateUser(user); err != nil {
		if errors.Is(err, errors2.ErrDuplicateEntry) {
			respondError(c, 409, "用户已存在")
		} else {
			respondError(c, 500, "注册失败")
		}
		return
	}

	c.JSON(201, utils.H{"message": "注册成功"})
}

func (h *UserHandler) Login(ctx context.Context, c *app.RequestContext) {
	var req model.LoginReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, utils.H{"error": "参数错误"})
		return
	}

	// 获取存储的密码哈希
	storedHash, userID, err := h.UserRepo.GetPasswordHash(req.Username)
	if err != nil {
		c.JSON(401, utils.H{"error": "用户不存在"})
		return
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
		c.JSON(401, utils.H{"error": "密码错误"})
		return
	}

	// 生成 JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": req.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(), // 过期时间
		"iss":      "my-digital-home",                     // 签发方
	})

	signedToken, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		c.JSON(500, utils.H{"error": "令牌生成失败"})
		return
	}

	c.JSON(200, utils.H{
		"token":    signedToken,
		"user_id":  userID,
		"username": req.Username,
	})
}

// 密码修改接口（增强验证）
func (h *UserHandler) ChangePassword(ctx context.Context, c *app.RequestContext) {
	claims, exist := c.Get("jwt_claims")
	if !exist {
		respondError(c, 401, "未授权访问")
		return
	}

	// 安全提取用户ID和用户名
	jwtClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		respondError(c, 401, "无效令牌类型")
		return
	}

	userID, ok := jwtClaims["user_id"].(float64)
	if !ok {
		respondError(c, 401, "用户信息解析失败")
		return
	}

	// 提取修改密码请求数据
	var req model.ChangePwdReq
	if err := c.BindAndValidate(&req); err != nil {
		respondError(c, 400, "参数错误: "+err.Error())
		return
	}

	// 严格校验新密码复杂度
	if !passwordRegex.MatchString(req.NewPassword) {
		respondError(c, 400, "新密码不符合复杂度要求")
		return
	}

	// 新密码哈希生成
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondError(c, 500, "系统错误")
		return
	}

	// 更新密码，带版本校验
	if err := h.UserRepo.UpdatePassword(uint(userID), string(newHash)); err != nil {
		if errors.Is(err, errors2.ErrUserNotFound) {
			respondError(c, 404, "用户不存在或已注销")
		} else if errors.Is(err, dao2.ErrDatabaseInternal) {
			respondError(c, 500, "数据库错误")
		} else {
			respondError(c, 500, "密码更新失败: "+err.Error())
		}
		return
	}

	c.JSON(200, utils.H{"message": "密码更新成功"})
}

func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("密码至少8位")
	}

	hasNumber := false
	hasLetter := false
	hasSpecial := false

	for _, c := range password {
		switch {
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsLetter(c):
			hasLetter = true
		case unicode.IsSymbol(c) || unicode.IsPunct(c):
			hasSpecial = true
		}
	}

	if !(hasNumber && hasLetter && hasSpecial) {
		return errors.New("需包含数字、字母和特殊字符")
	}

	return nil
}

// 统一错误响应方法
func respondError(c *app.RequestContext, code int, msg string) {
	c.JSON(code, utils.H{
		"error":   msg,
		"code":    code,
		"success": false,
	})
}
