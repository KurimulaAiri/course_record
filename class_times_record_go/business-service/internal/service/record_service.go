// Package service business-service 业务逻辑层 - 上课记录模块
//
// 对齐 Java business-service RecordServiceImpl
//
// 核心功能：
//   - 上课记录查询（按机构/学生/课程名/记录类型分页）
//   - 新增单条上课记录
//   - 批量新增上课记录（同时更新课卡剩余课时）
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/business-service/internal/mapper"
)

// ============================================================
// RecordService 上课记录服务
// ============================================================

// RecordService 上课记录服务（对齐 Java RecordServiceImpl）
//
// 查询：按机构/学生/课程名/记录类型分页
// 写操作：新增单条/批量新增（同时更新课卡剩余课时）
type RecordService struct {
	recordMapper       *mapper.RecordMapper
	courseRecordMapper *mapper.CourseRecordMapper
}

// NewRecordService 创建 RecordService
//
// 参数：
//   - recordMapper: 上课记录 Mapper
//   - courseRecordMapper: 课卡记录 Mapper（批量新增时更新剩余课时）
func NewRecordService(
	recordMapper *mapper.RecordMapper,
	courseRecordMapper *mapper.CourseRecordMapper,
) *RecordService {
	return &RecordService{
		recordMapper:       recordMapper,
		courseRecordMapper: courseRecordMapper,
	}
}

// RecordVO 上课记录视图对象（对齐前端 RecordResponse）
//
// 字段命名对齐前端 src/types/record.d.ts
//
// 重构说明（从扁平字段改为嵌套对象）：
//   - 旧字段 CourseID/StudentID/CourseName/CourseType/StudentName/TeacherName/CourseRestTime（扁平 JOIN 字段）已移除
//   - 新字段 CourseRecord CourseRecordVO（嵌套课卡对象），对齐前端 courseRecord: CourseRecordResponse
//   - 新字段 Student StudentBriefVO（嵌套学生对象），对齐前端 student: StudentResponse
//   - 新字段 Course CourseVO（嵌套课程对象），对齐前端 course: CourseResponse
//   - 新字段 OperatorTeacher TeacherBriefVO（嵌套教师对象），对齐前端 operatorTeacher: TeacherResponse
//
// 嵌套对象填充策略：
//   - 全部 4 个嵌套对象在 ToRecordVO 中从 DTO 的 JOIN 字段构造（一次 SQL 查询获取全部数据）
//   - CourseVO.CurrentStudentCourseRecord 保持 nil（上课记录场景不需要该字段，omitempty 省略）
type RecordVO struct {
	ID                  int64          `json:"id"`                  // 记录ID
	CourseRecordID      int64          `json:"courseRecordId"`      // 课卡记录ID
	RecordTimeStr       string         `json:"recordTimeStr"`       // 记录时间字符串
	RecordRemark        string         `json:"recordRemark"`        // 备注
	RecordType          int64          `json:"recordType"`          // 记录类型（1=增加, 2=减少）
	RecordChange        int64          `json:"recordChange"`        // 课时变更数量
	RestTimeAfterDeduct int64          `json:"restTimeAfterDeduct"` // 扣费后剩余课时
	DeductMode          string         `json:"deductMode"`          // 扣费模式（BY_STUDENT/BY_COURSE/BY_CLASS）
	ClassID             int64          `json:"classId"`             // 班级ID（按班级扣费时有值）
	CreateTimeStr       string         `json:"createTimeStr"`       // 创建时间字符串
	UpdateTimeStr       string         `json:"updateTimeStr"`       // 更新时间字符串
	CourseRecord        CourseRecordVO `json:"courseRecord"`        // 嵌套课卡对象（对齐前端 courseRecord: CourseRecordResponse）
	Student             StudentBriefVO `json:"student"`             // 嵌套学生对象（对齐前端 student: StudentResponse）
	Course              CourseVO       `json:"course"`              // 嵌套课程对象（对齐前端 course: CourseResponse）
	OperatorTeacher     TeacherBriefVO `json:"operatorTeacher"`     // 嵌套教师对象（对齐前端 operatorTeacher: TeacherResponse）
}

// ToRecordVO RecordDTO 转 VO
//
// 转换逻辑：
//  1. 将 DTO 的扁平字段映射到 VO 的对应字段
//  2. 从 DTO 的课卡字段构造嵌套 CourseRecord CourseRecordVO
//     （包含嵌套 Course CourseBriefVO，由 ToCourseRecordVO 内部构造）
//  3. 从 DTO 的学生字段构造嵌套 Student StudentBriefVO
//  4. 从 DTO 的课程字段构造嵌套 Course CourseVO（CurrentStudentCourseRecord 保持 nil）
//  5. 从 DTO 的教师字段构造嵌套 OperatorTeacher TeacherBriefVO
func ToRecordVO(dto *mapper.RecordDTO) *RecordVO {
	if dto == nil {
		return nil
	}
	return &RecordVO{
		ID:                  dto.ID,
		CourseRecordID:      dto.CourseRecordID,
		RecordTimeStr:       dto.RecordTime,
		RecordRemark:        dto.RecordRemark,
		RecordType:          dto.RecordType,
		RecordChange:        dto.RecordChange,
		RestTimeAfterDeduct: dto.RestTimeAfterDeduct,
		DeductMode:          dto.DeductMode,
		ClassID:             dto.ClassID,
		CreateTimeStr:       dto.CreateTime,
		UpdateTimeStr:       dto.UpdateTime,
		// 构造嵌套课卡对象（CourseRecordVO），从 DTO 的课卡 JOIN 字段填充
		// ToCourseRecordVO 内部会构造嵌套 Course CourseBriefVO 并计算 ExpireStatus
		CourseRecord: CourseRecordVO{
			ID:                dto.CourseRecordID,
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
			CreateTimeStr:     dto.CourseRecordCreateTime,
			UpdateTimeStr:     dto.CourseRecordUpdateTime,
			// 构造课卡内嵌的课程对象（CourseBriefVO）
			Course: CourseBriefVO{
				ID:          dto.CourseID,
				CourseName:  dto.CourseName,
				CourseType:  dto.CourseType,
				IsAvailable: dto.IsAvailable,
			},
			// 根据 expire_time 原始值计算过期状态
			ExpireStatus: calcCourseRecordExpireStatus(dto.ExpireTimeRaw),
		},
		// 构造嵌套学生对象（StudentBriefVO）
		Student: StudentBriefVO{
			ID:          dto.StudentID,
			StudentName: dto.StudentName,
			Sex:         dto.Sex,
			Avatar:      dto.Avatar,
		},
		// 构造嵌套课程对象（CourseVO，CurrentStudentCourseRecord 保持 nil）
		Course: CourseVO{
			ID:            dto.CourseID,
			CourseName:    dto.CourseName,
			CourseType:    dto.CourseType,
			IsAvailable:   dto.IsAvailable,
			UpdateTimeStr: dto.CourseCreateTime,
			CreateTimeStr: dto.CourseUpdateTime,
			// 构造课程内嵌的机构对象（InstitutionBriefVO）
			Institution: InstitutionBriefVO{
				ID:              dto.InstitutionID,
				InstitutionName: dto.InstitutionName,
				InstitutionCode: dto.InstitutionCode,
			},
			// CurrentStudentCourseRecord 保持 nil（上课记录场景不需要）
		},
		// 构造嵌套教师对象（TeacherBriefVO）
		OperatorTeacher: TeacherBriefVO{
			TeacherID: dto.OperateTeacherID,
			Username:  dto.TeacherName,
		},
	}
}

// ToRecordVOList RecordDTO 列表转 VO 列表
func ToRecordVOList(list []*mapper.RecordDTO) []*RecordVO {
	result := make([]*RecordVO, 0, len(list))
	for _, dto := range list {
		if vo := ToRecordVO(dto); vo != nil {
			result = append(result, vo)
		}
	}
	return result
}

// QueryRecordVO 上课记录查询响应包装（对齐前端 RecordListResponse）
type QueryRecordVO struct {
	Records []*RecordVO `json:"records"` // 记录列表
	Total   int64       `json:"total"`   // 总数
}

// NewGetRecord 查询上课记录列表（对齐 Java RecordServiceImpl.newGetRecord）
//
// 前端期望：data.records（数组）+ data.total
//
// 参数：
//   - institutionID: 机构ID（0 表示不过滤）
//   - studentID: 学生ID（0 表示不过滤）
//   - courseRecordID: 课卡记录ID（0 表示不过滤）
//   - courseName: 课程名称关键词
//   - recordType: 记录类型（0 表示不过滤，1=增加, 2=减少）
//   - currentPage: 当前页码
//   - pageSize: 每页条数
func (s *RecordService) NewGetRecord(institutionID, studentID, courseRecordID int64, courseName string, recordType int64, currentPage, pageSize int) *response.ResponseDTO {
	list, total, err := s.recordMapper.SelectList(institutionID, studentID, courseRecordID, courseName, recordType, currentPage, pageSize)
	if err != nil {
		log.Printf("查询上课记录列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := ToRecordVOList(list)
	return response.Success(&QueryRecordVO{
		Records: voList,
		Total:   total,
	})
}

// InsertRecord 新增单条上课记录（对齐 Java RecordServiceImpl.insertRecord）
//
// 前端期望：data 为成功消息字符串
//
// 参数：
//   - courseRecordID: 课卡记录ID
//   - recordTime: 记录时间（格式 yyyy-MM-dd HH:mm:ss，空字符串表示当前时间）
//   - recordType: 记录类型（1=增加, 2=减少）
//   - recordChange: 课时变更数量
//   - recordRemark: 备注
func (s *RecordService) InsertRecord(courseRecordID int64, recordTime string, recordType, recordChange int64, recordRemark string) *response.ResponseDTO {
	if courseRecordID == 0 {
		return response.Fail("课卡记录ID不能为空")
	}
	if recordType == 0 {
		return response.Fail("记录类型不能为空")
	}

	_, err := s.recordMapper.Insert(
		courseRecordID, // 课卡记录ID
		recordTime,     // 记录时间
		0,              // 操作教师ID（手动新增无操作教师）
		recordRemark,   // 备注
		recordType,     // 记录类型
		recordChange,   // 课时变更数量
		0,              // 扣费后剩余课时（手动新增不记录快照）
		"",             // 扣费模式
		0,              // 班级ID
	)
	if err != nil {
		log.Printf("新增上课记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	return response.Success("插入成功")
}

// InsertRecords 批量新增上课记录（对齐 Java RecordServiceImpl.insertRecords）
//
// 流程：
//  1. 遍历 courseRecordIDList，为每个课卡插入一条记录
//  2. 按 recordType 更新课卡剩余课时：
//     - recordType=1（消课/减少）：course_rest_time -= change，更新 course_last_time
//     - recordType=2（增加）：course_rest_time += change，course_total_time += change
//
// 前端期望：data 为成功消息字符串
//
// 参数：
//   - courseRecordIDs: 课卡记录ID列表
//   - recordTime: 记录时间
//   - recordType: 记录类型（1=消课, 2=增加）
//   - recordChange: 课时变更数量
//   - recordRemark: 备注
func (s *RecordService) InsertRecords(courseRecordIDs []int64, recordTime string, recordType, recordChange int64, recordRemark string) *response.ResponseDTO {
	if len(courseRecordIDs) == 0 {
		return response.Fail("课卡记录ID列表不能为空")
	}
	if recordType == 0 {
		return response.Fail("记录类型不能为空")
	}

	successCount := int64(0)
	// 遍历每个课卡记录ID，插入记录并更新课时
	for _, courseRecordID := range courseRecordIDs {
		// 1. 插入上课记录
		_, err := s.recordMapper.Insert(
			courseRecordID, // 课卡记录ID
			recordTime,     // 记录时间
			0,              // 操作教师ID
			recordRemark,   // 备注
			recordType,     // 记录类型
			recordChange,   // 课时变更数量
			0,              // 扣费后剩余课时
			"",             // 扣费模式
			0,              // 班级ID
		)
		if err != nil {
			log.Printf("批量新增上课记录失败（courseRecordID=%d）: %v", courseRecordID, err)
			continue // 跳过失败的记录，继续处理下一个
		}
		successCount++

		// 2. 查询当前课卡记录获取旧的剩余课时和总课时
		oldCR, err := s.courseRecordMapper.SelectByID(courseRecordID)
		if err != nil {
			log.Printf("查询课卡记录失败（courseRecordID=%d）: %v", courseRecordID, err)
			continue
		}
		if oldCR == nil {
			log.Printf("课卡记录不存在（courseRecordID=%d）", courseRecordID)
			continue
		}

		// 3. 按 recordType 更新课卡剩余课时（对齐 Java insertRecords 逻辑）
		// Java 原始逻辑：type=1 为消课（减课时），type=2 为增加课时
		var newRestTime, newTotalTime int64
		var lastTime string
		if recordType == 1 {
			// 消课：剩余课时减少，更新上次上课时间
			newRestTime = oldCR.CourseRestTime.Int64 - recordChange
			newTotalTime = 0 // 不更新总课时
			lastTime = recordTime
		} else if recordType == 2 {
			// 增加课时：剩余课时和总课时都增加
			newRestTime = oldCR.CourseRestTime.Int64 + recordChange
			newTotalTime = oldCR.CourseTotalTime.Int64 + recordChange
			lastTime = "" // 增加课时不更新上次上课时间
		}

		_, err = s.courseRecordMapper.UpdateRestAndTotalByID(courseRecordID, newRestTime, newTotalTime, lastTime)
		if err != nil {
			log.Printf("更新课卡剩余课时失败（courseRecordID=%d）: %v", courseRecordID, err)
		}
	}

	if successCount > 0 {
		return response.Success("插入成功，影响行数为" + int64ToString(successCount))
	}
	return response.Fail("插入失败")
}

// int64ToString 将 int64 转为字符串（简易实现，避免引入 strconv）
func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
