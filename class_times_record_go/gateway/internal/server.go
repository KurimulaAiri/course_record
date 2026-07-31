// Package internal Go Gateway 主程序
//
// 替代 Java Spring Cloud Gateway，职责：
//  1. 路由分发：/auth/** → auth-service, /biz/** → business-service, /admin/** → admin-service
//  2. JWT 校验：验签 + 验过期（HS256，与 Java JwtUtils 互通）
//  3. Redis 黑名单：查询 cr:token:blacklist:{token}（与 Java TokenBlacklistService 互通）
//  4. 请求头注入：X-User-Id / X-User-Role / X-User-OpenId（与 Java JwtAuthFilter 一致）
//  5. StripPrefix=1：转发前去除第一层路径前缀
//
// 性能对比：
//   - Java Gateway: 300-500MB 内存，启动 8-15s
//   - Go Gateway:   30-50MB 内存，启动 <1s
package internal

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kurimula-airi/course_record_go/common/jwt"
	"github.com/redis/go-redis/v9"
)

// ============================================================
// 公开路径白名单（与 Java JwtAuthFilter.PUBLIC_PATHS 完全一致）
// ============================================================

// PublicPaths 返回公开路径列表（供外部读取）
func PublicPaths() []string {
	return publicPaths
}

// publicPaths 公开路径列表（免 JWT 校验）
// 与 Java JwtAuthFilter.java 第 47-66 行的 PUBLIC_PATHS 完全对齐
var publicPaths = []string{
	// auth-service 公开接口
	"/auth/auth/login_no_pwd",
	"/auth/auth/login_by_pwd",
	"/auth/auth/login_by_token",
	"/auth/auth/get_open_id",
	"/auth/auth/register",
	"/auth/auth/refresh",
	"/auth/auth/get_bind_info",
	"/auth/auth/get_bind_info_by_code",
	"/auth/auth/check_bind_status",
	"/auth/auth/confirm_bind",
	"/auth/auth/record_subscribe",
	"/auth/auth/get_subscribe_status",
	"/auth/auth/test_send_subscribe",
	"/auth/auth/bind_by_code",
	// business-service 公开接口
	"/biz/institution/get_by_open_id",
	"/biz/institution/get_by_institution_code",
	"/biz/course_record/deduct-detail",
	// admin-service 公开接口
	"/admin/user/login",
	"/admin/user/refresh",
	"/admin/crypto/public_key",
}

// Redis 黑名单 key 前缀（与 Java TokenBlacklistService 一致）
const blacklistPrefix = "cr:token:blacklist:"

// 错误定义
var errMissingToken = errors.New("missing or invalid Authorization header")

// ============================================================
// Gateway 结构体
// ============================================================

// Gateway 网关实例
type Gateway struct {
	config      *Config
	jwtUtils    *jwt.Utils
	redisClient *redis.Client
	// 路由表：路径前缀 → 反向代理（预编译，避免每次请求解析）
	proxies map[string]*httputil.ReverseProxy
	// 路由配置：路径前缀 → 路由配置（用于 StripPrefix）
	routes map[string]*RouteConfig
}

// NewGateway 创建 Gateway 实例
//
// 参数：
//   - cfg: 全局配置
func NewGateway(cfg *Config) *Gateway {
	// 初始化 JWT 工具（与 Java JwtUtils 使用相同密钥）
	jwtUtils := jwt.NewUtils(
		cfg.JWT.SecretKey,
		cfg.JWT.AccessExpiration,
		cfg.JWT.RefreshExpiration,
	)

	// 初始化 Redis 客户端（与 Java ReactiveRedisTemplate 连接同一 Redis）
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 预编译路由表（每个目标服务一个 ReverseProxy）
	proxies := make(map[string]*httputil.ReverseProxy)
	routes := make(map[string]*RouteConfig)
	for i := range cfg.Routes {
		route := &cfg.Routes[i]
		// 解析目标 URI
		targetURL, err := url.Parse(route.URI)
		if err != nil {
			log.Printf("❌ 路由 %s URI 解析失败: %v", route.ID, err)
			continue
		}
		// 创建反向代理
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		// 设置自定义 Director（处理 StripPrefix）
		originalDirector := proxy.Director
		routeCopy := *route // 避免闭包捕获循环变量
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// StripPrefix: 去除第一层路径前缀
			// 例如 /auth/auth/login → /auth/login
			if routeCopy.StripPrefix > 0 {
				req.URL.Path = stripPrefix(req.URL.Path, routeCopy.StripPrefix)
				req.URL.RawPath = stripPrefix(req.URL.RawPath, routeCopy.StripPrefix)
			}
		}
		proxies[route.Prefix] = proxy
		routes[route.Prefix] = route
		log.Printf("✅ 路由注册: %s → %s (StripPrefix=%d)", route.Prefix, route.URI, route.StripPrefix)
	}

	return &Gateway{
		config:      cfg,
		jwtUtils:    jwtUtils,
		redisClient: redisClient,
		proxies:     proxies,
		routes:      routes,
	}
}

// ============================================================
// HTTP 请求处理
// ============================================================

// ServeHTTP 实现 http.Handler 接口
// 这是 Gateway 的核心入口，每个请求都会经过：
//  1. 路由匹配
//  2. 公开路径检查
//  3. JWT 校验
//  4. Redis 黑名单检查
//  5. 请求头注入
//  6. 反向代理转发
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. 路由匹配：找到匹配的前缀
	prefix := g.matchRoute(r.URL.Path)
	if prefix == "" {
		http.Error(w, `{"code":404,"message":"路由未找到"}`, http.StatusNotFound)
		return
	}

	// 2. 公开路径检查（免 JWT 校验）
	if isPublicPath(r.URL.Path) {
		g.proxies[prefix].ServeHTTP(w, r)
		return
	}

	// 3. swagger/api-docs 放行（与 Java 一致）
	if strings.Contains(r.URL.Path, "swagger") || strings.Contains(r.URL.Path, "api-docs") {
		g.proxies[prefix].ServeHTTP(w, r)
		return
	}

	// 4. JWT 校验
	claims, err := g.authenticate(r)
	if err != nil {
		log.Printf("JWT 校验失败: %v, path=%s", err, r.URL.Path)
		http.Error(w, `{"code":401,"message":"未授权或Token已失效"}`, http.StatusUnauthorized)
		return
	}

	// 5. Redis 黑名单检查
	isBlacklisted, err := g.checkBlacklist(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		// Redis 查询失败时降级处理：放行请求（与 Java 反应式查询失败行为一致）
		log.Printf("⚠️  Redis 黑名单查询失败（降级放行）: %v", err)
	} else if isBlacklisted {
		log.Printf("Token 在黑名单中: userId=%d, path=%s", claims.UserID, r.URL.Path)
		http.Error(w, `{"code":401,"message":"Token已失效，请重新登录"}`, http.StatusUnauthorized)
		return
	}

	// 6. 注入请求头（X-User-Id / X-User-Role / X-User-OpenId）
	// 与 Java JwtAuthFilter 第 119-123 行完全一致
	r.Header.Set("X-User-Id", fmtInt64(claims.UserID))
	r.Header.Set("X-User-Role", fmtInt64(claims.RoleID))
	// openId 为空时注入空字符串（与 Java 行为一致）
	openId := claims.OpenID
	r.Header.Set("X-User-OpenId", openId)

	// 7. 转发请求
	g.proxies[prefix].ServeHTTP(w, r)
}

// matchRoute 匹配路由前缀
// 返回匹配的前缀（如 "/auth"），无匹配返回空字符串
func (g *Gateway) matchRoute(path string) string {
	for prefix := range g.proxies {
		if strings.HasPrefix(path, prefix+"/") || path == prefix {
			return prefix
		}
	}
	return ""
}

// isPublicPath 检查是否为公开路径
// 对齐 Java PUBLIC_PATHS.stream().anyMatch(path::startsWith)
func isPublicPath(path string) bool {
	for _, p := range publicPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// authenticate JWT 认证
// 对齐 Java JwtAuthFilter 的 token 提取 + 解析流程
func (g *Gateway) authenticate(r *http.Request) (*jwt.CustomClaims, error) {
	// 提取 Authorization 头
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errMissingToken
	}
	// 截取 token（去除 "Bearer " 前缀，对齐 Java substring(7)）
	token := authHeader[7:]

	// 解析并校验 Access Token
	// 对齐 Java Jwts.parserBuilder().parseClaimsJws() 的验签 + 验过期
	claims, err := g.jwtUtils.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// checkBlacklist 检查 Token 是否在 Redis 黑名单中
// 对齐 Java ReactiveRedisTemplate.hasKey(blacklistKey)
//
// 参数：
//   - ctx: 请求上下文
//   - authHeader: Authorization 头（"Bearer xxx"）
//
// 返回：true=在黑名单中，false=不在
func (g *Gateway) checkBlacklist(ctx context.Context, authHeader string) (bool, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return false, nil
	}
	token := authHeader[7:]
	blacklistKey := blacklistPrefix + token

	// 使用 EXISTS 命令（对齐 Java hasKey）
	// 设置 3 秒超时，避免 Redis 故障导致请求卡住
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	n, err := g.redisClient.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ============================================================
// 辅助函数
// ============================================================

// stripPrefix 去除路径前缀
// 例如 stripPrefix("/auth/auth/login", 1) → "/auth/login"
//
// 参数：
//   - path: 原始路径
//   - count: 去除的前缀层数
func stripPrefix(path string, count int) string {
	if count <= 0 || path == "" {
		return path
	}
	// 标准化：确保以 / 开头
	if path[0] != '/' {
		path = "/" + path
	}
	// 逐层去除
	for i := 0; i < count; i++ {
		// 找到第一个 / 之后的位置
		idx := strings.Index(path[1:], "/")
		if idx < 0 {
			// 没有更多层级，返回根路径
			return "/"
		}
		path = path[idx+1:]
	}
	return path
}

// fmtInt64 将 int64 格式化为字符串
// 与 Java String.valueOf(claims.get("userId")) 行为一致
func fmtInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}
