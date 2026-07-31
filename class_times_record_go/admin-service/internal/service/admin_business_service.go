// Package service admin-service 业务管理透传业务逻辑层
//
// 对齐 Java admin-service AdminBusinessServiceImpl
// 直接操作业务表（c_student, c_teacher 等），非 RPC 调用
//
// 涵盖模块：
//   - 机构管理（c_institution + c_subscription_plan）
//   - 学生管理（c_student + c_parent_student + c_parent）
//   - 教师管理（c_teacher + c_user_auth）
//   - 课程管理（c_course）
//   - 班级管理（c_class + c_class_student + c_class_teacher）
//   - 课表管理（c_class_schedule）
//   - 课时记录管理（c_course_record）
//   - 上课记录管理（c_record）
//   - 小程序菜单管理（c_menu + c_role_menu）
package service

import (
	"encoding/json"
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// DTO 定义（对齐 Java QueryXxxDTO / InsertXxxDTO / UpdateXxxDTO）
// ============================================================

// --- 机构 ---

// QueryInstitutionRequest 机构列表查询请求（对齐 admin 前端 GetInstitutionListRequest）
type QueryInstitutionRequest struct {
	InstitutionID   int64  `json:"institutionId"`   // 机构ID精确匹配（0 不过滤）
	InstitutionName string `json:"institutionName"` // 机构名称模糊匹配（空不过滤）
	InstitutionCode string `json:"institutionCode"` // 机构编码模糊匹配（空不过滤）
	Status          int64  `json:"status"`          // 状态（0 不过滤）
	CurrentPage     int    `json:"currentPage"`     // 当前页码（从1开始）
	PageSize        int    `json:"pageSize"`        // 每页条数
}

// InsertInstitutionRequest 新增机构请求（对齐 admin 前端 InsertInstitutionRequest）
type InsertInstitutionRequest struct {
	InstitutionName    string `json:"institutionName"`    // 机构名称（必填）
	InstitutionAddress string `json:"institutionAddress"` // 机构地址
	ExpireTime         string `json:"expireTime"`         // 过期时间（"" 永久有效）
	SubscriptionPlanID int64  `json:"subscriptionPlanId"` // 订阅套餐ID（0 默认1）
}

// UpdateInstitutionRequest 更新机构请求（对齐 admin 前端 UpdateInstitutionRequest）
type UpdateInstitutionRequest struct {
	ID                 int64  `json:"id"`                 // 机构ID（必填）
	InstitutionName    string `json:"institutionName"`    // 机构名称（空不更新）
	InstitutionAddress string `json:"institutionAddress"` // 机构地址（空不更新）
	InstitutionCode    string `json:"institutionCode"`    // 机构编码（空不更新）
	Status             int64  `json:"status"`             // 状态（0 不更新）
	ExpireTime         string `json:"expireTime"`         // 过期时间（"" 不更新, null 永久有效）
	SubscriptionPlanID int64  `json:"subscriptionPlanId"` // 套餐ID（0 不更新）
}

// --- 学生 ---

// QueryStudentRequest 学生列表查询请求（对齐 admin 前端 GetStudentListRequest）
type QueryStudentRequest struct {
	InstitutionID int64  `json:"institutionId"` // 机构ID（0 不过滤）
	Keyword       string `json:"keyword"`       // 关键词（姓名或学校，空不过滤）
	Sex           int64  `json:"sex"`           // 性别（-1 不过滤）
	CurrentPage   int    `json:"currentPage"`   // 当前页码
	PageSize      int    `json:"pageSize"`      // 每页条数
}

// InsertStudentRequest 新增学生请求（对齐 admin 前端 InsertStudentRequest）
type InsertStudentRequest struct {
	StudentName   string `json:"studentName"`   // 学生姓名（必填）
	InstitutionID int64  `json:"institutionId"` // 机构ID（必填）
	Sex           int64  `json:"sex"`           // 性别（0=未知,1=男,2=女）
	Birth         string `json:"birth"`         // 出生日期（"yyyy-MM-dd"，空不设置）
	School        string `json:"school"`        // 学校
	Address       string `json:"address"`       // 地址
}

// UpdateStudentRequest 更新学生请求（对齐 admin 前端 UpdateStudentRequest）
type UpdateStudentRequest struct {
	ID         int64  `json:"id"`         // 学生ID（必填）
	StudentName string `json:"studentName"` // 学生姓名（空不更新）
	Sex        int64  `json:"sex"`        // 性别（-1 不更新）
	Birth      string `json:"birth"`      // 出生日期（"" 不更新, null 设为 NULL）
	School     string `json:"school"`     // 学校（空不更新）
	Address    string `json:"address"`    // 地址（空不更新）
}

// --- 教师 ---

// QueryTeacherRequest 教师列表查询请求（对齐 admin 前端 GetTeacherListRequest）
type QueryTeacherRequest struct {
	InstitutionID int64  `json:"institutionId"` // 机构ID（0 不过滤）
	Keyword       string `json:"keyword"`       // 关键词（暂未使用，预留字段）
	IsAvailable   int64  `json:"isAvailable"`   // 是否可用（-1 不过滤,0=不可用,1=可用）
	CurrentPage   int    `json:"currentPage"`   // 当前页码
	PageSize      int    `json:"pageSize"`      // 每页条数
}

// InsertTeacherRequest 新增教师请求（对齐 admin 前端 InsertTeacherRequest）
type InsertTeacherRequest struct {
	Username      string `json:"username"`      // 教师用户名（必填）
	InstitutionID int64  `json:"institutionId"` // 机构ID（必填）
	IsAvailable   int64  `json:"isAvailable"`   // 是否可用（0=不可用,1=可用，默认1）
}

// UpdateTeacherRequest 更新教师请求（对齐 admin 前端 UpdateTeacherRequest）
type UpdateTeacherRequest struct {
	ID          int64  `json:"id"`          // 教师ID（必填）
	Username    string `json:"username"`    // 教师用户名（空不更新）
	IsAvailable int64  `json:"isAvailable"` // 是否可用（-1 不更新,0=不可用,1=可用）
}

// --- 课程 ---

// QueryCourseRequest 课程列表查询请求（对齐 admin 前端 GetCourseListRequest）
type QueryCourseRequest struct {
	InstitutionID int64  `json:"institutionId"` // 机构ID（0 不过滤）
	Keyword       string `json:"keyword"`       // 课程名称关键词（空不过滤）
	CourseType    int64  `json:"courseType"`    // 课程类型（0 不过滤）
	IsAvailable   int64  `json:"isAvailable"`   // 是否可用（-1 不过滤,0=不可用,1=可用）
	CurrentPage   int    `json:"currentPage"`   // 当前页码
	PageSize      int    `json:"pageSize"`      // 每页条数
}

// InsertCourseRequest 新增课程请求（对齐 admin 前端 InsertCourseRequest）
type InsertCourseRequest struct {
	CourseName    string `json:"courseName"`    // 课程名称（必填）
	CourseType    int64  `json:"courseType"`    // 课程类型（1=按次,2=按天）
	InstitutionID int64  `json:"institutionId"` // 机构ID（必填）
	IsAvailable   int64  `json:"isAvailable"`   // 是否可用（0=不可用,1=可用，默认1）
}

// UpdateCourseRequest 更新课程请求（对齐 admin 前端 UpdateCourseRequest）
type UpdateCourseRequest struct {
	ID          int64  `json:"id"`          // 课程ID（必填）
	CourseName  string `json:"courseName"`  // 课程名称（空不更新）
	CourseType  int64  `json:"courseType"`  // 课程类型（0 不更新）
	IsAvailable int64  `json:"isAvailable"` // 是否可用（-1 不更新,0=不可用,1=可用）
}

// --- 班级 ---

// QueryClassRequest 班级列表查询请求（对齐 admin 前端 GetClassListRequest）
type QueryClassRequest struct {
	CourseID      int64  `json:"courseId"`      // 课程ID（0 不过滤）
	Keyword       string `json:"keyword"`       // 班级名称关键词（空不过滤）
	Status        int64  `json:"status"`        // 班级状态（-1 不过滤）
	InstitutionID int64  `json:"institutionId"` // 机构ID（0 不过滤，通过课程过滤）
	CurrentPage   int    `json:"currentPage"`   // 当前页码
	PageSize      int    `json:"pageSize"`      // 每页条数
}

// InsertClassRequest 新增班级请求（对齐 admin 前端 InsertClassRequest）
type InsertClassRequest struct {
	ClassName       string  `json:"className"`       // 班级名称（必填）
	CourseID        int64   `json:"courseId"`        // 课程ID（必填）
	StudentMaxCount int64   `json:"studentMaxCount"` // 班级最大人数（0 默认0）
	Status          int64   `json:"status"`          // 班级状态（0 默认1）
	TeacherIDs      []int64 `json:"teacherIds"`      // 教师ID列表（可选）
}

// UpdateClassRequest 更新班级请求（对齐 admin 前端 UpdateClassRequest）
type UpdateClassRequest struct {
	ClassID         int64  `json:"classId"`         // 班级ID（必填）
	ClassName       string `json:"className"`       // 班级名称（空不更新）
	StudentMaxCount int64  `json:"studentMaxCount"` // 班级最大人数（0 不更新）
	Status          int64  `json:"status"`          // 班级状态（-1 不更新）
}

// ClassStudentRequest 班级学生操作请求（对齐 admin 前端 AdminClassStudentDTO）
type ClassStudentRequest struct {
	ClassID   int64 `json:"classId"`   // 班级ID
	StudentID int64 `json:"studentId"` // 学生ID
}

// --- 课表 ---

// QueryClassScheduleRequest 课表列表查询请求
type QueryClassScheduleRequest struct {
	ClassID       int64 `json:"classId"`       // 班级ID（0 不过滤）
	DayOfWeek     int64 `json:"dayOfWeek"`     // 星期几（0 不过滤）
	InstitutionID int64 `json:"institutionId"` // 机构ID（0 不过滤）
}

// UpdateClassScheduleRequest 更新课表请求（对齐 admin 前端 UpdateClassScheduleRequest）
type UpdateClassScheduleRequest struct {
	ID          int64  `json:"id"`          // 课表ID（必填）
	StartDate   string `json:"startDate"`   // 开始日期（"" 不更新）
	EndDate     string `json:"endDate"`     // 结束日期（"" 不更新）
	DayOfWeek   int64  `json:"dayOfWeek"`   // 星期几（0 不更新）
	StartTime   string `json:"startTime"`   // 开始时间（"" 不更新）
	EndTime     string `json:"endTime"`     // 结束时间（"" 不更新）
	Remark      string `json:"remark"`      // 备注（"" 不更新）
}

// --- 课时记录 ---

// QueryCourseRecordRequest 课时记录列表查询请求（对齐 admin 前端 GetCourseRecordListRequest）
type QueryCourseRecordRequest struct {
	StudentID     int64 `json:"studentId"`     // 学生ID（0 不过滤）
	CourseID      int64 `json:"courseId"`      // 课程ID（0 不过滤）
	CourseStatus  int64 `json:"courseStatus"`  // 课程状态（0 不过滤）
	InstitutionID int64 `json:"institutionId"` // 机构ID（0 不过滤，通过课程过滤）
	CurrentPage   int   `json:"currentPage"`   // 当前页码
	PageSize      int   `json:"pageSize"`      // 每页条数
}

// InsertCourseRecordRequest 新增课时记录请求（对齐 admin 前端 InsertCourseRecordRequest）
type InsertCourseRecordRequest struct {
	StudentID     int64  `json:"studentId"`     // 学生ID（必填）
	CourseID      int64  `json:"courseId"`      // 课程ID（必填）
	CourseTotalTime int64 `json:"courseTotalTime"` // 课时总数
	CourseRestTime  int64 `json:"courseRestTime"`  // 剩余课时（0 默认为 totalTime）
	ExpireTime    string `json:"expireTime"`    // 过期时间（"" 永久有效）
	CourseStatus  int64  `json:"courseStatus"`  // 课程状态（0 默认1）
	CourseRemark  string `json:"courseRemark"`  // 备注
}

// UpdateCourseRecordRequest 更新课时记录请求（对齐 admin 前端 UpdateCourseRecordRequest）
type UpdateCourseRecordRequest struct {
	ID             int64  `json:"id"`             // 记录ID（必填）
	CourseRestTime int64  `json:"courseRestTime"` // 剩余课时（-1 不更新）
	CourseStatus   int64  `json:"courseStatus"`   // 课程状态（-1 不更新）
	CourseRemark   string `json:"courseRemark"`   // 备注（"" 不更新）
}

// --- 上课记录 ---

// QueryRecordRequest 上课记录列表查询请求（对齐 admin 前端 GetRecordListRequest）
type QueryRecordRequest struct {
	CourseRecordID int64 `json:"courseRecordId"` // 课卡记录ID（0 不过滤）
	RecordType     int64 `json:"recordType"`     // 记录类型（0 不过滤）
	InstitutionID  int64 `json:"institutionId"`  // 机构ID（0 不过滤）
	CurrentPage    int   `json:"currentPage"`    // 当前页码
	PageSize       int   `json:"pageSize"`       // 每页条数
}

// InsertRecordRequest 新增上课记录请求（对齐 admin 前端 InsertRecordRequest）
type InsertRecordRequest struct {
	CourseRecordID int64  `json:"courseRecordId"` // 课卡记录ID（必填）
	RecordType     int64  `json:"recordType"`     // 记录类型（1=增加,2=减少）
	RecordChange   int64  `json:"recordChange"`   // 课时变更数量
	RecordTime     string `json:"recordTime"`     // 记录时间（"" 使用 NOW()）
	RecordRemark   string `json:"recordRemark"`   // 备注
}

// --- 小程序菜单 ---

// InsertMiniMenuRequest 新增小程序菜单请求（对齐 admin 前端 InsertMiniMenuRequest）
type InsertMiniMenuRequest struct {
	MenuName  string  `json:"menuName"`  // 菜单名称（必填）
	Icon      string  `json:"icon"`      // 图标
	IconType  int64   `json:"iconType"`  // 图标类型（0=内置,1=路径）
	BgColor   string  `json:"bgColor"`   // 背景色
	Path      string  `json:"path"`      // 跳转路径
	SortOrder int64   `json:"sortOrder"` // 排序权值
	IsVisible *bool   `json:"isVisible"` // 是否显示（nil 默认 true）
	RoleIDs   []int64 `json:"roleIds"`   // 角色（权限）ID列表
}

// UpdateMiniMenuRequest 更新小程序菜单请求（对齐 admin 前端 UpdateMiniMenuRequest）
type UpdateMiniMenuRequest struct {
	ID        int64   `json:"id"`        // 菜单ID（必填）
	MenuName  string  `json:"menuName"`  // 菜单名称（"" 不更新）
	Icon      string  `json:"icon"`      // 图标（"" 不更新）
	IconType  int64   `json:"iconType"`  // 图标类型（-1 不更新）
	BgColor   string  `json:"bgColor"`   // 背景色（"" 不更新）
	Path      string  `json:"path"`      // 跳转路径（"" 不更新）
	SortOrder int64   `json:"sortOrder"` // 排序权值（-1 不更新）
	IsVisible *bool   `json:"isVisible"` // 是否显示（nil 不更新）
	RoleIDs   []int64 `json:"roleIds"`   // 角色（权限）ID列表（nil 不更新）
}

// ============================================================
// AdminBusinessService 业务管理透传服务
// ============================================================

// AdminBusinessService 业务管理透传服务（对齐 Java AdminBusinessServiceImpl）
//
// 注入：
//   - AdminBusinessMapper：业务表数据访问
//   - logService：操作日志服务（用于记录写操作日志）
type AdminBusinessService struct {
	bizMapper  *mapper.AdminBusinessMapper
	logService *SysOperationLogService
}

// NewAdminBusinessService 创建 AdminBusinessService
//
// 参数：
//   - bizMapper: 业务管理 Mapper
//   - logService: 操作日志服务（用于记录写操作日志）
func NewAdminBusinessService(bizMapper *mapper.AdminBusinessMapper, logService *SysOperationLogService) *AdminBusinessService {
	return &AdminBusinessService{
		bizMapper:  bizMapper,
		logService: logService,
	}
}

// recordLog 记录操作日志（不阻断主流程）
//
// 内部辅助方法，统一处理操作日志记录
//
// 参数：
//   - userID: 操作用户ID
//   - username: 操作用户名
//   - operation: 操作描述
//   - method: 方法名
//   - params: 请求参数（任意类型，会转为 JSON 字符串）
func (s *AdminBusinessService) recordLog(userID int64, username, operation, method string, params interface{}) {
	if s.logService == nil {
		return
	}
	paramsStr := ""
	if params != nil {
		if b, err := json.Marshal(params); err == nil {
			paramsStr = string(b)
		}
	}
	s.logService.RecordLog(&RecordOperationLogRequest{
		UserID:    userID,
		Username:  username,
		Operation: operation,
		Method:    method,
		Params:    paramsStr,
	})
}

// ============================================================
// 机构管理
// ============================================================

// ListInstitutions 机构分页列表（对齐 Java listInstitutions）
func (s *AdminBusinessService) ListInstitutions(req *QueryInstitutionRequest) *response.ResponseDTO {
	// 1. 参数默认值处理
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	// 2. 查询机构列表
	list, total, err := s.bizMapper.ListInstitutions(req.InstitutionID, req.InstitutionName, req.InstitutionCode, req.Status, offset, req.PageSize)
	if err != nil {
		log.Printf("查询机构列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminInstitutionRow{}
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertInstitution 新增机构（对齐 Java insertInstitution）
func (s *AdminBusinessService) InsertInstitution(req *InsertInstitutionRequest) *response.ResponseDTO {
	if req.InstitutionName == "" {
		return response.Fail("机构名称不能为空")
	}
	id, err := s.bizMapper.InsertInstitution(req.InstitutionName, req.InstitutionAddress, req.SubscriptionPlanID, req.ExpireTime)
	if err != nil {
		log.Printf("新增机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增机构失败")
	}
	// 重新查询返回完整机构信息
	inst, err := s.bizMapper.SelectInstitutionByID(id)
	if err != nil || inst == nil {
		// 查询失败时返回最小信息
		return response.Success(map[string]interface{}{
			"id":              id,
			"institutionName": req.InstitutionName,
			"institutionCode": "",
		})
	}
	return response.Success(inst)
}

// UpdateInstitution 更新机构（对齐 Java updateInstitution）
func (s *AdminBusinessService) UpdateInstitution(req *UpdateInstitutionRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("机构ID不能为空")
	}
	// 过期时间处理：前端传 null 表示永久有效，转为 "NULL" 标记
	expireTimeStr := req.ExpireTime
	if expireTimeStr == "null" {
		expireTimeStr = "NULL"
	}
	if err := s.bizMapper.UpdateInstitution(req.ID, req.InstitutionName, req.InstitutionAddress, req.InstitutionCode, req.Status, req.SubscriptionPlanID, expireTimeStr); err != nil {
		log.Printf("更新机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新机构失败")
	}
	// 返回更新后的机构信息
	inst, err := s.bizMapper.SelectInstitutionByID(req.ID)
	if err != nil || inst == nil {
		return response.Success(map[string]interface{}{"id": req.ID})
	}
	return response.Success(inst)
}

// ============================================================
// 学生管理
// ============================================================

// ListStudents 学生分页列表（对齐 Java listStudents）
func (s *AdminBusinessService) ListStudents(req *QueryStudentRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	// 默认 sex=-1 表示不过滤（前端默认 0=未知是有效值）
	if req.Sex == 0 {
		req.Sex = -1
	}

	list, total, err := s.bizMapper.ListStudents(req.InstitutionID, req.Keyword, req.Sex, offset, req.PageSize)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminStudentRow{}
	}

	// 注入家长信息（主/次联系人）—— 对齐 Java listStudents 中注入家长信息的逻辑
	for _, student := range list {
		parents, err := s.bizMapper.SelectParentInfoByStudentID(student.ID)
		if err != nil {
			log.Printf("查询学生家长信息失败: studentID=%d, err=%v", student.ID, err)
			continue
		}
		for _, p := range parents {
			if p.IsPrimary == 1 {
				student.PrimaryParent = p
			} else {
				student.SecondaryParent = p
			}
		}
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertStudent 新增学生（对齐 Java insertStudent）
func (s *AdminBusinessService) InsertStudent(req *InsertStudentRequest) *response.ResponseDTO {
	if req.StudentName == "" {
		return response.Fail("学生姓名不能为空")
	}
	if req.InstitutionID == 0 {
		return response.Fail("机构ID不能为空")
	}
	id, err := s.bizMapper.InsertStudent(req.StudentName, req.InstitutionID, req.Sex, req.Birth, req.School, req.Address)
	if err != nil {
		log.Printf("新增学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增学生失败")
	}
	return response.Success(map[string]interface{}{
		"studentId": id,
	})
}

// UpdateStudent 更新学生（对齐 Java updateStudent）
func (s *AdminBusinessService) UpdateStudent(req *UpdateStudentRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("学生ID不能为空")
	}
	// 过期时间处理：前端传 null 表示清空，转为 "NULL" 标记
	birthStr := req.Birth
	if birthStr == "null" {
		birthStr = "NULL"
	}
	if err := s.bizMapper.UpdateStudent(req.ID, req.StudentName, req.Sex, birthStr, req.School, req.Address); err != nil {
		log.Printf("更新学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新学生失败")
	}
	return response.Success(map[string]interface{}{
		"studentId": req.ID,
	})
}

// ============================================================
// 教师管理
// ============================================================

// ListTeachers 教师分页列表（对齐 Java listTeachers）
func (s *AdminBusinessService) ListTeachers(req *QueryTeacherRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	list, total, err := s.bizMapper.ListTeachers(req.InstitutionID, req.Keyword, req.IsAvailable, offset, req.PageSize)
	if err != nil {
		log.Printf("查询教师列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminTeacherRow{}
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertTeacher 新增教师（对齐 Java insertTeacher）
func (s *AdminBusinessService) InsertTeacher(req *InsertTeacherRequest) *response.ResponseDTO {
	if req.Username == "" {
		return response.Fail("教师用户名不能为空")
	}
	if req.InstitutionID == 0 {
		return response.Fail("机构ID不能为空")
	}
	// 默认 isAvailable=1（可用）
	isAvailable := req.IsAvailable != 0
	teacherID, _, err := s.bizMapper.InsertTeacher(req.Username, req.InstitutionID, isAvailable)
	if err != nil {
		log.Printf("新增教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增教师失败")
	}
	return response.Success(map[string]interface{}{
		"teacherId": teacherID,
	})
}

// UpdateTeacher 更新教师（对齐 Java updateTeacher）
func (s *AdminBusinessService) UpdateTeacher(req *UpdateTeacherRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("教师ID不能为空")
	}
	if err := s.bizMapper.UpdateTeacher(req.ID, req.Username, req.IsAvailable); err != nil {
		log.Printf("更新教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新教师失败")
	}
	return response.Success(map[string]interface{}{
		"teacherId": req.ID,
	})
}

// ============================================================
// 课程管理
// ============================================================

// ListCourses 课程分页列表（对齐 Java listCourses）
func (s *AdminBusinessService) ListCourses(req *QueryCourseRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	list, total, err := s.bizMapper.ListCourses(req.InstitutionID, req.Keyword, req.CourseType, req.IsAvailable, offset, req.PageSize)
	if err != nil {
		log.Printf("查询课程列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminCourseRow{}
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertCourse 新增课程（对齐 Java insertCourse）
func (s *AdminBusinessService) InsertCourse(req *InsertCourseRequest) *response.ResponseDTO {
	if req.CourseName == "" {
		return response.Fail("课程名称不能为空")
	}
	if req.InstitutionID == 0 {
		return response.Fail("机构ID不能为空")
	}
	isAvailable := req.IsAvailable != 0
	id, err := s.bizMapper.InsertCourse(req.CourseName, req.CourseType, req.InstitutionID, isAvailable)
	if err != nil {
		log.Printf("新增课程失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增课程失败")
	}
	return response.Success(map[string]interface{}{
		"courseId": id,
	})
}

// UpdateCourse 更新课程（对齐 Java updateCourse）
func (s *AdminBusinessService) UpdateCourse(req *UpdateCourseRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("课程ID不能为空")
	}
	if err := s.bizMapper.UpdateCourse(req.ID, req.CourseName, req.CourseType, req.IsAvailable); err != nil {
		log.Printf("更新课程失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新课程失败")
	}
	return response.Success(map[string]interface{}{
		"courseId": req.ID,
	})
}

// ============================================================
// 班级管理
// ============================================================

// ListClasses 班级分页列表（对齐 Java listClasses）
func (s *AdminBusinessService) ListClasses(req *QueryClassRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	// 默认 status=-1 表示不过滤
	if req.Status == 0 {
		req.Status = -1
	}

	list, total, err := s.bizMapper.ListClasses(req.CourseID, req.Keyword, req.Status, req.InstitutionID, offset, req.PageSize)
	if err != nil {
		log.Printf("查询班级列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminClassRow{}
	}

	// 注入教师信息（教师ID列表和教师用户名列表）
	for _, class := range list {
		teacherIDs, teacherNames, err := s.bizMapper.SelectClassTeachers(class.ID)
		if err != nil {
			log.Printf("查询班级教师失败: classID=%d, err=%v", class.ID, err)
			continue
		}
		class.TeacherIDs = teacherIDs
		class.TeacherNames = teacherNames
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertClass 新增班级（对齐 Java insertClass）
func (s *AdminBusinessService) InsertClass(req *InsertClassRequest) *response.ResponseDTO {
	if req.ClassName == "" {
		return response.Fail("班级名称不能为空")
	}
	if req.CourseID == 0 {
		return response.Fail("课程ID不能为空")
	}
	// 默认 status=1
	status := req.Status
	if status == 0 {
		status = 1
	}
	classID, err := s.bizMapper.InsertClass(req.ClassName, req.CourseID, req.StudentMaxCount, status)
	if err != nil {
		log.Printf("新增班级失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增班级失败")
	}
	return response.Success(map[string]interface{}{
		"classId": classID,
	})
}

// UpdateClass 更新班级（对齐 Java updateClass）
func (s *AdminBusinessService) UpdateClass(req *UpdateClassRequest) *response.ResponseDTO {
	if req.ClassID == 0 {
		return response.Fail("班级ID不能为空")
	}
	if err := s.bizMapper.UpdateClass(req.ClassID, req.ClassName, req.StudentMaxCount, req.Status); err != nil {
		log.Printf("更新班级失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新班级失败")
	}
	return response.Success(map[string]interface{}{
		"classId": req.ClassID,
	})
}

// GetClassByID 按ID查班级详情（含学生列表和教师列表）
func (s *AdminBusinessService) GetClassByID(classID int64) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}
	// 1. 查询班级基础信息
	classRow, err := s.bizMapper.SelectClassByID(classID)
	if err != nil {
		log.Printf("查询班级失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if classRow == nil {
		return response.Fail("班级不存在")
	}
	// 2. 查询班级教师列表
	teacherIDs, teacherNames, err := s.bizMapper.SelectClassTeachers(classID)
	if err != nil {
		log.Printf("查询班级教师失败: %v", err)
	} else {
		classRow.TeacherIDs = teacherIDs
		classRow.TeacherNames = teacherNames
	}
	// 3. 查询班级学生列表
	students, err := s.bizMapper.SelectStudentsByClassID(classID)
	if err != nil {
		log.Printf("查询班级学生失败: %v", err)
	}
	if students == nil {
		students = []*mapper.AdminStudentRow{}
	}
	// 4. 组装返回结果（对齐 Java AdminClassDetailVO 结构）
	return response.Success(map[string]interface{}{
		"id":              classRow.ID,
		"courseId":        classRow.CourseID,
		"className":       classRow.ClassName,
		"status":          classRow.Status,
		"studentCount":    classRow.StudentCount,
		"studentMaxCount": classRow.StudentMaxCount,
		"teacherIds":      classRow.TeacherIDs,
		"teacherNames":    classRow.TeacherNames,
		"students":        students,
		"createTimeStr":   classRow.CreateTimeStr,
		"updateTimeStr":   classRow.UpdateTimeStr,
	})
}

// AddStudentToClass 班级添加学生（对齐 Java addStudentToClass）
func (s *AdminBusinessService) AddStudentToClass(req *ClassStudentRequest) *response.ResponseDTO {
	if req.ClassID == 0 || req.StudentID == 0 {
		return response.Fail("班级ID和学生ID不能为空")
	}
	if err := s.bizMapper.InsertClassStudent(req.ClassID, req.StudentID); err != nil {
		log.Printf("班级添加学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "添加失败")
	}
	// 更新班级学生人数统计
	if err := s.bizMapper.UpdateClassStudentCount(req.ClassID); err != nil {
		log.Printf("更新班级学生人数失败: %v", err)
	}
	return response.Success("添加成功")
}

// RemoveStudentFromClass 班级移除学生（对齐 Java removeStudentFromClass）
func (s *AdminBusinessService) RemoveStudentFromClass(req *ClassStudentRequest) *response.ResponseDTO {
	if req.ClassID == 0 || req.StudentID == 0 {
		return response.Fail("班级ID和学生ID不能为空")
	}
	if err := s.bizMapper.DeleteClassStudent(req.ClassID, req.StudentID); err != nil {
		log.Printf("班级移除学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "移除失败")
	}
	// 更新班级学生人数统计
	if err := s.bizMapper.UpdateClassStudentCount(req.ClassID); err != nil {
		log.Printf("更新班级学生人数失败: %v", err)
	}
	return response.Success("移除成功")
}

// ============================================================
// 课表管理
// ============================================================

// ListClassSchedules 课表列表（对齐 Java listClassSchedules）
//
// 注意：本接口不分页，返回所有匹配记录
func (s *AdminBusinessService) ListClassSchedules(req *QueryClassScheduleRequest) *response.ResponseDTO {
	list, err := s.bizMapper.ListClassSchedules(req.ClassID, req.DayOfWeek, req.InstitutionID)
	if err != nil {
		log.Printf("查询课表列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminClassScheduleRow{}
	}
	return response.Success(map[string]interface{}{
		"list":  list,
		"total": int64(len(list)),
	})
}

// UpdateClassSchedule 更新课表（对齐 Java updateClassSchedule）
func (s *AdminBusinessService) UpdateClassSchedule(req *UpdateClassScheduleRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("课表ID不能为空")
	}
	// 备注处理：前端传 null 表示清空，转为 "NULL" 标记
	remark := req.Remark
	if remark == "null" {
		remark = "NULL"
	}
	if err := s.bizMapper.UpdateClassSchedule(req.ID, req.StartDate, req.EndDate, req.DayOfWeek, req.StartTime, req.EndTime, remark); err != nil {
		log.Printf("更新课表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新课表失败")
	}
	// 返回更新后的课表信息
	schedule, err := s.bizMapper.SelectClassScheduleByID(req.ID)
	if err != nil || schedule == nil {
		return response.Success(map[string]interface{}{"id": req.ID})
	}
	return response.Success(schedule)
}

// ============================================================
// 课时记录管理
// ============================================================

// ListCourseRecords 课时记录分页列表（对齐 Java listCourseRecords）
func (s *AdminBusinessService) ListCourseRecords(req *QueryCourseRecordRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	list, total, err := s.bizMapper.ListCourseRecords(req.StudentID, req.CourseID, req.CourseStatus, req.InstitutionID, offset, req.PageSize)
	if err != nil {
		log.Printf("查询课时记录列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminCourseRecordRow{}
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertCourseRecord 新增课时记录（对齐 Java insertCourseRecord）
func (s *AdminBusinessService) InsertCourseRecord(req *InsertCourseRecordRequest) *response.ResponseDTO {
	if req.StudentID == 0 || req.CourseID == 0 {
		return response.Fail("学生ID和课程ID不能为空")
	}
	id, err := s.bizMapper.InsertCourseRecord(req.StudentID, req.CourseID, req.CourseTotalTime, req.CourseRestTime, req.CourseStatus, req.ExpireTime, req.CourseRemark)
	if err != nil {
		log.Printf("新增课时记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增课时记录失败")
	}
	return response.Success(map[string]interface{}{
		"courseRecordId": id,
	})
}

// UpdateCourseRecord 更新课时记录（对齐 Java updateCourseRecord）
func (s *AdminBusinessService) UpdateCourseRecord(req *UpdateCourseRecordRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("记录ID不能为空")
	}
	// 备注处理：前端传 null 表示清空，转为 "NULL" 标记
	remark := req.CourseRemark
	if remark == "null" {
		remark = "NULL"
	}
	// 默认 CourseRestTime=-1 / CourseStatus=-1 表示不更新
	restTime := req.CourseRestTime
	if restTime == 0 {
		restTime = -1
	}
	status := req.CourseStatus
	if status == 0 {
		status = -1
	}
	if err := s.bizMapper.UpdateCourseRecord(req.ID, restTime, -1, status, remark); err != nil {
		log.Printf("更新课时记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新课时记录失败")
	}
	return response.Success(map[string]interface{}{
		"courseRecordId": req.ID,
	})
}

// ============================================================
// 上课记录管理
// ============================================================

// ListRecords 上课记录分页列表（对齐 Java listRecords）
func (s *AdminBusinessService) ListRecords(req *QueryRecordRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	list, total, err := s.bizMapper.ListRecords(req.CourseRecordID, req.RecordType, req.InstitutionID, offset, req.PageSize)
	if err != nil {
		log.Printf("查询上课记录列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminRecordRow{}
	}

	return response.Success(map[string]interface{}{
		"list":  list,
		"total": total,
	})
}

// InsertRecord 新增上课记录（对齐 Java insertRecord）
func (s *AdminBusinessService) InsertRecord(req *InsertRecordRequest) *response.ResponseDTO {
	if req.CourseRecordID == 0 {
		return response.Fail("课卡记录ID不能为空")
	}
	id, err := s.bizMapper.InsertRecord(req.CourseRecordID, req.RecordType, req.RecordChange, req.RecordTime, req.RecordRemark)
	if err != nil {
		log.Printf("新增上课记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增上课记录失败")
	}
	return response.Success(map[string]interface{}{
		"recordId": id,
	})
}

// ============================================================
// 小程序菜单管理
// ============================================================

// ListMiniMenus 查询所有小程序菜单（对齐 Java listMiniMenus）
func (s *AdminBusinessService) ListMiniMenus() *response.ResponseDTO {
	list, err := s.bizMapper.ListMiniMenus()
	if err != nil {
		log.Printf("查询小程序菜单失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.AdminMiniMenuRow{}
	}
	// 注入每个菜单的角色ID列表（从 c_role_menu 表查询）
	for _, menu := range list {
		roleIDs, err := s.bizMapper.SelectRoleIDsByMenuID(menu.ID)
		if err != nil {
			log.Printf("查询菜单角色ID失败: menuID=%d, err=%v", menu.ID, err)
			continue
		}
		menu.RoleIDs = roleIDs
	}
	return response.Success(list)
}

// InsertMiniMenu 新增小程序菜单（对齐 Java insertMiniMenu）
func (s *AdminBusinessService) InsertMiniMenu(req *InsertMiniMenuRequest) *response.ResponseDTO {
	if req.MenuName == "" {
		return response.Fail("菜单名称不能为空")
	}
	// 默认 isVisible=true
	isVisible := true
	if req.IsVisible != nil {
		isVisible = *req.IsVisible
	}
	id, err := s.bizMapper.InsertMiniMenu(req.MenuName, req.Icon, req.IconType, req.BgColor, req.Path, req.SortOrder, isVisible)
	if err != nil {
		log.Printf("新增小程序菜单失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增菜单失败")
	}
	// 保存菜单-角色关联（c_role_menu 表）
	for _, roleID := range req.RoleIDs {
		if err := s.bizMapper.InsertRoleMenu(roleID, id); err != nil {
			log.Printf("新增菜单角色关联失败: roleID=%d, menuID=%d, err=%v", roleID, id, err)
		}
	}
	return response.Success(map[string]interface{}{
		"id":       id,
		"menuName": req.MenuName,
	})
}

// UpdateMiniMenu 更新小程序菜单（对齐 Java updateMiniMenu）
func (s *AdminBusinessService) UpdateMiniMenu(req *UpdateMiniMenuRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("菜单ID不能为空")
	}
	if err := s.bizMapper.UpdateMiniMenu(req.ID, req.MenuName, req.Icon, req.IconType, req.BgColor, req.Path, req.SortOrder, req.IsVisible); err != nil {
		log.Printf("更新小程序菜单失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新菜单失败")
	}
	// 如果提供了 RoleIDs（非 nil），则更新菜单-角色关联
	if req.RoleIDs != nil {
		// 先删除旧关联，再插入新关联（对齐 Java deleteRoleMenuByMenuId + insertRoleMenu）
		if err := s.bizMapper.DeleteRoleMenuByMenuID(req.ID); err != nil {
			log.Printf("删除旧菜单角色关联失败: menuID=%d, err=%v", req.ID, err)
		}
		for _, roleID := range req.RoleIDs {
			if err := s.bizMapper.InsertRoleMenu(roleID, req.ID); err != nil {
				log.Printf("新增菜单角色关联失败: roleID=%d, menuID=%d, err=%v", roleID, req.ID, err)
			}
		}
	}
	return response.Success(map[string]interface{}{
		"id": req.ID,
	})
}

// DeleteMiniMenu 删除小程序菜单（对齐 Java deleteMiniMenu）
func (s *AdminBusinessService) DeleteMiniMenu(id int64) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("菜单ID不能为空")
	}
	// 先删除菜单-角色关联，再删除菜单（避免外键约束失败）
	if err := s.bizMapper.DeleteRoleMenuByMenuID(id); err != nil {
		log.Printf("删除菜单角色关联失败: menuID=%d, err=%v", id, err)
	}
	if err := s.bizMapper.DeleteMiniMenu(id); err != nil {
		log.Printf("删除小程序菜单失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "删除菜单失败")
	}
	return response.Success("删除成功")
}
