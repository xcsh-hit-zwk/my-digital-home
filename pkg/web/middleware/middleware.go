package middleware

import (
	"context"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/golang-jwt/jwt/v5"
	jwth "github.com/hertz-contrib/jwt"
	"my-digital-home/pkg/common/config"
	"regexp"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/cors"

// LoggerMiddleware 结构化的请求日志记录
func LoggerMiddleware() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		start := time.Now()
		ctx.Next(c) // 放行到后续处理器
		latency := time.Since(start)

		// 结构化日志输出
		hlog.CtxTracef(c, "| %3d | %13v | %15s | %-7s | %s | UA=%s",
			ctx.Response.StatusCode(),
			latency,
			ctx.ClientIP(),
			ctx.Method(),
			ctx.Path(),
			ctx.GetHeader("User-Agent"),
		)
	}
}

/*
	启动时指定环境变量
	export APP_ENV=production
	go run main.go
*/

// RecoveryMiddleware 增强型异常捕获（带配置依赖版本）
func RecoveryMiddleware(cfg *config.Config) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				// 获取调用堆栈
				stack := string(debug.Stack())

				hlog.CtxErrorf(c, "[PANIC RECOVERED] %v\n%s", err, stack)

				// 生产环境处理
				if cfg.IsProd() { // 使用注入的配置实例判断环境
					ctx.AbortWithStatusJSON(500, map[string]interface{}{
						"code":    500,
						"message": "internal server error",
					})
				} else { // 开发环境显示详细错误
					ctx.AbortWithStatusJSON(500, map[string]interface{}{
						"code":  500,
						"error": fmt.Sprintf("%v", err),     // 转换为字符串格式
						"stack": strings.Split(stack, "\n"), // 切割为字符串数组更易读
					})
				}
			}
		}()
		ctx.Next(c)
	}
}

// CORSMiddleware 安全的跨域配置
func CORSMiddleware(corsConfig config.CORSConfig) app.HandlerFunc {
	return cors.New(
		cors.Config{
			AllowOrigins:     corsConfig.AllowOrigins,
			AllowMethods:     corsConfig.AllowMethods,
			AllowHeaders:     corsConfig.AllowHeaders,
			ExposeHeaders:    corsConfig.ExposeHeaders,
			AllowCredentials: corsConfig.AllowCredentials,
			MaxAge:           corsConfig.MaxAge,
			// 动态校验来源
			AllowOriginFunc: func(origin string) bool {
				for _, domain := range corsConfig.TrustedDomains {
					if strings.Contains(origin, domain) {
						return true
					}
				}
				return false
			},
		},
	)
}

func TimeoutMiddleware(seconds int) app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		timeoutCtx, cancel := context.WithTimeout(c, time.Duration(seconds)*time.Second)
		defer cancel()

		// 通过goroutine执行后续处理器
		done := make(chan struct{})
		var panicErr interface{}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					panicErr = r
				}
				close(done)
			}()
			ctx.Next(timeoutCtx) // 关键：传入超时上下文
		}()

		// 监听超时或完成
		select {
		case <-timeoutCtx.Done():
			ctx.AbortWithStatusJSON(503, utils.H{
				"code":    503000,
				"message": "service unavailable",
			})
			hlog.CtxWarnf(timeoutCtx, "request timeout path=%s", ctx.Path())
		case <-done:
			if panicErr != nil {
				panic(panicErr) // 交给全局recovery处理
			}
		}
	}
}

// RateLimitMiddleware 令牌桶算法限流
func RateLimitMiddleware(rate int, interval time.Duration) app.HandlerFunc {
	limiter := NewTokenBucket(rate, interval)

	return func(c context.Context, ctx *app.RequestContext) {
		if !limiter.Allow() {
			hlog.CtxInfof(c, "[RATE LIMIT] path=%s", ctx.Path())
			ctx.AbortWithStatusJSON(429, map[string]interface{}{
				"code":    429001,
				"message": "too many requests",
			})
			return
		}
		ctx.Next(c)
	}
}

// 令牌桶实现
type TokenBucket struct {
	capacity int
	tokens   chan struct{}
	rate     time.Duration
}

func NewTokenBucket(rate int, interval time.Duration) *TokenBucket {
	tb := &TokenBucket{
		capacity: rate,
		tokens:   make(chan struct{}, rate),
		rate:     interval,
	}

	// 定时器生产令牌
	go func() {
		ticker := time.NewTicker(tb.rate)
		for range ticker.C {
			select {
			case tb.tokens <- struct{}{}:
			default:
			}
		}
	}()
	return tb
}

func (tb *TokenBucket) Allow() bool {
	select {
	case <-tb.tokens:
		return true
	default:
		return false
	}
}

// SecurityCheckMiddleware 全局安全校验中间件
func SecurityCheckMiddleware(maxBodySize int64) app.HandlerFunc {
	// 预编译恶意字符正则
	xssRegex := regexp.MustCompile(`<script.*?>|<\/script>|alert\(|onerror=`)
	sqlInjectRegex := regexp.MustCompile(`\b(union|select|drop|delete|insert)\b`)

	return func(c context.Context, ctx *app.RequestContext) {
		// 防护机制1：检查User-Agent
		if isInvalidUserAgent(ctx) {
			securityResponse(ctx, 400001, "missing required header: User-Agent", 400)
			return
		}

		// 防护机制2：请求体大小限制
		// 修复：将 ContentLength() 的返回值转换为 int64
		if int64(ctx.Request.Header.ContentLength()) > maxBodySize {
			securityResponse(ctx, 413001, "request body exceeds max size", 413)
			return
		}

		// 防护机制3：参数恶意字符检查
		if hasMaliciousContent(ctx, xssRegex, sqlInjectRegex) {
			securityResponse(ctx, 422001, "request contains invalid characters", 422)
			return
		}

		// 防护机制4：检查HTTP方法
		if !isAllowedMethod(ctx) {
			securityResponse(ctx, 405001, "method not allowed", 405)
			return
		}

		ctx.Next(c)
	}
}

// JWTAuthMiddleware 验证JWT令牌有效性
func JWTAuthMiddleware(secret string) app.HandlerFunc {
	authMiddleware, err := jwth.New(&jwth.Middleware{
		SigningKey:  []byte(secret),
		TokenLookup: "header:Authorization",
		TimeFunc:    time.Now,
	})
	if err != nil {
		panic(fmt.Sprintf("JWT 中间件初始化失败: %v", err))
	}
	return authMiddleware.MiddlewareFunc()
}

func InitJWTAuth(cfg *config.JWTAuthConfig) app.HandlerFunc {
	authMiddleware := jwt.New(&jwt.HertzJWTMiddleware{
		Realm:            cfg.Issuer,
		SigningAlgorithm: cfg.SigningMethod,
		Key:              []byte(cfg.Secret),
		Timeout:          cfg.ExpireDuration,
		TimeFunc:         time.Now,
		Authenticator:    authenticator, // TODO: 实际用户验证逻辑
		IdentityKey:      "user_id",
		Unauthorized:     handleJWTError,
	})

	if err != nil {
		hlog.Fatal("JWT Middleware init failed: %v", err)
	}

	return authMiddleware.MiddlewareFunc()
}

func authenticator(ctx context.Context, c *app.RequestContext) (interface{}, error) {
	var loginReq struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	if err := c.BindAndValidate(&loginReq); err != nil {
		return nil, jwt.ErrMissingLoginValues
	}

	// 查询数据库验证用户，此处需要实际数据访问
	// user, err := userRepository.FindByUsername(loginReq.Username)
	if loginReq.Username != "admin" || loginReq.Password != "password" {
		return nil, jwt.ErrFailedAuthentication
	}

	// 返回用户身份标识
	return map[string]interface{}{
		"user_id":  1,
		"username": loginReq.Username,
	}, nil
}

func handleJWTError(ctx context.Context, c *app.RequestContext, code int, message string) {
	hlog.Errorf("JWT Error (code=%d) path=%s: %s", code, c.Path(), message)
	c.JSON(code, utils.H{
		"code":    code,
		"message": message,
	})
}

// 辅助方法：判断User-Agent合法性
func isInvalidUserAgent(ctx *app.RequestContext) bool {
	ua := string(ctx.GetHeader("User-Agent"))
	// 示例检查逻辑：不允许空UA
	return ua == ""
}

// 带性能优化的版本
func hasMaliciousContent(ctx *app.RequestContext, xss *regexp.Regexp, sql *regexp.Regexp) bool {
	// 使用atomic包确保线程安全
	var found int32

	check := func(data []byte) bool {
		return xss.Match(data) || sql.Match(data)
	}

	visitor := func(key, value []byte) {
		if atomic.LoadInt32(&found) == 1 {
			return // 已经找到匹配，跳过后续检查
		}
		if check(key) || check(value) {
			atomic.StoreInt32(&found, 1)
		}
	}

	// 检查Query参数
	ctx.QueryArgs().VisitAll(visitor)
	if atomic.LoadInt32(&found) == 1 {
		return true
	}

	// 检查Post表单参数
	ctx.PostArgs().VisitAll(visitor)
	return atomic.LoadInt32(&found) == 1
}

// 辅助方法：允许的HTTP方法检查
func isAllowedMethod(ctx *app.RequestContext) bool {
	allowed := map[string]bool{
		"GET":  true,
		"POST": true,
		"PUT":  true,
	}
	return allowed[string(ctx.Method())]
}

// 安全响应统一处理
func securityResponse(ctx *app.RequestContext, code int, msg string, status int) {
	hlog.Warnf("SecurityAlert[code=%d]: %s", code, msg)
	ctx.AbortWithStatusJSON(status, map[string]interface{}{
		"code":    code,
		"message": msg,
	})
}
