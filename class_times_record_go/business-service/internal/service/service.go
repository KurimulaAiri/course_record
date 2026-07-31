// Package service business-service 业务逻辑层
//
// 对齐 Java business-service/src/main/java/com/shiroko/service/impl 包
//
// 核心功能：
//   - 机构查询（按ID/openId/编码/学生ID）
//   - 学生查询（按ID/家长ID/教师ID/机构ID）
//   - 教师查询（按ID/机构ID）
//   - 后续：班级/课程/课时记录/签到扣课等
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// InstitutionService 机构服务
// ============================================================

// InstitutionService 机构服务（对齐 Java InstitutionServiceImpl）
type InstitutionService struct {
	institutionMapper *mapper.InstitutionMapper
}

// NewInstitutionService 创建 InstitutionService
func NewInstitutionService(institutionMapper *mapper.InstitutionMapper) *InstitutionService {
	return &InstitutionService{institutionMapper: institutionMapper}
}

// GetInstitutionByID 按ID查机构
//
// 对齐 Java InstitutionServiceImpl.getInstitutionById
func (s *InstitutionService) GetInstitutionByID(id int64) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("机构ID不能为空")
	}

	inst, err := s.institutionMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if inst == nil {
		return response.Fail("机构不存在")
	}

	return response.Success(inst)
}

// GetInstitutionByOpenID 按openId查机构列表
//
// 对齐 Java InstitutionServiceImpl.getInstitutionByOpenId
// 用途：家长/教师登录后，根据 openId 查询其关联的所有机构
func (s *InstitutionService) GetInstitutionByOpenID(openID string) *response.ResponseDTO {
	if openID == "" {
		return response.Fail("openId 不能为空")
	}

	list, err := s.institutionMapper.SelectByOpenID(openID)
	if err != nil {
		log.Printf("查询机构列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success(list)
}

// GetInstitutionByCode 按机构编码查机构
//
// 对齐 Java InstitutionServiceImpl.getInstitutionByCode
func (s *InstitutionService) GetInstitutionByCode(code string) *response.ResponseDTO {
	if code == "" {
		return response.Fail("机构编码不能为空")
	}

	inst, err := s.institutionMapper.SelectByCode(code)
	if err != nil {
		log.Printf("查询机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if inst == nil {
		return response.Fail("机构不存在")
	}

	return response.Success(inst)
}

// GetInstitutionByStudentID 按学生ID查机构
//
// 对齐 Java InstitutionServiceImpl.getInstitutionByStudentId
func (s *InstitutionService) GetInstitutionByStudentID(studentID int64) *response.ResponseDTO {
	if studentID == 0 {
		return response.Fail("学生ID不能为空")
	}

	inst, err := s.institutionMapper.SelectByStudentID(studentID)
	if err != nil {
		log.Printf("查询机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if inst == nil {
		return response.Fail("机构不存在")
	}

	return response.Success(inst)
}

// ============================================================
// StudentService 学生服务
// ============================================================

// StudentService 学生服务（对齐 Java StudentServiceImpl）
type StudentService struct {
	studentMapper *mapper.StudentMapper
}

// NewStudentService 创建 StudentService
func NewStudentService(studentMapper *mapper.StudentMapper) *StudentService {
	return &StudentService{studentMapper: studentMapper}
}

// GetStudentByID 按ID查学生
func (s *StudentService) GetStudentByID(id int64) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("学生ID不能为空")
	}

	student, err := s.studentMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if student == nil {
		return response.Fail("学生不存在")
	}

	return response.Success(student)
}

// GetStudentByParentID 按家长ID查学生列表
func (s *StudentService) GetStudentByParentID(parentID int64) *response.ResponseDTO {
	if parentID == 0 {
		return response.Fail("家长ID不能为空")
	}

	list, err := s.studentMapper.SelectByParentID(parentID)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success(list)
}

// GetStudentByTeacherID 按教师ID查学生列表
func (s *StudentService) GetStudentByTeacherID(teacherID int64) *response.ResponseDTO {
	if teacherID == 0 {
		return response.Fail("教师ID不能为空")
	}

	list, err := s.studentMapper.SelectByTeacherID(teacherID)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success(list)
}

// GetStudentByInstitutionID 按机构ID查学生列表
func (s *StudentService) GetStudentByInstitutionID(institutionID int64) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.studentMapper.SelectByInstitutionID(institutionID)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success(list)
}

// ============================================================
// TeacherService 教师服务
// ============================================================

// TeacherService 教师服务（对齐 Java TeacherServiceImpl）
type TeacherService struct {
	teacherMapper *mapper.TeacherMapper
}

// NewTeacherService 创建 TeacherService
func NewTeacherService(teacherMapper *mapper.TeacherMapper) *TeacherService {
	return &TeacherService{teacherMapper: teacherMapper}
}

// GetTeacherByID 按ID查教师
func (s *TeacherService) GetTeacherByID(id int64) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("教师ID不能为空")
	}

	teacher, err := s.teacherMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if teacher == nil {
		return response.Fail("教师不存在")
	}

	return response.Success(teacher)
}

// GetTeacherByInstitutionID 按机构ID查教师列表
func (s *TeacherService) GetTeacherByInstitutionID(institutionID int64) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.teacherMapper.SelectByInstitutionID(institutionID)
	if err != nil {
		log.Printf("查询教师列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success(list)
}

// ============================================================
// 辅助：实体转 VO（对齐 Java MapStruct Converter）
// ============================================================

// InstitutionVO 机构视图对象（对齐 Java QueryInstitutionVO）
type InstitutionVO struct {
	ID                   int64  `json:"id"`                   // 机构ID
	InstitutionName      string `json:"institutionName"`      // 机构名称
	InstitutionAddress   string `json:"institutionAddress"`   // 机构地址
	InstitutionCode      string `json:"institutionCode"`      // 机构编码
	Status               int64  `json:"status"`               // 状态
	ExpireTime           string `json:"expireTime"`           // 过期时间
	SubscriptionPlanID   int64  `json:"subscriptionPlanId"`   // 订阅计划ID
	SubscriptionPlanName string `json:"subscriptionPlanName"` // 订阅计划名
}

// ToInstitutionVO 实体转 VO（对齐 Java InstitutionConverter）
func ToInstitutionVO(inst *entity.Institution) *InstitutionVO {
	if inst == nil {
		return nil
	}
	vo := &InstitutionVO{}
	if inst.ID.Valid {
		vo.ID = inst.ID.Int64
	}
	vo.InstitutionName = inst.InstitutionName.String
	vo.InstitutionAddress = inst.InstitutionAddress.String
	vo.InstitutionCode = inst.InstitutionCode.String
	if inst.Status.Valid {
		vo.Status = inst.Status.Int64
	}
	vo.ExpireTime = entity.FormatTime(inst.ExpireTime)
	if inst.SubscriptionPlanID.Valid {
		vo.SubscriptionPlanID = inst.SubscriptionPlanID.Int64
	}
	vo.SubscriptionPlanName = inst.SubscriptionPlanName.String
	return vo
}
