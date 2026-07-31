// Package mapper business-service 数据访问层 - 上课记录模块
//
// 对齐 Java com.shiroko.mapper.RecordMapper
//
// 表：c_record（上课记录表，每次上课/扣课时的明细记录）
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// RecordMapper 上课记录表操作
// ============================================================

// RecordMapper 上课记录表 c_record 的 Mapper
type RecordMapper struct {
	db *sql.DB
}

// NewRecordMapper 创建 RecordMapper
func NewRecordMapper(db *sql.DB) *RecordMapper {
	return &RecordMapper{db: db}
}

// RecordDTO 上课记录查询结果 DTO（对齐 Java RecordDTO）
//
// 包含记录基础信息 + 课卡信息 + 课程信息 + 学生信息 + 操作教师信息 + 机构信息
//
// 重构说明（嵌套对象化）：
//   - 新增课卡完整字段：CourseStatus/CourseLastTime/ExpireTime/ExpireTimeRaw/IsDelete/CourseRemark/
//     CourseOwnerUserID/PermissionType/CourseRecordCreateTime/CourseRecordUpdateTime
//     用于构造 RecordVO.CourseRecord 嵌套对象
//   - 新增课程补充字段：IsAvailable/CourseCreateTime/CourseUpdateTime
//     用于构造 RecordVO.Course 嵌套对象（CourseVO）
//   - 新增学生补充字段：Sex/Avatar
//     用于构造 RecordVO.Student 嵌套对象（StudentBriefVO）
//   - 新增机构字段：InstitutionID/InstitutionName/InstitutionCode
//     用于构造 RecordVO.Course.Institution 嵌套对象（InstitutionBriefVO）
type RecordDTO struct {
	ID                   int64         `json:"id"`                   // 记录ID
	CourseRecordID       int64         `json:"courseRecordId"`       // 课卡记录ID
	RecordTime           string        `json:"recordTime"`           // 记录时间（格式化字符串）
	RecordRemark         string        `json:"recordRemark"`         // 备注
	RecordType           int64         `json:"recordType"`           // 记录类型（1=增加, 2=减少）
	RecordChange         int64         `json:"recordChange"`         // 课时变更数量
	RestTimeAfterDeduct  int64         `json:"restTimeAfterDeduct"`  // 扣费后剩余课时
	DeductMode           string        `json:"deductMode"`           // 扣费模式（BY_STUDENT/BY_COURSE/BY_CLASS）
	ClassID              int64         `json:"classId"`              // 班级ID
	OperateTeacherID     int64         `json:"operateTeacherId"`     // 操作教师ID
	CreateTime           string        `json:"createTime"`           // 创建时间（格式化字符串）
	UpdateTime           string        `json:"updateTime"`           // 更新时间（格式化字符串）
	// JOIN 字段 - 课卡记录（c_course_record cr），用于构造 RecordVO.CourseRecord 嵌套对象
	StudentID            int64         `json:"studentId"`            // 学生ID（来自 cr.student_id）
	CourseTotalTime      int64         `json:"courseTotalTime"`      // 课时总数（来自 cr.course_total_time）
	CourseRestTime       int64         `json:"courseRestTime"`       // 剩余课时（来自 cr.course_rest_time）
	CourseStatus         int64         `json:"courseStatus"`         // 课程状态（来自 cr.course_status）
	CourseLastTime       string        `json:"courseLastTime"`       // 上次上课时间（格式化字符串，来自 cr.course_last_time）
	ExpireTime           string        `json:"expireTime"`           // 过期时间（格式化字符串，来自 cr.expire_time）
	ExpireTimeRaw        sql.NullTime  `json:"-"`                    // 过期时间原始值（用于计算 ExpireStatus，不序列化到 JSON）
	IsDelete             bool          `json:"isDelete"`             // 是否已删除（来自 cr.is_delete）
	CourseRemark         string        `json:"courseRemark"`         // 课程备注（来自 cr.course_remark）
	CourseOwnerUserID    int64         `json:"courseOwnerUserId"`    // 课程归属人ID（来自 cr.course_owner_user_id）
	PermissionType       int64         `json:"permissionType"`       // 权限类型（来自 cr.permission_type）
	CourseRecordCreateTime string      `json:"courseRecordCreateTime"` // 课卡创建时间（格式化字符串，来自 cr.create_time）
	CourseRecordUpdateTime string      `json:"courseRecordUpdateTime"` // 课卡更新时间（格式化字符串，来自 cr.update_time）
	// JOIN 字段 - 课程（c_course c），用于构造 RecordVO.Course 和 RecordVO.CourseRecord.Course 嵌套对象
	CourseID             int64         `json:"courseId"`             // 课程ID（来自 c.id）
	CourseName           string        `json:"courseName"`           // 课程名称（来自 c.course_name）
	CourseType           int64         `json:"courseType"`           // 课程类型（来自 c.course_type）
	IsAvailable          bool          `json:"isAvailable"`          // 是否可用（来自 c.is_available）
	CourseCreateTime     string        `json:"courseCreateTime"`     // 课程创建时间（格式化字符串，来自 c.create_time）
	CourseUpdateTime     string        `json:"courseUpdateTime"`     // 课程更新时间（格式化字符串，来自 c.update_time）
	// JOIN 字段 - 学生（c_student s），用于构造 RecordVO.Student 嵌套对象
	StudentName          string        `json:"studentName"`          // 学生姓名（来自 s.student_name）
	Sex                  int64         `json:"sex"`                  // 性别（来自 s.sex，0=未知,1=男,2=女）
	Avatar               string        `json:"avatar"`               // 头像URL（来自 s.avatar）
	// JOIN 字段 - 机构（c_institution i），用于构造 RecordVO.Course.Institution 嵌套对象
	InstitutionID        int64         `json:"institutionId"`        // 机构ID（来自 i.id）
	InstitutionName      string        `json:"institutionName"`      // 机构名称（来自 i.institution_name）
	InstitutionCode      string        `json:"institutionCode"`      // 机构编码（来自 i.institution_code）
	// JOIN 字段 - 教师（c_teacher t），用于构造 RecordVO.OperatorTeacher 嵌套对象
	TeacherName          string        `json:"teacherName"`          // 操作教师姓名（来自 t.username）
}

// SelectList 查询上课记录列表（对齐 Java RecordMapper.selectRecords）
//
// 支持按机构ID、学生ID、课卡ID、课程名称、记录类型过滤
//
// 参数：
//   - institutionID: 机构ID（0 表示不过滤）
//   - studentID: 学生ID（0 表示不过滤）
//   - courseRecordID: 课卡记录ID（0 表示不过滤）
//   - courseName: 课程名称关键词
//   - recordType: 记录类型（0 表示不过滤）
//   - currentPage: 当前页码（从1开始）
//   - pageSize: 每页条数
//
// 返回：列表 + 总数
func (m *RecordMapper) SelectList(institutionID, studentID, courseRecordID int64, courseName string, recordType int64, currentPage, pageSize int) ([]*RecordDTO, int64, error) {
	// 构建 WHERE 条件
	where := "WHERE 1=1"
	args := []interface{}{}
	if institutionID != 0 {
		where += " AND c.institution_id = ?"
		args = append(args, institutionID)
	}
	if studentID != 0 {
		where += " AND cr.student_id = ?"
		args = append(args, studentID)
	}
	if courseRecordID != 0 {
		where += " AND r.course_record_id = ?"
		args = append(args, courseRecordID)
	}
	if courseName != "" {
		where += " AND c.course_name LIKE CONCAT('%', ?, '%')"
		args = append(args, courseName)
	}
	if recordType != 0 {
		where += " AND r.record_type = ?"
		args = append(args, recordType)
	}

	// 查询总数
	// 重构说明：COUNT 查询也 JOIN c_institution，保持与分页查询的 WHERE 条件一致
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM c_record r
		LEFT JOIN c_course_record cr ON r.course_record_id = cr.id
		LEFT JOIN c_course c ON cr.course_id = c.id
		LEFT JOIN c_student s ON cr.student_id = s.id
		LEFT JOIN c_institution i ON c.institution_id = i.id
		%s
	`, where)
	var total int64
	err := m.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询上课记录总数失败: %w", err)
	}

	// 分页查询
	// 重构说明：
	//   - SQL 增加 c_institution JOIN，获取机构信息（institution_name, institution_code）用于构造 RecordVO.Course.Institution
	//   - 增加课卡完整字段：cr.course_status, cr.course_last_time, cr.expire_time, cr.is_delete,
	//     cr.course_remark, cr.course_owner_user_id, cr.permission_type, cr.create_time, cr.update_time
	//   - 增加课程补充字段：c.is_available, c.create_time, c.update_time
	//   - 增加学生补充字段：s.sex, s.avatar
	//   - 增加机构字段：i.id, i.institution_name, i.institution_code
	offset := (currentPage - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT r.id, r.course_record_id, r.record_time, r.record_remark, r.record_type,
		       r.record_change, r.rest_time_after_deduct, r.deduct_mode, r.class_id, r.operate_teacher_id,
		       r.create_time, r.update_time,
		       cr.student_id, cr.course_total_time, cr.course_rest_time, cr.course_status,
		       cr.course_last_time, cr.expire_time, cr.is_delete, cr.course_remark,
		       cr.course_owner_user_id, cr.permission_type, cr.create_time, cr.update_time,
		       c.id AS course_id, c.course_name, c.course_type, c.is_available, c.create_time, c.update_time,
		       s.student_name, s.sex, s.avatar,
		       i.id AS institution_id, i.institution_name, i.institution_code,
		       t.username AS teacher_name
		FROM c_record r
		LEFT JOIN c_course_record cr ON r.course_record_id = cr.id
		LEFT JOIN c_course c ON cr.course_id = c.id
		LEFT JOIN c_student s ON cr.student_id = s.id
		LEFT JOIN c_institution i ON c.institution_id = i.id
		LEFT JOIN c_teacher t ON r.operate_teacher_id = t.id
		%s
		ORDER BY r.record_time DESC, r.id DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, pageSize, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询上课记录列表失败: %w", err)
	}
	defer rows.Close()

	var list []*RecordDTO
	for rows.Next() {
		dto := &RecordDTO{}
		var (
			courseRecordID       sql.NullInt64
			recordTime           sql.NullTime
			recordRemark         sql.NullString
			recordType           sql.NullInt64
			recordChange         sql.NullInt64
			restAfterDeduct      sql.NullInt64
			deductMode           sql.NullString
			classID              sql.NullInt64
			operateTeacherID     sql.NullInt64
			createTime           sql.NullTime
			updateTime           sql.NullTime
			// 课卡记录字段
			studentID            sql.NullInt64
			courseTotalTime      sql.NullInt64
			courseRestTime       sql.NullInt64
			courseStatus         sql.NullInt64
			courseLastTime       sql.NullTime
			expireTime           sql.NullTime
			isDelete             sql.NullBool
			courseRemark         sql.NullString
			courseOwnerUserID    sql.NullInt64
			permissionType       sql.NullInt64
			crCreateTime         sql.NullTime
			crUpdateTime         sql.NullTime
			// 课程字段
			courseID             sql.NullInt64
			courseName           sql.NullString
			courseType           sql.NullInt64
			isAvailable          sql.NullBool
			courseCreateTime     sql.NullTime
			courseUpdateTime     sql.NullTime
			// 学生字段
			studentName          sql.NullString
			sex                  sql.NullInt64
			avatar               sql.NullString
			// 机构字段
			institutionID        sql.NullInt64
			institutionName      sql.NullString
			institutionCode      sql.NullString
			// 教师字段
			teacherName          sql.NullString
		)
		err := rows.Scan(
			&dto.ID, &courseRecordID, &recordTime, &recordRemark, &recordType,
			&recordChange, &restAfterDeduct, &deductMode, &classID, &operateTeacherID,
			&createTime, &updateTime,
			// 课卡记录字段
			&studentID, &courseTotalTime, &courseRestTime, &courseStatus,
			&courseLastTime, &expireTime, &isDelete, &courseRemark,
			&courseOwnerUserID, &permissionType, &crCreateTime, &crUpdateTime,
			// 课程字段
			&courseID, &courseName, &courseType, &isAvailable, &courseCreateTime, &courseUpdateTime,
			// 学生字段
			&studentName, &sex, &avatar,
			// 机构字段
			&institutionID, &institutionName, &institutionCode,
			// 教师字段
			&teacherName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描上课记录失败: %w", err)
		}
		// 记录基础字段
		dto.CourseRecordID = courseRecordID.Int64
		dto.RecordTime = entity.FormatTime(recordTime)
		dto.RecordRemark = recordRemark.String
		dto.RecordType = recordType.Int64
		dto.RecordChange = recordChange.Int64
		dto.RestTimeAfterDeduct = restAfterDeduct.Int64
		dto.DeductMode = deductMode.String
		dto.ClassID = classID.Int64
		dto.OperateTeacherID = operateTeacherID.Int64
		dto.CreateTime = entity.FormatTime(createTime)
		dto.UpdateTime = entity.FormatTime(updateTime)
		// 课卡记录字段
		dto.StudentID = studentID.Int64
		dto.CourseTotalTime = courseTotalTime.Int64
		dto.CourseRestTime = courseRestTime.Int64
		dto.CourseStatus = courseStatus.Int64
		dto.CourseLastTime = entity.FormatTime(courseLastTime)
		dto.ExpireTime = entity.FormatTime(expireTime)
		// 保存原始过期时间，用于 service 层计算 ExpireStatus
		dto.ExpireTimeRaw = expireTime
		dto.IsDelete = isDelete.Bool
		dto.CourseRemark = courseRemark.String
		dto.CourseOwnerUserID = courseOwnerUserID.Int64
		dto.PermissionType = permissionType.Int64
		dto.CourseRecordCreateTime = entity.FormatTime(crCreateTime)
		dto.CourseRecordUpdateTime = entity.FormatTime(crUpdateTime)
		// 课程字段
		dto.CourseID = courseID.Int64
		dto.CourseName = courseName.String
		dto.CourseType = courseType.Int64
		dto.IsAvailable = isAvailable.Bool
		dto.CourseCreateTime = entity.FormatTime(courseCreateTime)
		dto.CourseUpdateTime = entity.FormatTime(courseUpdateTime)
		// 学生字段
		dto.StudentName = studentName.String
		dto.Sex = sex.Int64
		dto.Avatar = avatar.String
		// 机构字段
		dto.InstitutionID = institutionID.Int64
		dto.InstitutionName = institutionName.String
		dto.InstitutionCode = institutionCode.String
		// 教师字段
		dto.TeacherName = teacherName.String
		list = append(list, dto)
	}
	return list, total, nil
}

// SelectByID 按主键查上课记录（用于查询扣费详情）
func (m *RecordMapper) SelectByID(id int64) (*entity.Record, error) {
	query := `SELECT id, course_record_id, record_time, record_remark, record_type, record_change, rest_time_after_deduct, deduct_mode, class_id, operate_teacher_id, create_time, update_time FROM c_record WHERE id = ?`
	row := m.db.QueryRow(query, id)

	r := &entity.Record{}
	err := row.Scan(
		&r.ID, &r.CourseRecordID, &r.RecordTime, &r.RecordRemark,
		&r.RecordType, &r.RecordChange, &r.RestTimeAfterDeduct,
		&r.DeductMode, &r.ClassID, &r.OperateTeacherID,
		&r.CreateTime, &r.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询上课记录失败: %w", err)
	}
	return r, nil
}

// DeductDetailDTO 扣费详情查询结果 DTO
//
// 包含上课记录 + 课卡记录 + 课程 + 学生 + 班级 + 课表 + 教师 的全部信息
// 用于 GetDeductDetail 接口，一次 JOIN 查询获取前端 DeductDetailResponse 所需的全部 21 个字段
//
// 字段来源：
//   - c_record r：record_time, record_remark, record_change, rest_time_after_deduct, deduct_mode, class_id, operate_teacher_id
//   - c_course_record cr：student_id, course_id, course_total_time, course_rest_time, expire_time
//   - c_course c：course_name, course_type
//   - c_student s：student_name
//   - c_class cl：class_name（通过 r.class_id 关联）
//   - c_teacher t：username（通过 r.operate_teacher_id 关联）
//   - c_class_schedule cs：排课描述（子查询 GROUP_CONCAT 聚合多个课表项）
type DeductDetailDTO struct {
	RecordID            int64          // 上课记录ID（c_record.id，主键，非空）
	CourseRecordID      sql.NullInt64  // 课卡记录ID（c_course_record.id）
	RecordTime          sql.NullTime   // 记录时间
	RecordRemark        sql.NullString // 备注
	RecordChange        sql.NullInt64  // 课时变更数量（扣减时为 deductCount）
	RestTimeAfterDeduct sql.NullInt64  // 扣费后剩余课时（快照值）
	DeductMode          sql.NullString // 扣费模式（BY_STUDENT/BY_COURSE/BY_CLASS）
	ClassID             sql.NullInt64  // 班级ID
	OperateTeacherID    sql.NullInt64  // 操作教师ID（c_teacher.id）
	StudentID           sql.NullInt64  // 学生ID（c_student.id）
	CourseID            sql.NullInt64  // 课程ID（c_course.id）
	CourseTotalTime     sql.NullInt64  // 课时总数（来自 c_course_record）
	CourseRestTime      sql.NullInt64  // 当前剩余课时（实时值，来自 c_course_record）
	ExpireTime          sql.NullTime   // 过期时间（NULL=永久有效，来自 c_course_record）
	CourseName          sql.NullString // 课程名称（JOIN c_course）
	CourseType          sql.NullInt64  // 课程类型（1=按次, 2=按天，JOIN c_course）
	StudentName         sql.NullString // 学生姓名（JOIN c_student）
	ClassName           sql.NullString // 班级名称（JOIN c_class，通过 r.class_id 关联）
	TeacherName         sql.NullString // 操作教师姓名（JOIN c_teacher，c_teacher.username）
	ScheduleDesc        sql.NullString // 排课时间描述（子查询 GROUP_CONCAT 聚合）
}

// SelectDeductDetailByID 按上课记录ID查扣费详情（联表查询全部字段）
//
// 对齐 Java CourseRecordServiceImpl.getDeductDetail
// 一次 JOIN 查询获取前端 DeductDetailResponse 所需的全部 21 个字段：
//   - c_record：记录基础信息（record_time, record_remark, record_change, ...）
//   - c_course_record：课卡信息（student_id, course_id, course_rest_time, expire_time, course_total_time）
//   - c_course：课程信息（course_name, course_type）
//   - c_student：学生姓名（student_name）
//   - c_class：班级名称（class_name，通过 c_record.class_id 关联）
//   - c_class_schedule：排课描述（子查询 GROUP_CONCAT 聚合，格式 "周一 09:00-10:00, 周三 14:00-15:00"）
//   - c_teacher：操作教师姓名（username，通过 c_record.operate_teacher_id 关联）
//
// 参数：
//   - recordID: 上课记录ID（c_record.id）
//
// 返回：扣费详情 DTO（未找到返回 nil）
func (m *RecordMapper) SelectDeductDetailByID(recordID int64) (*DeductDetailDTO, error) {
	// 联表查询：c_record 为主表，LEFT JOIN 获取关联数据
	// 排课描述使用子查询 + GROUP_CONCAT 聚合（一个班级可有多个课表项）
	query := `
		SELECT
			r.id AS record_id,
			r.course_record_id,
			r.record_time,
			r.record_remark,
			r.record_change,
			r.rest_time_after_deduct,
			r.deduct_mode,
			r.class_id,
			r.operate_teacher_id,
			cr.student_id,
			cr.course_id,
			cr.course_total_time,
			cr.course_rest_time,
			cr.expire_time,
			c.course_name,
			c.course_type,
			s.student_name,
			cl.class_name,
			t.username AS teacher_name,
			(SELECT GROUP_CONCAT(CONCAT(
				CASE cs.day_of_week
					WHEN 1 THEN '周一' WHEN 2 THEN '周二' WHEN 3 THEN '周三'
					WHEN 4 THEN '周四' WHEN 5 THEN '周五' WHEN 6 THEN '周六'
					WHEN 7 THEN '周日'
				END,
				' ', TIME_FORMAT(cs.start_time, '%H:%i'), '-', TIME_FORMAT(cs.end_time, '%H:%i')
			) SEPARATOR ', ')
			 FROM c_class_schedule cs WHERE cs.class_id = r.class_id) AS schedule_desc
		FROM c_record r
		LEFT JOIN c_course_record cr ON r.course_record_id = cr.id
		LEFT JOIN c_course c ON cr.course_id = c.id
		LEFT JOIN c_student s ON cr.student_id = s.id
		LEFT JOIN c_class cl ON r.class_id = cl.id
		LEFT JOIN c_teacher t ON r.operate_teacher_id = t.id
		WHERE r.id = ?
		LIMIT 1
	`
	row := m.db.QueryRow(query, recordID)

	dto := &DeductDetailDTO{}
	// 直接扫描到 DTO 的 sql.Null* 字段（RecordID 为主键非空，直接用 int64）
	err := row.Scan(
		&dto.RecordID, &dto.CourseRecordID, &dto.RecordTime, &dto.RecordRemark,
		&dto.RecordChange, &dto.RestTimeAfterDeduct, &dto.DeductMode,
		&dto.ClassID, &dto.OperateTeacherID,
		&dto.StudentID, &dto.CourseID, &dto.CourseTotalTime, &dto.CourseRestTime,
		&dto.ExpireTime,
		&dto.CourseName, &dto.CourseType,
		&dto.StudentName,
		&dto.ClassName,
		&dto.TeacherName,
		&dto.ScheduleDesc,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询扣费详情失败: %w", err)
	}
	return dto, nil
}

// Insert 新增上课记录（对齐 Java RecordMapper.insert）
//
// 用于扣课时记录流水明细
//
// 参数：
//   - courseRecordID: 课卡记录ID
//   - recordTime: 记录时间（空字符串表示使用当前时间）
//   - operateTeacherID: 操作教师ID
//   - remark: 备注
//   - recordType: 记录类型（1=增加, 2=减少）
//   - recordChange: 课时变更数量
//   - restTimeAfterDeduct: 扣费后剩余课时
//   - deductMode: 扣费模式（BY_STUDENT/BY_COURSE/BY_CLASS）
//   - classID: 班级ID（按班级扣费时有值，0 表示 NULL）
//
// 返回：记录ID
func (m *RecordMapper) Insert(courseRecordID int64, recordTime string, operateTeacherID int64, remark string, recordType, recordChange, restTimeAfterDeduct int64, deductMode string, classID int64) (int64, error) {
	// 处理可选参数的 NULL 值
	var recordTimeArg interface{}
	if recordTime != "" {
		recordTimeArg = recordTime
	} else {
		recordTimeArg = nil
	}

	var classIDArg interface{}
	if classID != 0 {
		classIDArg = classID
	} else {
		classIDArg = nil
	}

	var restTimeArg interface{}
	if restTimeAfterDeduct != 0 {
		restTimeArg = restTimeAfterDeduct
	} else {
		restTimeArg = nil
	}

	query := `INSERT INTO c_record (course_record_id, record_time, record_remark, record_type, record_change, rest_time_after_deduct, deduct_mode, class_id, operate_teacher_id, create_time, update_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, courseRecordID, recordTimeArg, remark, recordType, recordChange, restTimeArg, deductMode, classIDArg, operateTeacherID)
	if err != nil {
		return 0, fmt.Errorf("新增上课记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取上课记录ID失败: %w", err)
	}
	return id, nil
}

// DeleteByID 按主键删除上课记录（硬删除）
//
// 对齐 Java RecordController 中未实现但任务要求的 /record/delete 接口
//
// 参数：
//   - id: 记录ID
//
// 返回：影响行数
func (m *RecordMapper) DeleteByID(id int64) (int64, error) {
	query := `DELETE FROM c_record WHERE id = ?`
	result, err := m.db.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("删除上课记录失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}
