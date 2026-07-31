// Package service business-service 业务逻辑层
//
// 对齐 Java business-service/src/main/java/com/shiroko/service/impl 包
//
// 核心功能：
//   - 机构查询（按ID/openId/编码/学生ID）+ 更新
//   - 学生查询（按ID/家长ID/教师ID/机构ID/班级ID/课程ID）+ 增改/解绑/取消订阅
//   - 教师查询（按ID/机构ID）+ 增删改
//   - 班级/课表/课程/课时记录/上课记录等模块（独立 service 文件）
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/crypto"
	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// VO 包装结构（对齐 Java QueryXxxVO）
//
// 前端期望的响应结构：
//   - Institution: data.institutions（数组）
//   - Student:     data.list（数组）+ data.total
//   - Teacher:     data.teachers（数组）+ data.total
//
// 即使查询单个实体（如按ID查），也包装为单元素数组返回
// ============================================================

// QueryInstitutionVO 机构查询响应包装（对齐 Java QueryInstitutionVO）
//
// 前端类型定义（src/types/institution.d.ts）：
//
//	interface GetInstitutionByOpenIdResponse { institutions: InstitutionResponse[] }
type QueryInstitutionVO struct {
	Institutions []*InstitutionVO `json:"institutions"` // 机构列表
}

// QueryStudentVO 学生查询响应包装（对齐 Java QueryStudentVO）
//
// 前端类型定义（src/types/student.d.ts）：
//
//	interface StudentListResponse { list: StudentResponse[]; total: number }
type QueryStudentVO struct {
	List  []*StudentVO `json:"list"`  // 学生列表
	Total int64        `json:"total"` // 总数
}

// QueryTeacherVO 教师查询响应包装（对齐 Java QueryTeacherVO）
//
// 前端类型定义（src/types/teacher.d.ts）：
//
//	interface TeacherListResponse { teachers: TeacherResponse[]; total: number }
type QueryTeacherVO struct {
	Teachers []*TeacherVO `json:"teachers"` // 教师列表
	Total    int64        `json:"total"`    // 总数
}

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
//
// 前端期望：data.institutions[0]（单元素数组）
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

	// 包装为 institutions 数组返回（对齐前端 GetInstitutionByIdResponse）
	return response.Success(&QueryInstitutionVO{
		Institutions: []*InstitutionVO{ToInstitutionVO(inst)},
	})
}

// GetInstitutionByOpenID 按openId查机构列表
//
// 对齐 Java InstitutionServiceImpl.getInstitutionByOpenId
// 用途：家长/教师登录后，根据 openId 查询其关联的所有机构
//
// 前端期望：data.institutions（数组）
func (s *InstitutionService) GetInstitutionByOpenID(openID string) *response.ResponseDTO {
	if openID == "" {
		return response.Fail("openId 不能为空")
	}

	list, err := s.institutionMapper.SelectByOpenID(openID)
	if err != nil {
		log.Printf("查询机构列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 转为 VO 列表
	voList := make([]*InstitutionVO, 0, len(list))
	for _, inst := range list {
		if vo := ToInstitutionVO(inst); vo != nil {
			voList = append(voList, vo)
		}
	}

	// 包装为 institutions 字段返回（对齐前端 GetInstitutionByOpenIdResponse）
	return response.Success(&QueryInstitutionVO{
		Institutions: voList,
	})
}

// GetInstitutionByCode 按机构编码查机构
//
// 对齐 Java InstitutionServiceImpl.getInstitutionByCode
//
// 前端期望：data.institutions[0]（单元素数组）
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

	// 包装为 institutions 数组返回（对齐前端 GetInstitutionByCodeResponse）
	return response.Success(&QueryInstitutionVO{
		Institutions: []*InstitutionVO{ToInstitutionVO(inst)},
	})
}

// GetInstitutionByStudentID 按学生ID查机构
//
// 对齐 Java InstitutionServiceImpl.getInstitutionByStudentId
//
// 前端期望：data.institutions[0]（单元素数组）
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

	// 包装为 institutions 数组返回
	return response.Success(&QueryInstitutionVO{
		Institutions: []*InstitutionVO{ToInstitutionVO(inst)},
	})
}

// UpdateInstitution 更新机构信息
//
// 对齐 Java InstitutionServiceImpl.updateInstitution
//
// 前端期望：data.result（影响行数，非零表示成功）
//
// 参数：
//   - id: 机构ID
//   - name: 机构名称（空字符串表示不更新）
//   - address: 机构地址（空字符串表示不更新）
//   - status: 状态（-1 表示不更新）
//   - expireTime: 过期时间（空字符串表示不更新，"NULL" 表示设为 NULL）
func (s *InstitutionService) UpdateInstitution(id int64, name, address string, status int64, expireTime string) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("机构ID不能为空")
	}

	rows, err := s.institutionMapper.UpdateByID(id, name, address, status, expireTime)
	if err != nil {
		log.Printf("更新机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回影响行数（对齐前端 UpdateInstitutionResponse.result）
	return response.Success(&UpdateResultVO{Result: rows})
}

// UpdateResultVO 通用更新结果 VO（对齐前端 *Response.result 字段）
type UpdateResultVO struct {
	Result int64 `json:"result"` // 影响行数或操作结果
}

// ============================================================
// StudentService 学生服务
// ============================================================

// StudentService 学生服务（对齐 Java StudentServiceImpl）
//
// 查询：按ID/家长ID/教师ID/机构ID/班级ID/课程ID
// 写操作：新增/更新/解绑家长/取消订阅
type StudentService struct {
	studentMapper           *mapper.StudentMapper
	parentStudentMapper     *mapper.ParentStudentMapper
	parentMapper            *mapper.ParentMapper
	wxStudentSubscribeMapper *mapper.WxStudentSubscribeMapper
	wxSubscribeRecordMapper *mapper.WxSubscribeRecordMapper
	userPlatformMapper      *mapper.UserPlatformMapper
}

// NewStudentService 创建 StudentService
//
// 参数注入写操作所需的关联 Mapper（解绑、取消订阅流程需要）
func NewStudentService(
	studentMapper *mapper.StudentMapper,
	parentStudentMapper *mapper.ParentStudentMapper,
	parentMapper *mapper.ParentMapper,
	wxStudentSubscribeMapper *mapper.WxStudentSubscribeMapper,
	wxSubscribeRecordMapper *mapper.WxSubscribeRecordMapper,
	userPlatformMapper *mapper.UserPlatformMapper,
) *StudentService {
	return &StudentService{
		studentMapper:           studentMapper,
		parentStudentMapper:     parentStudentMapper,
		parentMapper:            parentMapper,
		wxStudentSubscribeMapper: wxStudentSubscribeMapper,
		wxSubscribeRecordMapper: wxSubscribeRecordMapper,
		userPlatformMapper:      userPlatformMapper,
	}
}

// GetStudentByID 按ID查学生
//
// 前端期望：data.list[0]（单元素数组）
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

	// 包装为 list 数组返回（对齐前端 StudentListResponse）
	voList := []*StudentVO{ToStudentVO(student)}
	return response.Success(&QueryStudentVO{
		List:  voList,
		Total: int64(len(voList)),
	})
}

// GetStudentByParentID 按家长ID查学生列表
//
// 前端期望：data.list（数组）+ data.total
func (s *StudentService) GetStudentByParentID(parentID int64) *response.ResponseDTO {
	if parentID == 0 {
		return response.Fail("家长ID不能为空")
	}

	list, err := s.studentMapper.SelectByParentID(parentID)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToStudentVOList(list)
	return response.Success(&QueryStudentVO{
		List:  voList,
		Total: int64(len(voList)),
	})
}

// GetStudentByTeacherID 按教师ID查学生列表
//
// 前端期望：data.list（数组）+ data.total
func (s *StudentService) GetStudentByTeacherID(teacherID int64) *response.ResponseDTO {
	if teacherID == 0 {
		return response.Fail("教师ID不能为空")
	}

	list, err := s.studentMapper.SelectByTeacherID(teacherID)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToStudentVOList(list)
	return response.Success(&QueryStudentVO{
		List:  voList,
		Total: int64(len(voList)),
	})
}

// GetStudentByInstitutionID 按机构ID查学生列表
//
// 前端期望：data.list（数组）+ data.total
func (s *StudentService) GetStudentByInstitutionID(institutionID int64) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.studentMapper.SelectByInstitutionID(institutionID)
	if err != nil {
		log.Printf("查询学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToStudentVOList(list)
	return response.Success(&QueryStudentVO{
		List:  voList,
		Total: int64(len(voList)),
	})
}

// GetStudentByClassID 按班级ID查学生列表
//
// 对齐 Java StudentServiceImpl.getStudentByClassId
// 前端期望：data.list（数组）+ data.total
func (s *StudentService) GetStudentByClassID(classID int64) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}

	list, err := s.studentMapper.SelectByClassID(classID)
	if err != nil {
		log.Printf("查询班级学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToStudentVOList(list)
	return response.Success(&QueryStudentVO{
		List:  voList,
		Total: int64(len(voList)),
	})
}

// GetStudentByCourseID 按课程ID查选修学生列表
//
// 对齐 Java StudentServiceImpl.getStudentByCourseId
// 前端期望：data.list（数组）+ data.total
func (s *StudentService) GetStudentByCourseID(courseID int64) *response.ResponseDTO {
	if courseID == 0 {
		return response.Fail("课程ID不能为空")
	}

	list, err := s.studentMapper.SelectByCourseID(courseID)
	if err != nil {
		log.Printf("查询课程学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToStudentVOList(list)
	return response.Success(&QueryStudentVO{
		List:  voList,
		Total: int64(len(voList)),
	})
}

// InsertStudent 新增学生
//
// 对齐 Java StudentServiceImpl.insertStudent（简化版：仅插入 c_student 表，不处理家长关联）
//
// 前端期望：data.studentId（新学生ID）
//
// 参数：
//   - avatar: 头像URL
//   - studentName: 学生姓名
//   - institutionID: 机构ID
//   - sex: 性别（0=未知,1=男,2=女）
//   - birth: 出生日期（空字符串表示 NULL）
//   - school: 学校
//   - address: 地址
func (s *StudentService) InsertStudent(avatar, studentName string, institutionID, sex int64, birth, school, address string) *response.ResponseDTO {
	if studentName == "" {
		return response.Fail("学生姓名不能为空")
	}
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	id, err := s.studentMapper.Insert(avatar, studentName, institutionID, sex, birth, school, address)
	if err != nil {
		log.Printf("新增学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回新学生ID（对齐前端 InsertStudentResponse.studentId）
	return response.Success(&InsertStudentVO{StudentID: id})
}

// InsertStudentVO 新增学生响应 VO（对齐前端 InsertStudentResponse）
type InsertStudentVO struct {
	StudentID int64 `json:"studentId"` // 新学生ID
}

// UpdateStudent 更新学生信息
//
// 对齐 Java StudentServiceImpl.updateStudent（简化版：仅更新 c_student 表）
//
// 前端期望：data.studentId（被更新的学生ID）
//
// 参数：
//   - id: 学生ID
//   - avatar: 头像URL（空字符串表示不更新）
//   - studentName: 学生姓名（空字符串表示不更新）
//   - sex: 性别（-1 表示不更新）
//   - birth: 出生日期（空字符串表示不更新，"NULL" 表示设为 NULL）
//   - school: 学校（空字符串表示不更新）
//   - address: 地址（空字符串表示不更新）
func (s *StudentService) UpdateStudent(id int64, avatar, studentName string, sex int64, birth, school, address string) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("学生ID不能为空")
	}

	_, err := s.studentMapper.UpdateByID(id, avatar, studentName, sex, birth, school, address)
	if err != nil {
		log.Printf("更新学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回学生ID（对齐前端 UpdateStudentResponse.studentId）
	return response.Success(&UpdateStudentVO{StudentID: id})
}

// UpdateStudentVO 更新学生响应 VO（对齐前端 UpdateStudentResponse）
type UpdateStudentVO struct {
	StudentID int64 `json:"studentId"` // 被更新的学生ID
}

// UnbindStudent 解绑家长-学生关系
//
// 对齐 Java StudentServiceImpl.unbindStudent
//
// 流程：
//  1. 查询 parent_student 关联记录，获取 isPrimary 定位联系人角色
//  2. 删除 wx_student_subscribe 中该 (studentId, isPrimary) 的订阅记录
//  3. 删除 parent_student 关联记录
//  4. 判断是否需要删除/重置 parent 记录本身
//
// 前端期望：data 为字符串消息
func (s *StudentService) UnbindStudent(parentID, studentID int64) *response.ResponseDTO {
	if parentID == 0 || studentID == 0 {
		return response.Fail("参数不能为空")
	}

	// 1. 查询关联记录，获取 isPrimary
	isPrimary, err := s.parentStudentMapper.SelectByParentAndStudent(parentID, studentID)
	if err != nil {
		log.Printf("查询家长学生关联失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	// SelectByParentAndStudent 返回 false 可能是未找到记录或 isPrimary=false，需区分
	// 这里简化处理：如果删除关联时影响行数为0，则认为未找到
	// 先尝试删除 wx_student_subscribe（按 studentId + isPrimary）
	_, err = s.wxStudentSubscribeMapper.DeleteByStudentAndIsPrimary(studentID, isPrimary)
	if err != nil {
		log.Printf("清理订阅关系失败: %v", err)
		// 不阻塞主流程，继续删除关联
	}

	// 2. 删除 parent_student 关联记录
	rows, err := s.parentStudentMapper.DeleteByParentAndStudent(parentID, studentID)
	if err != nil {
		log.Printf("删除家长学生关联失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if rows == 0 {
		return response.Fail("未找到绑定关系")
	}

	// 3. 判断是否需要删除/重置 parent 记录
	parent, err := s.parentMapper.SelectByID(parentID)
	if err != nil {
		log.Printf("查询家长失败: %v", err)
	} else if parent != nil {
		// 未绑定账号的占位 parent（is_bound=false 或 userId=NULL），直接删除
		isPlaceholder := !parent.IsBound.Bool || !parent.UserID.Valid
		if isPlaceholder {
			_, _ = s.parentMapper.DeleteByID(parentID)
		} else {
			// 已绑定的 parent，检查是否还有其他学生关联
			remaining, _ := s.parentStudentMapper.CountByParentID(parentID)
			if remaining == 0 {
				// 已无其他学生关联，重置为未绑定状态
				_, _ = s.parentMapper.ResetUnbound(parentID)
			}
		}
	}

	return response.Success("解绑成功")
}

// CancelStudentSubscribe 取消家长对学生的微信订阅通知
//
// 对齐 Java StudentServiceImpl.cancelStudentSubscribe
//
// 流程：
//  1. 查询 parent_student 关联记录，获取 isPrimary 定位联系人角色
//  2. 删除 wx_student_subscribe 中该 (studentId, isPrimary) 的订阅记录
//  3. 删除 wx_subscribe_record 中该家长所有 openId 的授权次数记录
//
// 前端期望：data 为字符串消息
func (s *StudentService) CancelStudentSubscribe(parentID, studentID int64) *response.ResponseDTO {
	if parentID == 0 || studentID == 0 {
		return response.Fail("参数不能为空")
	}

	// 1. 查询关联记录，获取 isPrimary
	isPrimary, err := s.parentStudentMapper.SelectByParentAndStudent(parentID, studentID)
	if err != nil {
		log.Printf("查询家长学生关联失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 删除 wx_student_subscribe 中该 (studentId, isPrimary) 的订阅记录
	_, err = s.wxStudentSubscribeMapper.DeleteByStudentAndIsPrimary(studentID, isPrimary)
	if err != nil {
		log.Printf("清理订阅关系失败: %v", err)
	}

	// 3. 删除 wx_subscribe_record 中该家长关联的所有 openId 的授权次数记录
	//    先通过 parent 的 userId 找到所有 openId
	parent, err := s.parentMapper.SelectByID(parentID)
	if err != nil {
		log.Printf("查询家长失败: %v", err)
		return response.Success("取消订阅成功")
	}
	if parent == nil || !parent.UserID.Valid {
		// 占位 parent 无 userId，无 openId 可清理
		return response.Success("取消订阅成功")
	}

	openIDs, err := s.userPlatformMapper.SelectOpenIDsByUserID(parent.UserID.Int64)
	if err != nil {
		log.Printf("查询家长 openId 失败: %v", err)
		return response.Success("取消订阅成功")
	}
	if len(openIDs) > 0 {
		_, _ = s.wxSubscribeRecordMapper.DeleteByOpenIDs(openIDs)
	}

	return response.Success("取消订阅成功")
}

// ============================================================
// TeacherService 教师服务
// ============================================================

// TeacherService 教师服务（对齐 Java TeacherServiceImpl）
//
// 查询：按ID/机构ID
// 写操作：新增/更新/删除（涉及 c_user、c_user_auth、c_teacher 多表）
type TeacherService struct {
	teacherMapper     *mapper.TeacherMapper
	userMapper        *mapper.UserMapper
	userAuthMapper    *mapper.UserAuthMapper
	classTeacherMapper *mapper.ClassTeacherMapper
	sm2PrivateKey     string // SM2 私钥（hex），用于解密前端传入的密码密文
}

// NewTeacherService 创建 TeacherService
//
// 参数：
//   - teacherMapper: 教师表 Mapper
//   - userMapper: 用户表 Mapper（新增/删除教师时维护 c_user）
//   - userAuthMapper: 用户认证表 Mapper（新增/删除/更新教师时维护 c_user_auth）
//   - classTeacherMapper: 班级-教师关联 Mapper（删除教师前校验是否关联班级）
//   - sm2PrivateKey: SM2 私钥 hex（来自 Nacos cr-auth-service.yaml），用于解密前端密码密文
func NewTeacherService(
	teacherMapper *mapper.TeacherMapper,
	userMapper *mapper.UserMapper,
	userAuthMapper *mapper.UserAuthMapper,
	classTeacherMapper *mapper.ClassTeacherMapper,
	sm2PrivateKey string,
) *TeacherService {
	return &TeacherService{
		teacherMapper:      teacherMapper,
		userMapper:         userMapper,
		userAuthMapper:     userAuthMapper,
		classTeacherMapper: classTeacherMapper,
		sm2PrivateKey:      sm2PrivateKey,
	}
}

// GetTeacherByID 按ID查教师
//
// 前端期望：data.teachers[0]（单元素数组）
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

	// 包装为 teachers 数组返回（对齐前端 TeacherListResponse）
	voList := []*TeacherVO{ToTeacherVO(teacher)}
	return response.Success(&QueryTeacherVO{
		Teachers: voList,
		Total:    int64(len(voList)),
	})
}

// GetTeacherByInstitutionID 按机构ID查教师列表
//
// 前端期望：data.teachers（数组）+ data.total
func (s *TeacherService) GetTeacherByInstitutionID(institutionID int64) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.teacherMapper.SelectByInstitutionID(institutionID)
	if err != nil {
		log.Printf("查询教师列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToTeacherVOList(list)
	return response.Success(&QueryTeacherVO{
		Teachers: voList,
		Total:    int64(len(voList)),
	})
}

// InsertTeacher 新增教师
//
// 对齐 Java TeacherServiceImpl.insertTeacher
//
// 流程：
//  1. 创建 c_user 记录（教师所属机构的用户记录）
//  2. 创建 c_user_auth 记录（账号密码，roleId=4 教师）
//     - 密码使用 SM2 解密前端密文，再 SM3 加盐哈希存储
//  3. 创建 c_teacher 记录（关联 userId）
//
// 前端期望：data.teacherId（新教师ID）
//
// 参数：
//   - username: 教师用户名
//   - account: 登录账号（手机号）
//   - passwordCipher: SM2 加密的密码密文（hex）
//   - institutionID: 机构ID
//   - phone: 手机号
func (s *TeacherService) InsertTeacher(username, account, passwordCipher string, institutionID int64, phone string) *response.ResponseDTO {
	if username == "" {
		return response.Fail("教师姓名不能为空")
	}
	if account == "" {
		return response.Fail("登录账号不能为空")
	}
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	// 1. SM2 解密密码密文
	passwordPlain, err := crypto.SM2Decrypt(passwordCipher, s.sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密密码失败: %v", err)
		return response.Fail("密码解密失败")
	}

	// 2. 生成盐值并计算 SM3 加盐哈希
	salt := crypto.GenerateSalt()
	hashedPassword := crypto.SM3DigestWithSalt(passwordPlain, salt)

	// 3. 创建 c_user 记录
	userID, err := s.userMapper.Insert(institutionID)
	if err != nil {
		log.Printf("创建用户记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 4. 创建 c_user_auth 记录（roleId=4 教师）
	_, err = s.userAuthMapper.Insert(userID, 4, account, hashedPassword, salt)
	if err != nil {
		log.Printf("创建用户认证记录失败: %v", err)
		// 回滚 c_user 记录
		_, _ = s.userMapper.DeleteByID(userID)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 5. 创建 c_teacher 记录
	teacherID, err := s.teacherMapper.Insert(userID, username, institutionID, phone)
	if err != nil {
		log.Printf("创建教师记录失败: %v", err)
		// 回滚 c_user_auth 和 c_user 记录
		_, _ = s.userAuthMapper.DeleteByID(userID)
		_, _ = s.userMapper.DeleteByID(userID)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回新教师ID（对齐前端 InsertTeacherResponse）
	return response.Success(&InsertTeacherVO{TeacherID: teacherID})
}

// InsertTeacherVO 新增教师响应 VO（对齐前端 InsertTeacherResponse）
type InsertTeacherVO struct {
	TeacherID int64 `json:"teacherId"` // 新教师ID
}

// UpdateTeacher 更新教师信息
//
// 对齐 Java TeacherServiceImpl.updateTeacher
//
// 流程：
//  1. 更新 c_teacher 表（username/phone/isAvailable/isInstitutionAdmin）
//  2. 如果提供了 account 或 password，更新 c_user_auth 表
//     - password 使用 SM2 解密前端密文，再 SM3 加盐哈希存储
//
// 前端期望：data.teacherId（被更新的教师ID）
//
// 参数：
//   - id: 教师ID
//   - username: 教师姓名（空字符串表示不更新）
//   - phone: 手机号（空字符串表示不更新）
//   - isAvailable: 是否可用（nil 表示不更新）
//   - isInstitutionAdmin: 是否机构管理员（nil 表示不更新）
//   - account: 登录账号（空字符串表示不更新）
//   - passwordCipher: SM2 加密的新密码密文（空字符串表示不更新）
func (s *TeacherService) UpdateTeacher(id int64, username, phone string, isAvailable *bool, isInstitutionAdmin *bool, account, passwordCipher string) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("教师ID不能为空")
	}

	// 1. 更新 c_teacher 表
	_, err := s.teacherMapper.UpdateByID(id, username, phone, isAvailable, isInstitutionAdmin)
	if err != nil {
		log.Printf("更新教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 如果需要更新账号或密码，先查教师关联的 userId 和 user_auth 记录
	if account != "" || passwordCipher != "" {
		// 查教师记录获取 userId
		teacher, err := s.teacherMapper.SelectByID(id)
		if err != nil {
			log.Printf("查询教师失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "系统异常")
		}
		if teacher == nil || !teacher.UserID.Valid {
			return response.Fail("教师记录不存在或未关联用户")
		}

		// 查 user_auth 记录
		userID := teacher.UserID.Int64
		auth, err := s.userAuthMapper.SelectByUserID(userID, 4)
		if err != nil {
			log.Printf("查询用户认证失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "系统异常")
		}
		if auth == nil {
			return response.Fail("用户认证记录不存在")
		}

		// 处理密码：SM2 解密后 SM3 加盐哈希
		newPassword := ""
		newSalt := ""
		if passwordCipher != "" {
			passwordPlain, err := crypto.SM2Decrypt(passwordCipher, s.sm2PrivateKey)
			if err != nil {
				log.Printf("SM2 解密密码失败: %v", err)
				return response.Fail("密码解密失败")
			}
			newSalt = crypto.GenerateSalt()
			newPassword = crypto.SM3DigestWithSalt(passwordPlain, newSalt)
		}

		// 更新 user_auth
		_, err = s.userAuthMapper.UpdateAccountAndPassword(auth.ID, account, newPassword, newSalt)
		if err != nil {
			log.Printf("更新用户认证失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "系统异常")
		}
	}

	// 返回教师ID（对齐前端 UpdateTeacherResponse）
	return response.Success(&UpdateTeacherVO{TeacherID: id})
}

// UpdateTeacherVO 更新教师响应 VO（对齐前端 UpdateTeacherResponse）
type UpdateTeacherVO struct {
	TeacherID int64 `json:"teacherId"` // 被更新的教师ID
}

// DeleteTeacher 删除教师
//
// 对齐 Java TeacherServiceImpl.deleteTeacher
//
// 流程：
//  1. 校验教师是否关联班级（c_class_teacher），已关联则禁止删除
//  2. 查教师关联的 userId
//  3. 删除 c_user_auth 记录
//  4. 删除 c_user 记录
//  5. 删除 c_teacher 记录
//
// 前端期望：data 为字符串消息
//
// 参数：
//   - id: 教师ID
func (s *TeacherService) DeleteTeacher(id int64) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("教师ID不能为空")
	}

	// 1. 校验教师是否关联班级
	classCount, err := s.classTeacherMapper.CountByTeacherID(id)
	if err != nil {
		log.Printf("查询教师班级关联失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if classCount > 0 {
		return response.Fail("教师已关联班级，无法删除")
	}

	// 2. 查教师记录获取 userId
	teacher, err := s.teacherMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if teacher == nil {
		return response.Fail("教师不存在")
	}

	// 3. 删除 c_user_auth 记录（如果有关联的 userId）
	if teacher.UserID.Valid {
		userID := teacher.UserID.Int64
		auth, err := s.userAuthMapper.SelectByUserID(userID, 4)
		if err != nil {
			log.Printf("查询用户认证失败: %v", err)
		} else if auth != nil {
			_, _ = s.userAuthMapper.DeleteByID(auth.ID)
		}

		// 4. 删除 c_user 记录
		_, _ = s.userMapper.DeleteByID(userID)
	}

	// 5. 删除 c_teacher 记录
	_, err = s.teacherMapper.DeleteByID(id)
	if err != nil {
		log.Printf("删除教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success("删除教师成功")
}

// ============================================================
// 辅助：实体转 VO（对齐 Java MapStruct Converter）
// ============================================================

// InstitutionVO 机构视图对象（对齐 Java QueryInstitutionVO）
//
// 字段命名对齐前端 InstitutionResponse 类型：
//   - 小程序 src/types/institution.d.ts
//   - admin   src/types/business.d.ts
type InstitutionVO struct {
	ID                   int64  `json:"id"`                   // 机构ID
	InstitutionName      string `json:"institutionName"`      // 机构名称
	InstitutionAddress   string `json:"institutionAddress"`   // 机构地址
	InstitutionCode      string `json:"institutionCode"`      // 机构编码
	Status               int64  `json:"status"`               // 状态
	ExpireTime           string `json:"expireTime"`           // 过期时间
	SubscriptionPlanID   int64  `json:"subscriptionPlanId"`   // 订阅计划ID
	SubscriptionPlanName string `json:"subscriptionPlanName"` // 订阅计划名
	CreateTimeStr        string `json:"createTimeStr"`        // 创建时间字符串（前端使用 xxxStr 命名）
	UpdateTimeStr        string `json:"updateTimeStr"`        // 更新时间字符串
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
	// 时间字段使用 xxxStr 命名（对齐前端类型定义）
	vo.CreateTimeStr = entity.FormatTime(inst.CreateTime)
	vo.UpdateTimeStr = entity.FormatTime(inst.UpdateTime)
	return vo
}

// StudentVO 学生视图对象（对齐 Java QueryStudentVO）
//
// 使用普通类型而非 sql.NullXxx，避免 JSON 序列化输出对象格式
//
// 字段命名对齐前端 StudentResponse 类型：
//   - 小程序 src/types/student.d.ts（birthStr/createTimeStr/updateTimeStr）
//   - admin   src/types/business.d.ts（birthStr/createTimeStr/updateTimeStr）
type StudentVO struct {
	ID            int64  `json:"id"`            // 学生ID
	Avatar        string `json:"avatar"`        // 头像URL
	StudentName   string `json:"studentName"`   // 学生姓名
	InstitutionID int64  `json:"institutionId"` // 机构ID
	Sex           int64  `json:"sex"`           // 性别（0=未知,1=男,2=女）
	BirthStr      string `json:"birthStr"`      // 出生日期字符串（前端使用 birthStr）
	School        string `json:"school"`        // 学校
	Address       string `json:"address"`       // 地址
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
}

// ToStudentVO 实体转 VO
func ToStudentVO(s *entity.Student) *StudentVO {
	if s == nil {
		return nil
	}
	vo := &StudentVO{}
	if s.ID.Valid {
		vo.ID = s.ID.Int64
	}
	vo.Avatar = s.Avatar.String
	vo.StudentName = s.StudentName.String
	if s.InstitutionID.Valid {
		vo.InstitutionID = s.InstitutionID.Int64
	}
	if s.Sex.Valid {
		vo.Sex = s.Sex.Int64
	}
	// 时间字段使用 xxxStr 命名（对齐前端类型定义）
	vo.BirthStr = entity.FormatTime(s.Birth)
	vo.School = s.School.String
	vo.Address = s.Address.String
	vo.CreateTimeStr = entity.FormatTime(s.CreateTime)
	vo.UpdateTimeStr = entity.FormatTime(s.UpdateTime)
	return vo
}

// ToStudentVOList 实体列表转 VO 列表
func ToStudentVOList(list []*entity.Student) []*StudentVO {
	result := make([]*StudentVO, 0, len(list))
	for _, s := range list {
		if vo := ToStudentVO(s); vo != nil {
			result = append(result, vo)
		}
	}
	return result
}

// TeacherVO 教师视图对象（对齐 Java QueryTeacherVO）
//
// 字段命名对齐前端 TeacherResponse 类型：
//   - 小程序 src/types/teacher.d.ts（包含 account 字段）
//   - admin   src/types/business.d.ts（包含 account 字段）
type TeacherVO struct {
	TeacherID          int64  `json:"teacherId"`          // 教师ID（主键）
	UserID             int64  `json:"userId"`             // 关联用户ID
	IsAvailable        bool   `json:"isAvailable"`        // 是否可用
	Username           string `json:"username"`           // 用户名
	Account            string `json:"account"`            // 登录账号（来自 c_user_auth.account）
	InstitutionID      int64  `json:"institutionId"`      // 机构ID
	IsInstitutionAdmin bool   `json:"isInstitutionAdmin"` // 是否机构管理员
	Phone              string `json:"phone"`              // 手机号
}

// ToTeacherVO 实体转 VO
//
// 注意：account 字段需要从 c_user_auth 表查询后填充，此处仅初始化为空字符串
// 调用方（如 TeacherService）应在查询后通过 SetTeacherAccount 方法填充
func ToTeacherVO(t *entity.Teacher) *TeacherVO {
	if t == nil {
		return nil
	}
	vo := &TeacherVO{}
	if t.TeacherID.Valid {
		vo.TeacherID = t.TeacherID.Int64
	}
	if t.UserID.Valid {
		vo.UserID = t.UserID.Int64
	}
	vo.IsAvailable = t.IsAvailable.Bool
	vo.Username = t.Username.String
	vo.Account = "" // 默认空，由调用方填充
	if t.InstitutionID.Valid {
		vo.InstitutionID = t.InstitutionID.Int64
	}
	vo.IsInstitutionAdmin = t.IsInstitutionAdmin.Bool
	vo.Phone = t.Phone.String
	return vo
}

// ToTeacherVOList 实体列表转 VO 列表
func ToTeacherVOList(list []*entity.Teacher) []*TeacherVO {
	result := make([]*TeacherVO, 0, len(list))
	for _, t := range list {
		if vo := ToTeacherVO(t); vo != nil {
			result = append(result, vo)
		}
	}
	return result
}
