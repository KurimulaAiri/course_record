// Package mapper business-service 数据访问层
//
// 对齐 Java business-service/src/main/java/com/shiroko/mapper 包
//
// 包含：
//   - InstitutionMapper：机构表 c_institution 查询
//   - StudentMapper：学生表 c_student 查询
//   - TeacherMapper：教师表 c_teacher 查询
//   - ClassMapper：班级表 c_class 查询
//   - CourseMapper：课程表 c_course 查询
//   - CourseRecordMapper：课时记录表 c_course_record 查询
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// InstitutionMapper 机构表操作
// ============================================================

// InstitutionMapper 机构表 c_institution 的 Mapper
type InstitutionMapper struct {
	db *sql.DB
}

// NewInstitutionMapper 创建 InstitutionMapper
func NewInstitutionMapper(db *sql.DB) *InstitutionMapper {
	return &InstitutionMapper{db: db}
}

// SelectByID 按主键查机构
func (m *InstitutionMapper) SelectByID(id int64) (*entity.Institution, error) {
	query := `SELECT id, institution_name, institution_address, institution_code, status, expire_time, subscription_plan_id, create_time, update_time FROM c_institution WHERE id = ?`
	row := m.db.QueryRow(query, id)

	inst := &entity.Institution{}
	err := row.Scan(
		&inst.ID,
		&inst.InstitutionName,
		&inst.InstitutionAddress,
		&inst.InstitutionCode,
		&inst.Status,
		&inst.ExpireTime,
		&inst.SubscriptionPlanID,
		&inst.CreateTime,
		&inst.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询机构失败: %w", err)
	}
	return inst, nil
}

// SelectByOpenID 按 openId 查机构列表
//
// 通过 c_user_platform 关联 c_user 再关联 c_institution
func (m *InstitutionMapper) SelectByOpenID(openID string) ([]*entity.Institution, error) {
	query := `
		SELECT DISTINCT i.id, i.institution_name, i.institution_address, i.institution_code, i.status, i.expire_time, i.subscription_plan_id, i.create_time, i.update_time
		FROM c_institution i
		INNER JOIN c_user u ON u.institution_id = i.id
		INNER JOIN c_user_platform up ON up.user_id = u.id
		WHERE up.open_id = ? AND up.is_available = 1
	`
	rows, err := m.db.Query(query, openID)
	if err != nil {
		return nil, fmt.Errorf("查询机构列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Institution
	for rows.Next() {
		inst := &entity.Institution{}
		err := rows.Scan(
			&inst.ID,
			&inst.InstitutionName,
			&inst.InstitutionAddress,
			&inst.InstitutionCode,
			&inst.Status,
			&inst.ExpireTime,
			&inst.SubscriptionPlanID,
			&inst.CreateTime,
			&inst.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描机构记录失败: %w", err)
		}
		list = append(list, inst)
	}
	return list, nil
}

// SelectByCode 按机构编码查
func (m *InstitutionMapper) SelectByCode(code string) (*entity.Institution, error) {
	query := `SELECT id, institution_name, institution_address, institution_code, status, expire_time, subscription_plan_id, create_time, update_time FROM c_institution WHERE institution_code = ?`
	row := m.db.QueryRow(query, code)

	inst := &entity.Institution{}
	err := row.Scan(
		&inst.ID,
		&inst.InstitutionName,
		&inst.InstitutionAddress,
		&inst.InstitutionCode,
		&inst.Status,
		&inst.ExpireTime,
		&inst.SubscriptionPlanID,
		&inst.CreateTime,
		&inst.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询机构失败: %w", err)
	}
	return inst, nil
}

// SelectByStudentID 按学生ID查机构
func (m *InstitutionMapper) SelectByStudentID(studentID int64) (*entity.Institution, error) {
	query := `
		SELECT i.id, i.institution_name, i.institution_address, i.institution_code, i.status, i.expire_time, i.subscription_plan_id, i.create_time, i.update_time
		FROM c_institution i
		INNER JOIN c_student s ON s.institution_id = i.id
		WHERE s.id = ?
	`
	row := m.db.QueryRow(query, studentID)

	inst := &entity.Institution{}
	err := row.Scan(
		&inst.ID,
		&inst.InstitutionName,
		&inst.InstitutionAddress,
		&inst.InstitutionCode,
		&inst.Status,
		&inst.ExpireTime,
		&inst.SubscriptionPlanID,
		&inst.CreateTime,
		&inst.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询机构失败: %w", err)
	}
	return inst, nil
}

// ============================================================
// StudentMapper 学生表操作
// ============================================================

// StudentMapper 学生表 c_student 的 Mapper
type StudentMapper struct {
	db *sql.DB
}

// NewStudentMapper 创建 StudentMapper
func NewStudentMapper(db *sql.DB) *StudentMapper {
	return &StudentMapper{db: db}
}

// SelectByID 按主键查学生
func (m *StudentMapper) SelectByID(id int64) (*entity.Student, error) {
	query := `SELECT id, avatar, student_name, institution_id, sex, birth, school, address, create_time, update_time FROM c_student WHERE id = ?`
	row := m.db.QueryRow(query, id)

	s := &entity.Student{}
	err := row.Scan(
		&s.ID,
		&s.Avatar,
		&s.StudentName,
		&s.InstitutionID,
		&s.Sex,
		&s.Birth,
		&s.School,
		&s.Address,
		&s.CreateTime,
		&s.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询学生失败: %w", err)
	}
	return s, nil
}

// SelectByInstitutionID 按机构ID查学生列表
func (m *StudentMapper) SelectByInstitutionID(institutionID int64) ([]*entity.Student, error) {
	query := `SELECT id, avatar, student_name, institution_id, sex, birth, school, address, create_time, update_time FROM c_student WHERE institution_id = ? ORDER BY id DESC`
	rows, err := m.db.Query(query, institutionID)
	if err != nil {
		return nil, fmt.Errorf("查询学生列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Student
	for rows.Next() {
		s := &entity.Student{}
		err := rows.Scan(
			&s.ID,
			&s.Avatar,
			&s.StudentName,
			&s.InstitutionID,
			&s.Sex,
			&s.Birth,
			&s.School,
			&s.Address,
			&s.CreateTime,
			&s.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}

// SelectByParentID 按家长ID查学生列表
func (m *StudentMapper) SelectByParentID(parentID int64) ([]*entity.Student, error) {
	query := `
		SELECT s.id, s.avatar, s.student_name, s.institution_id, s.sex, s.birth, s.school, s.address, s.create_time, s.update_time
		FROM c_student s
		INNER JOIN c_parent_student ps ON ps.student_id = s.id
		WHERE ps.parent_id = ?
		ORDER BY s.id DESC
	`
	rows, err := m.db.Query(query, parentID)
	if err != nil {
		return nil, fmt.Errorf("查询家长学生列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Student
	for rows.Next() {
		s := &entity.Student{}
		err := rows.Scan(
			&s.ID,
			&s.Avatar,
			&s.StudentName,
			&s.InstitutionID,
			&s.Sex,
			&s.Birth,
			&s.School,
			&s.Address,
			&s.CreateTime,
			&s.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}

// SelectByTeacherID 按教师ID查学生列表（通过班级关联）
func (m *StudentMapper) SelectByTeacherID(teacherID int64) ([]*entity.Student, error) {
	query := `
		SELECT DISTINCT s.id, s.avatar, s.student_name, s.institution_id, s.sex, s.birth, s.school, s.address, s.create_time, s.update_time
		FROM c_student s
		INNER JOIN c_class_student cs ON cs.student_id = s.id
		INNER JOIN c_class_teacher ct ON ct.class_id = cs.class_id
		WHERE ct.teacher_id = ?
		ORDER BY s.id DESC
	`
	rows, err := m.db.Query(query, teacherID)
	if err != nil {
		return nil, fmt.Errorf("查询教师学生列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Student
	for rows.Next() {
		s := &entity.Student{}
		err := rows.Scan(
			&s.ID,
			&s.Avatar,
			&s.StudentName,
			&s.InstitutionID,
			&s.Sex,
			&s.Birth,
			&s.School,
			&s.Address,
			&s.CreateTime,
			&s.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}

// ============================================================
// TeacherMapper 教师表操作
// ============================================================

// TeacherMapper 教师表 c_teacher 的 Mapper
type TeacherMapper struct {
	db *sql.DB
}

// NewTeacherMapper 创建 TeacherMapper
func NewTeacherMapper(db *sql.DB) *TeacherMapper {
	return &TeacherMapper{db: db}
}

// SelectByID 按主键查教师
func (m *TeacherMapper) SelectByID(id int64) (*entity.Teacher, error) {
	query := `SELECT id, user_id, is_available, username, institution_id, is_institution_admin, phone FROM c_teacher WHERE id = ?`
	row := m.db.QueryRow(query, id)

	t := &entity.Teacher{}
	err := row.Scan(
		&t.TeacherID,
		&t.UserID,
		&t.IsAvailable,
		&t.Username,
		&t.InstitutionID,
		&t.IsInstitutionAdmin,
		&t.Phone,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询教师失败: %w", err)
	}
	return t, nil
}

// SelectByInstitutionID 按机构ID查教师列表
func (m *TeacherMapper) SelectByInstitutionID(institutionID int64) ([]*entity.Teacher, error) {
	query := `SELECT id, user_id, is_available, username, institution_id, is_institution_admin, phone FROM c_teacher WHERE institution_id = ? ORDER BY id DESC`
	rows, err := m.db.Query(query, institutionID)
	if err != nil {
		return nil, fmt.Errorf("查询教师列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Teacher
	for rows.Next() {
		t := &entity.Teacher{}
		err := rows.Scan(
			&t.TeacherID,
			&t.UserID,
			&t.IsAvailable,
			&t.Username,
			&t.InstitutionID,
			&t.IsInstitutionAdmin,
			&t.Phone,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教师记录失败: %w", err)
		}
		list = append(list, t)
	}
	return list, nil
}

// Insert 新增教师（对齐 Java TeacherMapper.insert）
//
// 插入 c_teacher 表，返回自增主键 ID
//
// 参数：
//   - userID: 关联用户ID（c_user.id）
//   - username: 用户名
//   - institutionID: 机构ID
//   - phone: 手机号
//
// 返回：教师ID
func (m *TeacherMapper) Insert(userID int64, username string, institutionID int64, phone string) (int64, error) {
	query := `INSERT INTO c_teacher (user_id, is_available, username, institution_id, is_institution_admin, phone) VALUES (?, 1, ?, ?, 0, ?)`
	result, err := m.db.Exec(query, userID, username, institutionID, phone)
	if err != nil {
		return 0, fmt.Errorf("新增教师失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取教师ID失败: %w", err)
	}
	return id, nil
}

// UpdateByID 按ID更新教师信息（对齐 Java TeacherMapper.updateById）
//
// 仅更新非空字段
//
// 参数：
//   - id: 教师ID
//   - username: 用户名（空字符串表示不更新）
//   - phone: 手机号（空字符串表示不更新）
//   - isAvailable: 是否可用（nil 表示不更新）
//   - isInstitutionAdmin: 是否机构管理员（nil 表示不更新）
//
// 返回：影响行数
func (m *TeacherMapper) UpdateByID(id int64, username, phone string, isAvailable *bool, isInstitutionAdmin *bool) (int64, error) {
	setParts := []string{}
	args := []interface{}{}
	if username != "" {
		setParts = append(setParts, "username = ?")
		args = append(args, username)
	}
	if phone != "" {
		setParts = append(setParts, "phone = ?")
		args = append(args, phone)
	}
	if isAvailable != nil {
		setParts = append(setParts, "is_available = ?")
		args = append(args, *isAvailable)
	}
	if isInstitutionAdmin != nil {
		setParts = append(setParts, "is_institution_admin = ?")
		args = append(args, *isInstitutionAdmin)
	}

	if len(setParts) == 0 {
		return 0, nil // 没有字段需要更新
	}

	query := fmt.Sprintf("UPDATE c_teacher SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新教师失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// DeleteByID 按主键删除教师（对齐 Java TeacherMapper.deleteById）
func (m *TeacherMapper) DeleteByID(id int64) (int64, error) {
	query := `DELETE FROM c_teacher WHERE id = ?`
	result, err := m.db.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("删除教师失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// InstitutionMapper 写操作补充
// ============================================================

// UpdateByID 按ID更新机构信息（对齐 Java InstitutionMapper.updateById）
//
// 仅更新非空字段
//
// 参数：
//   - id: 机构ID
//   - name: 机构名称（空字符串表示不更新）
//   - address: 机构地址（空字符串表示不更新）
//   - status: 机构状态（-1 表示不更新）
//   - expireTime: 过期时间（空字符串表示不更新，"NULL" 表示设为 NULL）
//
// 返回：影响行数
func (m *InstitutionMapper) UpdateByID(id int64, name, address string, status int64, expireTime string) (int64, error) {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if name != "" {
		setParts = append(setParts, "institution_name = ?")
		args = append(args, name)
	}
	if address != "" {
		setParts = append(setParts, "institution_address = ?")
		args = append(args, address)
	}
	if status != -1 {
		setParts = append(setParts, "status = ?")
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

	query := fmt.Sprintf("UPDATE c_institution SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新机构失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// StudentMapper 写操作补充
// ============================================================

// SelectByClassID 按班级ID查学生列表（对齐 Java StudentMapper.selectStudentByClassId）
//
// 通过 c_class_student 关联查询，包含课卡剩余课时和家长信息
func (m *StudentMapper) SelectByClassID(classID int64) ([]*entity.Student, error) {
	query := `
		SELECT s.id, s.avatar, s.student_name, s.institution_id, s.sex, s.birth, s.school, s.address, s.create_time, s.update_time
		FROM c_class_student AS cs
		LEFT JOIN c_class c ON c.id = cs.class_id
		LEFT JOIN c_student s ON s.id = cs.student_id
		WHERE cs.class_id = ?
		ORDER BY s.id DESC
	`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("查询班级学生列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Student
	for rows.Next() {
		s := &entity.Student{}
		err := rows.Scan(
			&s.ID, &s.Avatar, &s.StudentName, &s.InstitutionID,
			&s.Sex, &s.Birth, &s.School, &s.Address,
			&s.CreateTime, &s.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}

// SelectByCourseID 按课程ID查选修学生列表（对齐 Java StudentMapper.selectStudentByCourseId）
//
// 通过 c_course_record 关联查询选修了该课程的学生
func (m *StudentMapper) SelectByCourseID(courseID int64) ([]*entity.Student, error) {
	query := `
		SELECT s.id, s.avatar, s.student_name, s.institution_id, s.sex, s.birth, s.school, s.address, s.create_time, s.update_time
		FROM c_student AS s
		LEFT JOIN c_course_record AS cr ON s.id = cr.student_id
		WHERE cr.course_id = ? AND cr.is_delete = 0
		ORDER BY s.update_time DESC, s.id DESC
	`
	rows, err := m.db.Query(query, courseID)
	if err != nil {
		return nil, fmt.Errorf("查询课程学生列表失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.Student
	for rows.Next() {
		s := &entity.Student{}
		err := rows.Scan(
			&s.ID, &s.Avatar, &s.StudentName, &s.InstitutionID,
			&s.Sex, &s.Birth, &s.School, &s.Address,
			&s.CreateTime, &s.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}

// Insert 新增学生（对齐 Java StudentMapper.insert）
//
// 插入 c_student 表，返回自增主键 ID
//
// 参数：
//   - avatar: 头像URL
//   - studentName: 学生姓名
//   - institutionID: 机构ID
//   - sex: 性别（0=未知,1=男,2=女）
//   - birth: 出生日期（空字符串表示 NULL）
//   - school: 学校
//   - address: 地址
//
// 返回：学生ID
func (m *StudentMapper) Insert(avatar, studentName string, institutionID, sex int64, birth, school, address string) (int64, error) {
	// 处理可选参数的 NULL 值
	var birthArg interface{}
	if birth != "" {
		birthArg = birth
	} else {
		birthArg = nil
	}

	query := `INSERT INTO c_student (avatar, student_name, institution_id, sex, birth, school, address, create_time, update_time) VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, avatar, studentName, institutionID, sex, birthArg, school, address)
	if err != nil {
		return 0, fmt.Errorf("新增学生失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取学生ID失败: %w", err)
	}
	return id, nil
}

// UpdateByID 按ID更新学生信息（对齐 Java StudentMapper.updateById）
//
// 仅更新非空字段
//
// 参数：
//   - id: 学生ID
//   - avatar: 头像URL（空字符串表示不更新）
//   - studentName: 学生姓名（空字符串表示不更新）
//   - sex: 性别（-1 表示不更新）
//   - birth: 出生日期（空字符串表示不更新，"NULL" 表示设为 NULL）
//   - school: 学校（空字符串表示不更新）
//   - address: 地址（空字符串表示不更新）
//
// 返回：影响行数
func (m *StudentMapper) UpdateByID(id int64, avatar, studentName string, sex int64, birth, school, address string) (int64, error) {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if avatar != "" {
		setParts = append(setParts, "avatar = ?")
		args = append(args, avatar)
	}
	if studentName != "" {
		setParts = append(setParts, "student_name = ?")
		args = append(args, studentName)
	}
	if sex != -1 {
		setParts = append(setParts, "sex = ?")
		args = append(args, sex)
	}
	if birth != "" {
		if birth == "NULL" {
			setParts = append(setParts, "birth = NULL")
		} else {
			setParts = append(setParts, "birth = ?")
			args = append(args, birth)
		}
	}
	if school != "" {
		setParts = append(setParts, "school = ?")
		args = append(args, school)
	}
	if address != "" {
		setParts = append(setParts, "address = ?")
		args = append(args, address)
	}

	query := fmt.Sprintf("UPDATE c_student SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新学生失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// ParentStudentMapper 家长-学生关联表操作
// ============================================================

// ParentStudentMapper 家长-学生关联表 c_parent_student 的 Mapper
type ParentStudentMapper struct {
	db *sql.DB
}

// NewParentStudentMapper 创建 ParentStudentMapper
func NewParentStudentMapper(db *sql.DB) *ParentStudentMapper {
	return &ParentStudentMapper{db: db}
}

// SelectByParentAndStudent 查询家长-学生关联记录（用于解绑和取消订阅）
//
// 返回 isPrimary 字段用于定位联系人角色
func (m *ParentStudentMapper) SelectByParentAndStudent(parentID, studentID int64) (bool, error) {
	query := `SELECT is_primary FROM c_parent_student WHERE parent_id = ? AND student_id = ? LIMIT 1`
	var isPrimary sql.NullBool
	err := m.db.QueryRow(query, parentID, studentID).Scan(&isPrimary)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // 未找到记录，返回 false
		}
		return false, fmt.Errorf("查询家长学生关联失败: %w", err)
	}
	return isPrimary.Bool, nil
}

// DeleteByParentAndStudent 删除家长-学生关联记录（对齐 Java unbindStudent 中的删除逻辑）
func (m *ParentStudentMapper) DeleteByParentAndStudent(parentID, studentID int64) (int64, error) {
	query := `DELETE FROM c_parent_student WHERE parent_id = ? AND student_id = ?`
	result, err := m.db.Exec(query, parentID, studentID)
	if err != nil {
		return 0, fmt.Errorf("删除家长学生关联失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// CountByParentID 统计家长还有多少个学生关联（用于判断是否需要重置家长为未绑定状态）
func (m *ParentStudentMapper) CountByParentID(parentID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM c_parent_student WHERE parent_id = ?`
	var count int64
	err := m.db.QueryRow(query, parentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计家长关联数失败: %w", err)
	}
	return count, nil
}

// ============================================================
// ParentMapper 家长表操作
// ============================================================

// ParentMapper 家长表 c_parent 的 Mapper
type ParentMapper struct {
	db *sql.DB
}

// NewParentMapper 创建 ParentMapper
func NewParentMapper(db *sql.DB) *ParentMapper {
	return &ParentMapper{db: db}
}

// SelectByID 按主键查家长
func (m *ParentMapper) SelectByID(id int64) (*entity.Parent, error) {
	query := `SELECT id, user_id, is_available, username, phone, is_bound, create_time, update_time FROM c_parent WHERE id = ?`
	row := m.db.QueryRow(query, id)

	p := &entity.Parent{}
	err := row.Scan(
		&p.ParentID, &p.UserID, &p.IsAvailable, &p.Username,
		&p.Phone, &p.IsBound, &p.CreateTime, &p.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询家长失败: %w", err)
	}
	return p, nil
}

// DeleteByID 按主键删除家长（用于解绑时删除占位 parent）
func (m *ParentMapper) DeleteByID(id int64) (int64, error) {
	query := `DELETE FROM c_parent WHERE id = ?`
	result, err := m.db.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("删除家长失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ResetUnbound 重置家长为未绑定状态（用于解绑后无剩余关联时）
func (m *ParentMapper) ResetUnbound(id int64) (int64, error) {
	query := `UPDATE c_parent SET is_bound = 0, user_id = NULL, update_time = NOW() WHERE id = ?`
	result, err := m.db.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("重置家长绑定状态失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// UserAuthMapper 用户认证表操作（教师删除/新增时使用）
// ============================================================

// UserAuthMapper 用户认证表 c_user_auth 的 Mapper
type UserAuthMapper struct {
	db *sql.DB
}

// NewUserAuthMapper 创建 UserAuthMapper
func NewUserAuthMapper(db *sql.DB) *UserAuthMapper {
	return &UserAuthMapper{db: db}
}

// SelectByTeacherID 按教师ID查用户认证记录（对齐 Java userAuthMapper.selectAuthByTeacherId）
//
// 通过 c_teacher.user_id 关联查询
func (m *UserAuthMapper) SelectByTeacherID(teacherID int64) (*entity.UserAuth, error) {
	query := `
		SELECT ua.id, ua.user_id, ua.role_id, ua.account, ua.password, ua.salt, ua.last_login_time
		FROM c_user_auth ua
		INNER JOIN c_teacher t ON t.user_id = ua.user_id
		WHERE t.id = ? AND ua.role_id = 4
		LIMIT 1
	`
	row := m.db.QueryRow(query, teacherID)

	ua := &entity.UserAuth{}
	err := row.Scan(
		&ua.ID, &ua.UserID, &ua.RoleID, &ua.Account,
		&ua.Password, &ua.Salt, &ua.LastLoginTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户认证失败: %w", err)
	}
	return ua, nil
}

// DeleteByID 按主键删除用户认证记录（用于删除教师时清理）
func (m *UserAuthMapper) DeleteByID(id int64) (int64, error) {
	query := `DELETE FROM c_user_auth WHERE id = ?`
	result, err := m.db.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("删除用户认证失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// Insert 新增用户认证记录（对齐 Java UserAuthMapper.insert，用于教师创建）
//
// 参数：
//   - userID: 关联用户ID（c_user.id）
//   - roleID: 角色ID（4=教师）
//   - account: 登录账号（手机号或用户名）
//   - password: 密码（SM3 加盐哈希后的值）
//   - salt: 盐值（32 位 UUID 去横杠）
//
// 返回：认证记录ID
func (m *UserAuthMapper) Insert(userID int64, roleID int64, account, password, salt string) (int64, error) {
	query := `INSERT INTO c_user_auth (user_id, role_id, account, password, salt, last_login_time) VALUES (?, ?, ?, ?, ?, NOW())`
	result, err := m.db.Exec(query, userID, roleID, account, password, salt)
	if err != nil {
		return 0, fmt.Errorf("新增用户认证失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取认证记录ID失败: %w", err)
	}
	return id, nil
}

// UpdateAccountAndPassword 更新账号和密码（对齐 Java 教师更新流程）
//
// 参数：
//   - id: 认证记录ID
//   - account: 新账号（空字符串表示不更新）
//   - password: 新密码（SM3 加盐哈希值，空字符串表示不更新）
//   - salt: 新盐值（空字符串表示不更新）
//
// 返回：影响行数
func (m *UserAuthMapper) UpdateAccountAndPassword(id int64, account, password, salt string) (int64, error) {
	setParts := []string{}
	args := []interface{}{}
	if account != "" {
		setParts = append(setParts, "account = ?")
		args = append(args, account)
	}
	if password != "" {
		setParts = append(setParts, "password = ?")
		args = append(args, password)
	}
	if salt != "" {
		setParts = append(setParts, "salt = ?")
		args = append(args, salt)
	}

	if len(setParts) == 0 {
		return 0, nil
	}

	query := fmt.Sprintf("UPDATE c_user_auth SET %s WHERE id = ?", joinStrings(setParts, ", "))
	args = append(args, id)

	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("更新用户认证失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// SelectByUserID 按用户ID查认证记录（用于教师更新时查找认证记录）
//
// 参数：
//   - userID: 用户ID（c_user.id）
//   - roleID: 角色ID（4=教师）
//
// 返回：认证记录，未找到返回 nil
func (m *UserAuthMapper) SelectByUserID(userID, roleID int64) (*entity.UserAuth, error) {
	query := `SELECT id, user_id, role_id, account, password, salt, last_login_time FROM c_user_auth WHERE user_id = ? AND role_id = ? LIMIT 1`
	row := m.db.QueryRow(query, userID, roleID)

	ua := &entity.UserAuth{}
	err := row.Scan(
		&ua.ID, &ua.UserID, &ua.RoleID, &ua.Account,
		&ua.Password, &ua.Salt, &ua.LastLoginTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户认证失败: %w", err)
	}
	return ua, nil
}

// ============================================================
// UserMapper 用户表操作（教师新增/删除时使用）
// ============================================================

// UserMapper 用户表 c_user 的 Mapper
type UserMapper struct {
	db *sql.DB
}

// NewUserMapper 创建 UserMapper
func NewUserMapper(db *sql.DB) *UserMapper {
	return &UserMapper{db: db}
}

// Insert 新增用户记录（对齐 Java userMapper.insert）
//
// 参数：
//   - institutionID: 机构ID
//
// 返回：用户ID
func (m *UserMapper) Insert(institutionID int64) (int64, error) {
	query := `INSERT INTO c_user (institution_id, create_time, update_time) VALUES (?, NOW(), NOW())`
	result, err := m.db.Exec(query, institutionID)
	if err != nil {
		return 0, fmt.Errorf("新增用户失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取用户ID失败: %w", err)
	}
	return id, nil
}

// DeleteByID 按主键删除用户（用于删除教师时清理）
func (m *UserMapper) DeleteByID(id int64) (int64, error) {
	query := `DELETE FROM c_user WHERE id = ?`
	result, err := m.db.Exec(query, id)
	if err != nil {
		return 0, fmt.Errorf("删除用户失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// WxStudentSubscribeMapper 微信学生订阅表操作
// ============================================================

// WxStudentSubscribeMapper 微信学生订阅表 c_wx_student_subscribe 的 Mapper
type WxStudentSubscribeMapper struct {
	db *sql.DB
}

// NewWxStudentSubscribeMapper 创建 WxStudentSubscribeMapper
func NewWxStudentSubscribeMapper(db *sql.DB) *WxStudentSubscribeMapper {
	return &WxStudentSubscribeMapper{db: db}
}

// DeleteByStudentAndIsPrimary 按学生ID和联系人角色删除订阅记录（对齐 Java cancelStudentSubscribe/unbindStudent）
//
// 参数：
//   - studentID: 学生ID
//   - isPrimary: 是否主联系人
//
// 返回：影响行数
func (m *WxStudentSubscribeMapper) DeleteByStudentAndIsPrimary(studentID int64, isPrimary bool) (int64, error) {
	query := `DELETE FROM c_wx_student_subscribe WHERE student_id = ? AND is_primary = ?`
	result, err := m.db.Exec(query, studentID, isPrimary)
	if err != nil {
		return 0, fmt.Errorf("删除学生订阅记录失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// WxSubscribeRecordMapper 微信订阅记录表操作
// ============================================================

// WxSubscribeRecordMapper 微信订阅记录表 c_wx_subscribe_record 的 Mapper
type WxSubscribeRecordMapper struct {
	db *sql.DB
}

// NewWxSubscribeRecordMapper 创建 WxSubscribeRecordMapper
func NewWxSubscribeRecordMapper(db *sql.DB) *WxSubscribeRecordMapper {
	return &WxSubscribeRecordMapper{db: db}
}

// DeleteByOpenIDs 按 openId 列表批量删除订阅授权记录（对齐 Java cancelStudentSubscribe 中的清理逻辑）
func (m *WxSubscribeRecordMapper) DeleteByOpenIDs(openIDs []string) (int64, error) {
	if len(openIDs) == 0 {
		return 0, nil
	}
	placeholders := ""
	args := make([]interface{}, 0, len(openIDs))
	for i, oid := range openIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, oid)
	}
	query := fmt.Sprintf("DELETE FROM c_wx_subscribe_record WHERE open_id IN (%s)", placeholders)
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("删除订阅授权记录失败: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("获取影响行数失败: %w", err)
	}
	return rows, nil
}

// ============================================================
// UserPlatformMapper 用户平台表操作（取消订阅时查 openId）
// ============================================================

// UserPlatformMapper 用户平台表 c_user_platform 的 Mapper
type UserPlatformMapper struct {
	db *sql.DB
}

// NewUserPlatformMapper 创建 UserPlatformMapper
func NewUserPlatformMapper(db *sql.DB) *UserPlatformMapper {
	return &UserPlatformMapper{db: db}
}

// SelectOpenIDsByUserID 按用户ID查所有有效的微信 openId（对齐 Java cancelStudentSubscribe 中的查询）
func (m *UserPlatformMapper) SelectOpenIDsByUserID(userID int64) ([]string, error) {
	query := `SELECT open_id FROM c_user_platform WHERE user_id = ? AND platform = 'WEIXIN' AND is_available = 1`
	rows, err := m.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户平台 openId 失败: %w", err)
	}
	defer rows.Close()

	var list []string
	for rows.Next() {
		var openID sql.NullString
		if err := rows.Scan(&openID); err != nil {
			return nil, fmt.Errorf("扫描 openId 失败: %w", err)
		}
		if openID.String != "" {
			list = append(list, openID.String)
		}
	}
	return list, nil
}
