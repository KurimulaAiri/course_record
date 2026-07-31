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

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/service"
)

// BusinessHandler 业务 HTTP 处理器
//
// 聚合机构、学生、教师等 Handler
type BusinessHandler struct {
	institutionService *service.InstitutionService
	studentService     *service.StudentService
	teacherService     *service.TeacherService
}

// NewBusinessHandler 创建 BusinessHandler
func NewBusinessHandler(
	institutionService *service.InstitutionService,
	studentService *service.StudentService,
	teacherService *service.TeacherService,
) *BusinessHandler {
	return &BusinessHandler{
		institutionService: institutionService,
		studentService:     studentService,
		teacherService:     teacherService,
	}
}

// RegisterRoutes 注册路由（对齐 Java 各 Controller 的 @RequestMapping）
//
// 路由前缀说明：
//   - Gateway 转发 /biz/** 到 business-service，StripPrefix=1 去除 /biz
//   - 所以 business-service 收到的路径是 /institution/**, /student/**, /teacher/**
func (h *BusinessHandler) RegisterRoutes(mux *http.ServeMux) {
	// 机构相关（对齐 Java InstitutionController @RequestMapping("/institution")）
	mux.HandleFunc("/institution/get_by_id", h.GetInstitutionByID)
	mux.HandleFunc("/institution/get_by_open_id", h.GetInstitutionByOpenID)
	mux.HandleFunc("/institution/get_by_institution_code", h.GetInstitutionByCode)
	mux.HandleFunc("/institution/get_institution_by_student_id", h.GetInstitutionByStudentID)

	// 学生相关（对齐 Java StudentController @RequestMapping("/student")）
	mux.HandleFunc("/student/get_by_student_id", h.GetStudentByID)
	mux.HandleFunc("/student/get_by_parent_id", h.GetStudentByParentID)
	mux.HandleFunc("/student/get_by_teacher_id", h.GetStudentByTeacherID)
	mux.HandleFunc("/student/get_by_institution_id", h.GetStudentByInstitutionID)

	// 教师相关（对齐 Java TeacherController @RequestMapping("/teacher")）
	mux.HandleFunc("/teacher/get_by_id", h.GetTeacherByID)
	mux.HandleFunc("/teacher/get_teacher_by_institution_id", h.GetTeacherByInstitutionID)
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
