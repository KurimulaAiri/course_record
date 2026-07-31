// Package service business-service 业务逻辑层 - 班级模块
//
// 对齐 Java business-service ClassServiceImpl
//
// 核心功能：
//   - 班级查询（按学生ID/教师ID/机构ID/班级ID）
//   - 班级新增（含课表和教师关联）
//   - 班级更新（含课表和教师关联的"先删后增"）
//   - 班级学生管理（添加/移除学生）
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// ClassService 班级服务
// ============================================================

// ClassService 班级服务（对齐 Java ClassServiceImpl）
//
// 查询：按学生ID/教师ID/机构ID/班级ID
// 写操作：新增/更新班级（含课表和教师关联）、添加/移除学生
//
// 重构说明（嵌套对象化）：
//   - 新增 courseRecordMapper 依赖：用于按学生ID查班级时填充 ClassVO.CourseRecord 嵌套对象
//   - 现有 classScheduleMapper 用于填充 ClassVO.ScheduleList 嵌套数组
type ClassService struct {
	classMapper         *mapper.ClassMapper
	classStudentMapper  *mapper.ClassStudentMapper
	classTeacherMapper  *mapper.ClassTeacherMapper
	classScheduleMapper *mapper.ClassScheduleMapper
	courseRecordMapper  *mapper.CourseRecordMapper // 课卡记录 Mapper（按学生维度查询时填充 CourseRecord 嵌套对象）
}

// NewClassService 创建 ClassService
//
// 参数：
//   - classMapper: 班级表 Mapper
//   - classStudentMapper: 班级-学生关联 Mapper
//   - classTeacherMapper: 班级-教师关联 Mapper
//   - classScheduleMapper: 班级课表 Mapper
//   - courseRecordMapper: 课卡记录 Mapper（用于按学生ID查班级时填充 CourseRecord）
func NewClassService(
	classMapper *mapper.ClassMapper,
	classStudentMapper *mapper.ClassStudentMapper,
	classTeacherMapper *mapper.ClassTeacherMapper,
	classScheduleMapper *mapper.ClassScheduleMapper,
	courseRecordMapper *mapper.CourseRecordMapper,
) *ClassService {
	return &ClassService{
		classMapper:         classMapper,
		classStudentMapper:  classStudentMapper,
		classTeacherMapper:  classTeacherMapper,
		classScheduleMapper: classScheduleMapper,
		courseRecordMapper:  courseRecordMapper,
	}
}

// ClassVO 班级视图对象（对齐前端 ClassResponse）
//
// 字段命名对齐前端 src/types/class.d.ts
//
// 重构说明（从扁平字段改为嵌套对象）：
//   - 旧字段 TeacherID/TeacherUsername（单值）已移除
//   - 新字段 Teachers []*TeacherBriefVO（教师数组），对齐前端 teachers: TeacherResponse[]
//   - 新字段 CourseRecord *CourseRecordVO（嵌套课卡对象，指针可为 nil）
//     对齐前端 courseRecord: CourseRecordResponse
//   - 新字段 ScheduleList []ClassScheduleVO（嵌套课表数组，可选）
//     对齐前端 scheduleList?: BackendScheduleItem[]
//
// 嵌套对象填充策略（在 service 层完成）：
//   - Teachers：在 ToClassVOList 中按 class_id 聚合多行 DTO 自动填充
//   - CourseRecord：仅在 GetClassByStudentID 中查询 c_course_record 填充；其他维度为 nil
//   - ScheduleList：所有查询方法均查询 c_class_schedule 填充
type ClassVO struct {
	ID              int64               `json:"id"`              // 班级ID
	ClassName       string              `json:"className"`       // 班级名称
	StudentCount    int64               `json:"studentCount"`    // 班级学生人数
	StudentMaxCount int64               `json:"studentMaxCount"` // 班级最大人数
	CourseID        int64               `json:"courseId"`        // 课程ID
	CourseName      string              `json:"courseName"`      // 课程名称
	CourseType      int64               `json:"courseType"`      // 课程类型
	Status          int64               `json:"status"`          // 班级状态
	Teachers        []*TeacherBriefVO   `json:"teachers"`        // 教师对象数组（替代旧的 TeacherID/TeacherUsername 单值）
	CourseRecord    *CourseRecordVO     `json:"courseRecord"`    // 嵌套课卡对象（按学生维度查询时填充，其他维度为 nil）
	ScheduleList    []*ClassScheduleVO  `json:"scheduleList,omitempty"` // 嵌套课表数组（在 service 层填充）
	CreateTimeStr   string              `json:"createTimeStr"`   // 创建时间字符串
	UpdateTimeStr   string              `json:"updateTimeStr"`   // 更新时间字符串
}

// ToClassVO ClassDTO 转 VO（不含教师聚合）
//
// 注意：本函数仅做单行 DTO → VO 转换，不处理多教师聚合。
// 若 dto 中的 TeacherID/TeacherUsername 有效，会被收集为单个 TeacherBriefVO。
// 对于"一个班级多个教师"的场景（SQL JOIN 返回多行），应使用 ToClassVOList
// 进行按 class_id 聚合，而非本函数。
//
// CourseRecord 和 ScheduleList 在本函数中不填充，由 service 层负责。
func ToClassVO(dto *mapper.ClassDTO) *ClassVO {
	if dto == nil {
		return nil
	}
	vo := &ClassVO{
		ID:              dto.ID,
		ClassName:       dto.ClassName,
		StudentCount:    dto.StudentCount,
		StudentMaxCount: dto.StudentMaxCount,
		CourseID:        dto.CourseID,
		CourseName:      dto.CourseName,
		CourseType:      dto.CourseType,
		Status:          dto.Status,
		CreateTimeStr:   dto.CreateTime,
		UpdateTimeStr:   dto.UpdateTime,
		// Teachers / CourseRecord / ScheduleList 由 ToClassVOList 或 service 层填充
	}
	// 若 DTO 携带教师信息，转为单元素数组（保持向后兼容）
	if dto.TeacherID != 0 {
		vo.Teachers = []*TeacherBriefVO{
			{
				TeacherID: dto.TeacherID,
				Username:  dto.TeacherUsername,
			},
		}
	}
	return vo
}

// ToClassVOList ClassDTO 列表转 VO 列表（按班级 ID 聚合教师）
//
// 由于一个班级可能有多个教师（通过 c_class_teacher 关联），SQL JOIN 后
// 同一班级会返回多行（每行对应一名教师）。本函数按 class_id 聚合，
// 将多行合并为一个 ClassVO，教师列表收集到 Teachers 字段。
//
// 聚合策略：
//  1. 遍历 DTO 列表，按 dto.ID 分组
//  2. 每个 id 的首行作为基础 VO（含班级名称、课程信息等）
//  3. 同一 id 的所有行的教师信息（TeacherID + TeacherUsername）合并到 Teachers 数组
//  4. 教师去重（同一教师可能在多行中出现）
//
// 注意：本函数不填充 CourseRecord 和 ScheduleList，由 service 层在调用后填充。
//
// 参数：
//   - list: ClassDTO 列表（可能含同一班级的多行）
//
// 返回：聚合后的 ClassVO 列表（一班级一项）
func ToClassVOList(list []*mapper.ClassDTO) []*ClassVO {
	// 按 class id 分组聚合
	voMap := make(map[int64]*ClassVO)
	order := make([]int64, 0, len(list)) // 保持原始顺序
	// 教师去重：每个 class id 维护一个已见 teacherID 集合
	seenTeachers := make(map[int64]map[int64]bool)

	for _, dto := range list {
		vo, exists := voMap[dto.ID]
		if !exists {
			// 首次遇到该班级，创建 VO 并填充基础字段
			vo = &ClassVO{
				ID:              dto.ID,
				ClassName:       dto.ClassName,
				StudentCount:    dto.StudentCount,
				StudentMaxCount: dto.StudentMaxCount,
				CourseID:        dto.CourseID,
				CourseName:      dto.CourseName,
				CourseType:      dto.CourseType,
				Status:          dto.Status,
				CreateTimeStr:   dto.CreateTime,
				UpdateTimeStr:   dto.UpdateTime,
				Teachers:        make([]*TeacherBriefVO, 0),
			}
			voMap[dto.ID] = vo
			order = append(order, dto.ID)
			seenTeachers[dto.ID] = make(map[int64]bool)
		}

		// 收集教师信息（去重）
		if dto.TeacherID != 0 && !seenTeachers[dto.ID][dto.TeacherID] {
			seenTeachers[dto.ID][dto.TeacherID] = true
			vo.Teachers = append(vo.Teachers, &TeacherBriefVO{
				TeacherID: dto.TeacherID,
				Username:  dto.TeacherUsername,
			})
		}
	}

	// 按 order 顺序输出聚合结果
	result := make([]*ClassVO, 0, len(order))
	for _, id := range order {
		result = append(result, voMap[id])
	}
	return result
}

// QueryClassVO 班级查询响应包装（对齐前端 ClassListResponse）
type QueryClassVO struct {
	ClassList []*ClassVO `json:"classList"` // 班级列表
	Total     int64      `json:"total"`     // 总数
}

// GetClassByStudentID 按学生ID查班级列表
//
// 对齐 Java ClassServiceImpl.getClassesByStudentId
// 前端期望：data.classList（数组）+ data.total
//
// 嵌套对象填充策略：
//   - Teachers：由 ToClassVOList 自动聚合（多教师按 class_id 合并）
//   - CourseRecord：通过 courseRecordMapper 查询该学生 + 该课程（dto.CourseID）的课卡记录填充
//   - ScheduleList：通过 classScheduleMapper 查询该班级的课表填充
func (s *ClassService) GetClassByStudentID(studentID int64) *response.ResponseDTO {
	if studentID == 0 {
		return response.Fail("学生ID不能为空")
	}

	list, err := s.classMapper.SelectByStudentID(studentID)
	if err != nil {
		log.Printf("查询学生班级列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 聚合教师列表（按 class_id 合并多行）
	voList := ToClassVOList(list)
	// 填充嵌套课表数组（所有维度均填充）
	s.fillScheduleList(voList)
	// 填充嵌套课卡对象（仅学生维度填充，按 studentID + courseID 查询）
	s.fillCourseRecord(voList, studentID)
	return response.Success(&QueryClassVO{
		ClassList: voList,
		Total:     int64(len(voList)),
	})
}

// GetClassByTeacherID 按教师ID查班级列表
//
// 对齐 Java ClassServiceImpl.getClassesByTeacherId
// 前端期望：data.classList（数组）+ data.total
//
// 嵌套对象填充策略：
//   - Teachers：由 ToClassVOList 自动聚合（多教师按 class_id 合并）
//   - CourseRecord：教师维度不填充（nil），因无法确定具体学生
//   - ScheduleList：通过 classScheduleMapper 查询该班级的课表填充
//
// 参数：
//   - teacherID: 教师ID
//   - classStatus: 班级状态（-1 表示不过滤）
//   - keyword: 班级名称关键词
func (s *ClassService) GetClassByTeacherID(teacherID int64, classStatus int64, keyword string) *response.ResponseDTO {
	if teacherID == 0 {
		return response.Fail("教师ID不能为空")
	}

	list, err := s.classMapper.SelectByTeacherID(teacherID, classStatus, keyword)
	if err != nil {
		log.Printf("查询教师班级列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 聚合教师列表 + 填充课表数组
	voList := ToClassVOList(list)
	s.fillScheduleList(voList)
	// CourseRecord 保持 nil（教师维度无法确定学生）
	return response.Success(&QueryClassVO{
		ClassList: voList,
		Total:     int64(len(voList)),
	})
}

// GetClassByInstitutionID 按机构ID查班级列表
//
// 对齐 Java ClassServiceImpl.getClassesByInstitutionId
// 前端期望：data.classList（数组）+ data.total
//
// 嵌套对象填充策略：
//   - Teachers：由 ToClassVOList 自动聚合（多教师按 class_id 合并）
//   - CourseRecord：机构维度不填充（nil），因无法确定具体学生
//   - ScheduleList：通过 classScheduleMapper 查询该班级的课表填充
//
// 参数：
//   - institutionID: 机构ID
//   - classStatus: 班级状态（-1 表示不过滤）
//   - keyword: 班级名称关键词
func (s *ClassService) GetClassByInstitutionID(institutionID int64, classStatus int64, keyword string) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.classMapper.SelectByInstitutionID(institutionID, classStatus, keyword)
	if err != nil {
		log.Printf("查询机构班级列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 聚合教师列表 + 填充课表数组
	voList := ToClassVOList(list)
	s.fillScheduleList(voList)
	// CourseRecord 保持 nil（机构维度无法确定学生）
	return response.Success(&QueryClassVO{
		ClassList: voList,
		Total:     int64(len(voList)),
	})
}

// GetClassByID 按班级ID查班级详情
//
// 对齐 Java ClassServiceImpl.getClassById
// 前端期望：data.classList[0]（单元素数组）
//
// 嵌套对象填充策略：
//   - Teachers：由 ToClassVOList 自动聚合（使用 SelectDTOListByID 返回多行）
//   - CourseRecord：班级ID维度不填充（nil），因无法确定具体学生
//   - ScheduleList：通过 classScheduleMapper 查询该班级的课表填充
func (s *ClassService) GetClassByID(classID int64) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}

	// 使用 SelectDTOListByID 返回多行（含全部教师），由 ToClassVOList 聚合
	list, err := s.classMapper.SelectDTOListByID(classID)
	if err != nil {
		log.Printf("查询班级详情失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if len(list) == 0 {
		return response.Fail("班级不存在")
	}

	// 聚合教师列表 + 填充课表数组
	voList := ToClassVOList(list)
	s.fillScheduleList(voList)
	// CourseRecord 保持 nil（班级ID维度无法确定学生）
	return response.Success(&QueryClassVO{
		ClassList: voList,
		Total:     int64(len(voList)),
	})
}

// ============================================================
// 嵌套对象填充辅助方法
// ============================================================

// fillScheduleList 为 ClassVO 列表填充嵌套课表数组 ScheduleList
//
// 遍历每个 ClassVO，按 vo.ID（班级ID）查询 c_class_schedule 表，
// 将课表 DTO 列表转为 ClassScheduleVO 列表后填充到 vo.ScheduleList。
//
// 错误处理：查询失败不中断主流程（仅记录日志），ScheduleList 保持 nil。
//
// 参数：
//   - voList: 班级 VO 列表
func (s *ClassService) fillScheduleList(voList []*ClassVO) {
	for _, vo := range voList {
		// 按班级ID查询课表列表（含教师 JOIN，可能多行）
		dtoList, err := s.classScheduleMapper.SelectByClassID(vo.ID)
		if err != nil {
			log.Printf("查询班级课表失败 classID=%d: %v", vo.ID, err)
			continue // 不中断主流程
		}
		// 转为 VO 列表（按 schedule id 聚合教师）
		vo.ScheduleList = ToClassScheduleVOList(dtoList)
	}
}

// fillCourseRecord 为 ClassVO 列表填充嵌套课卡对象 CourseRecord
//
// 仅在按学生ID查询时调用：遍历每个 ClassVO，按 (studentID, vo.CourseID)
// 查询 c_course_record 表，将课卡 DTO 转为 *CourseRecordVO 后填充到 vo.CourseRecord。
//
// 错误处理：查询失败或未找到课卡记录不中断主流程（仅记录日志），CourseRecord 保持 nil。
//
// 参数：
//   - voList: 班级 VO 列表
//   - studentID: 学生ID（用于按 studentID + courseID 联合查询课卡记录）
func (s *ClassService) fillCourseRecord(voList []*ClassVO, studentID int64) {
	for _, vo := range voList {
		// 按学生ID + 课程ID查询课卡记录 DTO（含 JOIN 数据）
		dto, err := s.courseRecordMapper.SelectDTOByStudentAndCourse(studentID, vo.CourseID)
		if err != nil {
			log.Printf("查询课卡记录失败 studentID=%d courseID=%d: %v", studentID, vo.CourseID, err)
			continue // 不中断主流程
		}
		if dto == nil {
			// 该学生未购买该课程的课卡，CourseRecord 保持 nil
			continue
		}
		// 转 VO 后填充到 ClassVO.CourseRecord
		vo.CourseRecord = ToCourseRecordVO(dto)
	}
}

// InsertClass 新增班级
//
// 对齐 Java ClassServiceImpl.insertClass
//
// 流程：
//  1. 创建 c_class 记录
//  2. 批量插入 c_class_teacher 关联（如果提供了 teacherIDs）
//  3. 批量插入 c_class_schedule 课表（如果提供了 schedules）
//
// 前端期望：data.classId + data.className
//
// 参数：
//   - className: 班级名称
//   - courseID: 课程ID
//   - maxCount: 班级最大人数
//   - teacherIDs: 关联教师ID列表
//   - schedules: 课表项列表
func (s *ClassService) InsertClass(className string, courseID int64, maxCount int64, teacherIDs []int64, schedules []*mapper.ScheduleItem) *response.ResponseDTO {
	if className == "" {
		return response.Fail("班级名称不能为空")
	}
	if courseID == 0 {
		return response.Fail("课程ID不能为空")
	}

	// 1. 创建班级
	classID, err := s.classMapper.Insert(className, courseID, maxCount)
	if err != nil {
		log.Printf("新增班级失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 批量插入班级-教师关联
	if len(teacherIDs) > 0 {
		_, err := s.classTeacherMapper.InsertBatch(classID, teacherIDs)
		if err != nil {
			log.Printf("插入班级教师关联失败: %v", err)
			// 不阻塞主流程，班级已创建
		}
	}

	// 3. 批量插入课表
	if len(schedules) > 0 {
		_, err := s.classScheduleMapper.InsertBatch(classID, schedules)
		if err != nil {
			log.Printf("插入班级课表失败: %v", err)
			// 不阻塞主流程，班级已创建
		}
	}

	// 返回班级ID和名称（对齐前端 InsertClassResponse）
	return response.Success(&InsertClassVO{
		ClassID:   classID,
		ClassName: className,
	})
}

// InsertClassVO 新增班级响应 VO（对齐前端 InsertClassResponse）
type InsertClassVO struct {
	ClassID   int64  `json:"classId"`   // 新班级ID
	ClassName string `json:"className"` // 班级名称
}

// UpdateClass 更新班级信息
//
// 对齐 Java ClassServiceImpl.updateClassById
//
// 流程：
//  1. 更新 c_class 基础信息（如果 onlyUpdateClassOwn=true 或提供了基础字段）
//  2. 如果提供了 teachers，先删除旧关联再批量插入新关联
//  3. 如果提供了 schedules，先删除旧课表再批量插入新课表
//
// 前端期望：data.result（影响行数）
//
// 参数：
//   - classID: 班级ID
//   - className: 班级名称（空字符串表示不更新）
//   - courseID: 课程ID（0 表示不更新）
//   - maxCount: 最大人数（0 表示不更新）
//   - status: 班级状态（-1 表示不更新）
//   - teacherIDs: 关联教师ID列表（nil 表示不更新教师关联）
//   - schedules: 课表项列表（nil 表示不更新课表）
func (s *ClassService) UpdateClass(classID int64, className string, courseID int64, maxCount int64, status int64, teacherIDs []int64, schedules []*mapper.ScheduleItem) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}

	// 1. 更新班级基础信息
	_, err := s.classMapper.UpdateByID(classID, className, courseID, maxCount, status)
	if err != nil {
		log.Printf("更新班级失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 更新教师关联（先删后增）
	if teacherIDs != nil {
		_, err := s.classTeacherMapper.DeleteByClassID(classID)
		if err != nil {
			log.Printf("删除班级教师关联失败: %v", err)
		}
		if len(teacherIDs) > 0 {
			_, err := s.classTeacherMapper.InsertBatch(classID, teacherIDs)
			if err != nil {
				log.Printf("插入班级教师关联失败: %v", err)
			}
		}
	}

	// 3. 更新课表（先删后增）
	if schedules != nil {
		_, err := s.classScheduleMapper.DeleteByClassID(classID)
		if err != nil {
			log.Printf("删除班级课表失败: %v", err)
		}
		if len(schedules) > 0 {
			_, err := s.classScheduleMapper.InsertBatch(classID, schedules)
			if err != nil {
				log.Printf("插入班级课表失败: %v", err)
			}
		}
	}

	// 返回影响行数（对齐前端 UpdateClassResponse.result）
	return response.Success(&UpdateResultVO{Result: 1})
}

// AddStudentToClass 添加学生到班级
//
// 对齐 Java ClassServiceImpl.addStudentToClass
//
// 流程：
//  1. 批量插入 c_class_student 关联
//  2. 更新班级学生人数（c_class.student_count）
//
// 前端期望：data.result（影响行数）
//
// 参数：
//   - classID: 班级ID
//   - studentIDs: 学生ID列表
func (s *ClassService) AddStudentToClass(classID int64, studentIDs []int64) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}
	if len(studentIDs) == 0 {
		return response.Fail("学生列表不能为空")
	}

	// 1. 批量插入班级-学生关联
	rows, err := s.classStudentMapper.InsertBatch(classID, studentIDs)
	if err != nil {
		log.Printf("添加学生到班级失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 更新班级学生人数
	err = s.classMapper.UpdateStudentCount(classID)
	if err != nil {
		log.Printf("更新班级学生人数失败: %v", err)
	}

	// 返回影响行数（对齐前端 AddStudentToClassResponse.result）
	return response.Success(&UpdateResultVO{Result: rows})
}

// RemoveStudentFromClass 从班级移除学生
//
// 对齐 Java ClassServiceImpl.removeStudentFromClass
//
// 流程：
//  1. 批量删除 c_class_student 关联
//  2. 更新班级学生人数
//
// 前端期望：data.result（影响行数）
//
// 参数：
//   - classID: 班级ID
//   - studentIDs: 学生ID列表
func (s *ClassService) RemoveStudentFromClass(classID int64, studentIDs []int64) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}
	if len(studentIDs) == 0 {
		return response.Fail("学生列表不能为空")
	}

	// 1. 批量删除班级-学生关联
	rows, err := s.classStudentMapper.DeleteBatch(classID, studentIDs)
	if err != nil {
		log.Printf("从班级移除学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 更新班级学生人数
	err = s.classMapper.UpdateStudentCount(classID)
	if err != nil {
		log.Printf("更新班级学生人数失败: %v", err)
	}

	// 返回影响行数（对齐前端 RemoveStudentFromClassResponse.result）
	return response.Success(&UpdateResultVO{Result: rows})
}
