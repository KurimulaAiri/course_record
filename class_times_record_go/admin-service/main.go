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

// SM2 密钥配置（对齐 Java Nacos cr-admin-service.yaml）
//
// 前端登录流程：
//   1. GET /admin/crypto/public_key 获取公钥
//   2. 前端用 sm-crypto 库以公钥加密密码（C1C3C2 模式，"04" 前缀）
//   3. POST /admin/user/login 提交加密后的密码
//   4. 后端用私钥 SM2 解密 → BCrypt 校验
//
// 这两个密钥必须与 Java 后端 Nacos 配置保持一致，否则 Go/Java 实例无法互通
const (
	sm2PublicKey  = "04ac715b7e653298c9667b366268e6ebdf67ca135259fc1c4183977df54e45bbe8efad05ba0fea995f45f0548ddb79426b6801fc11363de7d1662c19e4d9452fd1"
	sm2PrivateKey = "1f9fe7f47b0a27025ff42a0da5039827a629f3a26a5721d5a3ad04e3fc5d8969"
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
	// 注入 SM2 私钥用于登录时解密前端加密的密码（对齐 Java SM2Util.decrypt）
	adminUserMapper := mapper.NewAdminUserMapper(database)
	adminService := service.NewAdminService(adminUserMapper, jwtUtils, sm2PrivateKey)

	// 初始化系统角色 Mapper（对齐 Java SysRoleMapper / SysRoleMenuMapper）
	sysRoleMapper := mapper.NewSysRoleMapper(database)
	sysRoleMenuMapper := mapper.NewSysRoleMenuMapper(database)
	// 初始化系统菜单 Mapper（对齐 Java SysMenuMapper）
	sysMenuMapper := mapper.NewSysMenuMapper(database)
	// 初始化操作日志 Mapper（对齐 Java SysOperationLogMapper）
	sysOperationLogMapper := mapper.NewSysOperationLogMapper(database)

	// 初始化角色 Service（注入 roleMapper / roleMenuMapper / menuMapper / db，db 用于 save_menus 事务）
	roleService := service.NewSysRoleService(sysRoleMapper, sysRoleMenuMapper, sysMenuMapper, database)
	// 初始化菜单 Service（注入 menuMapper / roleMenuMapper / userMapper，userMapper 用于 user_tree 查询角色ID）
	menuService := service.NewSysMenuService(sysMenuMapper, sysRoleMenuMapper, adminUserMapper)
	// 初始化操作日志 Service
	logService := service.NewSysOperationLogService(sysOperationLogMapper)

	// ============================================================
	// 阶段四：业务管理透传 + 教师账号管理（对齐 Java AdminBusinessController / TeacherAuthController）
	// ============================================================
	// 业务管理 Mapper（直接操作业务表 c_student/c_teacher/c_course 等，非 RPC 调用）
	adminBusinessMapper := mapper.NewAdminBusinessMapper(database)
	// 业务管理透传 Service（注入 bizMapper / logService，logService 用于记录写操作日志）
	bizService := service.NewAdminBusinessService(adminBusinessMapper, logService)
	// 教师账号管理 Service（注入 bizMapper / sm2PrivateKey / logService，sm2PrivateKey 用于解密前端加密的新密码）
	// 教师账号（c_user_auth, role_id=4, SM3+salt）与系统管理员（sys_user, BCrypt）是不同身份
	teacherAuthService := service.NewTeacherAuthService(adminBusinessMapper, sm2PrivateKey, logService)

	// ============================================================
	// 阶段五：仪表盘 + 系统配置（对齐 Java SysDashboardController / SysConfigController）
	// ============================================================
	// 仪表盘 Mapper（统计查询：学生/教师/机构/课程/班级 总数 + 趋势 + 机构统计）
	dashboardMapper := mapper.NewDashboardMapper(database)
	// 仪表盘 Service
	dashboardService := service.NewDashboardService(dashboardMapper)
	// 系统配置 Mapper（操作 sys_config 表）
	sysConfigMapper := mapper.NewSysConfigMapper(database)
	// 系统配置 Service（注入 configMapper / logService，logService 用于记录写操作日志）
	configService := service.NewSysConfigService(sysConfigMapper, logService)

	// 4. 创建 HTTP 路由
	mux := http.NewServeMux()

	// 注入所有 Service 和 SM2 公钥到 Handler
	// SM2 公钥用于 /crypto/public_key 接口返回（对齐 Java CryptoController.getPublicKey）
	adminHandler := handler.NewAdminHandler(
		adminService,
		roleService,
		menuService,
		logService,
		bizService,
		teacherAuthService,
		dashboardService,
		configService,
		sm2PublicKey,
	)
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
