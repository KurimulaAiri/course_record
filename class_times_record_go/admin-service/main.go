// Package main admin-service 启动入口
//
// 对齐 Java AdminServiceApplication.java
// 启动方式：go run ./admin-service
//
// 端口：10003（与 Java admin-service 一致）
// 路由前缀：/admin（经 Gateway StripPrefix=1 后实际路径为 /{module}/**）
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/handler"
	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/admin-service/internal/service"
	commonctx "github.com/kurimula-airi/course_record_go/common/context"
	"github.com/kurimula-airi/course_record_go/common/db"
	"github.com/kurimula-airi/course_record_go/common/jwt"
)

// main admin-service 启动入口
func main() {
	// 1. 加载数据库配置
	dbCfg := db.DefaultConfig()
	if envDBHost := os.Getenv("DB_HOST"); envDBHost != "" {
		dbCfg.Host = envDBHost
	}

	database, err := db.NewMySQL(dbCfg)
	if err != nil {
		log.Fatalf("❌ MySQL 连接失败: %v", err)
	}
	defer database.Close()

	// 2. 初始化 JWT 工具（与 Java 使用相同密钥，保证 Token 互通）
	jwtUtils := jwt.NewUtils(
		"shiroko_project_secret_key_at_least_32_chars_long",
		5*60*1000,
		7*24*60*60*1000,
	)

	// 3. 初始化 Mapper 和 Service
	adminUserMapper := mapper.NewAdminUserMapper(database)
	adminService := service.NewAdminService(adminUserMapper, jwtUtils)

	// 4. 创建 HTTP 路由
	mux := http.NewServeMux()

	adminHandler := handler.NewAdminHandler(adminService)
	adminHandler.RegisterRoutes(mux)

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"admin-service"}`))
	})

	// 5. 应用中间件
	finalHandler := commonctxMiddleware(mux)

	// 6. 启动 HTTP 服务
	port := 10003
	if envPort := os.Getenv("ADMIN_PORT"); envPort != "" {
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

	log.Printf("🚀 Go admin-service 启动中，监听 %s", addr)
	log.Printf("   数据库: %s:%d/%s", dbCfg.Host, dbCfg.Port, dbCfg.Database)
	log.Printf("   路由前缀: /admin (StripPrefix=1)")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ admin-service 启动失败: %v", err)
	}
}

// commonctxMiddleware 用户上下文中间件
func commonctxMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIDHeader := r.Header.Get("X-User-Id")
		roleIDHeader := r.Header.Get("X-User-Role")
		openIDHeader := r.Header.Get("X-User-OpenId")

		user := commonctx.FromHeaders(userIDHeader, roleIDHeader, openIDHeader)
		if user != nil {
			ctx := commonctx.WithUser(r.Context(), user)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}
