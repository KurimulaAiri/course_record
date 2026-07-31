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

// SM2 私钥默认值（与 admin-service 一致，来自 Java Nacos cr-auth-service.yaml）
// 优先从环境变量 SM2_PRIVATE_KEY 读取，未设置时使用此默认值以便本地开发
const defaultSM2PrivateKey = "1f9fe7f47b0a27025ff42a0da5039827a629f3a26a5721d5a3ad04e3fc5d8969"

// main business-service 启动入口
//
// 职责：
//  1. 加载数据库配置
//  2. 初始化全部 17 个 Mapper
//  3. 读取 SM2 私钥（环境变量 SM2_PRIVATE_KEY，未设置则用默认值）
//  4. 初始化全部 8 个 Service
//  5. 注入 BusinessHandler 并注册路由
//  6. 启动 HTTP 服务
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

	// 2. 初始化全部 Mapper（17 个，对齐 Java 各 Mapper）
	// 基础业务 Mapper
	institutionMapper := mapper.NewInstitutionMapper(database)
	studentMapper := mapper.NewStudentMapper(database)
	teacherMapper := mapper.NewTeacherMapper(database)
	// 家长/用户体系 Mapper（StudentService 依赖）
	parentStudentMapper := mapper.NewParentStudentMapper(database)
	parentMapper := mapper.NewParentMapper(database)
	userAuthMapper := mapper.NewUserAuthMapper(database)
	userMapper := mapper.NewUserMapper(database)
	wxStudentSubscribeMapper := mapper.NewWxStudentSubscribeMapper(database)
	wxSubscribeRecordMapper := mapper.NewWxSubscribeRecordMapper(database)
	userPlatformMapper := mapper.NewUserPlatformMapper(database)
	// 班级体系 Mapper（ClassService / CourseRecordService 依赖）
	classMapper := mapper.NewClassMapper(database)
	classStudentMapper := mapper.NewClassStudentMapper(database)
	classTeacherMapper := mapper.NewClassTeacherMapper(database)
	classScheduleMapper := mapper.NewClassScheduleMapper(database)
	// 课程/课卡/上课记录 Mapper
	courseMapper := mapper.NewCourseMapper(database)
	courseRecordMapper := mapper.NewCourseRecordMapper(database)
	recordMapper := mapper.NewRecordMapper(database)

	// 3. 读取 SM2 私钥（环境变量优先，未设置则用默认值）
	// 用于 TeacherService 新增/更新教师时解密前端 SM2 加密的密码密文
	sm2PrivateKey := os.Getenv("SM2_PRIVATE_KEY")
	if sm2PrivateKey == "" {
		sm2PrivateKey = defaultSM2PrivateKey
		log.Printf("⚠️ 环境变量 SM2_PRIVATE_KEY 未设置，使用默认 SM2 私钥（仅供本地开发）")
	}

	// 4. 初始化全部 Service（8 个，对齐 Java 各 ServiceImpl）
	institutionService := service.NewInstitutionService(institutionMapper)
	studentService := service.NewStudentService(
		studentMapper,
		parentStudentMapper,
		parentMapper,
		wxStudentSubscribeMapper,
		wxSubscribeRecordMapper,
		userPlatformMapper,
	)
	teacherService := service.NewTeacherService(
		teacherMapper,
		userMapper,
		userAuthMapper,
		classTeacherMapper,
		sm2PrivateKey,
	)
	classService := service.NewClassService(
		classMapper,
		classStudentMapper,
		classTeacherMapper,
		classScheduleMapper,
		courseRecordMapper, // 注入课卡记录 Mapper（用于按学生ID查班级时填充 ClassVO.CourseRecord 嵌套对象）
	)
	classScheduleService := service.NewClassScheduleService(classScheduleMapper)
	// 注入 courseRecordMapper：用于按学生ID查课程时填充 CourseVO.CurrentStudentCourseRecord 嵌套对象
	courseService := service.NewCourseService(courseMapper, courseRecordMapper)
	courseRecordService := service.NewCourseRecordService(
		courseRecordMapper,
		classStudentMapper,
		recordMapper,
	)
	recordService := service.NewRecordService(recordMapper, courseRecordMapper)

	// 5. 创建 HTTP 路由并注入 Handler
	mux := http.NewServeMux()

	// 注册业务路由（全部 38 个接口）
	bizHandler := handler.NewBusinessHandler(
		institutionService,
		studentService,
		teacherService,
		classService,
		classScheduleService,
		courseService,
		courseRecordService,
		recordService,
	)
	bizHandler.RegisterRoutes(mux)

	// 健康检查接口
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"business-service"}`))
	})

	// 6. 应用中间件
	finalHandler := commonctxMiddleware(mux)

	// 7. 启动 HTTP 服务
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
// 从 Gateway 透传的 X-User-Id / X-User-Role / X-User-OpenId 头部解析用户上下文
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
