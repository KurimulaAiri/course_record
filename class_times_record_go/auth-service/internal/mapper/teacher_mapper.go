// Package mapper 教师表操作（对齐 Java TeacherMapper）
package mapper

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// TeacherMapper 教师表操作（对齐 Java TeacherMapper）
// ============================================================

// TeacherMapper 教师表 c_teacher 的 Mapper
//
// 对齐 Java IdentityServiceImpl.getByUserId("teacher", userId)
// 用途：登录后查询教师身份信息，构造 UserVO.IdentityInfo
type TeacherMapper struct {
	db *sql.DB
}

// NewTeacherMapper 创建 TeacherMapper
//
// 参数：
//   - db: 数据库连接
func NewTeacherMapper(db *sql.DB) *TeacherMapper {
	return &TeacherMapper{db: db}
}

// SelectByUserID 按用户ID查教师
//
// 对齐 Java IdentityServiceImpl.getByUserId("teacher", userId)
// 查询 c_teacher 表 WHERE user_id = ?
//
// 参数：
//   - userID: 用户ID（c_user.id）
//
// 返回：
//   - *entity.Teacher: 教师实体，未找到返回 nil
//   - error: 查询错误
func (m *TeacherMapper) SelectByUserID(userID int64) (*entity.Teacher, error) {
	// 查询 c_teacher 表，按 user_id 过滤
	query := `
		SELECT id, user_id, is_available, username, institution_id, is_institution_admin, phone
		FROM c_teacher
		WHERE user_id = ?
		LIMIT 1
	`
	row := m.db.QueryRow(query, userID)

	teacher := &entity.Teacher{}
	var phone sql.NullString
	err := row.Scan(
		&teacher.TeacherID,
		&teacher.UserID,
		&teacher.IsAvailable,
		&teacher.Username,
		&teacher.InstitutionID,
		&teacher.IsInstitutionAdmin,
		&phone,
	)
	if err != nil {
		// 未找到记录，返回 nil 不报错（对齐 Java getByUserId 返回 null 的行为）
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询教师失败: %w", err)
	}
	teacher.Phone = phone
	return teacher, nil
}
