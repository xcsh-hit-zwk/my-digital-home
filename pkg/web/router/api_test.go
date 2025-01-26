// pkg/web/router/api_test.go
package router_test

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"my-digital-home/pkg/web/router"
)

func TestHealthCheckRoute(t *testing.T) {
	// 正确初始化方式（注意去掉了config包）
	h := server.New()
	router.RegisterAPIs(h)

	w := ut.PerformRequest(h.Engine, "GET", "/health", nil)
	resp := w.Result()

	if resp.StatusCode() != 200 {
		t.Fatalf("Expected 200, got %d", resp.StatusCode())
	}
}
