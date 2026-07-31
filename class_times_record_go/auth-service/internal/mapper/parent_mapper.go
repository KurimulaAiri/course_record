// Package mapper 家长、机构、学生、家长-学生关联表操作
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// ParentMapper 家长表操作（对齐 Java ParentMapper）
// ============================================================

// ParentMapper 家长表 c_parent 的 Mapper
type ParentMapper struct {
	db *sql.DB
}

// NewParentMapper 创建 ParentMapper
func NewParentMapper(db *sql.DB) *ParentMapper {
	return &ParentMapper{db: db}
}

// SelectByUserID 按用户ID查家长
func (m *ParentMapper) SelectByUserID(userID int64) (*entity.Parent, error) {
	query := `
		SELECT id, user_id, is_available, username, phone, is_bound, create_time, update_time
		FROM c_parent
		WHERE user_id = ?
		ORDER BY id ASC
		LIMIT 1
	`
	row := m.db.QueryRow(query, userID)

	p := &entity.Parent{}
	err := row.Scan(
		&p.ParentID,
		&p.UserID,
		&p.IsAvailable,
		&p.Username,
		&p.Phone,
		&p.IsBound,
		&p.CreateTime,
		&p.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询家长失败: %w", err)
	}
	return p, nil
}

// SelectByID 按主键查家长
func (m *ParentMapper) SelectByID(parentID int64) (*entity.Parent, error) {
	query := `SELECT id, user_id, is_available, username, phone, is_bound, create_time, update_time FROM c_parent WHERE id = ?`
	row := m.db.QueryRow(query, parentID)

	p := &entity.Parent{}
	err := row.Scan(
		&p.ParentID,
		&p.UserID,
		&p.IsAvailable,
		&p.Username,
		&p.Phone,
		&p.IsBound,
		&p.CreateTime,
		&p.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询家长失败: %w", err)
	}
	return p, nil
}

// Insert 新增家长
func (m *ParentMapper) Insert(p *entity.Parent) (int64, error) {
	query := `INSERT INTO c_parent (user_id, is_available, username, phone, is_bound, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query,
		p.UserID,
		p.IsAvailable,
		p.Username,
		p.Phone,
		p.IsBound,
	)
	if err != nil {
		return 0, fmt.Errorf("新增家长失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取家长ID失败: %w", err)
	}
	return id, nil
}

// UpdateBindInfo 更新家长绑定信息（绑定微信用户后调用）
//
// 参数：
//   - parentID: 家长ID
//   - username: 家长姓名
//   - phone: 手机号
func (m *ParentMapper) UpdateBindInfo(parentID int64, username, phone string) error {
	query := `UPDATE c_parent SET username = ?, phone = ?, is_bound = 1, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, username, phone, parentID)
	if err != nil {
		return fmt.Errorf("更新家长绑定信息失败: %w", err)
	}
	return nil
}

// ============================================================
// InstitutionMapper 机构表操作（对齐 Java InstitutionMapper）
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

// SelectByCode 按机构编码查（登录时用）
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

// ============================================================
// StudentMapper 学生表操作（绑定流程需要）
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

// SelectByParentAndStudent 按家长ID+学生ID查关联
//
// 用途：绑定时去重（同一家长不能重复绑定同一学生）
func (m *ParentStudentMapper) SelectByParentAndStudent(parentID, studentID int64) (*entity.ParentStudent, error) {
	query := `SELECT id, parent_id, student_id, is_primary, relation, create_time, update_time FROM c_parent_student WHERE parent_id = ? AND student_id = ? LIMIT 1`
	row := m.db.QueryRow(query, parentID, studentID)

	ps := &entity.ParentStudent{}
	err := row.Scan(
		&ps.ID,
		&ps.ParentID,
		&ps.StudentID,
		&ps.IsPrimary,
		&ps.Relation,
		&ps.CreateTime,
		&ps.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询家长学生关联失败: %w", err)
	}
	return ps, nil
}

// SelectByStudentID 按学生ID查所有关联家长
func (m *ParentStudentMapper) SelectByStudentID(studentID int64) ([]*entity.ParentStudent, error) {
	query := `SELECT id, parent_id, student_id, is_primary, relation, create_time, update_time FROM c_parent_student WHERE student_id = ?`
	rows, err := m.db.Query(query, studentID)
	if err != nil {
		return nil, fmt.Errorf("查询学生家长关联失败: %w", err)
	}
	defer rows.Close()

	var list []*entity.ParentStudent
	for rows.Next() {
		ps := &entity.ParentStudent{}
		err := rows.Scan(
			&ps.ID,
			&ps.ParentID,
			&ps.StudentID,
			&ps.IsPrimary,
			&ps.Relation,
			&ps.CreateTime,
			&ps.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描家长学生关联失败: %w", err)
		}
		list = append(list, ps)
	}
	return list, nil
}

// Insert 新增家长-学生关联
func (m *ParentStudentMapper) Insert(ps *entity.ParentStudent) (int64, error) {
	query := `INSERT INTO c_parent_student (parent_id, student_id, is_primary, relation, create_time, update_time)
	          VALUES (?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query,
		ps.ParentID,
		ps.StudentID,
		ps.IsPrimary,
		ps.Relation,
	)
	if err != nil {
		return 0, fmt.Errorf("新增家长学生关联失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取关联ID失败: %w", err)
	}
	return id, nil
}
