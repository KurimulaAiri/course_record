// Package service business-service 业务逻辑层 - 课卡记录模块
//
// 对齐 Java business-service CourseRecordServiceImpl
//
// 核心功能：
//   - 课卡记录查询（按学生ID/机构ID/通用条件）
//   - 课卡记录新增
//   - 课卡记录更新
//   - 课时扣减（按学生ID/课程ID/班级ID，含双重校验）
//   - 扣费详情查询
//
// 扣费双重校验（对齐 Java CourseRecordServiceImpl.checkCourseRecordExpired）：
//  1. Service 层校验 expire_time（过期返回 code=1003 COURSE_EXPIRED）
//  2. SQL 层 WHERE 条件包含 (expire_time IS NULL OR expire_time > NOW()) 兜底
//  3. 余额不足返回 code=1001 COURSE_BALANCE_NOT_ENOUGH
package service

import (
	"database/sql"
	"log"
	"time"

	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// CourseRecordService 课卡记录服务
// ============================================================

// CourseRecordService 课卡记录服务（对齐 Java CourseRecordServiceImpl）
//
// 查询：按学生ID/机构ID/通用条件
// 写操作：新增/更新/扣课时（含双重校验）
type CourseRecordService struct {
	courseRecordMapper  *mapper.CourseRecordMapper
	classStudentMapper  *mapper.ClassStudentMapper
	recordMapper        *mapper.RecordMapper
}

// NewCourseRecordService 创建 CourseRecordService
//
// 参数：
//   - courseRecordMapper: 课卡记录 Mapper
//   - classStudentMapper: 班级-学生关联 Mapper（按班级扣课时查学生列表）
//   - recordMapper: 上课记录 Mapper（扣课时记录流水）
func NewCourseRecordService(
	courseRecordMapper *mapper.CourseRecordMapper,
	classStudentMapper *mapper.ClassStudentMapper,
	recordMapper *mapper.RecordMapper,
) *CourseRecordService {
	return &CourseRecordService{
		courseRecordMapper: courseRecordMapper,
		classStudentMapper: classStudentMapper,
		recordMapper:        recordMapper,
	}
}

// CourseRecordVO 课卡记录视图对象（对齐前端 CourseRecordResponse）
//
// 字段命名对齐前端 src/types/course-record.d.ts
//
// 重构说明（从扁平字段改为嵌套对象）：
//   - 旧字段 CourseID/CourseName/CourseType（扁平课程信息）已移除
//   - 新字段 Course CourseBriefVO（嵌套课程对象），对齐前端 course: CourseResponse
//     使用 CourseBriefVO 而非 CourseVO 以避免循环引用（见 vo.go 注释）
//   - 新字段 PermissionType int64（权限类型，对应 c_course_record.permission_type）
//   - 新字段 ExpireStatus string（过期状态，前端枚举 "expired"|"warning"|"valid"）
//     由 calcCourseRecordExpireStatus 根据 ExpireTimeRaw 计算
type CourseRecordVO struct {
	ID                int64          `json:"id"`                // 课卡记录ID
	StudentID         int64          `json:"studentId"`         // 学生ID
	StudentName       string         `json:"studentName"`       // 学生姓名（JOIN c_student 获取）
	CourseTotalTime   int64          `json:"courseTotalTime"`   // 课时总数
	CourseRestTime    int64          `json:"courseRestTime"`    // 剩余课时
	CourseStatus      int64          `json:"courseStatus"`      // 课程状态（0=默认,1=未完成,2=已完成）
	CourseLastTimeStr string         `json:"courseLastTimeStr"` // 上次上课时间字符串
	ExpireTimeStr     string         `json:"expireTimeStr"`     // 过期时间字符串（空=永久有效）
	IsDelete          bool           `json:"isDelete"`          // 是否已删除（逻辑删除标识）
	CourseRemark      string         `json:"courseRemark"`      // 课程备注
	CourseOwnerUserID int64          `json:"courseOwnerUserId"` // 课程归属人（c_user.id）
	PermissionType    int64          `json:"permissionType"`    // 权限类型（对应 c_course_record.permission_type）
	CreateTimeStr     string         `json:"createTimeStr"`     // 创建时间字符串
	UpdateTimeStr     string         `json:"updateTimeStr"`     // 更新时间字符串
	Course            CourseBriefVO  `json:"course"`            // 嵌套课程对象（对齐前端 course: CourseResponse）
	ExpireStatus      string         `json:"expireStatus"`      // 过期状态（expired=已过期, warning=即将过期, valid=有效）
}

// ToCourseRecordVO CourseRecordDTO 转 VO
//
// 转换逻辑：
//  1. 将 DTO 的扁平字段映射到 VO 的对应字段
//  2. 从 DTO 的课程字段（CourseID/CourseName/CourseType/IsAvailable）构造嵌套 Course CourseBriefVO
//  3. 根据 DTO.ExpireTimeRaw（sql.NullTime）计算 ExpireStatus（expired/warning/valid）
func ToCourseRecordVO(dto *mapper.CourseRecordDTO) *CourseRecordVO {
	if dto == nil {
		return nil
	}
	return &CourseRecordVO{
		ID:                dto.ID,
		StudentID:         dto.StudentID,
		StudentName:       dto.StudentName,
		CourseTotalTime:   dto.CourseTotalTime,
		CourseRestTime:    dto.CourseRestTime,
		CourseStatus:      dto.CourseStatus,
		CourseLastTimeStr: dto.CourseLastTime,
		ExpireTimeStr:     dto.ExpireTime,
		IsDelete:          dto.IsDelete,
		CourseRemark:      dto.CourseRemark,
		CourseOwnerUserID: dto.CourseOwnerUserID,
		PermissionType:    dto.PermissionType,
		CreateTimeStr:     dto.CreateTime,
		UpdateTimeStr:     dto.UpdateTime,
		// 构造嵌套课程对象（CourseBriefVO），从 DTO 的 JOIN 字段填充
		Course: CourseBriefVO{
			ID:          dto.CourseID,
			CourseName:  dto.CourseName,
			CourseType:  dto.CourseType,
			IsAvailable: dto.IsAvailable,
		},
		// 根据 expire_time 原始值计算过期状态（对齐前端 expireStatus 枚举）
		ExpireStatus: calcCourseRecordExpireStatus(dto.ExpireTimeRaw),
	}
}

// ToCourseRecordVOList CourseRecordDTO 列表转 VO 列表
func ToCourseRecordVOList(list []*mapper.CourseRecordDTO) []*CourseRecordVO {
	result := make([]*CourseRecordVO, 0, len(list))
	for _, dto := range list {
		if vo := ToCourseRecordVO(dto); vo != nil {
			result = append(result, vo)
		}
	}
	return result
}

// QueryCourseRecordVO 课卡记录查询响应包装（对齐前端 CourseRecordListResponse）
type QueryCourseRecordVO struct {
	CourseRecords []*CourseRecordVO `json:"courseRecords"` // 课卡记录列表
	Total         int64             `json:"total"`         // 总数
}

// GetCourseRecordList 查询课卡记录列表（对齐 Java CourseRecordServiceImpl.new_get）
//
// 前端期望：data.courseRecords（数组）+ data.total
//
// 参数：
//   - studentID: 学生ID（0 表示不过滤）
//   - institutionID: 机构ID（0 表示不过滤）
//   - courseName: 课程名称关键词
//   - stuName: 学生姓名关键词
//   - keyword: 通用关键词
//   - expireStatus: 过期状态（0=有效, 1=即将过期, 2=已过期，-1=不过滤）
//   - currentPage: 当前页码
//   - pageSize: 每页条数
func (s *CourseRecordService) GetCourseRecordList(studentID, institutionID int64, courseName, stuName, keyword string, expireStatus int64, currentPage, pageSize int) *response.ResponseDTO {
	list, total, err := s.courseRecordMapper.SelectList(studentID, institutionID, courseName, stuName, keyword, expireStatus, currentPage, pageSize)
	if err != nil {
		log.Printf("查询课卡记录列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToCourseRecordVOList(list)
	return response.Success(&QueryCourseRecordVO{
		CourseRecords: voList,
		Total:         total,
	})
}

// GetCourseRecordByStudentID 按学生ID查课卡记录列表
//
// 对齐 Java CourseRecordServiceImpl.getByStudentId
// 前端期望：data.courseRecords（数组）+ data.total
func (s *CourseRecordService) GetCourseRecordByStudentID(studentID int64, courseName string, expireStatus int64, currentPage, pageSize int) *response.ResponseDTO {
	if studentID == 0 {
		return response.Fail("学生ID不能为空")
	}

	list, total, err := s.courseRecordMapper.SelectList(studentID, 0, courseName, "", "", expireStatus, currentPage, pageSize)
	if err != nil {
		log.Printf("查询学生课卡记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToCourseRecordVOList(list)
	return response.Success(&QueryCourseRecordVO{
		CourseRecords: voList,
		Total:         total,
	})
}

// GetCourseRecordByInstitutionID 按机构ID查课卡记录列表
//
// 对齐 Java CourseRecordServiceImpl.getByInstitutionId
// 前端期望：data.courseRecords（数组）+ data.total
func (s *CourseRecordService) GetCourseRecordByInstitutionID(institutionID int64, keyword string, expireStatus int64, currentPage, pageSize int) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, total, err := s.courseRecordMapper.SelectList(0, institutionID, "", "", keyword, expireStatus, currentPage, pageSize)
	if err != nil {
		log.Printf("查询机构课卡记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToCourseRecordVOList(list)
	return response.Success(&QueryCourseRecordVO{
		CourseRecords: voList,
		Total:         total,
	})
}

// InsertCourseRecord 新增课卡记录
//
// 对齐 Java CourseRecordServiceImpl.insert
//
// 前端期望：data 为新创建的课卡记录对象
//
// 参数：
//   - studentID: 学生ID
//   - courseID: 课程ID
//   - totalTime: 课时总数
//   - restTime: 剩余课时
//   - expireTime: 过期时间（空字符串表示永久有效）
//   - ownerUserID: 课程归属用户ID
//   - remark: 备注
func (s *CourseRecordService) InsertCourseRecord(studentID, courseID, totalTime, restTime int64, expireTime string, ownerUserID int64, remark string) *response.ResponseDTO {
	if studentID == 0 {
		return response.Fail("学生ID不能为空")
	}
	if courseID == 0 {
		return response.Fail("课程ID不能为空")
	}

	id, err := s.courseRecordMapper.Insert(studentID, courseID, totalTime, restTime, expireTime, ownerUserID, remark)
	if err != nil {
		log.Printf("新增课卡记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回新创建的课卡记录（对齐前端 InsertCourseRecordResponse）
	return response.Success(&InsertCourseRecordVO{
		ID:              id,
		StudentID:       studentID,
		CourseID:        courseID,
		CourseTotalTime: totalTime,
		CourseRestTime:  restTime,
		ExpireTimeStr:   expireTime,
		CourseRemark:    remark,
	})
}

// InsertCourseRecordVO 新增课卡记录响应 VO（对齐前端 CourseRecordResponse）
type InsertCourseRecordVO struct {
	ID              int64  `json:"id"`              // 课卡记录ID
	StudentID       int64  `json:"studentId"`       // 学生ID
	CourseID        int64  `json:"courseId"`        // 课程ID
	CourseTotalTime int64  `json:"courseTotalTime"` // 课时总数
	CourseRestTime  int64  `json:"courseRestTime"`  // 剩余课时
	ExpireTimeStr   string `json:"expireTimeStr"`   // 过期时间字符串
	CourseRemark    string `json:"courseRemark"`    // 课程备注
}

// UpdateCourseRecord 更新课卡记录
//
// 对齐 Java CourseRecordServiceImpl.update
//
// 前端期望：data（影响行数）
//
// 参数：
//   - id: 课卡记录ID
//   - totalTime: 课时总数（0 表示不更新）
//   - restTime: 剩余课时（0 表示不更新）
//   - status: 课程状态（-1 表示不更新）
//   - expireTime: 过期时间（空字符串表示不更新，"NULL" 表示设为 NULL）
//   - remark: 备注（空字符串表示不更新）
func (s *CourseRecordService) UpdateCourseRecord(id int64, totalTime, restTime, status int64, expireTime, remark string) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("课卡记录ID不能为空")
	}

	rows, err := s.courseRecordMapper.UpdateByID(id, totalTime, restTime, status, expireTime, remark)
	if err != nil {
		log.Printf("更新课卡记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success(rows)
}

// DeductClassItem 按学生扣课时的班级子项（对齐前端 DeductClassDTO）
type DeductClassItem struct {
	ClassID     int64 // 班级ID
	CourseID    int64 // 课程ID
	DeductCount int64 // 扣减课时数
}

// DeductStudentItem 按课程扣课时的学生子项（对齐前端 DeductStudentDTO）
type DeductStudentItem struct {
	StudentID   int64 // 学生ID
	DeductCount int64 // 扣减课时数
}

// DeductByStudentID 按学生ID扣课时
//
// 对齐 Java CourseRecordServiceImpl.deductByStudentId
//
// 模式：mode=student
// 流程：遍历班级列表，对每个班级的所有学生扣减指定课时
//
// 前端期望：data.result（成功扣减的记录数）
//
// 参数：
//   - studentID: 学生ID
//   - classes: 班级扣课列表（每个项含 classId/courseId/deductCount）
//   - recordTime: 记录时间（空字符串表示当前时间）
//   - operateTeacherID: 操作教师ID
//   - remark: 备注
func (s *CourseRecordService) DeductByStudentID(studentID int64, classes []*DeductClassItem, recordTime string, operateTeacherID int64, remark string) *response.ResponseDTO {
	if studentID == 0 {
		return response.Fail("学生ID不能为空")
	}
	if len(classes) == 0 {
		return response.Fail("扣课班级列表不能为空")
	}

	// 记录时间默认为当前时间
	if recordTime == "" {
		recordTime = time.Now().Format("2006-01-02 15:04:05")
	}

	successCount := int64(0)
	// 遍历每个班级项，执行扣课
	for _, item := range classes {
		// 调用核心扣课逻辑
		result := s.deductOne(studentID, item.CourseID, item.DeductCount, recordTime, operateTeacherID, remark, item.ClassID, "BY_STUDENT")
		if result.Code == response.CodeSuccess {
			successCount++
		}
	}

	// 返回成功扣减数（对齐前端 FastDeductResponse.result）
	return response.Success(&DeductResultVO{Result: successCount})
}

// DeductByCourseID 按课程ID扣课时
//
// 对齐 Java CourseRecordServiceImpl.deductByCourseId
//
// 模式：mode=course
// 流程：遍历学生列表，对每个学生扣减指定课时的该课程
//
// 前端期望：data.result（成功扣减的记录数）
//
// 参数：
//   - courseID: 课程ID
//   - students: 学生扣课列表（每个项含 studentId/deductCount）
//   - recordTime: 记录时间
//   - operateTeacherID: 操作教师ID
//   - remark: 备注
func (s *CourseRecordService) DeductByCourseID(courseID int64, students []*DeductStudentItem, recordTime string, operateTeacherID int64, remark string) *response.ResponseDTO {
	if courseID == 0 {
		return response.Fail("课程ID不能为空")
	}
	if len(students) == 0 {
		return response.Fail("扣课学生列表不能为空")
	}

	if recordTime == "" {
		recordTime = time.Now().Format("2006-01-02 15:04:05")
	}

	successCount := int64(0)
	for _, item := range students {
		// 按课程扣课，classID=0 表示不关联班级
		result := s.deductOne(item.StudentID, courseID, item.DeductCount, recordTime, operateTeacherID, remark, 0, "BY_COURSE")
		if result.Code == response.CodeSuccess {
			successCount++
		}
	}

	return response.Success(&DeductResultVO{Result: successCount})
}

// DeductByClassID 按班级ID扣课时
//
// 对齐 Java CourseRecordServiceImpl.deductByClassId
//
// 模式：mode=class
// 流程：查班级所有学生，对每个学生扣减指定课时
//
// 前端期望：data.result（成功扣减的记录数）
//
// 参数：
//   - classID: 班级ID
//   - courseID: 课程ID
//   - deductCount: 扣减课时数
//   - recordTime: 记录时间
//   - operateTeacherID: 操作教师ID
//   - remark: 备注
func (s *CourseRecordService) DeductByClassID(classID, courseID, deductCount int64, recordTime string, operateTeacherID int64, remark string) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}
	if courseID == 0 {
		return response.Fail("课程ID不能为空")
	}
	if deductCount <= 0 {
		return response.Fail("扣减课时数必须大于0")
	}

	if recordTime == "" {
		recordTime = time.Now().Format("2006-01-02 15:04:05")
	}

	// 1. 查班级所有学生ID
	studentIDs, err := s.classStudentMapper.SelectStudentIDsByClassID(classID)
	if err != nil {
		log.Printf("查询班级学生列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if len(studentIDs) == 0 {
		return response.Fail("班级没有学生")
	}

	// 2. 对每个学生执行扣课
	successCount := int64(0)
	for _, studentID := range studentIDs {
		result := s.deductOne(studentID, courseID, deductCount, recordTime, operateTeacherID, remark, classID, "BY_CLASS")
		if result.Code == response.CodeSuccess {
			successCount++
		}
	}

	return response.Success(&DeductResultVO{Result: successCount})
}

// DeductResultVO 扣课结果 VO（对齐前端 FastDeductResponse）
type DeductResultVO struct {
	Result int64 `json:"result"` // 成功扣减的记录数
}

// deductOne 扣减单个学生的单个课程课时（核心：双重校验）
//
// 对齐 Java CourseRecordServiceImpl 内部扣课逻辑
//
// 双重校验：
//  1. Service 层校验：查课卡记录，判断 expire_time 是否过期
//     - 已过期 → 返回 code=1003 COURSE_EXPIRED
//  2. SQL 层校验：UpdateRestTime 的 WHERE 条件包含
//     - course_rest_time >= totalCount（余额充足）
//     - (expire_time IS NULL OR expire_time > NOW())（未过期）
//     - rows=0 表示余额不足或已过期
//
// 参数：
//   - studentID: 学生ID
//   - courseID: 课程ID
//   - deductCount: 扣减课时数
//   - recordTime: 记录时间
//   - operateTeacherID: 操作教师ID
//   - remark: 备注
//   - classID: 班级ID（0 表示不关联班级）
//   - deductMode: 扣费模式（BY_STUDENT/BY_COURSE/BY_CLASS）
//
// 返回：扣课结果响应
func (s *CourseRecordService) deductOne(studentID, courseID, deductCount int64, recordTime string, operateTeacherID int64, remark string, classID int64, deductMode string) *response.ResponseDTO {
	// 1. Service 层过期校验（对齐 Java checkCourseRecordExpired）
	cr, err := s.courseRecordMapper.SelectByStudentAndCourse(studentID, courseID)
	if err != nil {
		log.Printf("查询课卡记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if cr == nil {
		return response.Fail("课卡记录不存在")
	}

	// 校验过期时间：如果 expire_time 有效且已过期，返回 COURSE_EXPIRED
	if cr.ExpireTime.Valid {
		now := time.Now()
		if cr.ExpireTime.Time.Before(now) {
			// 已过期，禁止扣课（对齐 Java COURSE_EXPIRED code=1003）
			return response.FailWithCode(response.CodeCourseExpired, "课时已过期")
		}
	}

	// 2. SQL 层扣减课时（含双重校验：余额 + 过期时间）
	rows, updatedID, err := s.courseRecordMapper.UpdateRestTime(studentID, courseID, deductCount)
	if err != nil {
		log.Printf("扣减课时失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 3. rows=0 表示扣减失败（余额不足或已过期）
	if rows == 0 {
		// 区分错误：再次查课卡记录判断是余额不足还是已过期
		// 由于 SQL 已包含过期校验，此时大概率是余额不足
		if cr.CourseRestTime.Int64 < deductCount {
			// 余额不足（对齐 Java COURSE_BALANCE_NOT_ENOUGH code=1001）
			return response.FailWithCode(response.CodeCourseBalanceEmpty, "课时余额不足")
		}
		// 其他原因（可能在 Service 层校验和 SQL 执行之间过期）
		return response.FailWithCode(response.CodeCourseExpired, "课时已过期")
	}

	// 4. 查询扣减后的课卡记录（获取最新剩余课时）
	updatedCR, err := s.courseRecordMapper.SelectByID(updatedID)
	if err != nil {
		log.Printf("查询更新后课卡记录失败: %v", err)
	}
	restAfterDeduct := int64(0)
	if updatedCR != nil {
		restAfterDeduct = updatedCR.CourseRestTime.Int64
	}

	// 5. 插入上课记录（c_record 表）
	// cr 为 entity.CourseRecord，ID 为 sql.NullInt64，需取 .Int64 字段
	_, err = s.recordMapper.Insert(
		cr.ID.Int64,                 // courseRecordID
		recordTime,                  // recordTime
		operateTeacherID,            // operateTeacherID
		remark,                      // remark
		2,                           // recordType=2 表示减少
		deductCount,                 // recordChange
		restAfterDeduct,             // restTimeAfterDeduct
		deductMode,                  // deductMode
		classID,                     // classID
	)
	if err != nil {
		log.Printf("插入上课记录失败: %v", err)
		// 不阻塞主流程，扣课已成功
	}

	return response.Success(&DeductResultVO{Result: 1})
}

// GetDeductDetail 查询扣费详情
//
// 对齐 Java CourseRecordServiceImpl.getDeductDetail
// 用途：家长端通知点击后调用，查询扣费详情
//
// 前端期望：data 为扣费详情对象（DeductDetailResponse，21 个字段）
//
// 实现方式：一次 JOIN 查询获取全部关联数据，避免多次查表
//   - c_record + c_course_record + c_course + c_student + c_class + c_class_schedule + c_teacher
//
// 参数：
//   - recordID: 上课记录ID（c_record.id）
func (s *CourseRecordService) GetDeductDetail(recordID int64) *response.ResponseDTO {
	if recordID == 0 {
		return response.Fail("记录ID不能为空")
	}

	// 一次 JOIN 查询获取扣费详情全部字段（对齐前端 21 个字段）
	dto, err := s.recordMapper.SelectDeductDetailByID(recordID)
	if err != nil {
		log.Printf("查询扣费详情失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if dto == nil {
		return response.Fail("记录不存在")
	}

	// 构造扣费详情 VO，从 DTO 提取并格式化字段
	detail := &DeductDetailVO{
		CourseRecordID:      dto.CourseRecordID.Int64,
		RecordID:            dto.RecordID,
		CourseID:            dto.CourseID.Int64,
		CourseName:          dto.CourseName.String,
		CourseType:          dto.CourseType.Int64,
		StudentID:           dto.StudentID.Int64,
		StudentName:         dto.StudentName.String,
		DeductCount:         dto.RecordChange.Int64, // 扣减时 record_change 即为 deductCount
		CourseRestTime:      dto.CourseRestTime.Int64,
		RestTimeAfterDeduct: dto.RestTimeAfterDeduct.Int64,
		CourseTotalTime:     dto.CourseTotalTime.Int64,
		ExpireTime:          entity.FormatTime(dto.ExpireTime),
		RecordTime:          entity.FormatTime(dto.RecordTime),
		Remark:              dto.RecordRemark.String,
		ClassID:             dto.ClassID.Int64,
		ClassName:           dto.ClassName.String,
		ScheduleDesc:        dto.ScheduleDesc.String,
		TeacherID:           dto.OperateTeacherID.Int64,
		TeacherName:         dto.TeacherName.String,
		DeductMode:          dto.DeductMode.String,
		ExpireStatus:        calcExpireStatus(dto.ExpireTime),
	}

	return response.Success(detail)
}

// calcExpireStatus 计算过期状态（对齐前端 DeductDetailResponse.expireStatus 字段逻辑）
//
// 用于 DeductDetailVO 的 expireStatus 字段，返回值枚举：normal/expired/warning
//
// 规则：
//   - expire_time 为空或 NULL：返回 "normal"（永久有效）
//   - expire_time 已过期（< 当前时间）：返回 "expired"
//   - expire_time 在 7 天内过期（< 当前时间 + 7天）：返回 "warning"
//   - 否则：返回 "normal"
//
// 参数：
//   - expireTime: 过期时间（sql.NullTime，Valid=false 表示永久有效）
//
// 返回：过期状态字符串（normal/expired/warning）
func calcExpireStatus(expireTime sql.NullTime) string {
	// 过期时间为空表示永久有效
	if !expireTime.Valid {
		return "normal"
	}
	now := time.Now()
	// 已过期：过期时间早于当前时间
	if expireTime.Time.Before(now) {
		return "expired"
	}
	// 即将过期：7 天内过期（过期时间早于当前时间 + 7 天）
	warningThreshold := now.Add(7 * 24 * time.Hour)
	if expireTime.Time.Before(warningThreshold) {
		return "warning"
	}
	// 正常：过期时间在 7 天之后
	return "normal"
}

// calcCourseRecordExpireStatus 计算课卡记录的过期状态（对齐前端 CourseRecordResponse.expireStatus 字段）
//
// 用于 CourseRecordVO 的 expireStatus 字段，返回值枚举：valid/expired/warning
// 与 calcExpireStatus 的区别：永久有效/正常状态返回 "valid" 而非 "normal"，
// 对齐前端 CourseRecordResponse 的 expireStatus 类型定义 "expired" | "warning" | "valid"
//
// 规则：
//   - expire_time 为空或 NULL：返回 "valid"（永久有效）
//   - expire_time 已过期（< 当前时间）：返回 "expired"
//   - expire_time 在 7 天内过期（< 当前时间 + 7天）：返回 "warning"
//   - 否则：返回 "valid"
//
// 参数：
//   - expireTime: 过期时间（sql.NullTime，Valid=false 表示永久有效）
//
// 返回：过期状态字符串（valid/expired/warning）
func calcCourseRecordExpireStatus(expireTime sql.NullTime) string {
	// 过期时间为空表示永久有效
	if !expireTime.Valid {
		return "valid"
	}
	now := time.Now()
	// 已过期：过期时间早于当前时间
	if expireTime.Time.Before(now) {
		return "expired"
	}
	// 即将过期：7 天内过期（过期时间早于当前时间 + 7 天）
	warningThreshold := now.Add(7 * 24 * time.Hour)
	if expireTime.Time.Before(warningThreshold) {
		return "warning"
	}
	// 正常：过期时间在 7 天之后
	return "valid"
}

// DeductDetailVO 扣费详情 VO（对齐前端 DeductDetailResponse）
//
// 共 21 个字段，对齐前端 src/pages/main/parent/deduct-detail/index.d.ts
// 字段顺序与前端接口定义保持一致，便于前端直接消费
type DeductDetailVO struct {
	CourseRecordID      int64  `json:"courseRecordId"`      // 课卡记录ID
	RecordID            int64  `json:"recordId"`            // 上课记录ID
	CourseID            int64  `json:"courseId"`            // 课程ID
	CourseName          string `json:"courseName"`          // 课程名称
	CourseType          int64  `json:"courseType"`          // 课程类型（1=按次, 2=按天）
	StudentID           int64  `json:"studentId"`           // 学生ID
	StudentName         string `json:"studentName"`         // 学生姓名
	DeductCount         int64  `json:"deductCount"`         // 本次扣减课时数
	CourseRestTime      int64  `json:"courseRestTime"`      // 当前剩余课时数（实时值）
	RestTimeAfterDeduct int64  `json:"restTimeAfterDeduct"` // 该次扣费后剩余课时（快照值）
	CourseTotalTime     int64  `json:"courseTotalTime"`     // 课时总数
	ExpireTime          string `json:"expireTime"`          // 到期时间（格式化字符串，空=永久有效）
	RecordTime          string `json:"recordTime"`          // 上课时间（格式化字符串）
	Remark              string `json:"remark"`              // 备注
	ClassID             int64  `json:"classId"`             // 班级ID
	ClassName           string `json:"className"`           // 班级名称
	ScheduleDesc        string `json:"scheduleDesc"`        // 排课时间描述（如 "周一 09:00-10:00, 周三 14:00-15:00"）
	TeacherID           int64  `json:"teacherId"`           // 操作老师ID
	TeacherName         string `json:"teacherName"`         // 操作老师姓名
	DeductMode          string `json:"deductMode"`          // 扣费模式（BY_STUDENT=按学生, BY_COURSE=按课程, BY_CLASS=按班级）
	ExpireStatus        string `json:"expireStatus"`        // 到期状态（normal=正常, expired=已过期, warning=即将过期）
}
