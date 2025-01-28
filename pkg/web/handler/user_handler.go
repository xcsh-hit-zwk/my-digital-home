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
	"my-digital-home/pkg/core/user/repository/dao"
	"my-digital-home/pkg/web/model"
	"regexp"
	"time"
	"unicode"
)



type UserHandler struct {
	UserRepo  *dao.UserRepository // 使用具体接口
	JWTSecret string
}

var (
	DefaultUserHandler UserHandler
)

func NewUserHandler(cfg *config.Config){
	DefaultUserHandler = UserHandler{
		UserRepo:   /* 注入实际的仓储实现 */,
		JWTSecret: cfg.Middleware.JWT.Secret,
	}
	return
}

// 密码规则：同时包含数字、字母和特殊字符，最少8位
var passwordRegex = regexp.MustCompile(`^(?=.*[0-9])(?=.*[a-zA-Z])(?=.*[\W_]).{8,}$`)

func (h *UserHandler) Register(ctx context.Context, c *app.RequestContext) {
	var req model.RegisterReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, utils.H{"error": "参数校验失败"})
		return
	}

	// 使用新的密码验证方法
	if err := validatePasswordStrength(req.Password); err != nil {
		respondError(c, 400, err.Error())
		return
	}

	// 检查用户名重复
	if exists, _ := h.UserRepo.IsUsernameExists(req.Username); exists {
		c.JSON(409, utils.H{"error": "用户名已存在"})
		return
	}

	// 检查邮箱重复
	if exists, _ := h.UserRepo.IsEmailExists(req.Email); exists {
		c.JSON(409, utils.H{"error": "邮箱已被注册"})
		return
	}

	// 密码加密
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, utils.H{"error": "系统错误"})
		return
	}

	// 创建用户记录
	if err := h.UserRepo.CreateUser(req.Username, req.Email, string(hashedPwd)); err != nil {
		c.JSON(500, utils.H{"error": "注册失败"})
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
		"iss":      "my-digital-home",                      // 签发方
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

func (h *UserHandler) ChangePassword(ctx context.Context, c *app.RequestContext) {
	// 从JWT中获取用户信息
	claims, ok := c.Get("jwt_claims")
	if !ok {
		c.JSON(401, utils.H{"error": "未授权访问"})
		return
	}
	userID := claims.(map[string]interface{})["user_id"].(uint)

	var req model.ChangePwdReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, utils.H{"error": "参数错误"})
		return
	}

	// 检查新密码格式
	if !passwordRegex.MatchString(req.NewPassword) {
		c.JSON(400, utils.H{"error": "新密码不符合复杂度要求"})
		return
	}

	// 更新密码
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, utils.H{"error": "系统错误"})
		return
	}

	if err := h.UserRepo.UpdatePassword(userID, string(newHash)); err != nil {
		c.JSON(500, utils.H{"error": "密码更新失败"})
		return
	}

	c.JSON(200, utils.H{"message": "密码已更新"})
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
