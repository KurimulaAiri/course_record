// Package main auth-service 启动入口
//
// 对齐 Java AuthServiceApplication.java
// 启动方式：go run ./auth-service
//
// 端口：10002（与 Java auth-service 一致）
// 路由前缀：/auth（经 Gateway StripPrefix=1 后实际路径为 /auth/**）
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kurimula-airi/course_record_go/auth-service/internal/handler"
	"github.com/kurimula-airi/course_record_go/auth-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/auth-service/internal/service"
	commonctx "github.com/kurimula-airi/course_record_go/common/context"
	"github.com/kurimula-airi/course_record_go/common/db"
	"github.com/kurimula-airi/course_record_go/common/jwt"
	commonredis "github.com/kurimula-airi/course_record_go/common/redis"
)

// main auth-service 启动入口
//
// 职责：
//  1. 加载数据库、Redis、JWT 配置
//  2. 初始化各 Mapper
//  3. 初始化 AuthService
//  4. 注册路由并启动 HTTP 服务
func main() {
	// 1. 加载数据库配置（对齐 Java common-db.yaml）
	dbCfg := db.DefaultConfig()
	// 支持环境变量覆盖（生产部署用）
	if envDBHost := os.Getenv("DB_HOST"); envDBHost != "" {
		dbCfg.Host = envDBHost
	}
	if envDBPort := os.Getenv("DB_PORT"); envDBPort != "" {
		if port, err := strconv.Atoi(envDBPort); err == nil {
			dbCfg.Port = port
		}
	}

	// 连接 MySQL（对齐 Java HikariCP 数据源）
	database, err := db.NewMySQL(dbCfg)
	if err != nil {
		log.Fatalf("❌ MySQL 连接失败: %v", err)
	}
	defer database.Close()

	// 2. 加载 Redis 配置（对齐 Java common-redis.yaml）
	redisCfg := commonredis.DefaultConfig()
	if envRedisAddr := os.Getenv("REDIS_ADDR"); envRedisAddr != "" {
		redisCfg.Addr = envRedisAddr
	}

	redisClient, err := commonredis.NewRedis(redisCfg)
	if err != nil {
		log.Fatalf("❌ Redis 连接失败: %v", err)
	}
	defer redisClient.Close()

	// 3. 初始化 JWT 工具（对齐 Java JwtUtils，密钥与 Java 一致）
	jwtUtils := jwt.NewUtils(
		"shiroko_project_secret_key_at_least_32_chars_long",
		5*60*1000,              // Access Token 5 分钟
		7*24*60*60*1000,        // Refresh Token 7 天
	)

	// 4. 初始化各 Mapper（对齐 Java @Autowired 注入）
	userMapper := mapper.NewUserMapper(database)
	userAuthMapper := mapper.NewUserAuthMapper(database)
	userPlatformMapper := mapper.NewUserPlatformMapper(database)
	parentMapper := mapper.NewParentMapper(database)
	institutionMapper := mapper.NewInstitutionMapper(database)
	studentMapper := mapper.NewStudentMapper(database)
	parentStudentMapper := mapper.NewParentStudentMapper(database)
	wxSubscribeRecordMapper := mapper.NewWxSubscribeRecordMapper(database)
	wxStudentSubscribeMapper := mapper.NewWxStudentSubscribeMapper(database)

	// 5. 初始化 AuthService（对齐 Java AuthServiceImpl）
	authService := service.NewAuthService(
		userMapper,
		userAuthMapper,
		userPlatformMapper,
		parentMapper,
		institutionMapper,
		studentMapper,
		parentStudentMapper,
		wxSubscribeRecordMapper,
		wxStudentSubscribeMapper,
		jwtUtils,
		redisClient,
	)

	// 6. 创建 HTTP 路由（对齐 Java Spring MVC @RequestMapping）
	mux := http.NewServeMux()

	// 注册 auth 路由
	authHandler := handler.NewAuthHandler(authService)
	authHandler.RegisterRoutes(mux)

	// 健康检查接口（便于运维监控）
	mux.HandleFunc("/auth/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"auth-service"}`))
	})

	// 7. 应用中间件
	// 中间件链：UserContextMiddleware（从 header 解析用户信息）→ Handler
	finalHandler := commonctxMiddleware(mux)

	// 8. 启动 HTTP 服务
	port := 10002 // 与 Java auth-service 一致
	if envPort := os.Getenv("AUTH_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	addr := ":" + strconv.Itoa(port)
	server := &http.Server{
		Addr:         addr,
		Handler:      finalHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("🚀 Go auth-service 启动中，监听 %s", addr)
	log.Printf("   数据库: %s:%d/%s", dbCfg.Host, dbCfg.Port, dbCfg.Database)
	log.Printf("   Redis: %s", redisCfg.Addr)
	log.Printf("   路由前缀: /auth")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ auth-service 启动失败: %v", err)
	}
}

// commonctxMiddleware 用户上下文中间件
//
// 对齐 Java GatewayUserFilter + UserInterceptor
//
// 职责：
//  1. 从 X-User-Id / X-User-Role / X-User-OpenId header 解析用户信息
//  2. 写入 context.Context（对齐 Java UserContext.setUser）
//  3. 请求结束后 context 自动清理（对齐 Java UserInterceptor.afterCompletion）
func commonctxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从 header 解析用户信息（对齐 Java GatewayUserFilter.doFilterInternal）
		userIDHeader := r.Header.Get("X-User-Id")
		roleIDHeader := r.Header.Get("X-User-Role")
		openIDHeader := r.Header.Get("X-User-OpenId")

		user := commonctx.FromHeaders(userIDHeader, roleIDHeader, openIDHeader)
		if user != nil {
			// 写入 context（对齐 Java UserContext.setUser）
			ctx := commonctx.WithUser(r.Context(), user)
			r = r.WithContext(ctx)
		}

		// 转发请求
		next.ServeHTTP(w, r)

		// context 随请求结束自动清理，无需手动 remove（对齐 Java UserInterceptor.afterCompletion）
	})
}
