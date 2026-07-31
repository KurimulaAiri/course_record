// Package handler admin-service HTTP 处理层
//
// 对齐 Java admin-service/src/main/java/com/shiroko/controller 包
//
// 所有接口路径前缀 /admin（经 Gateway StripPrefix=1 后实际路径为 /{module}/**）
// 公开接口（免 JWT）：user/login, user/refresh, crypto/public_key
//
// 已实现接口（共 59 个）：
//   - 用户管理（10）：login, refresh, info, list, get_by_id, insert, update, delete, reset_password, get_roles
//   - 角色管理（7）：list, get_by_id, insert, update, delete, get_menus, save_menus
//   - 菜单管理（6）：list, tree, user_tree, insert, update, delete
//   - 操作日志（3）：list, delete, clear
//   - 加密（1）：crypto/public_key
//   - 业务管理透传（22）：institution/student/teacher/course/class/class_schedule/course_record/record/mini_menu 的 CRUD
//   - 教师账号管理（4）：get, update_account, update_password, toggle_institution_admin
//   - 仪表盘（3）：data, trend, institution/stats
//   - 系统配置（4）：list, insert, update, delete
package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/service"
	commonctx "github.com/kurimula-airi/course_record_go/common/context"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// AdminHandler 管理端 HTTP 处理器
//
// 聚合各业务 Service，统一注册路由
//
// 字段说明：
//   - adminService: 用户管理服务（登录/刷新/用户 CRUD）
//   - roleService: 角色管理服务（角色 CRUD + 菜单授权）
//   - menuService: 菜单管理服务（菜单 CRUD + 树形构建）
//   - logService: 操作日志服务（日志查询/删除/清空 + 记录工具方法）
//   - bizService: 业务管理透传服务（机构/学生/教师/课程/班级/课表/课时记录/上课记录/小程序菜单 CRUD）
//   - teacherAuthService: 教师账号管理服务（账号查询/更新/密码修改/机构管理员切换）
//   - dashboardService: 仪表盘服务（汇总数据/趋势/机构统计）
//   - configService: 系统配置服务（CRUD）
//   - sm2PublicKey: SM2 公钥（hex，用于 /crypto/public_key 接口返回）
type AdminHandler struct {
	adminService       *service.AdminService
	roleService        *service.SysRoleService
	menuService        *service.SysMenuService
	logService         *service.SysOperationLogService
	bizService         *service.AdminBusinessService
	teacherAuthService *service.TeacherAuthService
	dashboardService   *service.DashboardService
	configService      *service.SysConfigService
	sm2PublicKey       string // SM2 公钥（hex 编码，来自 Nacos 配置，前端用于加密密码）
}

// NewAdminHandler 创建 AdminHandler
//
// 参数：
//   - adminService: 用户管理服务
//   - roleService: 角色管理服务
//   - menuService: 菜单管理服务
//   - logService: 操作日志服务
//   - bizService: 业务管理透传服务
//   - teacherAuthService: 教师账号管理服务
//   - dashboardService: 仪表盘服务
//   - configService: 系统配置服务
//   - sm2PublicKey: SM2 公钥（hex 编码，来自 Nacos 配置，前端用于加密密码）
func NewAdminHandler(
	adminService *service.AdminService,
	roleService *service.SysRoleService,
	menuService *service.SysMenuService,
	logService *service.SysOperationLogService,
	bizService *service.AdminBusinessService,
	teacherAuthService *service.TeacherAuthService,
	dashboardService *service.DashboardService,
	configService *service.SysConfigService,
	sm2PublicKey string,
) *AdminHandler {
	return &AdminHandler{
		adminService:       adminService,
		roleService:        roleService,
		menuService:        menuService,
		logService:         logService,
		bizService:         bizService,
		teacherAuthService: teacherAuthService,
		dashboardService:   dashboardService,
		configService:      configService,
		sm2PublicKey:       sm2PublicKey,
	}
}

// RegisterRoutes 注册路由（对齐 Java 各 Controller 的 @RequestMapping）
//
// 路由前缀说明：
//   - Gateway 转发 /admin/** 到 admin-service，StripPrefix=1 去除 /admin
//   - 所以 admin-service 收到的路径是 /user/**, /role/**, /menu/**, /operation_log/**, /crypto/**
//
// 路由分组：
//   - /user/**：用户管理（对齐 Java AdminUserController @RequestMapping("/user")）
//   - /role/**：角色管理（对齐 Java SysRoleController @RequestMapping("/role")）
//   - /menu/**：菜单管理（对齐 Java SysMenuController @RequestMapping("/menu")）
//   - /operation_log/**：操作日志（对齐 Java SysOperationLogController @RequestMapping("/operation_log")）
//   - /crypto/**：加密相关（对齐 Java CryptoController @RequestMapping("/crypto")）
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	// ============================================================
	// 用户管理（对齐 Java AdminUserController）
	// ============================================================
	mux.HandleFunc("/user/login", h.Login)
	mux.HandleFunc("/user/refresh", h.RefreshToken)
	mux.HandleFunc("/user/info", h.GetUserInfo)
	mux.HandleFunc("/user/list", h.GetUserList)
	mux.HandleFunc("/user/get_by_id", h.GetUserByID)
	mux.HandleFunc("/user/insert", h.InsertUser)
	mux.HandleFunc("/user/update", h.UpdateUser)
	mux.HandleFunc("/user/delete", h.DeleteUser)
	mux.HandleFunc("/user/reset_password", h.ResetPassword)
	mux.HandleFunc("/user/get_roles", h.GetUserRoles)

	// ============================================================
	// 角色管理（对齐 Java SysRoleController）
	// ============================================================
	mux.HandleFunc("/role/list", h.GetRoleList)
	mux.HandleFunc("/role/get_by_id", h.GetRoleByID)
	mux.HandleFunc("/role/insert", h.InsertRole)
	mux.HandleFunc("/role/update", h.UpdateRole)
	mux.HandleFunc("/role/delete", h.DeleteRole)
	mux.HandleFunc("/role/get_menus", h.GetRoleMenus)
	mux.HandleFunc("/role/save_menus", h.SaveRoleMenus)

	// ============================================================
	// 菜单管理（对齐 Java SysMenuController）
	// ============================================================
	mux.HandleFunc("/menu/list", h.GetMenuList)
	mux.HandleFunc("/menu/tree", h.GetMenuTree)
	mux.HandleFunc("/menu/user_tree", h.GetUserMenuTree)
	mux.HandleFunc("/menu/insert", h.InsertMenu)
	mux.HandleFunc("/menu/update", h.UpdateMenu)
	mux.HandleFunc("/menu/delete", h.DeleteMenu)

	// ============================================================
	// 操作日志（对齐 Java SysOperationLogController）
	// ============================================================
	mux.HandleFunc("/operation_log/list", h.GetOperationLogList)
	mux.HandleFunc("/operation_log/delete", h.DeleteOperationLog)
	mux.HandleFunc("/operation_log/clear", h.ClearOperationLogs)

	// ============================================================
	// 加密相关（对齐 Java CryptoController）
	// /crypto/public_key 为公开接口，已在 Gateway publicPaths 白名单中
	// ============================================================
	mux.HandleFunc("/crypto/public_key", h.GetPublicKey)

	// ============================================================
	// 业务管理透传（对齐 Java AdminBusinessController @RequestMapping("/business")）
	// 直接操作业务表（c_student, c_teacher 等），非 RPC 调用
	// ============================================================

	// 机构管理（3 接口）
	mux.HandleFunc("/business/institution/list", h.ListInstitutions)
	mux.HandleFunc("/business/institution/insert", h.InsertInstitution)
	mux.HandleFunc("/business/institution/update", h.UpdateInstitution)

	// 学生管理（3 接口）
	mux.HandleFunc("/business/student/list", h.ListStudents)
	mux.HandleFunc("/business/student/insert", h.InsertStudent)
	mux.HandleFunc("/business/student/update", h.UpdateStudent)

	// 教师管理（3 接口）
	mux.HandleFunc("/business/teacher/list", h.ListTeachers)
	mux.HandleFunc("/business/teacher/insert", h.InsertTeacher)
	mux.HandleFunc("/business/teacher/update", h.UpdateTeacher)

	// 课程管理（3 接口）
	mux.HandleFunc("/business/course/list", h.ListCourses)
	mux.HandleFunc("/business/course/insert", h.InsertCourse)
	mux.HandleFunc("/business/course/update", h.UpdateCourse)

	// 班级管理（6 接口）
	mux.HandleFunc("/business/class/list", h.ListClasses)
	mux.HandleFunc("/business/class/insert", h.InsertClass)
	mux.HandleFunc("/business/class/update", h.UpdateClass)
	mux.HandleFunc("/business/class/get_by_id", h.GetClassByID)
	mux.HandleFunc("/business/class/add_student", h.AddStudentToClass)
	mux.HandleFunc("/business/class/remove_student", h.RemoveStudentFromClass)

	// 课表管理（2 接口）
	mux.HandleFunc("/business/class_schedule/list", h.ListClassSchedules)
	mux.HandleFunc("/business/class_schedule/update", h.UpdateClassSchedule)

	// 课时记录管理（3 接口）
	mux.HandleFunc("/business/course_record/list", h.ListCourseRecords)
	mux.HandleFunc("/business/course_record/insert", h.InsertCourseRecord)
	mux.HandleFunc("/business/course_record/update", h.UpdateCourseRecord)

	// 上课记录管理（2 接口）
	mux.HandleFunc("/business/record/list", h.ListRecords)
	mux.HandleFunc("/business/record/insert", h.InsertRecord)

	// 小程序菜单管理（4 接口）
	mux.HandleFunc("/business/mini_menu/list", h.ListMiniMenus)
	mux.HandleFunc("/business/mini_menu/insert", h.InsertMiniMenu)
	mux.HandleFunc("/business/mini_menu/update", h.UpdateMiniMenu)
	mux.HandleFunc("/business/mini_menu/delete", h.DeleteMiniMenu)

	// ============================================================
	// 教师账号管理（对齐 Java TeacherAuthController @RequestMapping("/teacher_auth")）
	// 操作 c_teacher 表的账号相关字段（username, password, is_institution_admin）
	// ============================================================
	mux.HandleFunc("/teacher_auth/get", h.GetTeacherAuth)
	mux.HandleFunc("/teacher_auth/update_account", h.UpdateTeacherAccount)
	mux.HandleFunc("/teacher_auth/update_password", h.UpdateTeacherPassword)
	mux.HandleFunc("/teacher_auth/toggle_institution_admin", h.ToggleInstitutionAdmin)

	// ============================================================
	// 仪表盘（对齐 Java SysDashboardController @RequestMapping("/dashboard")）
	// ============================================================
	mux.HandleFunc("/dashboard/data", h.GetDashboardData)
	mux.HandleFunc("/dashboard/trend", h.GetDashboardTrend)
	mux.HandleFunc("/dashboard/institution/stats", h.GetInstitutionStats)

	// ============================================================
	// 系统配置（对齐 Java SysConfigController @RequestMapping("/config")）
	// 管理端通过此接口管理系统运行时参数（JWT 过期时间、缓存 TTL 等）
	// ============================================================
	mux.HandleFunc("/config/list", h.ListConfigs)
	mux.HandleFunc("/config/insert", h.InsertConfig)
	mux.HandleFunc("/config/update", h.UpdateConfig)
	mux.HandleFunc("/config/delete", h.DeleteConfig)
}

// ============================================================
// 请求/响应辅助函数
// ============================================================

// readBody 读取请求体并反序列化为指定结构
//
// 参数：
//   - r: HTTP 请求
//   - v: 反序列化目标对象指针
func readBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

// writeResponse 写入 JSON 响应
//
// 参数：
//   - w: HTTP ResponseWriter
//   - resp: 响应对象
func writeResponse(w http.ResponseWriter, resp *response.ResponseDTO) {
	response.WriteJSON(w, resp)
}

// ============================================================
// 用户管理 Handler（对齐 Java AdminUserController）
// ============================================================

// Login 管理员登录
//
// POST /user/login
// 请求体：{ username: string, password: string（SM2 加密密文） }
// 响应：LoginVO（含 accessToken/refreshToken/userInfo）
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.Login(&req))
}

// RefreshToken 刷新 Token
//
// POST /user/refresh
// 请求体：{ refreshToken: string }
// 响应：LoginVO（含新 accessToken/refreshToken/userInfo）
func (h *AdminHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.RefreshToken(req.RefreshToken))
}

// GetUserInfo 查询当前用户信息
//
// POST /user/info
// 请求体：{ userId: number }
// 响应：SysUserVO
func (h *AdminHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.GetUserInfo(req.UserID))
}

// GetUserList 查询用户列表（分页）
//
// 对齐 admin 前端 src/api/user/index.ts getUserList
// 路径：POST /admin/user/list
// 请求体：{ username?, phone?, status?, currentPage, pageSize }
// 响应：{ list: SysUserVO[], total: number }
func (h *AdminHandler) GetUserList(w http.ResponseWriter, r *http.Request) {
	var req service.GetUserListRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.GetUserList(&req))
}

// GetUserByID 按 ID 查系统用户
//
// 对齐 admin 前端 src/api/user/index.ts getUserById
// 路径：POST /admin/user/get_by_id
// 请求体：{ id: number }
// 响应：SysUserVO
func (h *AdminHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.GetUserByID(req.ID))
}

// InsertUser 新增系统用户
//
// 对齐 admin 前端 src/api/user/index.ts insertUser
// 路径：POST /admin/user/insert
// 请求体：InsertUserRequest（password 为 SM2 加密密文）
// 响应：SysUserVO（含新用户ID）
func (h *AdminHandler) InsertUser(w http.ResponseWriter, r *http.Request) {
	var req service.InsertUserRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.InsertUser(&req))
}

// UpdateUser 更新系统用户
//
// 对齐 admin 前端 src/api/user/index.ts updateUser
// 路径：POST /admin/user/update
// 请求体：UpdateUserRequest
// 响应：SysUserVO
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateUserRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.UpdateUser(&req))
}

// DeleteUser 删除系统用户
//
// 对齐 admin 前端 src/api/user/index.ts deleteUser
// 路径：POST /admin/user/delete
// 请求体：{ id: number }
// 响应：操作结果消息
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.DeleteUser(req.ID))
}

// ResetPassword 重置用户密码
//
// 对齐 admin 前端 src/api/user/index.ts resetPassword
// 路径：POST /admin/user/reset_password
// 请求体：ResetPasswordRequest（password 为 SM2 加密密文）
// 响应：操作结果消息
func (h *AdminHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req service.ResetPasswordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.ResetPassword(&req))
}

// GetUserRoles 查询用户角色列表
//
// 对齐 admin 前端 src/api/user/index.ts getUserRoles
// 路径：POST /admin/user/get_roles
// 请求体：{ userId: number }
// 响应：SysRoleVO[]（前端实际用法 res.data.map(r => r.id) 提取 roleIds）
func (h *AdminHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.GetUserRoles(req.UserID))
}

// ============================================================
// 角色管理 Handler（对齐 Java SysRoleController）
// ============================================================

// GetRoleList 角色列表（分页）
//
// 对齐 admin 前端 src/api/role/index.ts getRoleList
// 路径：POST /admin/role/list
// 请求体：GetRoleListRequest
// 响应：{ list: SysRoleVO[], total: number }
func (h *AdminHandler) GetRoleList(w http.ResponseWriter, r *http.Request) {
	var req service.QueryRoleListRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.roleService.ListRoles(&req))
}

// GetRoleByID 按 ID 查角色
//
// 对齐 admin 前端 src/api/role/index.ts getRoleById
// 路径：POST /admin/role/get_by_id
// 请求体：{ id: number }
// 响应：SysRoleVO（含 menuIds）
func (h *AdminHandler) GetRoleByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.roleService.GetRoleByID(req.ID))
}

// InsertRole 新增角色
//
// 对齐 admin 前端 src/api/role/index.ts insertRole
// 路径：POST /admin/role/insert
// 请求体：InsertRoleRequest
// 响应：SysRoleVO
func (h *AdminHandler) InsertRole(w http.ResponseWriter, r *http.Request) {
	var req service.InsertRoleRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.roleService.InsertRole(&req))
}

// UpdateRole 更新角色
//
// 对齐 admin 前端 src/api/role/index.ts updateRole
// 路径：POST /admin/role/update
// 请求体：UpdateRoleRequest
// 响应：SysRoleVO
func (h *AdminHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateRoleRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.roleService.UpdateRole(&req))
}

// DeleteRole 删除角色
//
// 对齐 admin 前端 src/api/role/index.ts deleteRole
// 路径：POST /admin/role/delete
// 请求体：{ id: number }
// 响应：操作结果消息
func (h *AdminHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.roleService.DeleteRole(req.ID))
}

// GetRoleMenus 查询角色已分配菜单
//
// 对齐 admin 前端 src/api/role/index.ts getRoleMenus
// 路径：POST /admin/role/get_menus
// 请求体：{ roleId: number }
// 响应：SysMenuVO[]（前端实际用法 res.data.map(m => m.id) 提取 menuIds）
func (h *AdminHandler) GetRoleMenus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RoleID int64 `json:"roleId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.roleService.GetRoleMenus(req.RoleID))
}

// SaveRoleMenus 保存角色菜单授权
//
// 路径：POST /admin/role/save_menus
// 请求体：SaveRoleMenusRequest（roleId + menuIds）
// 响应：操作结果消息
//
// 注意：本接口使用事务保证"删旧+插新"原子性
// 前端通常通过 updateRole 的 menuIds 字段间接保存，本接口为独立保存入口
func (h *AdminHandler) SaveRoleMenus(w http.ResponseWriter, r *http.Request) {
	var req service.SaveRoleMenusRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// menuIds 为 nil 时转为空切片（表示清空所有关联）
	menuIDs := req.MenuIDs
	if menuIDs == nil {
		menuIDs = []int64{}
	}
	writeResponse(w, h.roleService.SaveRoleMenus(req.RoleID, menuIDs))
}

// ============================================================
// 菜单管理 Handler（对齐 Java SysMenuController）
// ============================================================

// GetMenuList 菜单扁平列表（带树形构建）
//
// 对齐 admin 前端 src/api/menu/index.ts getMenuList
// 路径：POST /admin/menu/list
// 请求体：GetMenuListRequest（可选筛选条件）
// 响应：SysMenuVO[]（树形结构）
func (h *AdminHandler) GetMenuList(w http.ResponseWriter, r *http.Request) {
	var req service.QueryMenuListRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.menuService.ListMenus(&req))
}

// GetMenuTree 完整菜单树
//
// 对齐 admin 前端 src/api/menu/index.ts getMenuTree
// 路径：POST /admin/menu/tree
// 请求体：GetMenuListRequest（可选筛选条件）
// 响应：SysMenuVO[]（树形结构）
func (h *AdminHandler) GetMenuTree(w http.ResponseWriter, r *http.Request) {
	// 兼容前端可能传入的筛选条件（虽然 tree 通常返回全部）
	var req service.QueryMenuListRequest
	if err := readBody(r, &req); err != nil {
		// 解析失败时忽略错误，返回完整菜单树
	}
	// tree 接口忽略筛选条件，返回完整菜单树（对齐 Java getMenuTree）
	writeResponse(w, h.menuService.GetMenuTree())
}

// GetUserMenuTree 当前用户菜单树（按 roleIds 过滤）
//
// 对齐 admin 前端 src/api/menu/index.ts getUserMenuTree
// 路径：POST /admin/menu/user_tree
// 请求体：空（从 JWT/UserContext 提取 userID）
// 响应：SysMenuVO[]（树形结构，仅含当前用户有权访问的菜单）
func (h *AdminHandler) GetUserMenuTree(w http.ResponseWriter, r *http.Request) {
	// 从 UserContext 提取当前用户ID（由 Gateway 注入 X-User-Id header，中间件写入 context）
	userID := commonctx.GetUserID(r.Context())
	writeResponse(w, h.menuService.GetUserMenuTree(userID))
}

// InsertMenu 新增菜单
//
// 对齐 admin 前端 src/api/menu/index.ts insertMenu
// 路径：POST /admin/menu/insert
// 请求体：InsertMenuRequest
// 响应：SysMenuVO
func (h *AdminHandler) InsertMenu(w http.ResponseWriter, r *http.Request) {
	var req service.InsertMenuRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.menuService.InsertMenu(&req))
}

// UpdateMenu 更新菜单
//
// 对齐 admin 前端 src/api/menu/index.ts updateMenu
// 路径：POST /admin/menu/update
// 请求体：UpdateMenuRequest
// 响应：SysMenuVO
func (h *AdminHandler) UpdateMenu(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateMenuRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.menuService.UpdateMenu(&req))
}

// DeleteMenu 删除菜单
//
// 对齐 admin 前端 src/api/menu/index.ts deleteMenu
// 路径：POST /admin/menu/delete
// 请求体：{ id: number }
// 响应：操作结果消息
func (h *AdminHandler) DeleteMenu(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.menuService.DeleteMenu(req.ID))
}

// ============================================================
// 操作日志 Handler（对齐 Java SysOperationLogController）
// ============================================================

// GetOperationLogList 操作日志列表（分页+筛选）
//
// 对齐 admin 前端 src/api/log/index.ts getOperationLogList
// 路径：POST /admin/operation_log/list
// 请求体：GetOperationLogListRequest
// 响应：{ list: SysOperationLogVO[], total: number }
func (h *AdminHandler) GetOperationLogList(w http.ResponseWriter, r *http.Request) {
	var req service.QueryOperationLogListRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.logService.ListLogs(&req))
}

// DeleteOperationLog 删除单条操作日志
//
// 对齐 admin 前端 src/api/log/index.ts deleteOperationLog
// 路径：POST /admin/operation_log/delete
// 请求体：{ id: number }
// 响应：操作结果消息
func (h *AdminHandler) DeleteOperationLog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.logService.DeleteLog(req.ID))
}

// ClearOperationLogs 清空全部操作日志
//
// 对齐 admin 前端 src/api/log/index.ts clearOperationLogs
// 路径：POST /admin/operation_log/clear
// 请求体：空
// 响应：操作结果消息
func (h *AdminHandler) ClearOperationLogs(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, h.logService.ClearLogs())
}

// ============================================================
// 加密相关 Handler（对齐 Java CryptoController）
// ============================================================

// GetPublicKey 获取 SM2 公钥
//
// 对齐 Java CryptoController.getPublicKey
// GET /crypto/public_key
//
// 前端登录前调用此接口获取 SM2 公钥，用于加密密码
// 响应：{ "publicKey": "<sm2-public-key>" }
func (h *AdminHandler) GetPublicKey(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, response.Success(map[string]string{
		"publicKey": h.sm2PublicKey,
	}))
}
