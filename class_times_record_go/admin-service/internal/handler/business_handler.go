// Package handler admin-service 业务管理透传 / 教师账号 / 仪表盘 / 系统配置 HTTP 处理层
//
// 本文件聚合阶段四（业务管理透传 29 接口）+ 阶段五（仪表盘 3 + 系统配置 4 接口）共 36 个 Handler 方法，
// 对齐 Java AdminBusinessController / TeacherAuthController / SysDashboardController / SysConfigController。
//
// 路由注册在 admin_handler.go 的 RegisterRoutes 方法中完成，本文件只提供 Handler 方法实现。
//
// 设计要点：
//   - 所有 Handler 统一使用 readBody 解析请求体、writeResponse 输出响应
//   - 请求参数校验由 Service 层负责，Handler 仅做反序列化与转发
//   - 写操作的操作日志记录由 Service 层通过 logService.RecordLog 完成
package handler

import (
	"net/http"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/service"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// 业务管理透传 Handler（对齐 Java AdminBusinessController @RequestMapping("/business")）
// 直接操作业务表（c_student, c_teacher 等），非 RPC 调用
// ============================================================

// -------------------- 机构管理（3 接口） --------------------

// ListInstitutions 机构分页列表
//
// POST /admin/business/institution/list
// 请求体：QueryInstitutionRequest
// 响应：{ list: InstitutionResponse[], total: number }
func (h *AdminHandler) ListInstitutions(w http.ResponseWriter, r *http.Request) {
	var req service.QueryInstitutionRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListInstitutions(&req))
}

// InsertInstitution 新增机构
//
// POST /admin/business/institution/insert
// 请求体：InsertInstitutionRequest
// 响应：InstitutionResponse（含机构编码）
func (h *AdminHandler) InsertInstitution(w http.ResponseWriter, r *http.Request) {
	var req service.InsertInstitutionRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertInstitution(&req))
}

// UpdateInstitution 更新机构
//
// POST /admin/business/institution/update
// 请求体：UpdateInstitutionRequest
// 响应：InstitutionResponse（更新后的完整信息）
func (h *AdminHandler) UpdateInstitution(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateInstitutionRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateInstitution(&req))
}

// -------------------- 学生管理（3 接口） --------------------

// ListStudents 学生分页列表
//
// POST /admin/business/student/list
// 请求体：QueryStudentRequest
// 响应：{ list: StudentResponse[], total: number }
func (h *AdminHandler) ListStudents(w http.ResponseWriter, r *http.Request) {
	var req service.QueryStudentRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListStudents(&req))
}

// InsertStudent 新增学生
//
// POST /admin/business/student/insert
// 请求体：InsertStudentRequest
// 响应：{ studentId: number }
func (h *AdminHandler) InsertStudent(w http.ResponseWriter, r *http.Request) {
	var req service.InsertStudentRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertStudent(&req))
}

// UpdateStudent 更新学生
//
// POST /admin/business/student/update
// 请求体：UpdateStudentRequest
// 响应：{ studentId: number }
func (h *AdminHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateStudentRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateStudent(&req))
}

// -------------------- 教师管理（3 接口） --------------------

// ListTeachers 教师分页列表
//
// POST /admin/business/teacher/list
// 请求体：QueryTeacherRequest
// 响应：{ list: TeacherResponse[], total: number }
func (h *AdminHandler) ListTeachers(w http.ResponseWriter, r *http.Request) {
	var req service.QueryTeacherRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListTeachers(&req))
}

// InsertTeacher 新增教师
//
// POST /admin/business/teacher/insert
// 请求体：InsertTeacherRequest
// 响应：{ teacherId: number }
func (h *AdminHandler) InsertTeacher(w http.ResponseWriter, r *http.Request) {
	var req service.InsertTeacherRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertTeacher(&req))
}

// UpdateTeacher 更新教师
//
// POST /admin/business/teacher/update
// 请求体：UpdateTeacherRequest
// 响应：{ teacherId: number }
func (h *AdminHandler) UpdateTeacher(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateTeacherRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateTeacher(&req))
}

// -------------------- 课程管理（3 接口） --------------------

// ListCourses 课程分页列表
//
// POST /admin/business/course/list
// 请求体：QueryCourseRequest
// 响应：{ list: CourseResponse[], total: number }
func (h *AdminHandler) ListCourses(w http.ResponseWriter, r *http.Request) {
	var req service.QueryCourseRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListCourses(&req))
}

// InsertCourse 新增课程
//
// POST /admin/business/course/insert
// 请求体：InsertCourseRequest
// 响应：{ courseId: number }
func (h *AdminHandler) InsertCourse(w http.ResponseWriter, r *http.Request) {
	var req service.InsertCourseRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertCourse(&req))
}

// UpdateCourse 更新课程
//
// POST /admin/business/course/update
// 请求体：UpdateCourseRequest
// 响应：{ courseId: number }
func (h *AdminHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateCourseRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateCourse(&req))
}

// -------------------- 班级管理（6 接口） --------------------

// ListClasses 班级分页列表
//
// POST /admin/business/class/list
// 请求体：QueryClassRequest
// 响应：{ list: ClassResponse[], total: number }
func (h *AdminHandler) ListClasses(w http.ResponseWriter, r *http.Request) {
	var req service.QueryClassRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListClasses(&req))
}

// InsertClass 新增班级
//
// POST /admin/business/class/insert
// 请求体：InsertClassRequest
// 响应：{ classId: number }
func (h *AdminHandler) InsertClass(w http.ResponseWriter, r *http.Request) {
	var req service.InsertClassRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertClass(&req))
}

// UpdateClass 更新班级
//
// POST /admin/business/class/update
// 请求体：UpdateClassRequest
// 响应：{ classId: number }
func (h *AdminHandler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateClassRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateClass(&req))
}

// GetClassByID 按ID查班级详情（含学生列表和教师列表）
//
// POST /admin/business/class/get_by_id
// 请求体：{ id: number }
// 响应：班级详情（含 students/teacherIds/teacherNames）
func (h *AdminHandler) GetClassByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"` // 班级ID（兼容 id 字段名，对齐其他 get_by_id 接口）
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.GetClassByID(req.ID))
}

// AddStudentToClass 班级添加学生
//
// POST /admin/business/class/add_student
// 请求体：ClassStudentRequest（classId + studentId）
// 响应：操作结果消息
func (h *AdminHandler) AddStudentToClass(w http.ResponseWriter, r *http.Request) {
	var req service.ClassStudentRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.AddStudentToClass(&req))
}

// RemoveStudentFromClass 班级移除学生
//
// POST /admin/business/class/remove_student
// 请求体：ClassStudentRequest（classId + studentId）
// 响应：操作结果消息
func (h *AdminHandler) RemoveStudentFromClass(w http.ResponseWriter, r *http.Request) {
	var req service.ClassStudentRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.RemoveStudentFromClass(&req))
}

// -------------------- 课表管理（2 接口） --------------------

// ListClassSchedules 课表列表
//
// POST /admin/business/class_schedule/list
// 请求体：QueryClassScheduleRequest
// 响应：{ list: ClassScheduleResponse[], total: number }
func (h *AdminHandler) ListClassSchedules(w http.ResponseWriter, r *http.Request) {
	var req service.QueryClassScheduleRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListClassSchedules(&req))
}

// UpdateClassSchedule 更新课表
//
// POST /admin/business/class_schedule/update
// 请求体：UpdateClassScheduleRequest
// 响应：ClassScheduleResponse（更新后的完整信息）
func (h *AdminHandler) UpdateClassSchedule(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateClassScheduleRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateClassSchedule(&req))
}

// -------------------- 课时记录管理（3 接口） --------------------

// ListCourseRecords 课时记录分页列表
//
// POST /admin/business/course_record/list
// 请求体：QueryCourseRecordRequest
// 响应：{ list: CourseRecordResponse[], total: number }
func (h *AdminHandler) ListCourseRecords(w http.ResponseWriter, r *http.Request) {
	var req service.QueryCourseRecordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListCourseRecords(&req))
}

// InsertCourseRecord 新增课时记录
//
// POST /admin/business/course_record/insert
// 请求体：InsertCourseRecordRequest
// 响应：{ courseRecordId: number }
func (h *AdminHandler) InsertCourseRecord(w http.ResponseWriter, r *http.Request) {
	var req service.InsertCourseRecordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertCourseRecord(&req))
}

// UpdateCourseRecord 更新课时记录
//
// POST /admin/business/course_record/update
// 请求体：UpdateCourseRecordRequest
// 响应：{ courseRecordId: number }
func (h *AdminHandler) UpdateCourseRecord(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateCourseRecordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateCourseRecord(&req))
}

// -------------------- 上课记录管理（2 接口） --------------------

// ListRecords 上课记录分页列表
//
// POST /admin/business/record/list
// 请求体：QueryRecordRequest
// 响应：{ list: RecordResponse[], total: number }
func (h *AdminHandler) ListRecords(w http.ResponseWriter, r *http.Request) {
	var req service.QueryRecordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.ListRecords(&req))
}

// InsertRecord 新增上课记录
//
// POST /admin/business/record/insert
// 请求体：InsertRecordRequest
// 响应：{ recordId: number }
func (h *AdminHandler) InsertRecord(w http.ResponseWriter, r *http.Request) {
	var req service.InsertRecordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertRecord(&req))
}

// -------------------- 小程序菜单管理（4 接口） --------------------

// ListMiniMenus 查询所有小程序菜单
//
// POST /admin/business/mini_menu/list
// 请求体：空
// 响应：MiniMenuResponse[]（含 roleIds）
func (h *AdminHandler) ListMiniMenus(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, h.bizService.ListMiniMenus())
}

// InsertMiniMenu 新增小程序菜单
//
// POST /admin/business/mini_menu/insert
// 请求体：InsertMiniMenuRequest
// 响应：{ id: number, menuName: string }
func (h *AdminHandler) InsertMiniMenu(w http.ResponseWriter, r *http.Request) {
	var req service.InsertMiniMenuRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.InsertMiniMenu(&req))
}

// UpdateMiniMenu 更新小程序菜单
//
// POST /admin/business/mini_menu/update
// 请求体：UpdateMiniMenuRequest
// 响应：{ id: number }
func (h *AdminHandler) UpdateMiniMenu(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateMiniMenuRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.UpdateMiniMenu(&req))
}

// DeleteMiniMenu 删除小程序菜单
//
// POST /admin/business/mini_menu/delete
// 请求体：{ id: number }
// 响应：操作结果消息
func (h *AdminHandler) DeleteMiniMenu(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"` // 菜单ID
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.bizService.DeleteMiniMenu(req.ID))
}

// ============================================================
// 教师账号管理 Handler（对齐 Java TeacherAuthController @RequestMapping("/teacher_auth")）
// 操作 c_teacher 表的账号相关字段（username, password, is_institution_admin）
// 教师账号（c_user_auth, role_id=4, SM3+salt）与系统管理员（sys_user, BCrypt）是不同身份
// ============================================================

// GetTeacherAuth 查询教师账号信息
//
// POST /admin/teacher_auth/get
// 请求体：{ teacherId: number }
// 响应：{ id, userId, account, lastLoginTime }
func (h *AdminHandler) GetTeacherAuth(w http.ResponseWriter, r *http.Request) {
	var req service.TeacherAuthRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherAuthService.GetTeacherAuth(req.TeacherID))
}

// UpdateTeacherAccount 更新教师登录账号
//
// POST /admin/teacher_auth/update_account
// 请求体：UpdateTeacherAccountRequest（teacherId + account）
// 响应：操作结果消息
func (h *AdminHandler) UpdateTeacherAccount(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateTeacherAccountRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherAuthService.UpdateTeacherAccount(&req))
}

// UpdateTeacherPassword 修改教师密码
//
// POST /admin/teacher_auth/update_password
// 请求体：UpdateTeacherPasswordRequest（teacherId + password，password 为 SM2 加密密文）
// 响应：操作结果消息
func (h *AdminHandler) UpdateTeacherPassword(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateTeacherPasswordRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherAuthService.UpdateTeacherPassword(&req))
}

// ToggleInstitutionAdmin 切换教师机构管理员身份
//
// POST /admin/teacher_auth/toggle_institution_admin
// 请求体：ToggleInstitutionAdminRequest（teacherId + isInstitutionAdmin）
// 响应：操作结果消息
//
// 注意：机构管理员（c_teacher.is_institution_admin）与系统管理员（sys_user）是不同身份
func (h *AdminHandler) ToggleInstitutionAdmin(w http.ResponseWriter, r *http.Request) {
	var req service.ToggleInstitutionAdminRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherAuthService.ToggleInstitutionAdmin(&req))
}

// ============================================================
// 仪表盘 Handler（对齐 Java SysDashboardController @RequestMapping("/dashboard")）
// ============================================================

// GetDashboardData 获取仪表盘汇总数据
//
// POST /admin/dashboard/data
// 请求体：空
// 响应：{ studentCount, teacherCount, institutionCount, courseCount, classCount }
func (h *AdminHandler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, h.dashboardService.GetDashboardData())
}

// GetDashboardTrend 获取趋势数据
//
// POST /admin/dashboard/trend
// 请求体：{ range: string }（week/month/halfyear/year，默认 year）
// 响应：趋势数据（按 range 决定粒度与刻度）
func (h *AdminHandler) GetDashboardTrend(w http.ResponseWriter, r *http.Request) {
	var req service.DashboardTrendRequest
	if err := readBody(r, &req); err != nil {
		// 解析失败时使用默认值（range=year），不阻断请求
		req = service.DashboardTrendRequest{}
	}
	writeResponse(w, h.dashboardService.GetTrend(&req))
}

// GetInstitutionStats 获取机构统计列表
//
// POST /admin/dashboard/institution/stats
// 请求体：{ limit: number }（<=0 表示不限制）
// 响应：InstitutionStatRow[]（按学生数降序）
func (h *AdminHandler) GetInstitutionStats(w http.ResponseWriter, r *http.Request) {
	var req service.InstitutionStatsRequest
	if err := readBody(r, &req); err != nil {
		// 解析失败时使用默认值（不限制数量），不阻断请求
		req = service.InstitutionStatsRequest{}
	}
	writeResponse(w, h.dashboardService.GetInstitutionStats(&req))
}

// ============================================================
// 系统配置 Handler（对齐 Java SysConfigController @RequestMapping("/config")）
// 管理端通过此接口管理系统运行时参数（JWT 过期时间、缓存 TTL 等）
// ============================================================

// ListConfigs 查询系统配置列表
//
// POST /admin/config/list
// 请求体：QuerySysConfigRequest（可选筛选条件）
// 响应：SysConfigResponse[]
func (h *AdminHandler) ListConfigs(w http.ResponseWriter, r *http.Request) {
	var req service.QuerySysConfigRequest
	if err := readBody(r, &req); err != nil {
		// 解析失败时返回全部配置，不阻断请求
		req = service.QuerySysConfigRequest{}
	}
	writeResponse(w, h.configService.ListConfigs(&req))
}

// InsertConfig 新增系统配置
//
// POST /admin/config/insert
// 请求体：InsertSysConfigRequest
// 响应：SysConfigResponse（新增后的完整配置）
func (h *AdminHandler) InsertConfig(w http.ResponseWriter, r *http.Request) {
	var req service.InsertSysConfigRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.configService.InsertConfig(&req))
}

// UpdateConfig 更新系统配置
//
// POST /admin/config/update
// 请求体：UpdateSysConfigRequest
// 响应：SysConfigResponse（更新后的完整配置）
func (h *AdminHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req service.UpdateSysConfigRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.configService.UpdateConfig(&req))
}

// DeleteConfig 删除系统配置
//
// POST /admin/config/delete
// 请求体：{ id: number }
// 响应：操作结果消息
func (h *AdminHandler) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID int64 `json:"id"` // 配置ID
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.configService.DeleteConfig(req.ID))
}
