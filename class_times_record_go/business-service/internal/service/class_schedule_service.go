// Package service business-service 业务逻辑层 - 班级课表模块
//
// 对齐 Java business-service ClassScheduleServiceImpl
//
// 核心功能：
//   - 课表查询（按班级ID/机构ID/教师ID/课表ID）
//   - 课表更新（按ID更新单条课表）
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// ClassScheduleService 班级课表服务
// ============================================================

// ClassScheduleService 班级课表服务（对齐 Java ClassScheduleServiceImpl）
//
// 查询：按班级ID/机构ID/教师ID/课表ID
// 写操作：按ID更新课表
type ClassScheduleService struct {
	classScheduleMapper *mapper.ClassScheduleMapper
}

// NewClassScheduleService 创建 ClassScheduleService
func NewClassScheduleService(classScheduleMapper *mapper.ClassScheduleMapper) *ClassScheduleService {
	return &ClassScheduleService{classScheduleMapper: classScheduleMapper}
}

// ClassScheduleVO 课表视图对象（对齐前端 ClassScheduleResponse）
//
// 字段命名对齐前端 src/types/class-schedule.d.ts
//
// 重构说明（从扁平字段改为嵌套对象）：
//   - 旧字段 TeacherID/TeacherName（单值）已移除
//   - 新字段 Teachers []*TeacherBriefVO（教师数组），对齐前端 teachers: TeacherResponse[]
//   - 新字段 Classroom string（教室，对齐前端 classroom 字段；当前数据库 c_class_schedule
//     表暂无此字段，返回空字符串）
//   - 新字段 Color string（颜色，对齐前端 color 字段；当前数据库表暂无此字段，返回空字符串）
type ClassScheduleVO struct {
	ID            int64             `json:"id"`            // 课表ID
	ClassID       int64             `json:"classId"`       // 班级ID
	ClassName     string            `json:"className"`     // 班级名称（JOIN c_class）
	Classroom     string            `json:"classroom"`     // 教室（数据库暂无此字段，固定为空字符串）
	DayOfWeek     int64             `json:"dayOfWeek"`     // 上课时间（1-7代表周一到周日）
	StartDateStr  string            `json:"startDateStr"`  // 开始日期字符串（YYYY-MM-DD）
	EndDateStr    string            `json:"endDateStr"`    // 结束日期字符串（YYYY-MM-DD）
	StartTimeStr  string            `json:"startTimeStr"`  // 开始时间字符串（HH:MM:SS）
	EndTimeStr    string            `json:"endTimeStr"`    // 结束时间字符串（HH:MM:SS）
	Teachers      []*TeacherBriefVO `json:"teachers"`      // 教师对象数组（替代旧的 TeacherID/TeacherName 单值）
	Remark        string            `json:"remark"`        // 备注
	Color         string            `json:"color"`         // 颜色标识（数据库暂无此字段，固定为空字符串）
	CreateTimeStr string            `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string            `json:"updateTimeStr"` // 更新时间字符串
}

// ToClassScheduleVO ClassScheduleDTO 转 VO（不含教师聚合）
//
// 注意：本函数仅做单行 DTO → VO 转换，不处理多教师聚合。
// 若 dto 中的 TeacherID/TeacherName 有效，会被收集为单个 TeacherBriefVO。
// 对于"一个课表多个教师"的场景（SQL JOIN 返回多行），应使用 ToClassScheduleVOList
// 进行按 id 聚合，而非本函数。
func ToClassScheduleVO(dto *mapper.ClassScheduleDTO) *ClassScheduleVO {
	if dto == nil {
		return nil
	}
	vo := &ClassScheduleVO{
		ID:            dto.ID,
		ClassID:       dto.ClassID,
		ClassName:     dto.ClassName,
		Classroom:     "", // 数据库 c_class_schedule 表暂无 classroom 字段
		DayOfWeek:     dto.DayOfWeek,
		StartDateStr:  dto.StartDate,
		EndDateStr:    dto.EndDate,
		StartTimeStr:  dto.StartTime,
		EndTimeStr:    dto.EndTime,
		Remark:        dto.Remark,
		Color:         "", // 数据库 c_class_schedule 表暂无 color 字段
		CreateTimeStr: dto.CreateTime,
		UpdateTimeStr: dto.UpdateTime,
	}
	// 若 DTO 携带教师信息，转为单元素数组（保持向后兼容）
	if dto.TeacherID != 0 {
		vo.Teachers = []*TeacherBriefVO{
			{
				TeacherID: dto.TeacherID,
				Username:  dto.TeacherName,
			},
		}
	}
	return vo
}

// ToClassScheduleVOList ClassScheduleDTO 列表转 VO 列表（按课表 id 聚合教师）
//
// 由于一个课表所属班级可能有多个教师（通过 c_class_teacher 关联），
// SQL JOIN 后同一课表会返回多行（每行对应一名教师）。本函数按课表 id
// 聚合，将多行合并为一个 ClassScheduleVO，教师列表收集到 Teachers 字段。
//
// 聚合策略：
//  1. 遍历 DTO 列表，按 dto.ID 分组
//  2. 每个 id 的首行作为基础 VO（含班级名称、日期、时间等）
//  3. 同一 id 的所有行的教师信息（TeacherID + TeacherName）合并到 Teachers 数组
//  4. 教师去重（同一教师可能在多行中出现）
//
// 参数：
//   - list: ClassScheduleDTO 列表（可能含同一课表的多行）
//
// 返回：聚合后的 ClassScheduleVO 列表（一课表一项）
func ToClassScheduleVOList(list []*mapper.ClassScheduleDTO) []*ClassScheduleVO {
	// 按 schedule id 分组聚合
	voMap := make(map[int64]*ClassScheduleVO)
	order := make([]int64, 0, len(list)) // 保持原始顺序
	// 教师去重：每个 schedule id 维护一个已见 teacherID 集合
	seenTeachers := make(map[int64]map[int64]bool)

	for _, dto := range list {
		vo, exists := voMap[dto.ID]
		if !exists {
			// 首次遇到该课表，创建 VO 并填充基础字段
			vo = &ClassScheduleVO{
				ID:            dto.ID,
				ClassID:       dto.ClassID,
				ClassName:     dto.ClassName,
				Classroom:     "", // 数据库暂无 classroom 字段
				DayOfWeek:     dto.DayOfWeek,
				StartDateStr:  dto.StartDate,
				EndDateStr:    dto.EndDate,
				StartTimeStr:  dto.StartTime,
				EndTimeStr:    dto.EndTime,
				Remark:        dto.Remark,
				Color:         "", // 数据库暂无 color 字段
				CreateTimeStr: dto.CreateTime,
				UpdateTimeStr: dto.UpdateTime,
				Teachers:      make([]*TeacherBriefVO, 0),
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
				Username:  dto.TeacherName,
			})
		}
	}

	// 按 order 顺序输出聚合结果
	result := make([]*ClassScheduleVO, 0, len(order))
	for _, id := range order {
		result = append(result, voMap[id])
	}
	return result
}

// QueryClassScheduleVO 课表查询响应包装（对齐前端 ClassScheduleListResponse）
type QueryClassScheduleVO struct {
	ClassSchedules []*ClassScheduleVO `json:"classSchedules"` // 课表列表
	Total          int64              `json:"total"`          // 总数
}

// GetClassScheduleByClassID 按班级ID查课表列表
//
// 对齐 Java ClassScheduleServiceImpl.getByClassId
// 前端期望：data.classSchedules（数组）+ data.total
func (s *ClassScheduleService) GetClassScheduleByClassID(classID int64) *response.ResponseDTO {
	if classID == 0 {
		return response.Fail("班级ID不能为空")
	}

	list, err := s.classScheduleMapper.SelectByClassID(classID)
	if err != nil {
		log.Printf("查询班级课表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToClassScheduleVOList(list)
	return response.Success(&QueryClassScheduleVO{
		ClassSchedules: voList,
		Total:          int64(len(voList)),
	})
}

// GetClassScheduleByInstitutionID 按机构ID查课表列表
//
// 对齐 Java ClassScheduleServiceImpl.getByInstitutionId
// 前端期望：data.classSchedules（数组）+ data.total
func (s *ClassScheduleService) GetClassScheduleByInstitutionID(institutionID int64) *response.ResponseDTO {
	if institutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	list, err := s.classScheduleMapper.SelectByInstitutionID(institutionID)
	if err != nil {
		log.Printf("查询机构课表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToClassScheduleVOList(list)
	return response.Success(&QueryClassScheduleVO{
		ClassSchedules: voList,
		Total:          int64(len(voList)),
	})
}

// GetClassScheduleByTeacherID 按教师ID查课表列表
//
// 对齐 Java ClassScheduleServiceImpl.getByTeacherId
// 前端期望：data.classSchedules（数组）+ data.total
func (s *ClassScheduleService) GetClassScheduleByTeacherID(teacherID int64) *response.ResponseDTO {
	if teacherID == 0 {
		return response.Fail("教师ID不能为空")
	}

	list, err := s.classScheduleMapper.SelectByTeacherID(teacherID)
	if err != nil {
		log.Printf("查询教师课表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToClassScheduleVOList(list)
	return response.Success(&QueryClassScheduleVO{
		ClassSchedules: voList,
		Total:          int64(len(voList)),
	})
}

// GetClassScheduleByID 按课表ID查课表详情
//
// 对齐 Java ClassScheduleServiceImpl.getById
// 前端期望：data.classSchedules[0]（单元素数组）
//
// 实现说明（重构后）：
//   - 旧实现使用 SelectByID 返回 *entity.ClassSchedule，仅含基础字段，
//     手工构造 VO 时缺失 ClassName/StartDate/EndDate/StartTime/EndTime/Teachers
//   - 新实现使用 SelectDTOByID 返回 []*ClassScheduleDTO（含 JOIN 教师信息），
//     通过 ToClassScheduleVOList 聚合教师列表，VO 字段完整
//
// 参数：
//   - scheduleID: 课表ID
func (s *ClassScheduleService) GetClassScheduleByID(scheduleID int64) *response.ResponseDTO {
	if scheduleID == 0 {
		return response.Fail("课表ID不能为空")
	}

	// 按ID查课表 DTO（含教师 JOIN 信息）
	list, err := s.classScheduleMapper.SelectDTOByID(scheduleID)
	if err != nil {
		log.Printf("查询课表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if len(list) == 0 {
		return response.Fail("课表不存在")
	}

	// 转为 VO 列表（按 id 聚合教师，结果应只有一项）
	voList := ToClassScheduleVOList(list)
	return response.Success(&QueryClassScheduleVO{
		ClassSchedules: voList,
		Total:          int64(len(voList)),
	})
}

// UpdateClassScheduleByID 按ID更新课表
//
// 对齐 Java ClassScheduleServiceImpl.updateById
//
// 前端期望：data.classSchedules（更新后的课表列表）
//
// 参数：
//   - scheduleID: 课表ID
//   - dayOfWeek: 星期几（0 表示不更新）
//   - startTime: 开始时间（空字符串表示不更新）
//   - endTime: 结束时间（空字符串表示不更新）
//   - remark: 备注（空字符串表示不更新）
func (s *ClassScheduleService) UpdateClassScheduleByID(scheduleID int64, dayOfWeek int64, startDate, endDate, startTime, endTime, remark string) *response.ResponseDTO {
	if scheduleID == 0 {
		return response.Fail("课表ID不能为空")
	}

	// 更新课表
	_, err := s.classScheduleMapper.UpdateByID(scheduleID, dayOfWeek, startDate, endDate, startTime, endTime, remark)
	if err != nil {
		log.Printf("更新课表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 返回更新后的课表（对齐前端 UpdateClassScheduleResponse）
	// 简化处理：返回空列表，前端通常只关心 code=200
	return response.Success(&QueryClassScheduleVO{
		ClassSchedules: []*ClassScheduleVO{},
		Total:          0,
	})
}
