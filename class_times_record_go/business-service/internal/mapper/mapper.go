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
