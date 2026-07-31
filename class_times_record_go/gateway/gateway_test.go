// package main_test Go Gateway 联调测试
//
// 验证 Go Gateway 能正确转发请求到 Java 后端服务
// 测试场景：
//  1. 公开路径直接转发（免 JWT）
//  2. 受保护路径无 Token → 401
//  3. 受保护路径带有效 Token → 转发
//  4. 受保护路径带黑名单 Token → 401
//
// 前置条件：Java auth-service(10002) 和 business-service(10001) 已启动
package main_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kurimula-airi/course_record_go/common/jwt"
	"github.com/kurimula-airi/course_record_go/gateway/internal"
	"github.com/redis/go-redis/v9"
)

// 测试用 Redis 和 JWT 配置（与 Java 侧一致）
const (
	testRedisAddr     = "121.196.229.10:6379"
	testRedisPassword = "shiroko114514"
)

// newTestGateway 创建测试用 Gateway 实例
func newTestGateway(t *testing.T) *internal.Gateway {
	cfg := internal.DefaultConfig()
	return internal.NewGateway(cfg)
}

// TestGateway_PublicPathForwarding 验证公开路径直接转发
// Go Gateway → Java auth-service（/auth/auth/get_bind_info?token=invalid）
func TestGateway_PublicPathForwarding(t *testing.T) {
	gw := newTestGateway(t)

	// 模拟请求 /auth/auth/get_bind_info?token=invalid（公开路径）
	req := httptest.NewRequest("GET", "/auth/auth/get_bind_info?token=invalid_poc_test", nil)
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	// 验证：Java 应返回 200 + 业务错误（token 无效）
	if w.Code != 200 {
		t.Errorf("公开路径转发失败: status=%d, body=%s", w.Code, w.Body.String())
		return
	}

	body := w.Body.String()
	if !strings.Contains(body, "二维码已过期") && !strings.Contains(body, "无效") {
		t.Errorf("预期业务错误响应，实际: %s", body)
		return
	}

	t.Logf("✅ 公开路径转发成功: %s", truncate(body, 150))
}

// TestGateway_ProtectedPathNoToken 验证受保护路径无 Token 返回 401
func TestGateway_ProtectedPathNoToken(t *testing.T) {
	gw := newTestGateway(t)

	// /auth/auth/get_subscribe_status 不在 PUBLIC_PATHS 中（注意路径前缀）
	// 实际上所有 /auth/auth/ 开头的都在 PUBLIC_PATHS，改用 /biz/student/query
	req := httptest.NewRequest("POST", "/biz/student/query_by_teacher_id", nil)
	w := httptest.NewRecorder()

	gw.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("无 Token 应返回 401, 实际: %d", w.Code)
		return
	}

	t.Logf("✅ 无 Token 拒绝: status=%d", w.Code)
}

// TestGateway_ProtectedPathWithValidToken 验证有效 Token 转发
func TestGateway_ProtectedPathWithValidToken(t *testing.T) {
	gw := newTestGateway(t)

	// 生成与 Java 兼容的 Access Token
	jwtUtils := jwt.NewUtils(
		"shiroko_project_secret_key_at_least_32_chars_long",
		5*60*1000,
		7*24*60*60*1000,
	)
	token, err := jwtUtils.CreateAccessToken(99999, 4, "test_openid_poc")
	if err != nil {
		t.Fatalf("生成 Token 失败: %v", err)
	}

	// 调用受保护路径（/biz/student/query_by_teacher_id 需要 JWT）
	req := httptest.NewRequest("POST", "/biz/student/query_by_teacher_id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// 验证：Token 有效，请求被转发到 Java business-service
	// Java 会返回业务错误（缺少签名），但不会是 401
	body := w.Body.String()
	if w.Code == 401 {
		t.Errorf("有效 Token 不应返回 401: %s", body)
		return
	}

	t.Logf("✅ 有效 Token 转发成功: status=%d, body=%s", w.Code, truncate(body, 150))
}

// TestGateway_BlacklistedToken 验证黑名单 Token 返回 401
func TestGateway_BlacklistedToken(t *testing.T) {
	gw := newTestGateway(t)

	// 生成 Token
	jwtUtils := jwt.NewUtils(
		"shiroko_project_secret_key_at_least_32_chars_long",
		5*60*1000,
		7*24*60*60*1000,
	)
	token, err := jwtUtils.CreateAccessToken(99998, 4, "blacklisted_openid")
	if err != nil {
		t.Fatalf("生成 Token 失败: %v", err)
	}

	// 将 Token 加入 Redis 黑名单（模拟登出）
	rdb := redis.NewClient(&redis.Options{
		Addr:     testRedisAddr,
		Password: testRedisPassword,
		DB:       0,
	})
	ctx := context.Background()
	blacklistKey := "cr:token:blacklist:" + token
	err = rdb.Set(ctx, blacklistKey, "1", 5*time.Minute).Err()
	if err != nil {
		t.Fatalf("写入黑名单失败: %v", err)
	}
	defer rdb.Del(ctx, blacklistKey) // 测试结束清理

	// 调用受保护路径
	req := httptest.NewRequest("POST", "/biz/student/query_by_teacher_id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// 验证：黑名单 Token 应返回 401
	if w.Code != 401 {
		t.Errorf("黑名单 Token 应返回 401, 实际: %d, body=%s", w.Code, w.Body.String())
		return
	}

	t.Logf("✅ 黑名单 Token 拒绝: status=%d", w.Code)
}

// TestGateway_StripPrefix 验证路径前缀去除
// Go Gateway 应将 /auth/auth/login 转发为 /auth/login（StripPrefix=1）
func TestGateway_StripPrefix(t *testing.T) {
	gw := newTestGateway(t)

	// 通过公开路径验证转发后的路径
	// /auth/auth/get_bind_info → Java 收到 /auth/get_bind_info
	req := httptest.NewRequest("GET", "/auth/auth/get_bind_info?token=strip_test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// Java 能正确响应说明路径转发正确
	if w.Code == 200 {
		t.Logf("✅ StripPrefix 转发正确: %s", truncate(w.Body.String(), 100))
	} else {
		t.Errorf("StripPrefix 转发失败: status=%d", w.Code)
	}
}

// truncate 截断字符串
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
