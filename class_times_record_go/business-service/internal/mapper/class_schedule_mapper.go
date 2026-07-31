// Package mapper business-service 数据访问层 - 班级课表模块
//
// 对齐 Java com.shiroko.mapper.ClassScheduleMapper
//
// 表：c_class_schedule（班级排班日程表）
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// ClassScheduleMapper 班级课表操作
// ============================================================

// ClassScheduleMapper 班级课表 c_class_schedule 的 Mapper
type ClassScheduleMapper struct {
	db *sql.DB
}

// NewClassScheduleMapper 创建 ClassScheduleMapper
func NewClassScheduleMapper(db *sql.DB) *ClassScheduleMapper {
	return &ClassScheduleMapper{db: db}
}

// ClassScheduleDTO 课表查询结果 DTO（对齐 Java ClassScheduleDTO）
//
// 包含课表基础信息 + 班级名称 + 教师列表（通过 c_class_teacher 关联）
type ClassScheduleDTO struct {
	ID            int64  `json:"id"`            // 课表ID
	ClassID       int64  `json:"classId"`       // 班级ID
	ClassName     string `json:"className"`     // 班级名称（JOIN c_class）
	DayOfWeek     int64  `json:"dayOfWeek"`     // 上课时间（1-7代表周一到周日）
	StartDate     string `json:"startDate"`     // 开始日期
	EndDate       string `json:"endDate"`       // 结束日期
	StartTime     string `json:"startTime"`     // 开始时间
	EndTime       string `json:"endTime"`       // 结束时间
	Remark        string `json:"remark"`        // 备注
	TeacherID     int64  `json:"teacherId"`     // 教师ID
	TeacherName   string `json:"teacherName"`   // 教师用户名
	CreateTime    string `json:"createTime"`    // 创建时间
	UpdateTime    string `json:"updateTime"`    // 更新时间
}

// classScheduleSelectColumns 课表查询的公共字段列表
const classScheduleSelectColumns = `
	cs.id, cs.class_id, cl.class_name, cs.day_of_week, cs.start_date, cs.end_date,
	cs.start_time, cs.end_time, cs.remark, cs.create_time, cs.update_time,
	t.id AS tid, t.username AS teacher_name
`

// scanClassScheduleDTO 扫描 ClassScheduleDTO
func scanClassScheduleDTO(rows *sql.Rows) (*ClassScheduleDTO, error) {
	dto := &ClassScheduleDTO{}
	var (
		classID    sql.NullInt64
		className  sql.NullString
		dayOfWeek  sql.NullInt64
		startDate  sql.NullTime
		endDate    sql.NullTime
		startTime  sql.NullTime
		endTime    sql.NullTime
		remark     sql.NullString
		createTime sql.NullTime
		updateTime sql.NullTime
		teacherID  sql.NullInt64
		teacherName sql.NullString
	)
	err := rows.Scan(
		&dto.ID, &classID, &className, &dayOfWeek, &startDate, &endDate,
		&startTime, &endTime, &remark, &createTime, &updateTime,
		&teacherID, &teacherName,
	)
	if err != nil {
		return nil, fmt.Errorf("扫描课表记录失败: %w", err)
	}
	dto.ClassID = classID.Int64
	dto.ClassName = className.String
	dto.DayOfWeek = dayOfWeek.Int64
	// 日期字段格式化为 YYYY-MM-DD，时间字段格式化为 HH:MM
	dto.StartDate = formatDate(startDate)
	dto.EndDate = formatDate(endDate)
	dto.StartTime = formatTimeOnly(startTime)
	dto.EndTime = formatTimeOnly(endTime)
	dto.Remark = remark.String
	dto.TeacherID = teacherID.Int64
	dto.TeacherName = teacherName.String
	dto.CreateTime = entity.FormatTime(createTime)
	dto.UpdateTime = entity.FormatTime(updateTime)
	return dto, nil
}

// SelectByClassID 按班级ID查课表列表（对齐 Java ClassScheduleMapper 按 classId 分页查询）
func (m *ClassScheduleMapper) SelectByClassID(classID int64) ([]*ClassScheduleDTO, error) {
	query := `
		SELECT ` + classScheduleSelectColumns + `
		FROM c_class_schedule cs
		LEFT JOIN c_class cl ON cs.class_id = cl.id
		LEFT JOIN c_class_teacher ct ON cl.id = ct.class_id
		LEFT JOIN c_teacher t ON ct.teacher_id = t.id
		WHERE cs.class_id = ?
		ORDER BY cs.id DESC
	`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("查询班级课表失败: %w", err)
	}
	defer rows.Close()

	return scanClassScheduleDTOList(rows)
}

// SelectByInstitutionID 按机构ID查课表列表（对齐 Java selectClassScheduleByInstitutionId）
//
// 通过 c_class.course_id → c_course.institution_id 关联查询
func (m *ClassScheduleMapper) SelectByInstitutionID(institutionID int64) ([]*ClassScheduleDTO, error) {
	query := `
		SELECT ` + classScheduleSelectColumns + `
		FROM c_class_schedule cs
		LEFT JOIN c_class cl ON cs.class_id = cl.id
		LEFT JOIN c_class_teacher ct ON cl.id = ct.class_id
		LEFT JOIN c_teacher t ON ct.teacher_id = t.id
		LEFT JOIN c_course co ON cl.course_id = co.id
		WHERE co.institution_id = ?
		ORDER BY cs.id DESC
	`
	rows, err := m.db.Query(query, institutionID)
	if err != nil {
		return nil, fmt.Errorf("查询机构课表失败: %w", err)
	}
	defer rows.Close()

	return scanClassScheduleDTOList(rows)
}

// SelectByTeacherID 按教师ID查课表列表（对齐 Java selectClassScheduleByTeacherId）
//
// 通过 c_class_teacher 关联查询教师的所有课表
func (m *ClassScheduleMapper) SelectByTeacherID(teacherID int64) ([]*ClassScheduleDTO, error) {
	query := `
		SELECT ` + classScheduleSelectColumns + `
		FROM c_class_schedule cs
		LEFT JOIN c_class cl ON cs.class_id = cl.id
		LEFT JOIN c_class_teacher ct ON cl.id = ct.class_id
		LEFT JOIN c_teacher t ON ct.teacher_id = t.id
		WHERE ct.teacher_id = ?
		ORDER BY cs.day_of_week ASC, cs.start_time ASC
	`
	rows, err := m.db.Query(query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("查询教师课表失败: %w", err)
	}
	defer rows.Close()

	return scanClassScheduleDTOList(rows)
}

// SelectByID 按主键查课表详情（对齐 Java ClassScheduleMapper.selectById）
//
// 返回 c_class_schedule 表基础字段，不含 JOIN 数据（教师列表/班级名称）
// 用于不需要教师信息的场景；如需教师信息请使用 SelectDTOByID
func (m *ClassScheduleMapper) SelectByID(id int64) (*entity.ClassSchedule, error) {
	query := `SELECT id, class_id, start_date, end_date, day_of_week, start_time, end_time, remark, create_time, update_time FROM c_class_schedule WHERE id = ?`
	row := m.db.QueryRow(query, id)

	cs := &entity.ClassSchedule{}
	err := row.Scan(
		&cs.ID, &cs.ClassID, &cs.StartDate, &cs.EndDate,
		&cs.DayOfWeek, &cs.StartTime, &cs.EndTime, &cs.Remark,
		&cs.CreateTime, &cs.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课表失败: %w", err)
	}
	return cs, nil
}

// SelectDTOByID 按主键查课表 DTO（含班级名称和教师信息）
//
// 与 SelectByID 不同，本方法通过 LEFT JOIN c_class + c_class_teacher + c_teacher
// 返回完整的 ClassScheduleDTO，包含教师信息（可能多行，每个教师一行）。
// 用于 GetClassScheduleByID 接口，需要展示教师列表的场景。
//
// 参数：
//   - id: 课表ID
//
// 返回：ClassScheduleDTO 切片（按 id 聚合后通常只有一项，但若班级有多个教师则有多行）
func (m *ClassScheduleMapper) SelectDTOByID(id int64) ([]*ClassScheduleDTO, error) {
	query := `
		SELECT ` + classScheduleSelectColumns + `
		FROM c_class_schedule cs
		LEFT JOIN c_class cl ON cs.class_id = cl.id
		LEFT JOIN c_class_teacher ct ON cl.id = ct.class_id
		LEFT JOIN c_teacher t ON ct.teacher_id = t.id
		WHERE cs.id = ?
	`
	rows, err := m.db.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("查询课表详情失败: %w", err)
	}
	defer rows.Close()

	return scanClassScheduleDTOList(rows)
}

// InsertBatch 批量插入课表（对齐 Java ClassScheduleMapper.insertBatch）
//
// 参数：
//   - classID: 班级ID
//   - schedules: 课表项列表（dayOfWeek, startDate, endDate, startTime, endTime, remark）
//
// 返回：影响行数
type ScheduleItem struct {
	DayOfWeek int64
	StartDate string
	EndDate   string
	StartTime string
	EndTime   string
	Remark    string
}

// InsertBatch 批量插入课表记录
func (m *ClassScheduleMapper) InsertBatch(classID int64, items []*ScheduleItem) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	placeholders := ""
	args := make([]interface{}, 0, len(items)*6)
	for i, item := range items {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "(?, ?, ?, ?, ?, ?, ?)"
		args = append(args, classID, item.DayOfWeek, item.StartDate, item.EndDate, item.StartTime, item.EndTime, item.Remark)
	}
	query := fmt.Sprintf("INSERT INTO c_class_schedule (class_id, day_of_week, start_date, end_date, start_time, end_time, remark) VALUES %s", placeholders)
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量插入课表失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// DeleteByClassID 按班级ID删除所有课表（对齐 Java ClassScheduleMapper.deleteByClassId）
func (m *ClassScheduleMapper) DeleteByClassID(classID int64) (int64, error) {
	query := `DELETE FROM c_class_schedule WHERE class_id = ?`
	result, err := m.db.Exec(query, classID)
	if err != nil {
		return 0, fmt.Errorf("删除班级课表失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// UpdateByID 按ID更新课表（对齐 Java ClassScheduleMapper.updateById）
//
// 参数：
//   - id: 课表ID
//   - dayOfWeek: 星期几（0 表示不更新）
//   - startTime: 开始时间（空字符串表示不更新）
//   - endTime: 结束时间（空字符串表示不更新）
//   - remark: 备注（空字符串表示不更新）
//
// 返回：影响行数
func (m *ClassScheduleMapper) UpdateByID(id int64, dayOfWeek int64, startDate, endDate, startTime, endTime, remark string) (int64, error) {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if dayOfWeek != 0 {
		setParts = append(setParts, "day_of_week = ?")
		args = append(args, dayOfWeek)
	}
	if startDate != "" {
		setParts = append(setParts, "start_date = ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		setParts = append(setParts, "end_date = ?")
		args = append(args, endDate)
	}
	if startTime != "" {
		setParts = append(setParts, "start_time = ?")
		args = append(args, startTime)
	}
	if endTime != "" {
		setParts = append(setParts, "end_time = ?")
		args = append(args, endTime)
	}
	if remark != "" {
		setParts = append(setParts, "remark = ?")
		args = append(args, remark)
	}

	query := fmt.Sprintf("UPDATE c_class_schedule SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新课表失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// scanClassScheduleDTOList 扫描多行 ClassScheduleDTO，保留所有行（不去重）
//
// 由于一个课表所属的班级可能有多个教师（通过 c_class_teacher 关联），
// SQL JOIN 后同一课表会返回多行（每行对应一名教师）。本函数返回全部行，
// 由上层 service（ToClassScheduleVOList）按 schedule id 聚合教师列表。
//
// 设计变更说明（对齐前端 ClassScheduleResponse.teachers 数组结构）：
//   - 旧实现按 id 去重，仅保留首个教师 → 前端拿不到完整教师列表
//   - 新实现保留所有行，由 service 层按 id 聚合并构造 Teachers 数组
func scanClassScheduleDTOList(rows *sql.Rows) ([]*ClassScheduleDTO, error) {
	var list []*ClassScheduleDTO
	for rows.Next() {
		dto, err := scanClassScheduleDTO(rows)
		if err != nil {
			return nil, err
		}
		// 保留所有行，由上层按 schedule id 聚合教师列表
		list = append(list, dto)
	}
	return list, nil
}

// formatDate 格式化日期为 YYYY-MM-DD
func formatDate(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02")
}

// formatTimeOnly 格式化时间为 HH:MM:SS 或 HH:MM
func formatTimeOnly(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("15:04:05")
}
