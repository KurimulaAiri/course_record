// Package mapper business-service 数据访问层 - 班级模块
//
// 对齐 Java com.shiroko.mapper.ClazzMapper + ClassStudentMapper + ClassTeacherMapper
//
// 表：
//   - c_class：班级主表
//   - c_class_student：班级-学生关联表
//   - c_class_teacher：班级-教师关联表
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// ClassMapper 班级表操作
// ============================================================

// ClassMapper 班级表 c_class 的 Mapper（对齐 Java ClazzMapper）
type ClassMapper struct {
	db *sql.DB
}

// NewClassMapper 创建 ClassMapper
func NewClassMapper(db *sql.DB) *ClassMapper {
	return &ClassMapper{db: db}
}

// ClassDTO 班级查询结果 DTO（对齐 Java ClassDTO）
//
// 包含班级基础信息 + 课程信息 + 教师列表
// 注意：课卡剩余课时（course_rest_time）不在此 DTO 中，因为机构/教师/班级维度
//      无法确定具体学生，无法 JOIN c_course_record。学生维度如需课卡信息，
//      应通过 /biz/course_record/get_by_student_id 接口单独查询。
type ClassDTO struct {
	ID              int64  `json:"id"`              // 班级ID
	CourseID        int64  `json:"courseId"`        // 课程ID
	ClassName       string `json:"className"`       // 班级名称
	CourseName      string `json:"courseName"`      // 课程名称（JOIN c_course）
	CourseType      int64  `json:"courseType"`      // 课程类型（JOIN c_course）
	Status          int64  `json:"status"`          // 班级状态
	StudentCount    int64  `json:"studentCount"`    // 班级学生人数
	StudentMaxCount int64  `json:"studentMaxCount"` // 班级最大人数
	TeacherID       int64  `json:"teacherId"`       // 教师ID（JOIN c_class_teacher）
	TeacherUsername string `json:"teacherUsername"` // 教师用户名（JOIN c_teacher）
	CreateTime      string `json:"createTime"`      // 创建时间字符串
	UpdateTime      string `json:"updateTime"`      // 更新时间字符串
}

// classSelectColumns 班级查询的公共字段列表
//
// 对齐 Java ClazzMapper.xml：所有查询都不 JOIN c_course_record 表
// （Java 端只有 getClassesByStudentId 查询 cr 字段，但前端 ClassResponse
//	不包含 courseRestTime，所以 Go 端统一不查询）
const classSelectColumns = `
	cl.id AS class_id, cl.course_id, cl.class_name, cl.status, cl.student_count, cl.student_max_count,
	cl.create_time, cl.update_time,
	co.course_name, co.course_type,
	t.id AS teacher_id, t.username AS teacher_username
`

// scanClassDTO 扫描 ClassDTO（处理 NULL 值和聚合教师）
//
// 由于一个班级可能有多个教师，查询结果会出现多行，调用方需要按 class_id 去重聚合
func scanClassDTO(rows *sql.Rows) (*ClassDTO, error) {
	dto := &ClassDTO{}
	var (
		courseID    sql.NullInt64
		className   sql.NullString
		courseName  sql.NullString
		courseType  sql.NullInt64
		status      sql.NullInt64
		studentCount sql.NullInt64
		studentMax  sql.NullInt64
		createTime  sql.NullTime
		updateTime  sql.NullTime
		teacherID   sql.NullInt64
		teacherName sql.NullString
	)
	err := rows.Scan(
		&dto.ID, &courseID, &className, &status, &studentCount, &studentMax,
		&createTime, &updateTime,
		&courseName, &courseType,
		&teacherID, &teacherName,
	)
	if err != nil {
		return nil, fmt.Errorf("扫描班级记录失败: %w", err)
	}
	dto.CourseID = courseID.Int64
	dto.ClassName = className.String
	dto.CourseName = courseName.String
	dto.CourseType = courseType.Int64
	dto.Status = status.Int64
	dto.StudentCount = studentCount.Int64
	dto.StudentMaxCount = studentMax.Int64
	dto.TeacherID = teacherID.Int64
	dto.TeacherUsername = teacherName.String
	dto.CreateTime = entity.FormatTime(createTime)
	dto.UpdateTime = entity.FormatTime(updateTime)
	return dto, nil
}

// SelectByStudentID 按学生ID查班级列表（对齐 Java ClazzMapper.getClassesByStudentId）
//
// 通过 c_class_student 关联 c_class，再 JOIN c_course/c_class_teacher/c_teacher
// 注意：不再 JOIN c_course_record，因为前端 ClassResponse 不包含 courseRestTime 字段
func (m *ClassMapper) SelectByStudentID(studentID int64) ([]*ClassDTO, error) {
	query := `
		SELECT ` + classSelectColumns + `
		FROM c_class_student AS cs
		LEFT JOIN c_class AS cl ON cs.class_id = cl.id
		LEFT JOIN c_course AS co ON cl.course_id = co.id
		LEFT JOIN c_class_teacher AS ct ON ct.class_id = cl.id
		LEFT JOIN c_teacher AS t ON ct.teacher_id = t.id
		WHERE cs.student_id = ?
	`
	rows, err := m.db.Query(query, studentID)
	if err != nil {
		return nil, fmt.Errorf("查询学生班级列表失败: %w", err)
	}
	defer rows.Close()

	return scanClassDTOList(rows)
}

// SelectByTeacherID 按教师ID查班级列表（对齐 Java ClazzMapper.getClassesByTeacherId）
//
// 通过 c_class_teacher 关联 c_class，支持按状态和关键词过滤
func (m *ClassMapper) SelectByTeacherID(teacherID int64, classStatus int64, keyword string) ([]*ClassDTO, error) {
	query := `
		SELECT ` + classSelectColumns + `
		FROM c_class_teacher AS ct_filter
		LEFT JOIN c_class AS cl ON ct_filter.class_id = cl.id
		LEFT JOIN c_course AS co ON cl.course_id = co.id
		LEFT JOIN c_class_teacher AS ct ON ct.class_id = cl.id
		LEFT JOIN c_teacher AS t ON ct.teacher_id = t.id
		WHERE ct_filter.teacher_id = ?
	`
	args := []interface{}{teacherID}
	// 状态过滤（-1 表示不过滤）
	if classStatus != -1 {
		query += " AND cl.status = ?"
		args = append(args, classStatus)
	}
	// 关键词过滤（按班级名称模糊匹配）
	if keyword != "" {
		query += " AND cl.class_name LIKE CONCAT('%', ?, '%')"
		args = append(args, keyword)
	}
	query += " ORDER BY cl.update_time DESC"

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询教师班级列表失败: %w", err)
	}
	defer rows.Close()

	return scanClassDTOList(rows)
}

// SelectByInstitutionID 按机构ID查班级列表（对齐 Java ClazzMapper.getClassesByInstitutionId）
//
// 通过 c_course.institution_id 关联查询，支持按状态和关键词过滤
func (m *ClassMapper) SelectByInstitutionID(institutionID int64, classStatus int64, keyword string) ([]*ClassDTO, error) {
	query := `
		SELECT ` + classSelectColumns + `
		FROM c_class AS cl
		LEFT JOIN c_course AS co ON cl.course_id = co.id
		LEFT JOIN c_class_teacher AS ct ON ct.class_id = cl.id
		LEFT JOIN c_teacher AS t ON ct.teacher_id = t.id
		WHERE co.institution_id = ?
	`
	args := []interface{}{institutionID}
	if classStatus != -1 {
		query += " AND cl.status = ?"
		args = append(args, classStatus)
	}
	if keyword != "" {
		query += " AND cl.class_name LIKE CONCAT('%', ?, '%')"
		args = append(args, keyword)
	}
	query += " ORDER BY cl.update_time DESC"

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询机构班级列表失败: %w", err)
	}
	defer rows.Close()

	return scanClassDTOList(rows)
}

// SelectByID 按班级ID查详情（对齐 Java ClazzMapper.selectByClassId）
//
// 注意：本方法仅返回首行 DTO（不含多教师聚合）。若一个班级有多个教师，
// SQL JOIN 会返回多行，但本方法只返回第一行。需要完整教师列表的场景
// 请使用 SelectDTOListByID。
func (m *ClassMapper) SelectByID(classID int64) (*ClassDTO, error) {
	query := `
		SELECT ` + classSelectColumns + `
		FROM c_class AS cl
		LEFT JOIN c_course AS co ON cl.course_id = co.id
		LEFT JOIN c_class_teacher AS ct ON ct.class_id = cl.id
		LEFT JOIN c_teacher AS t ON ct.teacher_id = t.id
		WHERE cl.id = ?
	`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("查询班级详情失败: %w", err)
	}
	defer rows.Close()

	list, err := scanClassDTOList(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// SelectDTOListByID 按班级ID查 DTO 列表（含全部教师行，用于多教师聚合）
//
// 与 SelectByID 不同，本方法返回 []*ClassDTO（保留全部行），用于 service 层
// 按 class_id 聚合教师列表（ToClassVOList）。一个班级若关联 N 名教师，
// SQL JOIN 会返回 N 行，本方法返回全部 N 行。
//
// 参数：
//   - classID: 班级ID
//
// 返回：ClassDTO 列表（同一班级多行，每行对应一名教师），空列表表示班级不存在
func (m *ClassMapper) SelectDTOListByID(classID int64) ([]*ClassDTO, error) {
	query := `
		SELECT ` + classSelectColumns + `
		FROM c_class AS cl
		LEFT JOIN c_course AS co ON cl.course_id = co.id
		LEFT JOIN c_class_teacher AS ct ON ct.class_id = cl.id
		LEFT JOIN c_teacher AS t ON ct.teacher_id = t.id
		WHERE cl.id = ?
	`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("查询班级详情列表失败: %w", err)
	}
	defer rows.Close()

	return scanClassDTOList(rows)
}

// SelectEntityByID 按主键查班级实体（用于校验班级是否存在）
//
// 返回 c_class 表的基础字段，不含 JOIN 数据
func (m *ClassMapper) SelectEntityByID(id int64) (*entity.Class, error) {
	query := `SELECT id, course_id, class_name, status, student_count, student_max_count, create_time, update_time FROM c_class WHERE id = ?`
	row := m.db.QueryRow(query, id)

	c := &entity.Class{}
	err := row.Scan(
		&c.ID, &c.CourseID, &c.ClassName, &c.Status,
		&c.StudentCount, &c.StudentMaxCount,
		&c.CreateTime, &c.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询班级失败: %w", err)
	}
	return c, nil
}

// Insert 新增班级（对齐 Java ClazzMapper.insert）
//
// 插入 c_class 表，返回自增主键 ID
//
// 参数：
//   - className: 班级名称
//   - courseID: 课程ID
//   - maxCount: 班级最大人数
//
// 返回：班级ID
func (m *ClassMapper) Insert(className string, courseID int64, maxCount int64) (int64, error) {
	query := `INSERT INTO c_class (course_id, class_name, student_max_count, create_time, update_time) VALUES (?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, courseID, className, maxCount)
	if err != nil {
		return 0, fmt.Errorf("新增班级失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取班级ID失败: %w", err)
	}
	return id, nil
}

// UpdateByID 按ID更新班级基础信息（对齐 Java ClazzMapper.updateById）
//
// 仅更新非空字段（className, courseId, maxCount, status）
//
// 参数：
//   - id: 班级ID
//   - className: 班级名称（空字符串表示不更新）
//   - courseID: 课程ID（0 表示不更新）
//   - maxCount: 最大人数（0 表示不更新）
//   - status: 班级状态（-1 表示不更新）
//
// 返回：影响行数
func (m *ClassMapper) UpdateByID(id int64, className string, courseID int64, maxCount int64, status int64) (int64, error) {
	// 动态构建 SET 子句
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if className != "" {
		setParts = append(setParts, "class_name = ?")
		args = append(args, className)
	}
	if courseID != 0 {
		setParts = append(setParts, "course_id = ?")
		args = append(args, courseID)
	}
	if maxCount != 0 {
		setParts = append(setParts, "student_max_count = ?")
		args = append(args, maxCount)
	}
	if status != -1 {
		setParts = append(setParts, "status = ?")
		args = append(args, status)
	}

	query := fmt.Sprintf("UPDATE c_class SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新班级失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// UpdateStudentCount 更新班级学生人数（对齐 Java @UpdateStudentCount 注解逻辑）
//
// 统计 c_class_student 表中该班级的学生数，更新到 c_class.student_count
func (m *ClassMapper) UpdateStudentCount(classID int64) error {
	query := `UPDATE c_class SET student_count = (SELECT COUNT(*) FROM c_class_student WHERE class_id = ?), update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, classID, classID)
	if err != nil {
		return fmt.Errorf("更新班级学生人数失败: %w", err)
	}
	return nil
}

// scanClassDTOList 扫描多行 ClassDTO，保留所有行（不去重）
//
// 由于一个班级可能有多个教师（通过 c_class_teacher 关联），SQL JOIN 后
// 同一班级会返回多行（每行对应一名教师）。本函数返回全部行，由上层
// service（ToClassVOList）按 class_id 聚合教师列表。
//
// 设计变更说明（对齐前端 ClassResponse.teachers 数组结构）：
//   - 旧实现按 class_id 去重，仅保留首个教师 → 前端拿不到完整教师列表
//   - 新实现保留所有行，由 service 层按 class_id 聚合并构造 Teachers 数组
func scanClassDTOList(rows *sql.Rows) ([]*ClassDTO, error) {
	var list []*ClassDTO
	for rows.Next() {
		dto, err := scanClassDTO(rows)
		if err != nil {
			return nil, err
		}
		// 保留所有行，由上层按 class_id 聚合教师列表
		list = append(list, dto)
	}
	return list, nil
}

// ============================================================
// ClassStudentMapper 班级-学生关联表操作
// ============================================================

// ClassStudentMapper 班级-学生关联表 c_class_student 的 Mapper
type ClassStudentMapper struct {
	db *sql.DB
}

// NewClassStudentMapper 创建 ClassStudentMapper
func NewClassStudentMapper(db *sql.DB) *ClassStudentMapper {
	return &ClassStudentMapper{db: db}
}

// InsertBatch 批量插入班级-学生关联（对齐 Java ClassStudentMapper.insertBatch）
//
// 参数：
//   - classID: 班级ID
//   - studentIDs: 学生ID列表
//
// 返回：影响行数
func (m *ClassStudentMapper) InsertBatch(classID int64, studentIDs []int64) (int64, error) {
	if len(studentIDs) == 0 {
		return 0, nil
	}
	// 构建批量插入 SQL：INSERT INTO c_class_student (class_id, student_id, create_time) VALUES (?, ?, NOW()), ...
	placeholders := ""
	args := make([]interface{}, 0, len(studentIDs)*2)
	for i, sid := range studentIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "(?, ?, NOW())"
		args = append(args, classID, sid)
	}
	query := fmt.Sprintf("INSERT INTO c_class_student (class_id, student_id, create_time) VALUES %s", placeholders)
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量插入班级学生关联失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// DeleteBatch 批量删除班级-学生关联（对齐 Java ClassStudentMapper.deleteBatch）
//
// 参数：
//   - classID: 班级ID
//   - studentIDs: 学生ID列表
//
// 返回：影响行数
func (m *ClassStudentMapper) DeleteBatch(classID int64, studentIDs []int64) (int64, error) {
	if len(studentIDs) == 0 {
		return 0, nil
	}
	// 构建 IN 子句占位符
	placeholders := ""
	args := []interface{}{classID}
	for i, sid := range studentIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, sid)
	}
	query := fmt.Sprintf("DELETE FROM c_class_student WHERE class_id = ? AND student_id IN (%s)", placeholders)
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量删除班级学生关联失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// SelectStudentIDsByClassID 按班级ID查所有学生ID（用于按班级扣课）
func (m *ClassStudentMapper) SelectStudentIDsByClassID(classID int64) ([]int64, error) {
	query := `SELECT student_id FROM c_class_student WHERE class_id = ?`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("查询班级学生ID列表失败: %w", err)
	}
	defer rows.Close()

	var list []int64
	for rows.Next() {
		var sid int64
		if err := rows.Scan(&sid); err != nil {
			return nil, fmt.Errorf("扫描学生ID失败: %w", err)
		}
		list = append(list, sid)
	}
	return list, nil
}

// ============================================================
// ClassTeacherMapper 班级-教师关联表操作
// ============================================================

// ClassTeacherMapper 班级-教师关联表 c_class_teacher 的 Mapper
type ClassTeacherMapper struct {
	db *sql.DB
}

// NewClassTeacherMapper 创建 ClassTeacherMapper
func NewClassTeacherMapper(db *sql.DB) *ClassTeacherMapper {
	return &ClassTeacherMapper{db: db}
}

// InsertBatch 批量插入班级-教师关联（对齐 Java ClassTeacherMapper.insertBatch）
//
// 参数：
//   - classID: 班级ID
//   - teacherIDs: 教师ID列表
//
// 返回：影响行数
func (m *ClassTeacherMapper) InsertBatch(classID int64, teacherIDs []int64) (int64, error) {
	if len(teacherIDs) == 0 {
		return 0, nil
	}
	placeholders := ""
	args := make([]interface{}, 0, len(teacherIDs)*2)
	for i, tid := range teacherIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "(?, ?, NOW())"
		args = append(args, classID, tid)
	}
	query := fmt.Sprintf("INSERT INTO c_class_teacher (class_id, teacher_id, create_time) VALUES %s", placeholders)
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("批量插入班级教师关联失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// DeleteByClassID 按班级ID删除所有教师关联（对齐 Java ClassTeacherMapper.deleteByClassId）
//
// 用于更新班级时的"先删后增"策略
func (m *ClassTeacherMapper) DeleteByClassID(classID int64) (int64, error) {
	query := `DELETE FROM c_class_teacher WHERE class_id = ?`
	result, err := m.db.Exec(query, classID)
	if err != nil {
		return 0, fmt.Errorf("删除班级教师关联失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// CountByTeacherID 统计教师关联的班级数（用于删除教师前校验）
func (m *ClassTeacherMapper) CountByTeacherID(teacherID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM c_class_teacher WHERE teacher_id = ?`
	var count int64
	err := m.db.QueryRow(query, teacherID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计教师班级数失败: %w", err)
	}
	return count, nil
}

// ============================================================
// 辅助函数
// ============================================================

// joinStrings 连接字符串切片（避免引入 strings 包的简易实现）
func joinStrings(parts []string, sep string) string {
	result := ""
	for i, s := range parts {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
