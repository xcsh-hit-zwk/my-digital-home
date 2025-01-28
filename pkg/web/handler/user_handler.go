// ----------- pkg/web/handler/user_handler.go -----------
package handler

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"golang.org/x/crypto/bcrypt"
	"my-digital-home/pkg/common/config"
	"my-digital-home/pkg/web/model"
	"regexp"
)

type UserHandler struct {
	// 依赖接口（实际项目需要通过DI注入）
	UserRepo interface {
		IsUsernameExists(username string) (bool, error)
		IsEmailExists(email string) (bool, error)
		CreateUser(username, email, hashedPwd string) error
		GetPasswordHash(username string) (string, uint, error)
		UpdatePassword(userID uint, newPwdHash string) error
	}
	JWTSecret string // JWT签名密钥
}

var (
	DefaultUserHandler UserHandler
)

func NewUserHandler(cfg *config.Config){
	DefaultUserHandler = UserHandler{
		UserRepo:   /* 注入实际的仓储实现 */,
		JWTSecret: cfg.JWT.Secret,
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

	// 校验密码强度
	if !passwordRegex.MatchString(req.Password) {
		c.JSON(400, utils.H{"error": "密码需要数字、字母和特殊字符组合"})
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

	// TODO: 生成JWT Token（此处需补充具体实现）
	token := "generated_jwt_token"

	c.JSON(200, utils.H{
		"token":    token,
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
