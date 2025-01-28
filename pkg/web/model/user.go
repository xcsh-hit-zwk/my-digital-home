package model

// 请求/响应数据结构
type (
	RegisterReq struct {
		Username string `json:"username" binding:"required,min=4,max=20"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	LoginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	ChangePwdReq struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	UserRes struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
)

// 领域模型（不直接对接数据库）
type User struct {
	ID       uint
	Username string
	Email    string
	password string // 小写表示不直接暴露
}
