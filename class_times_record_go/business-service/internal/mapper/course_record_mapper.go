// Package mapper business-service 数据访问层 - 课卡记录模块
//
// 对齐 Java com.shiroko.mapper.CourseRecordMapper
//
// 表：c_course_record（课卡记录表，记录学生在某课程的课时持有情况）
//
// 扣费双重校验：
//   1. Go service 层校验 expire_time（过期返回 code=1003 COURSE_EXPIRED）
//   2. SQL 层 WHERE 条件包含 (expire_time IS NULL OR expire_time > NOW()) 兜底
//   3. 余额不足返回 code=1001 COURSE_BALANCE_NOT_ENOUGH
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// CourseRecordMapper 课卡记录表操作
// ============================================================

// CourseRecordMapper 课卡记录表 c_course_record 的 Mapper
type CourseRecordMapper struct {
	db *sql.DB
}

// NewCourseRecordMapper 创建 CourseRecordMapper
func NewCourseRecordMapper(db *sql.DB) *CourseRecordMapper {
	return &CourseRecordMapper{db: db}
}

// CourseRecordDTO 课卡记录查询结果 DTO（对齐 Java CourseRecordDTO）
//
// 包含课卡基础信息 + 课程信息 + 学生姓名
//
// 重构说明（嵌套对象化）：
//   - 新增 CourseOwnerUserID：课程归属人ID（对应 cr.course_owner_user_id）
//   - 新增 PermissionType：权限类型（对应 cr.permission_type，用于 CourseRecordVO.PermissionType）
//   - 新增 IsAvailable：课程是否可用（JOIN c_course.is_available，用于嵌套 CourseBriefVO）
//   - 新增 ExpireTimeRaw：过期时间原始值（sql.NullTime，用于计算 ExpireStatus）
//     与 ExpireTime（格式化字符串）并存：前者用于逻辑判断，后者用于前端展示
type CourseRecordDTO struct {
	ID                int64         `json:"id"`                // 课卡记录ID
	StudentID         int64         `json:"studentId"`         // 学生ID
	CourseID          int64         `json:"courseId"`          // 课程ID
	CourseTotalTime   int64         `json:"courseTotalTime"`   // 课时总数
	CourseRestTime    int64         `json:"courseRestTime"`    // 剩余课时
	CourseStatus      int64         `json:"courseStatus"`      // 课程状态
	CourseLastTime    string        `json:"courseLastTime"`    // 上次上课时间（格式化字符串）
	ExpireTime        string        `json:"expireTime"`        // 过期时间（格式化字符串，空=永久有效）
	ExpireTimeRaw     sql.NullTime  `json:"-"`                 // 过期时间原始值（用于计算 ExpireStatus，不序列化到 JSON）
	IsDelete          bool          `json:"isDelete"`          // 是否已删除
	CourseName        string        `json:"courseName"`        // 课程名称（JOIN c_course）
	CourseType        int64         `json:"courseType"`        // 课程类型（JOIN c_course）
	IsAvailable       bool          `json:"isAvailable"`       // 课程是否可用（JOIN c_course.is_available，用于嵌套 CourseBriefVO）
	StudentName       string        `json:"studentName"`       // 学生姓名（JOIN c_student）
	CourseRemark      string        `json:"courseRemark"`      // 课程备注
	CourseOwnerUserID int64         `json:"courseOwnerUserId"` // 课程归属人ID（对应 cr.course_owner_user_id）
	PermissionType    int64         `json:"permissionType"`    // 权限类型（对应 cr.permission_type）
	CreateTime        string        `json:"createTime"`        // 创建时间（格式化字符串）
	UpdateTime        string        `json:"updateTime"`        // 更新时间（格式化字符串）
}

// SelectList 查询课卡记录列表（对齐 Java CourseRecordMapper.selectCourseRecords）
//
// 支持按学生ID、机构ID、课程名称、学生姓名、关键词、过期状态过滤
//
// 参数：
//   - studentID: 学生ID（0 表示不过滤）
//   - institutionID: 机构ID（0 表示不过滤）
//   - courseName: 课程名称关键词
//   - stuName: 学生姓名关键词
//   - keyword: 通用关键词（匹配课程名或学生名）
//   - expireStatus: 过期状态（0=有效, 1=即将过期, 2=已过期，-1=不过滤）
//   - currentPage: 当前页码（从1开始）
//   - pageSize: 每页条数
//
// 返回：列表 + 总数
func (m *CourseRecordMapper) SelectList(studentID, institutionID int64, courseName, stuName, keyword string, expireStatus int64, currentPage, pageSize int) ([]*CourseRecordDTO, int64, error) {
	// 构建 WHERE 条件
	where := "WHERE cr.is_delete = 0"
	args := []interface{}{}
	if studentID != 0 {
		where += " AND cr.student_id = ?"
		args = append(args, studentID)
	}
	if institutionID != 0 {
		where += " AND c.institution_id = ?"
		args = append(args, institutionID)
	}
	if courseName != "" {
		where += " AND c.course_name LIKE CONCAT('%', ?, '%')"
		args = append(args, courseName)
	}
	if stuName != "" {
		where += " AND s.student_name LIKE CONCAT('%', ?, '%')"
		args = append(args, stuName)
	}
	if keyword != "" {
		where += " AND (c.course_name LIKE CONCAT('%', ?, '%') OR s.student_name LIKE CONCAT('%', ?, '%'))"
		args = append(args, keyword, keyword)
	}
	if expireStatus != -1 {
		switch expireStatus {
		case 0: // 有效：过期时间在7天后或为空
			where += " AND (cr.expire_time IS NULL OR cr.expire_time > DATE_ADD(NOW(), INTERVAL 7 DAY))"
		case 1: // 即将过期：7天内过期
			where += " AND cr.expire_time > NOW() AND cr.expire_time <= DATE_ADD(NOW(), INTERVAL 7 DAY)"
		case 2: // 已过期
			where += " AND cr.expire_time <= NOW()"
		}
	}

	// 查询总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM c_course_record cr
		LEFT JOIN c_course c ON cr.course_id = c.id
		LEFT JOIN c_student s ON cr.student_id = s.id
		%s
	`, where)
	var total int64
	err := m.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课卡记录总数失败: %w", err)
	}

	// 分页查询
	// 重构说明：增加 cr.permission_type、cr.course_owner_user_id、c.is_available 列，
	// 用于填充 CourseRecordVO.PermissionType、CourseRecordVO.CourseOwnerUserID 和嵌套 CourseBriefVO.IsAvailable
	offset := (currentPage - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT cr.id, cr.student_id, cr.course_id, cr.course_total_time, cr.course_rest_time,
		       cr.course_status, cr.course_last_time, cr.expire_time, cr.is_delete, cr.course_remark,
		       cr.course_owner_user_id, cr.permission_type,
		       cr.create_time, cr.update_time,
		       c.course_name, c.course_type, c.is_available,
		       s.student_name
		FROM c_course_record cr
		LEFT JOIN c_course c ON cr.course_id = c.id
		LEFT JOIN c_student s ON cr.student_id = s.id
		%s
		ORDER BY cr.update_time DESC, cr.id DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, pageSize, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课卡记录列表失败: %w", err)
	}
	defer rows.Close()

	var list []*CourseRecordDTO
	for rows.Next() {
		dto := &CourseRecordDTO{}
		var (
			studentID         sql.NullInt64
			courseID          sql.NullInt64
			courseTotalTime   sql.NullInt64
			courseRestTime    sql.NullInt64
			courseStatus      sql.NullInt64
			courseLastTime    sql.NullTime
			expireTime        sql.NullTime
			isDelete          sql.NullBool
			courseRemark      sql.NullString
			courseOwnerUserID sql.NullInt64
			permissionType    sql.NullInt64
			createTime        sql.NullTime
			updateTime        sql.NullTime
			courseName        sql.NullString
			courseType        sql.NullInt64
			isAvailable       sql.NullBool
			studentName       sql.NullString
		)
		err := rows.Scan(
			&dto.ID, &studentID, &courseID, &courseTotalTime, &courseRestTime,
			&courseStatus, &courseLastTime, &expireTime, &isDelete, &courseRemark,
			&courseOwnerUserID, &permissionType,
			&createTime, &updateTime,
			&courseName, &courseType, &isAvailable,
			&studentName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描课卡记录失败: %w", err)
		}
		dto.StudentID = studentID.Int64
		dto.CourseID = courseID.Int64
		dto.CourseTotalTime = courseTotalTime.Int64
		dto.CourseRestTime = courseRestTime.Int64
		dto.CourseStatus = courseStatus.Int64
		dto.CourseLastTime = entity.FormatTime(courseLastTime)
		dto.ExpireTime = entity.FormatTime(expireTime)
		// 保存原始过期时间，用于 service 层计算 ExpireStatus
		dto.ExpireTimeRaw = expireTime
		dto.IsDelete = isDelete.Bool
		dto.CourseName = courseName.String
		dto.CourseType = courseType.Int64
		dto.IsAvailable = isAvailable.Bool
		dto.StudentName = studentName.String
		dto.CourseRemark = courseRemark.String
		dto.CourseOwnerUserID = courseOwnerUserID.Int64
		dto.PermissionType = permissionType.Int64
		dto.CreateTime = entity.FormatTime(createTime)
		dto.UpdateTime = entity.FormatTime(updateTime)
		list = append(list, dto)
	}
	return list, total, nil
}

// SelectByStudentAndCourse 按学生ID和课程ID查课卡记录（用于扣课前过期校验）
//
// 对齐 Java CourseRecordServiceImpl.checkCourseRecordExpired
//
// 仅查未删除的记录
func (m *CourseRecordMapper) SelectByStudentAndCourse(studentID, courseID int64) (*entity.CourseRecord, error) {
	query := `SELECT id, student_id, course_id, course_total_time, course_rest_time, course_status, course_last_time, expire_time, course_owner_user_id, course_remark, is_delete, create_time, update_time FROM c_course_record WHERE student_id = ? AND course_id = ? AND is_delete = 0 LIMIT 1`
	row := m.db.QueryRow(query, studentID, courseID)

	cr := &entity.CourseRecord{}
	err := row.Scan(
		&cr.ID, &cr.StudentID, &cr.CourseID, &cr.CourseTotalTime, &cr.CourseRestTime,
		&cr.CourseStatus, &cr.CourseLastTime, &cr.ExpireTime, &cr.CourseOwnerUserID,
		&cr.CourseRemark, &cr.IsDelete, &cr.CreateTime, &cr.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课卡记录失败: %w", err)
	}
	return cr, nil
}

// SelectDTOByStudentAndCourse 按学生ID和课程ID查课卡记录 DTO（含 JOIN 数据）
//
// 与 SelectByStudentAndCourse 不同，本方法返回 *CourseRecordDTO，
// 通过 LEFT JOIN c_course / c_student 获取课程名称、课程类型、学生姓名等额外字段。
// 用于 ClassVO.CourseRecord 和 CourseVO.CurrentStudentCourseRecord 嵌套对象的填充
// （需要完整字段以对齐前端 CourseRecordResponse）。
//
// 重构说明：SQL 增加 cr.course_owner_user_id、cr.permission_type、c.is_available 列，
// 用于填充 CourseRecordVO.CourseOwnerUserID、CourseRecordVO.PermissionType 和嵌套 CourseBriefVO.IsAvailable。
//
// 参数：
//   - studentID: 学生ID
//   - courseID: 课程ID
//
// 返回：课卡记录 DTO（含 JOIN 数据），未找到返回 nil
func (m *CourseRecordMapper) SelectDTOByStudentAndCourse(studentID, courseID int64) (*CourseRecordDTO, error) {
	query := `
		SELECT cr.id, cr.student_id, cr.course_id, cr.course_total_time, cr.course_rest_time,
		       cr.course_status, cr.course_last_time, cr.expire_time, cr.is_delete, cr.course_remark,
		       cr.course_owner_user_id, cr.permission_type,
		       cr.create_time, cr.update_time,
		       c.course_name, c.course_type, c.is_available,
		       s.student_name
		FROM c_course_record cr
		LEFT JOIN c_course c ON cr.course_id = c.id
		LEFT JOIN c_student s ON cr.student_id = s.id
		WHERE cr.student_id = ? AND cr.course_id = ? AND cr.is_delete = 0
		LIMIT 1
	`
	row := m.db.QueryRow(query, studentID, courseID)

	dto := &CourseRecordDTO{}
	// 注意：局部变量使用 Val 后缀，避免与函数参数 studentID/courseID 同名冲突
	var (
		studentIDVal      sql.NullInt64
		courseIDVal       sql.NullInt64
		courseTotalTime   sql.NullInt64
		courseRestTime    sql.NullInt64
		courseStatus      sql.NullInt64
		courseLastTime    sql.NullTime
		expireTime        sql.NullTime
		isDelete          sql.NullBool
		courseRemark      sql.NullString
		courseOwnerUserID sql.NullInt64
		permissionType    sql.NullInt64
		createTime        sql.NullTime
		updateTime        sql.NullTime
		courseName        sql.NullString
		courseType        sql.NullInt64
		isAvailable       sql.NullBool
		studentName       sql.NullString
	)
	err := row.Scan(
		&dto.ID, &studentIDVal, &courseIDVal, &courseTotalTime, &courseRestTime,
		&courseStatus, &courseLastTime, &expireTime, &isDelete, &courseRemark,
		&courseOwnerUserID, &permissionType,
		&createTime, &updateTime,
		&courseName, &courseType, &isAvailable,
		&studentName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课卡记录 DTO 失败: %w", err)
	}
	// 将 sql.NullXxx 转换为 DTO 字段
	dto.StudentID = studentIDVal.Int64
	dto.CourseID = courseIDVal.Int64
	dto.CourseTotalTime = courseTotalTime.Int64
	dto.CourseRestTime = courseRestTime.Int64
	dto.CourseStatus = courseStatus.Int64
	dto.CourseLastTime = entity.FormatTime(courseLastTime)
	dto.ExpireTime = entity.FormatTime(expireTime)
	// 保存原始过期时间，用于 service 层计算 ExpireStatus
	dto.ExpireTimeRaw = expireTime
	dto.IsDelete = isDelete.Bool
	dto.CourseName = courseName.String
	dto.CourseType = courseType.Int64
	dto.IsAvailable = isAvailable.Bool
	dto.StudentName = studentName.String
	dto.CourseRemark = courseRemark.String
	dto.CourseOwnerUserID = courseOwnerUserID.Int64
	dto.PermissionType = permissionType.Int64
	dto.CreateTime = entity.FormatTime(createTime)
	dto.UpdateTime = entity.FormatTime(updateTime)
	return dto, nil
}

// SelectByID 按主键查课卡记录（用于扣课后获取剩余课时快照）
func (m *CourseRecordMapper) SelectByID(id int64) (*entity.CourseRecord, error) {
	query := `SELECT id, student_id, course_id, course_total_time, course_rest_time, course_status, course_last_time, expire_time, course_owner_user_id, course_remark, is_delete, create_time, update_time FROM c_course_record WHERE id = ?`
	row := m.db.QueryRow(query, id)

	cr := &entity.CourseRecord{}
	err := row.Scan(
		&cr.ID, &cr.StudentID, &cr.CourseID, &cr.CourseTotalTime, &cr.CourseRestTime,
		&cr.CourseStatus, &cr.CourseLastTime, &cr.ExpireTime, &cr.CourseOwnerUserID,
		&cr.CourseRemark, &cr.IsDelete, &cr.CreateTime, &cr.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课卡记录失败: %w", err)
	}
	return cr, nil
}

// Insert 新增课卡记录（对齐 Java CourseRecordMapper.insert）
//
// 参数：
//   - studentID: 学生ID
//   - courseID: 课程ID
//   - totalTime: 课时总数
//   - restTime: 剩余课时
//   - expireTime: 过期时间（空字符串表示永久有效）
//   - ownerUserID: 课程归属用户ID
//   - remark: 备注
//
// 返回：课卡记录ID
func (m *CourseRecordMapper) Insert(studentID, courseID, totalTime, restTime int64, expireTime string, ownerUserID int64, remark string) (int64, error) {
	var expireArg interface{}
	if expireTime != "" {
		expireArg = expireTime
	} else {
		expireArg = nil
	}
	query := `INSERT INTO c_course_record (student_id, course_id, course_total_time, course_rest_time, course_status, expire_time, course_owner_user_id, course_remark, is_delete, create_time, update_time) VALUES (?, ?, ?, ?, 0, ?, ?, ?, 0, NOW(), NOW())`
	result, err := m.db.Exec(query, studentID, courseID, totalTime, restTime, expireArg, ownerUserID, remark)
	if err != nil {
		return 0, fmt.Errorf("新增课卡记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取课卡记录ID失败: %w", err)
	}
	return id, nil
}

// UpdateByID 按ID更新课卡记录（对齐 Java CourseRecordMapper.updateById）
//
// 仅更新非空字段
//
// 参数：
//   - id: 课卡记录ID
//   - totalTime: 课时总数（0 表示不更新）
//   - restTime: 剩余课时（0 表示不更新）
//   - status: 课程状态（-1 表示不更新）
//   - expireTime: 过期时间（空字符串表示不更新，"NULL" 表示设为 NULL）
//   - remark: 备注（空字符串表示不更新）
//
// 返回：影响行数
func (m *CourseRecordMapper) UpdateByID(id int64, totalTime, restTime, status int64, expireTime, remark string) (int64, error) {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if totalTime != 0 {
		setParts = append(setParts, "course_total_time = ?")
		args = append(args, totalTime)
	}
	if restTime != 0 {
		setParts = append(setParts, "course_rest_time = ?")
		args = append(args, restTime)
	}
	if status != -1 {
		setParts = append(setParts, "course_status = ?")
		args = append(args, status)
	}
	if expireTime != "" {
		if expireTime == "NULL" {
			setParts = append(setParts, "expire_time = NULL")
		} else {
			setParts = append(setParts, "expire_time = ?")
			args = append(args, expireTime)
		}
	}
	if remark != "" {
		setParts = append(setParts, "course_remark = ?")
		args = append(args, remark)
	}

	query := fmt.Sprintf("UPDATE c_course_record SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新课卡记录失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// UpdateRestAndTotalByID 按ID显式设置剩余课时和总课时（对齐 Java insertRecords 中的课时更新逻辑）
//
// 与 UpdateByID 不同，本方法直接写入传入的值（包括 0），不将 0 视为"不更新"。
// 用于 RecordService.insertRecords 流程：手动增减课时后同步课卡记录。
//
// 参数：
//   - id: 课卡记录ID
//   - restTime: 新的剩余课时（直接写入）
//   - totalTime: 新的总课时（0 表示不更新总课时）
//   - lastTime: 上次上课时间（空字符串表示不更新）
//
// 返回：影响行数
func (m *CourseRecordMapper) UpdateRestAndTotalByID(id, restTime, totalTime int64, lastTime string) (int64, error) {
	setParts := []string{"course_rest_time = ?", "update_time = NOW()"}
	args := []interface{}{restTime}
	if totalTime != 0 {
		setParts = append(setParts, "course_total_time = ?")
		args = append(args, totalTime)
	}
	if lastTime != "" {
		setParts = append(setParts, "course_last_time = ?")
		args = append(args, lastTime)
	}

	query := fmt.Sprintf("UPDATE c_course_record SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新课卡剩余课时失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// UpdateRestTime 扣减课时余额（核心：扣费 SQL 双重校验）
//
// 对齐 Java CourseRecordMapper.updateRestTime
//
// SQL 层双重校验：
//   1. course_rest_time >= totalCount：余额充足校验
//   2. (expire_time IS NULL OR expire_time > NOW())：未过期校验
//
// 使用 LAST_INSERT_ID(id) 记录被更新行的 ID，供调用方获取
//
// 参数：
//   - studentID: 学生ID
//   - courseID: 课程ID
//   - totalCount: 扣减课时数
//
// 返回：
//   - rows: 影响行数（0 表示余额不足或已过期）
//   - updatedID: 被更新记录的 ID（影响行数为0时无意义）
//   - err: 错误
func (m *CourseRecordMapper) UpdateRestTime(studentID, courseID, totalCount int64) (rows int64, updatedID int64, err error) {
	// 扣减课时余额，同时校验余额充足且课时未过期
	// LAST_INSERT_ID(id) 记录被更新行的 ID，供后续查询获取
	query := `
		UPDATE c_course_record
		SET
			course_rest_time = course_rest_time - ?,
			course_last_time = NOW(),
			id = LAST_INSERT_ID(id)
		WHERE
			course_id = ?
			AND student_id = ?
			AND course_rest_time >= ?
			AND (expire_time IS NULL OR expire_time > NOW())
			AND is_delete = 0
	`
	result, err := m.db.Exec(query, totalCount, courseID, studentID, totalCount)
	if err != nil {
		return 0, 0, fmt.Errorf("扣减课时失败: %w", err)
	}
	rows, err = result.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	if rows > 0 {
		// 获取 LAST_INSERT_ID（被更新记录的 ID）
		idRow := m.db.QueryRow("SELECT LAST_INSERT_ID()")
		err = idRow.Scan(&updatedID)
		if err != nil {
			return rows, 0, fmt.Errorf("获取更新记录ID失败: %w", err)
		}
	}
	return rows, updatedID, nil
}
