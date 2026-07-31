// Package mapper 用户认证表操作（对齐 Java UserAuthMapper）
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// UserAuthMapper 用户认证表 c_user_auth 的 Mapper
type UserAuthMapper struct {
	db *sql.DB
}

// NewUserAuthMapper 创建 UserAuthMapper
func NewUserAuthMapper(db *sql.DB) *UserAuthMapper {
	return &UserAuthMapper{db: db}
}

// SelectAuthByAccountAndInstitution 按账号+角色+机构查认证记录
//
// 对齐 Java UserAuthMapper.selectAuthByAccountAndInstitution
// 用途：登录时校验账号密码；绑定时查找已有家长账号
//
// 参数：
//   - account: 账号（手机号或用户名）
//   - roleId: 角色ID（3=parent, 4=teacher）
//   - institutionId: 机构ID
//
// 返回：认证记录，未找到返回 nil
func (m *UserAuthMapper) SelectAuthByAccountAndInstitution(account string, roleId, institutionId int64) (*entity.UserAuth, error) {
	query := `
		SELECT ua.id, ua.user_id, ua.role_id, ua.account, ua.password, ua.salt, ua.last_login_time
		FROM c_user_auth ua
		INNER JOIN c_user u ON ua.user_id = u.id
		WHERE ua.account = ? AND ua.role_id = ? AND u.institution_id = ?
		LIMIT 1
	`
	row := m.db.QueryRow(query, account, roleId, institutionId)

	auth := &entity.UserAuth{}
	err := row.Scan(
		&auth.ID,
		&auth.UserID,
		&auth.RoleID,
		&auth.Account,
		&auth.Password,
		&auth.Salt,
		&auth.LastLoginTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询认证记录失败: %w", err)
	}
	return auth, nil
}

// SelectByUserID 按用户ID查认证记录
func (m *UserAuthMapper) SelectByUserID(userID int64) (*entity.UserAuth, error) {
	query := `SELECT id, user_id, role_id, account, password, salt, last_login_time FROM c_user_auth WHERE user_id = ? LIMIT 1`
	row := m.db.QueryRow(query, userID)

	auth := &entity.UserAuth{}
	err := row.Scan(
		&auth.ID,
		&auth.UserID,
		&auth.RoleID,
		&auth.Account,
		&auth.Password,
		&auth.Salt,
		&auth.LastLoginTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询认证记录失败: %w", err)
	}
	return auth, nil
}

// SelectAuthByTeacherId 按教师ID查认证记录
//
// 对齐 Java UserAuthMapper.selectAuthByTeacherId
func (m *UserAuthMapper) SelectAuthByTeacherId(teacherId int64) (*entity.UserAuth, error) {
	query := `
		SELECT ua.id, ua.user_id, ua.role_id, ua.account, ua.password, ua.salt, ua.last_login_time
		FROM c_user_auth ua
		INNER JOIN c_teacher t ON ua.user_id = t.user_id
		WHERE t.id = ? AND ua.role_id = 4
		LIMIT 1
	`
	row := m.db.QueryRow(query, teacherId)

	auth := &entity.UserAuth{}
	err := row.Scan(
		&auth.ID,
		&auth.UserID,
		&auth.RoleID,
		&auth.Account,
		&auth.Password,
		&auth.Salt,
		&auth.LastLoginTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询教师认证记录失败: %w", err)
	}
	return auth, nil
}

// ExistsByInstitutionAndAccountAndRole 检查账号是否已存在
//
// 对齐 Java UserAuthMapper.existsByInstitutionAndAccountAndRole
// 用途：注册时去重
//
// 参数：
//   - institutionId: 机构ID
//   - account: 账号
//   - roleId: 角色ID
//
// 返回：存在返回 true
func (m *UserAuthMapper) ExistsByInstitutionAndAccountAndRole(institutionId int64, account string, roleId int64) (bool, error) {
	query := `
		SELECT COUNT(1)
		FROM c_user_auth ua
		INNER JOIN c_user u ON ua.user_id = u.id
		WHERE u.institution_id = ? AND ua.account = ? AND ua.role_id = ?
	`
	var count int64
	err := m.db.QueryRow(query, institutionId, account, roleId).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查账号存在性失败: %w", err)
	}
	return count > 0, nil
}

// ExistsByUserIDAndRole 检查用户ID+角色是否已有认证记录
//
// 用途：注册时去重（同 userId + role 不能重复注册）
func (m *UserAuthMapper) ExistsByUserIDAndRole(userID, roleId int64) (bool, error) {
	query := `SELECT COUNT(1) FROM c_user_auth WHERE user_id = ? AND role_id = ?`
	var count int64
	err := m.db.QueryRow(query, userID, roleId).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户认证存在性失败: %w", err)
	}
	return count > 0, nil
}

// Insert 新增认证记录
func (m *UserAuthMapper) Insert(auth *entity.UserAuth) (int64, error) {
	query := `INSERT INTO c_user_auth (user_id, role_id, account, password, salt, last_login_time)
	          VALUES (?, ?, ?, ?, ?, NOW())`
	result, err := m.db.Exec(query,
		auth.UserID,
		auth.RoleID,
		auth.Account,
		auth.Password,
		auth.Salt,
	)
	if err != nil {
		return 0, fmt.Errorf("新增认证记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取认证记录ID失败: %w", err)
	}
	return id, nil
}

// UpdateLastLoginTime 更新最后登录时间
func (m *UserAuthMapper) UpdateLastLoginTime(authID int64) error {
	query := `UPDATE c_user_auth SET last_login_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, authID)
	if err != nil {
		return fmt.Errorf("更新登录时间失败: %w", err)
	}
	return nil
}

// UpdatePassword 更新密码
func (m *UserAuthMapper) UpdatePassword(authID int64, hashedPassword, salt string) error {
	query := `UPDATE c_user_auth SET password = ?, salt = ? WHERE id = ?`
	_, err := m.db.Exec(query, hashedPassword, salt, authID)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return nil
}
