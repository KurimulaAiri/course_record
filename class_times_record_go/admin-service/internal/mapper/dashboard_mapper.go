// Package mapper admin-service 仪表盘统计 Mapper
//
// 对齐 Java admin-service SysDashboardServiceImpl
// 使用 JdbcTemplate 风格的查询，仅返回基础统计结果
//
// 涵盖接口：
//   - 仪表盘汇总数据（学生/教师/机构/课程/班级 总数）
//   - 趋势数据（按天或按月统计新增学生/教师数量）
//   - 机构统计（按机构分组统计学生/教师/课程/班级数量）
package mapper

import (
	"database/sql"
	"fmt"
	"time"
)

// ============================================================
// VO 定义（对齐 Java DashboardVO / DashboardTrendVO / InstitutionStatVO）
// ============================================================

// DashboardRow 仪表盘汇总数据行（对齐 Java DashboardVO）
type DashboardRow struct {
	StudentCount     int64 `json:"studentCount"`     // 学生总数
	TeacherCount     int64 `json:"teacherCount"`     // 教师总数
	InstitutionCount int64 `json:"institutionCount"` // 机构总数
	CourseCount      int64 `json:"courseCount"`      // 课程总数
	ClassCount       int64 `json:"classCount"`       // 班级总数
}

// DashboardTrendRow 仪表盘趋势数据行（对齐 Java DashboardTrendVO）
type DashboardTrendRow struct {
	Months           []string `json:"months"`           // 时间刻度字符串数组（对齐前端 DashboardTrendResponse.months）
	NewStudentCounts []int64  `json:"newStudentCounts"` // 各刻度对应的新增学生数
	NewTeacherCounts []int64  `json:"newTeacherCounts"` // 各刻度对应的新增教师数
}

// InstitutionStatRow 机构统计行（对齐 Java InstitutionStatVO）
type InstitutionStatRow struct {
	InstitutionID   int64  `json:"institutionId"`   // 机构ID（对齐前端 InstitutionStatResponse.institutionId）
	InstitutionName string `json:"institutionName"` // 机构名称
	StudentCount    int64  `json:"studentCount"`    // 学生数
	TeacherCount    int64  `json:"teacherCount"`    // 教师数
	CourseCount     int64  `json:"courseCount"`     // 课程数
	ClassCount      int64  `json:"classCount"`      // 班级数
}

// ============================================================
// DashboardMapper 仪表盘 Mapper
// ============================================================

// DashboardMapper 仪表盘统计 Mapper
//
// 对齐 Java SysDashboardServiceImpl 使用 JdbcTemplate 直接查询
type DashboardMapper struct {
	db *sql.DB
}

// NewDashboardMapper 创建 DashboardMapper
func NewDashboardMapper(db *sql.DB) *DashboardMapper {
	return &DashboardMapper{db: db}
}

// CountTable 统计指定表的总数
//
// 对齐 Java countTable 方法
//
// 参数：
//   - tableName: 表名（如 "c_student"）
//
// 返回：表总记录数（查询失败返回 0）
func (m *DashboardMapper) CountTable(tableName string) int64 {
	// tableName 由代码内部传入，非用户输入，无 SQL 注入风险
	var count int64
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if err := m.db.QueryRow(query).Scan(&count); err != nil {
		return 0
	}
	return count
}

// GetDashboardData 获取仪表盘汇总数据
//
// 对齐 Java getDashboardData
//
// 返回：学生/教师/机构/课程/班级 总数
func (m *DashboardMapper) GetDashboardData() (*DashboardRow, error) {
	row := &DashboardRow{
		StudentCount:     m.CountTable("c_student"),
		TeacherCount:     m.CountTable("c_teacher"),
		InstitutionCount: m.CountTable("c_institution"),
		CourseCount:      m.CountTable("c_course"),
		ClassCount:       m.CountTable("c_class"),
	}
	return row, nil
}

// countByPeriod 按时间刻度统计新增数量
//
// 对齐 Java countByPeriod 方法
//
// 参数：
//   - sql: 查询语句（含 GROUP BY 和 ? 占位符）
//   - startDate: 起始日期字符串（yyyy-MM-dd）
//
// 返回：刻度label -> count 映射
func (m *DashboardMapper) countByPeriod(query, startDate string) map[string]int64 {
	result := make(map[string]int64)
	rows, err := m.db.Query(query, startDate)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var count int64
		// DATE_FORMAT 结果可能为 NULL（旧数据无 create_time），使用 sql.NullString 兜底
		var p sql.NullString
		if err := rows.Scan(&p, &count); err != nil {
			continue
		}
		if p.Valid {
			result[p.String] = count
		}
	}
	return result
}

// GetTrend 获取趋势数据
//
// 对齐 Java getTrend
// 根据时间范围决定粒度与刻度：
//   - week/month: 按天统计（7天/30天）
//   - halfyear/year: 按月统计（6个月/12个月）
//
// 参数：
//   - rangeStr: 时间范围（"week"/"month"/"halfyear"/"year"，默认 "year"）
func (m *DashboardMapper) GetTrend(rangeStr string) (*DashboardTrendRow, error) {
	// 根据时间范围决定粒度：week/month 用天，halfyear/year 用月
	daily := rangeStr == "week" || rangeStr == "month"

	var labels []string
	var startDateStr string
	var sqlFmt string

	now := time.Now()
	if daily {
		// 按天统计
		days := 7
		if rangeStr == "month" {
			days = 30
		}
		// 起始日期 = 今天 - (days - 1)
		start := now.AddDate(0, 0, -(days - 1))
		for i := 0; i < days; i++ {
			labels = append(labels, start.AddDate(0, 0, i).Format("2006-01-02"))
		}
		startDateStr = start.Format("2006-01-02")
		sqlFmt = "%Y-%m-%d"
	} else {
		// 按月统计
		months := 6
		if rangeStr == "year" {
			months = 12
		}
		// 起始月份 = 本月第一天 - (months - 1) 个月
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -(months - 1), 0)
		for i := 0; i < months; i++ {
			labels = append(labels, start.AddDate(0, i, 0).Format("2006-01"))
		}
		startDateStr = start.Format("2006-01-02")
		sqlFmt = "%Y-%m"
	}

	// 统计新增学生数（按 create_time 分组）
	studentSQL := fmt.Sprintf(
		"SELECT DATE_FORMAT(create_time, '%s') AS p, COUNT(*) AS c FROM c_student WHERE create_time >= ? GROUP BY p",
		sqlFmt,
	)
	studentMap := m.countByPeriod(studentSQL, startDateStr)

	// 统计新增教师数（教师表无 create_time，JOIN c_user 取创建时间）
	teacherSQL := fmt.Sprintf(
		"SELECT DATE_FORMAT(u.create_time, '%s') AS p, COUNT(*) AS c FROM c_teacher t JOIN c_user u ON t.user_id = u.id WHERE u.create_time >= ? GROUP BY p",
		sqlFmt,
	)
	teacherMap := m.countByPeriod(teacherSQL, startDateStr)

	// 按刻度顺序填充统计值，缺失刻度补 0
	newStudentCounts := make([]int64, len(labels))
	newTeacherCounts := make([]int64, len(labels))
	for i, label := range labels {
		newStudentCounts[i] = studentMap[label] // map 中不存在时返回 0
		newTeacherCounts[i] = teacherMap[label]
	}

	return &DashboardTrendRow{
		Months:           labels,
		NewStudentCounts: newStudentCounts,
		NewTeacherCounts: newTeacherCounts,
	}, nil
}

// countGroupByInstitution 按机构分组统计
//
// 对齐 Java countGroupByInstitution 方法
//
// 参数：
//   - query: 查询语句（返回 institution_id 和 count）
//
// 返回：institutionId -> count 映射
func (m *DashboardMapper) countGroupByInstitution(query string) map[int64]int64 {
	result := make(map[int64]int64)
	rows, err := m.db.Query(query)
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var count int64
		// institution_id 可能为 NULL（数据不完整），使用 sql.NullInt64 兜底
		var nullIID sql.NullInt64
		if err := rows.Scan(&nullIID, &count); err != nil {
			continue
		}
		if nullIID.Valid {
			result[nullIID.Int64] = count
		}
	}
	return result
}

// GetInstitutionStats 获取机构统计列表
//
// 对齐 Java getInstitutionStats
// 查询所有机构（id > 0），并统计每个机构的学生/教师/课程/班级数量
// 结果按学生数降序排序，支持 limit 限制返回数量
//
// 参数：
//   - limit: 限制返回数量（<=0 表示不限制）
func (m *DashboardMapper) GetInstitutionStats(limit int) ([]*InstitutionStatRow, error) {
	// 1. 查询所有机构基础信息（过滤 id <= 0 的占位数据）
	instRows, err := m.db.Query(`SELECT id, institution_name FROM c_institution WHERE id > 0 ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("查询机构列表失败: %w", err)
	}
	defer instRows.Close()

	type instInfo struct {
		ID   int64
		Name string
	}
	var institutions []instInfo
	for instRows.Next() {
		var info instInfo
		var name sql.NullString
		if err := instRows.Scan(&info.ID, &name); err != nil {
			continue
		}
		info.Name = name.String
		institutions = append(institutions, info)
	}

	// 2. 按机构分组统计各实体数量
	studentMap := m.countGroupByInstitution(
		"SELECT institution_id AS iid, COUNT(*) AS c FROM c_student GROUP BY institution_id")
	teacherMap := m.countGroupByInstitution(
		"SELECT institution_id AS iid, COUNT(*) AS c FROM c_teacher GROUP BY institution_id")
	courseMap := m.countGroupByInstitution(
		"SELECT institution_id AS iid, COUNT(*) AS c FROM c_course GROUP BY institution_id")
	// 班级表无 institution_id，通过 JOIN c_course 获取机构ID
	classMap := m.countGroupByInstitution(
		"SELECT co.institution_id AS iid, COUNT(*) AS c FROM c_class c JOIN c_course co ON c.course_id = co.id GROUP BY co.institution_id")

	// 3. 组装结果
	result := make([]*InstitutionStatRow, 0, len(institutions))
	for _, inst := range institutions {
		result = append(result, &InstitutionStatRow{
			InstitutionID:   inst.ID,
			InstitutionName: inst.Name,
			StudentCount:    studentMap[inst.ID],
			TeacherCount:    teacherMap[inst.ID],
			CourseCount:     courseMap[inst.ID],
			ClassCount:      classMap[inst.ID],
		})
	}

	// 4. 按学生数降序排序（便于 Top-N 展示）
	// 使用简单的冒泡排序，机构数量通常较少
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].StudentCount > result[i].StudentCount {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	// 5. 应用 limit 限制
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
