// Package mapper business-service 数据访问层 - 课程模块
//
// 对齐 Java com.shiroko.mapper.CourseMapper
//
// 表：c_course（课程表）
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// CourseMapper 课程表操作
// ============================================================

// CourseMapper 课程表 c_course 的 Mapper
type CourseMapper struct {
	db *sql.DB
}

// NewCourseMapper 创建 CourseMapper
func NewCourseMapper(db *sql.DB) *CourseMapper {
	return &CourseMapper{db: db}
}

// CourseDTO 课程查询结果 DTO（对齐 Java CourseDTO）
//
// 包含课程基础信息 + 机构信息 + 学生课卡信息（按学生查询时）
type CourseDTO struct {
	ID             int64  `json:"id"`             // 课程ID
	CourseName     string `json:"courseName"`     // 课程名称
	CourseType     int64  `json:"courseType"`     // 课程类型（1=按次, 2=按天）
	IsAvailable    bool   `json:"isAvailable"`    // 是否可用
	InstitutionID  int64  `json:"institutionId"`  // 机构ID
	InstitutionName string `json:"institutionName"` // 机构名称
	InstitutionAddress string `json:"institutionAddress"` // 机构地址
	InstitutionCode string `json:"institutionCode"` // 机构编码
	Status         int64  `json:"status"`         // 机构状态
	CourseRecordID int64  `json:"courseRecordId"` // 课卡记录ID（按学生查询时）
	CourseRestTime int64  `json:"courseRestTime"` // 剩余课时（按学生查询时）
	CourseTotalTime int64 `json:"courseTotalTime"` // 总课时（按学生查询时）
	ExpireTime     string `json:"expireTime"`     // 过期时间（按学生查询时）
	CreateTime     string `json:"createTime"`     // 创建时间
	UpdateTime     string `json:"updateTime"`     // 更新时间
}

// SelectByInstitutionID 按机构ID查课程列表（对齐 Java CourseMapper.selectCourseByInstitutionId）
//
// 参数：
//   - institutionID: 机构ID
//   - keyword: 课程名称关键词（空字符串表示不过滤）
//
// 重构说明：SQL 增加 i.institution_code 列，用于填充 CourseVO.Institution.InstitutionCode
// （对齐前端 CourseResponse.institution.institutionCode 字段）
func (m *CourseMapper) SelectByInstitutionID(institutionID int64, keyword string) ([]*CourseDTO, error) {
	query := `
		SELECT c.id, c.course_name, c.course_type, c.is_available,
		       i.id AS institution_id, i.institution_name, i.institution_address, i.institution_code, i.status,
		       c.create_time, c.update_time
		FROM c_course AS c
		LEFT JOIN c_institution AS i ON c.institution_id = i.id
		WHERE c.institution_id = ?
	`
	args := []interface{}{institutionID}
	if keyword != "" {
		query += " AND c.course_name LIKE CONCAT('%', ?, '%')"
		args = append(args, keyword)
	}
	query += " ORDER BY c.id DESC"

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询机构课程列表失败: %w", err)
	}
	defer rows.Close()

	var list []*CourseDTO
	for rows.Next() {
		dto := &CourseDTO{}
		var (
			courseName     sql.NullString
			courseType     sql.NullInt64
			isAvailable    sql.NullBool
			instID         sql.NullInt64
			instName       sql.NullString
			instAddress    sql.NullString
			instCode       sql.NullString
			status         sql.NullInt64
			createTime     sql.NullTime
			updateTime     sql.NullTime
		)
		err := rows.Scan(
			&dto.ID, &courseName, &courseType, &isAvailable,
			&instID, &instName, &instAddress, &instCode, &status,
			&createTime, &updateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描课程记录失败: %w", err)
		}
		dto.CourseName = courseName.String
		dto.CourseType = courseType.Int64
		dto.IsAvailable = isAvailable.Bool
		dto.InstitutionID = instID.Int64
		dto.InstitutionName = instName.String
		dto.InstitutionAddress = instAddress.String
		dto.InstitutionCode = instCode.String
		dto.Status = status.Int64
		dto.CreateTime = entity.FormatTime(createTime)
		dto.UpdateTime = entity.FormatTime(updateTime)
		list = append(list, dto)
	}
	return list, nil
}

// SelectByStudentID 按学生ID查课程列表（对齐 Java CourseMapper.selectCourseByStudentId）
//
// 通过 c_course_record 关联查询学生报名的课程
func (m *CourseMapper) SelectByStudentID(studentID int64) ([]*CourseDTO, error) {
	query := `
		SELECT c.id, c.course_name, c.course_type, c.is_available,
		       cr.id AS course_record_id, cr.course_rest_time, cr.course_total_time, cr.expire_time,
		       i.id AS institution_id, i.institution_name, i.institution_address, i.institution_code, i.status,
		       c.create_time, c.update_time
		FROM c_course AS c
		LEFT JOIN c_institution AS i ON c.institution_id = i.id
		LEFT JOIN c_course_record AS cr ON c.id = cr.course_id
		WHERE cr.student_id = ? AND cr.is_delete = 0
		ORDER BY c.id DESC
	`
	rows, err := m.db.Query(query, studentID)
	if err != nil {
		return nil, fmt.Errorf("查询学生课程列表失败: %w", err)
	}
	defer rows.Close()

	var list []*CourseDTO
	for rows.Next() {
		dto := &CourseDTO{}
		var (
			courseName     sql.NullString
			courseType     sql.NullInt64
			isAvailable    sql.NullBool
			crID           sql.NullInt64
			crRestTime     sql.NullInt64
			crTotalTime    sql.NullInt64
			expireTime     sql.NullTime
			instID         sql.NullInt64
			instName       sql.NullString
			instAddress    sql.NullString
			instCode       sql.NullString
			status         sql.NullInt64
			createTime     sql.NullTime
			updateTime     sql.NullTime
		)
		err := rows.Scan(
			&dto.ID, &courseName, &courseType, &isAvailable,
			&crID, &crRestTime, &crTotalTime, &expireTime,
			&instID, &instName, &instAddress, &instCode, &status,
			&createTime, &updateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学生课程记录失败: %w", err)
		}
		dto.CourseName = courseName.String
		dto.CourseType = courseType.Int64
		dto.IsAvailable = isAvailable.Bool
		dto.CourseRecordID = crID.Int64
		dto.CourseRestTime = crRestTime.Int64
		dto.CourseTotalTime = crTotalTime.Int64
		dto.ExpireTime = entity.FormatTime(expireTime)
		dto.InstitutionID = instID.Int64
		dto.InstitutionName = instName.String
		dto.InstitutionAddress = instAddress.String
		dto.InstitutionCode = instCode.String
		dto.Status = status.Int64
		dto.CreateTime = entity.FormatTime(createTime)
		dto.UpdateTime = entity.FormatTime(updateTime)
		list = append(list, dto)
	}
	return list, nil
}

// SelectEntityByID 按主键查课程实体（用于扣课校验和课程名查询）
func (m *CourseMapper) SelectEntityByID(id int64) (*entity.Course, error) {
	query := `SELECT id, course_name, course_type, institution_id, is_available, create_time, update_time FROM c_course WHERE id = ?`
	row := m.db.QueryRow(query, id)

	c := &entity.Course{}
	err := row.Scan(
		&c.ID, &c.CourseName, &c.CourseType, &c.InstitutionID,
		&c.IsAvailable, &c.CreateTime, &c.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课程失败: %w", err)
	}
	return c, nil
}

// Insert 新增课程（对齐 Java CourseMapper.insert）
//
// 参数：
//   - courseName: 课程名称
//   - courseType: 课程类型（1=按次, 2=按天）
//   - institutionID: 机构ID
//   - isAvailable: 是否可用
//
// 返回：课程ID
func (m *CourseMapper) Insert(courseName string, courseType int64, institutionID int64, isAvailable bool) (int64, error) {
	query := `INSERT INTO c_course (course_name, course_type, institution_id, is_available, create_time, update_time) VALUES (?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, courseName, courseType, institutionID, isAvailable)
	if err != nil {
		return 0, fmt.Errorf("新增课程失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取课程ID失败: %w", err)
	}
	return id, nil
}

// UpdateByID 按ID更新课程（对齐 Java CourseMapper.updateById）
//
// 仅更新非空字段
//
// 参数：
//   - id: 课程ID
//   - courseName: 课程名称（空字符串表示不更新）
//   - courseType: 课程类型（0 表示不更新）
//   - isAvailable: 是否可用（nil 表示不更新）
//
// 返回：影响行数
func (m *CourseMapper) UpdateByID(id int64, courseName string, courseType int64, isAvailable *bool) (int64, error) {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if courseName != "" {
		setParts = append(setParts, "course_name = ?")
		args = append(args, courseName)
	}
	if courseType != 0 {
		setParts = append(setParts, "course_type = ?")
		args = append(args, courseType)
	}
	if isAvailable != nil {
		setParts = append(setParts, "is_available = ?")
		args = append(args, *isAvailable)
	}

	query := fmt.Sprintf("UPDATE c_course SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新课程失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}
