// Package handler business-service HTTP 处理层
//
// 对齐 Java business-service/src/main/java/com/shiroko/controller 包
//
// 所有接口路径前缀 /biz（经 Gateway StripPrefix=1 后实际路径为 /{module}/**）
// 公开接口（免 JWT）：institution/get_by_open_id, institution/get_by_institution_code,
//                   course_record/deduct-detail
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/business-service/internal/service"
)

// BusinessHandler 业务 HTTP 处理器
//
// 聚合机构、学生、教师、班级、课表、课程、课卡记录、上课记录等全部模块的 Handler
type BusinessHandler struct {
	institutionService   *service.InstitutionService
	studentService       *service.StudentService
	teacherService       *service.TeacherService
	classService         *service.ClassService
	classScheduleService *service.ClassScheduleService
	courseService        *service.CourseService
	courseRecordService  *service.CourseRecordService
	recordService        *service.RecordService
}

// NewBusinessHandler 创建 BusinessHandler
//
// 参数注入全部 8 个 Service
func NewBusinessHandler(
	institutionService *service.InstitutionService,
	studentService *service.StudentService,
	teacherService *service.TeacherService,
	classService *service.ClassService,
	classScheduleService *service.ClassScheduleService,
	courseService *service.CourseService,
	courseRecordService *service.CourseRecordService,
	recordService *service.RecordService,
) *BusinessHandler {
	return &BusinessHandler{
		institutionService:   institutionService,
		studentService:       studentService,
		teacherService:       teacherService,
		classService:         classService,
		classScheduleService: classScheduleService,
		courseService:        courseService,
		courseRecordService:  courseRecordService,
		recordService:        recordService,
	}
}

// RegisterRoutes 注册全部路由（对齐 Java 各 Controller 的 @RequestMapping）
//
// 路由前缀说明：
//   - Gateway 转发 /biz/** 到 business-service，StripPrefix=1 去除 /biz
//   - 所以 business-service 收到的路径是 /institution/**, /student/**, /teacher/**
//
// 使用 Go 1.22+ 的方法路由模式：POST /path 或 GET /path
func (h *BusinessHandler) RegisterRoutes(mux *http.ServeMux) {
	// ==================== 机构相关（对齐 Java InstitutionController） ====================
	mux.HandleFunc("POST /institution/get_by_id", h.GetInstitutionByID)
	mux.HandleFunc("POST /institution/get_by_open_id", h.GetInstitutionByOpenID)
	mux.HandleFunc("POST /institution/get_by_institution_code", h.GetInstitutionByCode)
	mux.HandleFunc("POST /institution/get_institution_by_student_id", h.GetInstitutionByStudentID)
	mux.HandleFunc("POST /institution/update", h.UpdateInstitution)

	// ==================== 学生相关（对齐 Java StudentController） ====================
	mux.HandleFunc("POST /student/get_by_student_id", h.GetStudentByID)
	mux.HandleFunc("POST /student/get_by_parent_id", h.GetStudentByParentID)
	mux.HandleFunc("POST /student/get_by_teacher_id", h.GetStudentByTeacherID)
	mux.HandleFunc("POST /student/get_by_institution_id", h.GetStudentByInstitutionID)
	mux.HandleFunc("POST /student/get_by_class_id", h.GetStudentByClassID)
	mux.HandleFunc("POST /student/get_by_course_id", h.GetStudentByCourseID)
	mux.HandleFunc("POST /student/insert", h.InsertStudent)
	mux.HandleFunc("POST /student/update", h.UpdateStudent)
	mux.HandleFunc("POST /student/unbind", h.UnbindStudent)
	mux.HandleFunc("POST /student/cancel_subscribe", h.CancelStudentSubscribe)

	// ==================== 教师相关（对齐 Java TeacherController） ====================
	mux.HandleFunc("POST /teacher/get_by_id", h.GetTeacherByID)
	mux.HandleFunc("POST /teacher/get_teacher_by_institution_id", h.GetTeacherByInstitutionID)
	mux.HandleFunc("POST /teacher/insert", h.InsertTeacher)
	mux.HandleFunc("POST /teacher/update_by_id", h.UpdateTeacher)
	mux.HandleFunc("POST /teacher/delete", h.DeleteTeacher)

	// ==================== 班级相关（对齐 Java ClassController） ====================
	mux.HandleFunc("POST /class/get_classes_by_student_id", h.GetClassByStudentID)
	mux.HandleFunc("POST /class/get_classes_by_teacher_id", h.GetClassByTeacherID)
	mux.HandleFunc("POST /class/get_classes_by_institution_id", h.GetClassByInstitutionID)
	mux.HandleFunc("POST /class/get_class_by_id", h.GetClassByID)
	mux.HandleFunc("POST /class/insert", h.InsertClass)
	mux.HandleFunc("POST /class/update_by_id", h.UpdateClass)
	mux.HandleFunc("POST /class/add_student_to_class", h.AddStudentToClass)
	mux.HandleFunc("POST /class/remove_student_from_class", h.RemoveStudentFromClass)

	// ==================== 班级课表相关（对齐 Java ClassScheduleController） ====================
	mux.HandleFunc("POST /class_schedule/get_by_class_id", h.GetClassScheduleByClassID)
	mux.HandleFunc("POST /class_schedule/get_by_institution_id", h.GetClassScheduleByInstitutionID)
	mux.HandleFunc("POST /class_schedule/get_by_teacher_id", h.GetClassScheduleByTeacherID)
	mux.HandleFunc("POST /class_schedule/get_by_id", h.GetClassScheduleByID)
	mux.HandleFunc("POST /class_schedule/update_by_id", h.UpdateClassSchedule)

	// ==================== 课程相关（对齐 Java CourseController） ====================
	mux.HandleFunc("POST /course/get_course_by_institution_id", h.GetCourseByInstitutionID)
	mux.HandleFunc("POST /course/get_course_by_student_id", h.GetCourseByStudentID)
	mux.HandleFunc("POST /course/add_course", h.InsertCourse)
	mux.HandleFunc("POST /course/update_by_id", h.UpdateCourse)

	// ==================== 课卡记录相关（对齐 Java CourseRecordController） ====================
	mux.HandleFunc("POST /course_record/new_get", h.GetCourseRecordList)
	mux.HandleFunc("POST /course_record/get_by_student_id", h.GetCourseRecordByStudentID)
	mux.HandleFunc("POST /course_record/get_by_institution_id", h.GetCourseRecordByInstitutionID)
	mux.HandleFunc("POST /course_record/insert", h.InsertCourseRecord)
	mux.HandleFunc("POST /course_record/update", h.UpdateCourseRecord)
	mux.HandleFunc("POST /course_record/deduct_by_student_id", h.DeductByStudentID)
	mux.HandleFunc("POST /course_record/deduct_by_course_id", h.DeductByCourseID)
	mux.HandleFunc("POST /course_record/deduct_by_class_id", h.DeductByClassID)
	mux.HandleFunc("GET /course_record/deduct-detail", h.GetDeductDetail)

	// ==================== 上课记录相关（对齐 Java RecordController） ====================
	mux.HandleFunc("POST /record/new_get", h.NewGetRecord)
	mux.HandleFunc("POST /record/add", h.InsertRecord)
	mux.HandleFunc("POST /record/add_all", h.InsertRecords)
}

// ============================================================
// 请求/响应辅助
// ============================================================

// readBody 读取请求体并解析 JSON
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

// writeResponse 写入响应
func writeResponse(w http.ResponseWriter, resp *response.ResponseDTO) {
	response.WriteJSON(w, resp)
}

// parseInt64 从字符串解析 int64（用于 GET 查询参数）
func parseInt64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// ============================================================
// 机构相关 Handler
// ============================================================

// GetInstitutionByID 按ID查机构
// POST /institution/get_by_id
func (h *BusinessHandler) GetInstitutionByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64 `json:"institutionId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.institutionService.GetInstitutionByID(req.InstitutionID))
}

// GetInstitutionByOpenID 按openId查机构列表
// POST /institution/get_by_open_id
func (h *BusinessHandler) GetInstitutionByOpenID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OpenID   string `json:"openId"`
		Platform string `json:"platform"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.institutionService.GetInstitutionByOpenID(req.OpenID))
}

// GetInstitutionByCode 按机构编码查
// POST /institution/get_by_institution_code
func (h *BusinessHandler) GetInstitutionByCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionCode string `json:"institutionCode"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.institutionService.GetInstitutionByCode(req.InstitutionCode))
}

// GetInstitutionByStudentID 按学生ID查机构
// POST /institution/get_institution_by_student_id
func (h *BusinessHandler) GetInstitutionByStudentID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int64 `json:"studentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.institutionService.GetInstitutionByStudentID(req.StudentID))
}

// UpdateInstitution 更新机构信息
// POST /institution/update
//
// 前端请求字段：id, institutionName, address, contact, phone
// 对齐 Java InstitutionController.updateInstitution
func (h *BusinessHandler) UpdateInstitution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID               int64  `json:"id"`
		InstitutionName  string `json:"institutionName"`
		Address          string `json:"address"`
		Status           int64  `json:"status"`
		ExpireTime       string `json:"expireTime"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// status 默认 -1 表示不更新；expireTime 默认空字符串表示不更新
	status := req.Status
	if status == 0 {
		status = -1
	}
	writeResponse(w, h.institutionService.UpdateInstitution(req.ID, req.InstitutionName, req.Address, status, req.ExpireTime))
}

// ============================================================
// 学生相关 Handler
// ============================================================

// GetStudentByID 按ID查学生
// POST /student/get_by_student_id
func (h *BusinessHandler) GetStudentByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int64 `json:"studentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.GetStudentByID(req.StudentID))
}

// GetStudentByParentID 按家长ID查学生列表
// POST /student/get_by_parent_id
func (h *BusinessHandler) GetStudentByParentID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentID int64 `json:"parentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.GetStudentByParentID(req.ParentID))
}

// GetStudentByTeacherID 按教师ID查学生列表
// POST /student/get_by_teacher_id
func (h *BusinessHandler) GetStudentByTeacherID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID int64 `json:"teacherId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.GetStudentByTeacherID(req.TeacherID))
}

// GetStudentByInstitutionID 按机构ID查学生列表
// POST /student/get_by_institution_id
func (h *BusinessHandler) GetStudentByInstitutionID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64 `json:"institutionId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.GetStudentByInstitutionID(req.InstitutionID))
}

// GetStudentByClassID 按班级ID查学生列表
// POST /student/get_by_class_id
func (h *BusinessHandler) GetStudentByClassID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID int64 `json:"classId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.GetStudentByClassID(req.ClassID))
}

// GetStudentByCourseID 按课程ID查选修学生列表
// POST /student/get_by_course_id
func (h *BusinessHandler) GetStudentByCourseID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CourseID int64 `json:"courseId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.GetStudentByCourseID(req.CourseID))
}

// InsertStudent 新增学生
// POST /student/insert
//
// 前端请求字段（对齐 InsertStudentRequest）：studentName, institutionId, sex, birth, school, address
// 注：前端还传 primaryParent/secondaryParent，Go 端简化处理仅插入 c_student 表
func (h *BusinessHandler) InsertStudent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentName   string `json:"studentName"`
		InstitutionID int64  `json:"institutionId"`
		Sex           int64  `json:"sex"`
		Birth         string `json:"birth"`
		School        string `json:"school"`
		Address       string `json:"address"`
		Avatar        string `json:"avatar"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.InsertStudent(req.Avatar, req.StudentName, req.InstitutionID, req.Sex, req.Birth, req.School, req.Address))
}

// UpdateStudent 更新学生信息
// POST /student/update
//
// 前端请求字段（对齐 UpdateStudentRequest）：id, avatar, studentName, sex, birthStr, school, address
// 注：前端传 birthStr 字段，映射到 service 的 birth 参数
func (h *BusinessHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          int64  `json:"id"`
		Avatar      string `json:"avatar"`
		StudentName string `json:"studentName"`
		Sex         int64  `json:"sex"`
		BirthStr    string `json:"birthStr"`
		School      string `json:"school"`
		Address     string `json:"address"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// birthStr 映射到 service 的 birth 参数
	writeResponse(w, h.studentService.UpdateStudent(req.ID, req.Avatar, req.StudentName, req.Sex, req.BirthStr, req.School, req.Address))
}

// UnbindStudent 解绑家长-学生关系
// POST /student/unbind
func (h *BusinessHandler) UnbindStudent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentID  int64 `json:"parentId"`
		StudentID int64 `json:"studentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.UnbindStudent(req.ParentID, req.StudentID))
}

// CancelStudentSubscribe 取消家长对学生的微信订阅通知
// POST /student/cancel_subscribe
func (h *BusinessHandler) CancelStudentSubscribe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentID  int64 `json:"parentId"`
		StudentID int64 `json:"studentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.studentService.CancelStudentSubscribe(req.ParentID, req.StudentID))
}

// ============================================================
// 教师相关 Handler
// ============================================================

// GetTeacherByID 按ID查教师
// POST /teacher/get_by_id
func (h *BusinessHandler) GetTeacherByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID int64 `json:"teacherId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherService.GetTeacherByID(req.TeacherID))
}

// GetTeacherByInstitutionID 按机构ID查教师列表
// POST /teacher/get_teacher_by_institution_id
func (h *BusinessHandler) GetTeacherByInstitutionID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64 `json:"institutionId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherService.GetTeacherByInstitutionID(req.InstitutionID))
}

// InsertTeacher 新增教师
// POST /teacher/insert
//
// 前端请求字段（对齐 InsertTeacherRequest）：username, account, password, institutionId, phone
// password 为 SM2 加密的密文，service 层会解密后 SM3 加盐哈希存储
func (h *BusinessHandler) InsertTeacher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username      string `json:"username"`
		Account       string `json:"account"`
		Password      string `json:"password"` // SM2 加密密文
		InstitutionID int64  `json:"institutionId"`
		Phone         string `json:"phone"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherService.InsertTeacher(req.Username, req.Account, req.Password, req.InstitutionID, req.Phone))
}

// UpdateTeacher 更新教师信息
// POST /teacher/update_by_id
//
// 前端请求字段（对齐 UpdateTeacherByIdRequest）：teacherId, username, account, isAvailable, password, phone
func (h *BusinessHandler) UpdateTeacher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID    int64  `json:"teacherId"`
		Username     string `json:"username"`
		Account      string `json:"account"`
		IsAvailable  *bool  `json:"isAvailable"`
		Password     string `json:"password"` // SM2 加密密文（空字符串表示不更新）
		Phone        string `json:"phone"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// isInstitutionAdmin 不由前端直接设置，传 nil 表示不更新
	writeResponse(w, h.teacherService.UpdateTeacher(req.TeacherID, req.Username, req.Phone, req.IsAvailable, nil, req.Account, req.Password))
}

// DeleteTeacher 删除教师
// POST /teacher/delete
func (h *BusinessHandler) DeleteTeacher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID int64 `json:"teacherId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.teacherService.DeleteTeacher(req.TeacherID))
}

// ============================================================
// 班级相关 Handler
// ============================================================

// GetClassByStudentID 按学生ID查班级列表
// POST /class/get_classes_by_student_id
func (h *BusinessHandler) GetClassByStudentID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int64 `json:"studentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.classService.GetClassByStudentID(req.StudentID))
}

// GetClassByTeacherID 按教师ID查班级列表
// POST /class/get_classes_by_teacher_id
func (h *BusinessHandler) GetClassByTeacherID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID   int64  `json:"teacherId"`
		ClassStatus int64  `json:"classStatus"`
		Keyword     string `json:"keyword"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// classStatus 默认 -1 表示不过滤
	status := req.ClassStatus
	if status == 0 {
		status = -1
	}
	writeResponse(w, h.classService.GetClassByTeacherID(req.TeacherID, status, req.Keyword))
}

// GetClassByInstitutionID 按机构ID查班级列表
// POST /class/get_classes_by_institution_id
func (h *BusinessHandler) GetClassByInstitutionID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64  `json:"institutionId"`
		ClassStatus   int64  `json:"classStatus"`
		Keyword       string `json:"keyword"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	status := req.ClassStatus
	if status == 0 {
		status = -1
	}
	writeResponse(w, h.classService.GetClassByInstitutionID(req.InstitutionID, status, req.Keyword))
}

// GetClassByID 按班级ID查班级详情
// POST /class/get_class_by_id
func (h *BusinessHandler) GetClassByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID int64 `json:"classId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.classService.GetClassByID(req.ClassID))
}

// InsertClass 新增班级
// POST /class/insert
//
// 前端请求字段（对齐 InsertClassRequest）：className, courseId, maxCount, schedules, teachers
// schedules 为课表数组，teachers 为教师ID数组（每个项含 teacherId）
func (h *BusinessHandler) InsertClass(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassName string `json:"className"`
		CourseID  int64  `json:"courseId"`
		MaxCount  int64  `json:"maxCount"`
		Schedules []struct {
			DayOfWeek int64  `json:"dayOfWeek"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
			Remark    string `json:"remark"`
		} `json:"schedules"`
		Teachers []struct {
			TeacherID int64 `json:"teacherId"`
		} `json:"teachers"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 转换教师ID列表
	teacherIDs := make([]int64, 0, len(req.Teachers))
	for _, t := range req.Teachers {
		teacherIDs = append(teacherIDs, t.TeacherID)
	}

	// 转换课表项列表
	schedules := make([]*mapper.ScheduleItem, 0, len(req.Schedules))
	for _, s := range req.Schedules {
		schedules = append(schedules, &mapper.ScheduleItem{
			DayOfWeek: s.DayOfWeek,
			StartDate: s.StartDate,
			EndDate:   s.EndDate,
			StartTime: s.StartTime,
			EndTime:   s.EndTime,
			Remark:    s.Remark,
		})
	}

	writeResponse(w, h.classService.InsertClass(req.ClassName, req.CourseID, req.MaxCount, teacherIDs, schedules))
}

// UpdateClass 更新班级信息
// POST /class/update_by_id
//
// 前端请求字段（对齐 UpdateClassRequest）：classId, className, courseId, maxCount, status, schedules, teachers
// onlyUpdateClassOwn=true 时仅更新班级基础信息，schedules/teachers 可为 null
func (h *BusinessHandler) UpdateClass(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID   int64  `json:"classId"`
		ClassName string `json:"className"`
		CourseID  int64  `json:"courseId"`
		MaxCount  int64  `json:"maxCount"`
		Status    int64  `json:"status"`
		Schedules []struct {
			DayOfWeek int64  `json:"dayOfWeek"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
			StartTime string `json:"startTime"`
			EndTime   string `json:"endTime"`
			Remark    string `json:"remark"`
		} `json:"schedules"`
		Teachers []struct {
			TeacherID int64 `json:"teacherId"`
		} `json:"teachers"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 转换教师ID列表（nil 表示不更新教师关联）
	var teacherIDs []int64
	if req.Teachers != nil {
		teacherIDs = make([]int64, 0, len(req.Teachers))
		for _, t := range req.Teachers {
			teacherIDs = append(teacherIDs, t.TeacherID)
		}
	}

	// 转换课表项列表（nil 表示不更新课表）
	var schedules []*mapper.ScheduleItem
	if req.Schedules != nil {
		schedules = make([]*mapper.ScheduleItem, 0, len(req.Schedules))
		for _, s := range req.Schedules {
			schedules = append(schedules, &mapper.ScheduleItem{
				DayOfWeek: s.DayOfWeek,
				StartDate: s.StartDate,
				EndDate:   s.EndDate,
				StartTime: s.StartTime,
				EndTime:   s.EndTime,
				Remark:    s.Remark,
			})
		}
	}

	// status 默认 -1 表示不更新
	status := req.Status
	if status == 0 {
		status = -1
	}

	writeResponse(w, h.classService.UpdateClass(req.ClassID, req.ClassName, req.CourseID, req.MaxCount, status, teacherIDs, schedules))
}

// AddStudentToClass 添加学生到班级
// POST /class/add_student_to_class
//
// 前端请求字段：classId, students（学生对象数组，每个含 id 字段）
func (h *BusinessHandler) AddStudentToClass(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID  int64 `json:"classId"`
		Students []struct {
			ID int64 `json:"id"`
		} `json:"students"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 提取学生ID列表
	studentIDs := make([]int64, 0, len(req.Students))
	for _, s := range req.Students {
		studentIDs = append(studentIDs, s.ID)
	}

	writeResponse(w, h.classService.AddStudentToClass(req.ClassID, studentIDs))
}

// RemoveStudentFromClass 从班级移除学生
// POST /class/remove_student_from_class
func (h *BusinessHandler) RemoveStudentFromClass(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID  int64 `json:"classId"`
		Students []struct {
			ID int64 `json:"id"`
		} `json:"students"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	studentIDs := make([]int64, 0, len(req.Students))
	for _, s := range req.Students {
		studentIDs = append(studentIDs, s.ID)
	}

	writeResponse(w, h.classService.RemoveStudentFromClass(req.ClassID, studentIDs))
}

// ============================================================
// 班级课表相关 Handler
// ============================================================

// GetClassScheduleByClassID 按班级ID查课表列表
// POST /class_schedule/get_by_class_id
func (h *BusinessHandler) GetClassScheduleByClassID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ClassID int64 `json:"classId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.classScheduleService.GetClassScheduleByClassID(req.ClassID))
}

// GetClassScheduleByInstitutionID 按机构ID查课表列表
// POST /class_schedule/get_by_institution_id
func (h *BusinessHandler) GetClassScheduleByInstitutionID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64 `json:"institutionId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.classScheduleService.GetClassScheduleByInstitutionID(req.InstitutionID))
}

// GetClassScheduleByTeacherID 按教师ID查课表列表
// POST /class_schedule/get_by_teacher_id
func (h *BusinessHandler) GetClassScheduleByTeacherID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID int64 `json:"teacherId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.classScheduleService.GetClassScheduleByTeacherID(req.TeacherID))
}

// GetClassScheduleByID 按课表ID查课表详情
// POST /class_schedule/get_by_id
//
// 前端请求字段：scheduleId（对齐 GetClassScheduleByIdRequest）
func (h *BusinessHandler) GetClassScheduleByID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScheduleID int64 `json:"scheduleId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.classScheduleService.GetClassScheduleByID(req.ScheduleID))
}

// UpdateClassSchedule 按ID更新课表
// POST /class_schedule/update_by_id
//
// 前端请求字段（对齐 UpdateClassScheduleRequest）：scheduleId, weekDay, startTime, endTime, remark
func (h *BusinessHandler) UpdateClassSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScheduleID int64  `json:"scheduleId"`
		WeekDay    int64  `json:"weekDay"` // 前端字段名 weekDay，映射到 service 的 dayOfWeek
		StartTime  string `json:"startTime"`
		EndTime    string `json:"endTime"`
		Remark     string `json:"remark"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// weekDay 映射到 dayOfWeek，空字符串表示不更新 startDate/endDate
	writeResponse(w, h.classScheduleService.UpdateClassScheduleByID(req.ScheduleID, req.WeekDay, "", "", req.StartTime, req.EndTime, req.Remark))
}

// ============================================================
// 课程相关 Handler
// ============================================================

// GetCourseByInstitutionID 按机构ID查课程列表
// POST /course/get_course_by_institution_id
func (h *BusinessHandler) GetCourseByInstitutionID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64  `json:"institutionId"`
		Keyword       string `json:"keyword"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.courseService.GetCourseByInstitutionID(req.InstitutionID, req.Keyword))
}

// GetCourseByStudentID 按学生ID查课程列表
// POST /course/get_course_by_student_id
func (h *BusinessHandler) GetCourseByStudentID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int64 `json:"studentId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.courseService.GetCourseByStudentID(req.StudentID))
}

// InsertCourse 新增课程
// POST /course/add_course
//
// 前端请求字段（对齐 InsertCourseRequest）：courseName, courseType, institutionId
func (h *BusinessHandler) InsertCourse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CourseName    string `json:"courseName"`
		CourseType    int64  `json:"courseType"`
		InstitutionID int64  `json:"institutionId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// isAvailable 默认为 true（新建课程默认可用）
	writeResponse(w, h.courseService.InsertCourse(req.CourseName, req.CourseType, req.InstitutionID, true))
}

// UpdateCourse 更新课程信息
// POST /course/update_by_id
//
// 前端请求字段（对齐 UpdateCourseRequest）：id, courseName, courseType, isAvailable
func (h *BusinessHandler) UpdateCourse(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID          int64  `json:"id"`
		CourseName  string `json:"courseName"`
		CourseType  int64  `json:"courseType"`
		IsAvailable *bool  `json:"isAvailable"` // nil 表示不更新
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.courseService.UpdateCourse(req.ID, req.CourseName, req.CourseType, req.IsAvailable))
}

// ============================================================
// 课卡记录相关 Handler
// ============================================================

// GetCourseRecordList 查询课卡记录列表
// POST /course_record/new_get
//
// 前端请求字段（对齐 GetCourseRecordListRequest）：studentId, stuName, courseName, courseRemark, courseStatus, currentPage, pageSize
func (h *BusinessHandler) GetCourseRecordList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID    int64  `json:"studentId"`
		InstitutionID int64 `json:"institutionId"`
		StuName      string `json:"stuName"`
		CourseName   string `json:"courseName"`
		Keyword      string `json:"keyword"`
		ExpireStatus int64  `json:"expireStatus"`
		CurrentPage  int    `json:"currentPage"`
		PageSize     int    `json:"pageSize"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// expireStatus 默认 -1 表示不过滤
	expireStatus := req.ExpireStatus
	if expireStatus == 0 {
		expireStatus = -1
	}
	writeResponse(w, h.courseRecordService.GetCourseRecordList(req.StudentID, req.InstitutionID, req.CourseName, req.StuName, req.Keyword, expireStatus, req.CurrentPage, req.PageSize))
}

// GetCourseRecordByStudentID 按学生ID查课卡记录列表
// POST /course_record/get_by_student_id
func (h *BusinessHandler) GetCourseRecordByStudentID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID    int64  `json:"studentId"`
		CourseName   string `json:"courseName"`
		ExpireStatus int64  `json:"expireStatus"`
		CurrentPage  int    `json:"currentPage"`
		PageSize     int    `json:"pageSize"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	expireStatus := req.ExpireStatus
	if expireStatus == 0 {
		expireStatus = -1
	}
	writeResponse(w, h.courseRecordService.GetCourseRecordByStudentID(req.StudentID, req.CourseName, expireStatus, req.CurrentPage, req.PageSize))
}

// GetCourseRecordByInstitutionID 按机构ID查课卡记录列表
// POST /course_record/get_by_institution_id
func (h *BusinessHandler) GetCourseRecordByInstitutionID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64  `json:"institutionId"`
		Keyword       string `json:"keyword"`
		ExpireStatus  int64  `json:"expireStatus"`
		CurrentPage   int    `json:"currentPage"`
		PageSize      int    `json:"pageSize"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	expireStatus := req.ExpireStatus
	if expireStatus == 0 {
		expireStatus = -1
	}
	writeResponse(w, h.courseRecordService.GetCourseRecordByInstitutionID(req.InstitutionID, req.Keyword, expireStatus, req.CurrentPage, req.PageSize))
}

// InsertCourseRecord 新增课卡记录
// POST /course_record/insert
//
// 前端请求字段（对齐 InsertCourseRecordRequest）：studentId, courseId, expireTime, courseTotalTime, courseRestTime, courseRemark
func (h *BusinessHandler) InsertCourseRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID       int64  `json:"studentId"`
		CourseID        int64  `json:"courseId"`
		ExpireTime      string `json:"expireTime"`
		CourseTotalTime int64  `json:"courseTotalTime"`
		CourseRestTime  int64  `json:"courseRestTime"`
		CourseRemark    string `json:"courseRemark"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// ownerUserID 暂传 0（前端未传该字段，可从 UserContext 获取）
	writeResponse(w, h.courseRecordService.InsertCourseRecord(req.StudentID, req.CourseID, req.CourseTotalTime, req.CourseRestTime, req.ExpireTime, 0, req.CourseRemark))
}

// UpdateCourseRecord 更新课卡记录
// POST /course_record/update
//
// 前端请求字段（对齐 UpdateCourseRecordRequest）：id, courseTotalTime, courseRestTime, expireTime, courseStatus, courseRemark
func (h *BusinessHandler) UpdateCourseRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID              int64  `json:"id"`
		CourseTotalTime int64  `json:"courseTotalTime"`
		CourseRestTime  int64  `json:"courseRestTime"`
		ExpireTime      string `json:"expireTime"`
		CourseStatus    int64  `json:"courseStatus"`
		CourseRemark    string `json:"courseRemark"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	// courseStatus 默认 -1 表示不更新
	status := req.CourseStatus
	if status == 0 {
		status = -1
	}
	writeResponse(w, h.courseRecordService.UpdateCourseRecord(req.ID, req.CourseTotalTime, req.CourseRestTime, status, req.ExpireTime, req.CourseRemark))
}

// DeductByStudentID 按学生ID扣课时
// POST /course_record/deduct_by_student_id
//
// 前端请求字段（对齐 FastDeductRequest mode=student）：recordTime, operatorId, remark, studentId, classes
func (h *BusinessHandler) DeductByStudentID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordTime    string `json:"recordTime"`
		OperatorID    int64  `json:"operatorId"`
		Remark        string `json:"remark"`
		StudentID     int64  `json:"studentId"`
		Classes       []struct {
			ClassID     int64 `json:"classId"`
			CourseID    int64 `json:"courseId"`
			DeductCount int64 `json:"deductCount"`
		} `json:"classes"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 转换班级扣课列表
	classes := make([]*service.DeductClassItem, 0, len(req.Classes))
	for _, c := range req.Classes {
		classes = append(classes, &service.DeductClassItem{
			ClassID:     c.ClassID,
			CourseID:    c.CourseID,
			DeductCount: c.DeductCount,
		})
	}

	writeResponse(w, h.courseRecordService.DeductByStudentID(req.StudentID, classes, req.RecordTime, req.OperatorID, req.Remark))
}

// DeductByCourseID 按课程ID扣课时
// POST /course_record/deduct_by_course_id
//
// 前端请求字段（对齐 FastDeductRequest mode=course）：recordTime, operatorId, remark, courseId, students
func (h *BusinessHandler) DeductByCourseID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordTime string `json:"recordTime"`
		OperatorID int64  `json:"operatorId"`
		Remark     string `json:"remark"`
		CourseID   int64  `json:"courseId"`
		Students   []struct {
			StudentID   int64 `json:"studentId"`
			DeductCount int64 `json:"deductCount"`
		} `json:"students"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 转换学生扣课列表
	students := make([]*service.DeductStudentItem, 0, len(req.Students))
	for _, s := range req.Students {
		students = append(students, &service.DeductStudentItem{
			StudentID:   s.StudentID,
			DeductCount: s.DeductCount,
		})
	}

	writeResponse(w, h.courseRecordService.DeductByCourseID(req.CourseID, students, req.RecordTime, req.OperatorID, req.Remark))
}

// DeductByClassID 按班级ID扣课时
// POST /course_record/deduct_by_class_id
//
// 前端请求字段（对齐 FastDeductRequest mode=class）：recordTime, operatorId, remark, classId, deductCount
func (h *BusinessHandler) DeductByClassID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecordTime  string `json:"recordTime"`
		OperatorID  int64  `json:"operatorId"`
		Remark      string `json:"remark"`
		ClassID     int64  `json:"classId"`
		DeductCount int64  `json:"deductCount"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.courseRecordService.DeductByClassID(req.ClassID, 0, req.DeductCount, req.RecordTime, req.OperatorID, req.Remark))
}

// GetDeductDetail 查询扣费详情
// GET /course_record/deduct-detail?recordId=xxx
//
// 前端使用 GET 方法，通过 query 参数传递 recordId
func (h *BusinessHandler) GetDeductDetail(w http.ResponseWriter, r *http.Request) {
	// 从 query 参数获取 recordId
	recordIDStr := r.URL.Query().Get("recordId")
	recordID := parseInt64(recordIDStr)
	if recordID == 0 {
		writeResponse(w, response.Fail("记录ID不能为空"))
		return
	}
	writeResponse(w, h.courseRecordService.GetDeductDetail(recordID))
}

// ============================================================
// 上课记录相关 Handler
// ============================================================

// NewGetRecord 查询上课记录列表
// POST /record/new_get
//
// 前端请求字段（对齐 GetRecordListRequest）：institutionId, studentId, courseName, recordType, currentPage, pageSize
func (h *BusinessHandler) NewGetRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstitutionID int64  `json:"institutionId"`
		StudentID     int64  `json:"studentId"`
		CourseName    string `json:"courseName"`
		RecordType    int64  `json:"recordType"`
		CurrentPage   int    `json:"currentPage"`
		PageSize      int    `json:"pageSize"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.recordService.NewGetRecord(req.InstitutionID, req.StudentID, 0, req.CourseName, req.RecordType, req.CurrentPage, req.PageSize))
}

// InsertRecord 新增单条上课记录
// POST /record/add
//
// 前端请求字段（对齐 InsertRecordDTO）：courseRecordId, recordTime, recordType, recordChange, recordRemark
func (h *BusinessHandler) InsertRecord(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CourseRecordID int64  `json:"courseRecordId"`
		RecordTime     string `json:"recordTime"`
		RecordType     int64  `json:"recordType"`
		RecordChange   int64  `json:"recordChange"`
		RecordRemark   string `json:"recordRemark"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.recordService.InsertRecord(req.CourseRecordID, req.RecordTime, req.RecordType, req.RecordChange, req.RecordRemark))
}

// InsertRecords 批量新增上课记录
// POST /record/add_all
//
// 前端请求字段（对齐 InsertRecordsDTO）：courseRecordIdList, recordType, recordRemark, recordTime, recordChange
func (h *BusinessHandler) InsertRecords(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CourseRecordIDList []int64 `json:"courseRecordIdList"`
		RecordType         int64   `json:"recordType"`
		RecordRemark       string  `json:"recordRemark"`
		RecordTime         string  `json:"recordTime"`
		RecordChange       int64   `json:"recordChange"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.recordService.InsertRecords(req.CourseRecordIDList, req.RecordTime, req.RecordType, req.RecordChange, req.RecordRemark))
}
