// Package service business-service 业务逻辑层 - 课程模块
//
// 对齐 Java business-service CourseServiceImpl
//
// 核心功能：
//   - 课程查询（按机构ID/学生ID）
//   - 课程新增
//   - 课程更新
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// CourseService 课程服务
// ============================================================

// CourseService 课程服务（对齐 Java CourseServiceImpl）
//
// 查询：按机构ID/学生ID
// 写操作：新增/更新课程
//
// 重构说明（嵌套对象化）：
//   - 新增 courseRecordMapper 依赖：用于按学生ID查课程时填充 CourseVO.CurrentStudentCourseRecord 嵌套对象
type CourseService struct {
	courseMapper        *mapper.CourseMapper
	courseRecordMapper  *mapper.CourseRecordMapper // 课卡记录 Mapper（按学生维度查询时填充 CurrentStudentCourseRecord）
}

// NewCourseService 创建 CourseService
//
// 参数：
//   - courseMapper: 课程表 Mapper
//   - courseRecordMapper: 课卡记录 Mapper（用于按学生ID查课程时填充 CurrentStudentCourseRecord 嵌套对象）
func NewCourseService(courseMapper *mapper.CourseMapper, courseRecordMapper *mapper.CourseRecordMapper) *CourseService {
	return &CourseService{
		courseMapper:       courseMapper,
		courseRecordMapper: courseRecordMapper,
	}
}

// CourseVO 课程视图对象（对齐前端 CourseResponse）
//
// 字段命名对齐前端 src/types/course.d.ts
//
// 重构说明（从扁平字段改为嵌套对象）：
//   - 旧字段 InstitutionID（扁平机构ID）已移除，改为 Institution InstitutionBriefVO 嵌套机构对象
//     对齐前端 institution: InstitutionResponse
//   - 旧字段 CourseRecordID/CourseRestTime/CourseTotalTime/ExpireTimeStr（扁平课卡信息）已移除
//     改为 CurrentStudentCourseRecord *CourseRecordVO 嵌套课卡对象（指针，可为 nil）
//     对齐前端 currentStudentCourseRecord?: CourseRecordResponse
//   - CurrentStudentCourseRecord 仅在 GetCourseByStudentID 中填充（按学生维度查询时），
//     其他查询维度（如 GetCourseByInstitutionID）中为 nil（omitempty 序列化时省略）
type CourseVO struct {
	ID                          int64              `json:"id"`                          // 课程ID
	CourseName                  string             `json:"courseName"`                  // 课程名称
	CourseType                  int64              `json:"courseType"`                  // 课程类型（1=按次, 2=按天）
	IsAvailable                 bool               `json:"isAvailable"`                 // 是否可用
	Institution                 InstitutionBriefVO `json:"institution"`                 // 嵌套机构对象（对齐前端 institution: InstitutionResponse）
	CurrentStudentCourseRecord  *CourseRecordVO    `json:"currentStudentCourseRecord,omitempty"` // 当前学生的课卡记录（仅按学生查询时填充，对齐前端 currentStudentCourseRecord?）
	UpdateTimeStr               string             `json:"updateTimeStr"`               // 更新时间字符串
	CreateTimeStr               string             `json:"createTimeStr"`               // 创建时间字符串
}

// ToCourseVO CourseDTO 转 VO（对齐前端 CourseResponse）
//
// 转换逻辑：
//  1. 将 DTO 的扁平字段映射到 VO 的对应字段
//  2. 从 DTO 的机构字段（InstitutionID/InstitutionName/InstitutionCode）构造嵌套 Institution InstitutionBriefVO
//  3. CurrentStudentCourseRecord 在本函数中不填充（保持 nil），由 service 层按查询维度决定是否填充
func ToCourseVO(dto *mapper.CourseDTO) *CourseVO {
	if dto == nil {
		return nil
	}
	return &CourseVO{
		ID:            dto.ID,
		CourseName:    dto.CourseName,
		CourseType:    dto.CourseType,
		IsAvailable:   dto.IsAvailable,
		UpdateTimeStr: dto.UpdateTime,
		CreateTimeStr: dto.CreateTime,
		// 构造嵌套机构对象（InstitutionBriefVO），从 DTO 的 JOIN 字段填充
		Institution: InstitutionBriefVO{
			ID:              dto.InstitutionID,
			InstitutionName: dto.InstitutionName,
			InstitutionCode: dto.InstitutionCode,
		},
		// CurrentStudentCourseRecord 由 service 层填充（仅按学生查询时），此处保持 nil
	}
}

// ToCourseVOList CourseDTO 列表转 VO 列表
func ToCourseVOList(list []*mapper.CourseDTO) []*CourseVO {
	result := make([]*CourseVO, 0, len(list))
	for _, dto := range list {
		if vo := ToCourseVO(dto); vo != nil {
			result = append(result, vo)
		}
	}
	return result
}

// QueryCourseVO 课程查询响应包装（对齐前端 CourseListResponse）
type QueryCourseVO struct {
	Courses []*CourseVO `json:"courses"` // 课程列表
	Total   int64       `json:"total"`   // 总数
}

// GetCourseByInstitutionID 按机构ID查课程列表
//
// 对齐 Java CourseServiceImpl.getCourseByInstitutionId
// 前端期望：data.courses（数组）+ data.total
//
// 重构说明（嵌套对象化）：
//   - CurrentStudentCourseRecord 保持 nil（按机构维度查询不关联具体学生）
//     omitempty 序列化时该字段被省略，对齐前端 currentStudentCourseRecord?（可选字段）
//
// 参数：
//   - institutionID: 机构ID
//   - keyword: 课程名称关键词
func (s *CourseService) GetCourseByInstitutionID(institutionID int64, keyword string) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.courseMapper.SelectByInstitutionID(institutionID, keyword)
	if err != nil {
		log.Printf("查询机构课程列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToCourseVOList(list)
	return response.Success(&QueryCourseVO{
		Courses: voList,
		Total:   int64(len(voList)),
	})
}

// GetCourseByStudentID 按学生ID查课程列表
//
// 对齐 Java CourseServiceImpl.getCourseByStudentId
// 前端期望：data.courses（数组）+ data.total
//
// 重构说明（嵌套对象化）：
//   - 查询课程列表后，遍历每个课程，按 (studentID, courseID) 查询 c_course_record
//   - 将课卡 DTO 转为 *CourseRecordVO 后填充到 CourseVO.CurrentStudentCourseRecord
//   - 对齐前端 CourseResponse.currentStudentCourseRecord?: CourseRecordResponse
//   - 查询失败或未找到课卡记录不中断主流程（仅记录日志），CurrentStudentCourseRecord 保持 nil
func (s *CourseService) GetCourseByStudentID(studentID int64) *response.ResponseDTO {
	if studentID == 0 {
		return response.Fail("学生ID不能为空")
	}

	list, err := s.courseMapper.SelectByStudentID(studentID)
	if err != nil {
		log.Printf("查询学生课程列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToCourseVOList(list)
	// 为每个课程填充 CurrentStudentCourseRecord 嵌套对象（按学生维度查询时）
	s.fillCurrentStudentCourseRecord(voList, studentID)
	return response.Success(&QueryCourseVO{
		Courses: voList,
		Total:   int64(len(voList)),
	})
}

// fillCurrentStudentCourseRecord 为 CourseVO 列表填充当前学生的课卡记录嵌套对象
//
// 仅在按学生ID查询课程时调用：遍历每个 CourseVO，按 (studentID, vo.ID)
// 查询 c_course_record 表，将课卡 DTO 转为 *CourseRecordVO 后填充到 vo.CurrentStudentCourseRecord。
//
// 错误处理：查询失败或未找到课卡记录不中断主流程（仅记录日志），CurrentStudentCourseRecord 保持 nil。
//
// 参数：
//   - voList: 课程 VO 列表
//   - studentID: 学生ID（用于按 studentID + courseID 联合查询课卡记录）
func (s *CourseService) fillCurrentStudentCourseRecord(voList []*CourseVO, studentID int64) {
	for _, vo := range voList {
		// 按学生ID + 课程ID查询课卡记录 DTO（含 JOIN 数据）
		dto, err := s.courseRecordMapper.SelectDTOByStudentAndCourse(studentID, vo.ID)
		if err != nil {
			log.Printf("查询课卡记录失败 studentID=%d courseID=%d: %v", studentID, vo.ID, err)
			continue // 不中断主流程
		}
		if dto == nil {
			// 该学生未购买该课程的课卡，CurrentStudentCourseRecord 保持 nil
			continue
		}
		// 转 VO 后填充到 CourseVO.CurrentStudentCourseRecord
		vo.CurrentStudentCourseRecord = ToCourseRecordVO(dto)
	}
}

// InsertCourse 新增课程
//
// 对齐 Java CourseServiceImpl.insertCourse
//
// 前端期望：data.courseId（新课程ID）
//
// 参数：
//   - courseName: 课程名称
//   - courseType: 课程类型（1=按次, 2=按天）
//   - institutionID: 机构ID
//   - isAvailable: 是否可用
func (s *CourseService) InsertCourse(courseName string, courseType int64, institutionID int64, isAvailable bool) *response.ResponseDTO {
	if courseName == "" {
		return response.Fail("课程名称不能为空")
	}
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	courseID, err := s.courseMapper.Insert(courseName, courseType, institutionID, isAvailable)
	if err != nil {
		log.Printf("新增课程失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回新课程ID（对齐前端 InsertCourseResponse）
	return response.Success(&InsertCourseVO{CourseID: courseID})
}

// InsertCourseVO 新增课程响应 VO（对齐前端 InsertCourseResponse）
type InsertCourseVO struct {
	CourseID int64 `json:"courseId"` // 新课程ID
}

// UpdateCourse 更新课程信息
//
// 对齐 Java CourseServiceImpl.updateCourse
//
// 前端期望：data.effect（影响行数）
//
// 参数：
//   - id: 课程ID
//   - courseName: 课程名称（空字符串表示不更新）
//   - courseType: 课程类型（0 表示不更新）
//   - isAvailable: 是否可用（nil 表示不更新）
func (s *CourseService) UpdateCourse(id int64, courseName string, courseType int64, isAvailable *bool) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("课程ID不能为空")
	}

	rows, err := s.courseMapper.UpdateByID(id, courseName, courseType, isAvailable)
	if err != nil {
		log.Printf("更新课程失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回影响行数（对齐前端 UpdateCourseResponse.effect）
	return response.Success(&UpdateCourseVO{Effect: rows})
}

// UpdateCourseVO 更新课程响应 VO（对齐前端 UpdateCourseResponse）
type UpdateCourseVO struct {
	Effect int64 `json:"effect"` // 影响行数
}
