// Package main business-service 启动入口
//
// 对齐 Java BusinessServiceApplication.java
// 启动方式：go run ./business-service
//
// 端口：10001（与 Java business-service 一致）
// 路由前缀：/biz（经 Gateway StripPrefix=1 后实际路径为 /{module}/**）
package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kurimula-airi/course_record_go/business-service/internal/handler"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/business-service/internal/service"
	commonctx "github.com/kurimula-airi/course_record_go/common/context"
	"github.com/kurimula-airi/course_record_go/common/db"
)

// main business-service 启动入口
//
// 职责：
//  1. 加载数据库配置
//  2. 初始化各 Mapper
//  3. 初始化各 Service
//  4. 注册路由并启动 HTTP 服务
func main() {
	// 1. 加载数据库配置（对齐 Java common-db.yaml）
	dbCfg := db.DefaultConfig()
	if envDBHost := os.Getenv("DB_HOST"); envDBHost != "" {
		dbCfg.Host = envDBHost
	}

	// 连接 MySQL
	database, err := db.NewMySQL(dbCfg)
	if err != nil {
		log.Fatalf("❌ MySQL 连接失败: %v", err)
	}
	defer database.Close()

	// 2. 初始化各 Mapper
	institutionMapper := mapper.NewInstitutionMapper(database)
	studentMapper := mapper.NewStudentMapper(database)
	teacherMapper := mapper.NewTeacherMapper(database)

	// 3. 初始化各 Service
	institutionService := service.NewInstitutionService(institutionMapper)
	studentService := service.NewStudentService(studentMapper)
	teacherService := service.NewTeacherService(teacherMapper)

	// 4. 创建 HTTP 路由
	mux := http.NewServeMux()

	// 注册业务路由
	bizHandler := handler.NewBusinessHandler(institutionService, studentService, teacherService)
	bizHandler.RegisterRoutes(mux)

	// 健康检查接口
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"business-service"}`))
	})

	// 5. 应用中间件
	finalHandler := commonctxMiddleware(mux)

	// 6. 启动 HTTP 服务
	port := 10001 // 与 Java business-service 一致
	if envPort := os.Getenv("BIZ_PORT"); envPort != "" {
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

	log.Printf("🚀 Go business-service 启动中，监听 %s", addr)
	log.Printf("   数据库: %s:%d/%s", dbCfg.Host, dbCfg.Port, dbCfg.Database)
	log.Printf("   路由前缀: /biz (StripPrefix=1)")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ business-service 启动失败: %v", err)
	}
}

// commonctxMiddleware 用户上下文中间件
//
// 对齐 Java GatewayUserFilter + UserInterceptor
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
