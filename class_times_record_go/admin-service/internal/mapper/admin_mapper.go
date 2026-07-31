// Package mapper admin-service 数据访问层
//
// 对齐 Java admin-service/src/main/java/com/shiroko/mapper 包
//
// 包含：
//   - AdminUserMapper：系统用户表 sys_user 操作
//   - AdminRoleMapper：系统角色表 sys_role 操作
//   - AdminMenuMapper：系统菜单表 sys_menu 操作
package mapper

import (
	"database/sql"
	"fmt"
)

// ============================================================
// AdminUserMapper 系统用户表操作
// ============================================================

// AdminUserMapper 系统用户表 sys_user 的 Mapper
type AdminUserMapper struct {
	db *sql.DB
}

// NewAdminUserMapper 创建 AdminUserMapper
func NewAdminUserMapper(db *sql.DB) *AdminUserMapper {
	return &AdminUserMapper{db: db}
}

// AdminUser 系统用户实体（对齐 Java SysUser）
type AdminUser struct {
	ID           int64          `json:"id"`           // 主键
	Username     string         `json:"username"`     // 用户名
	Password     string         `json:"password"`     // 密码（BCrypt 哈希）
	Nickname     string         `json:"nickname"`     // 昵称
	Avatar       string         `json:"avatar"`       // 头像
	Email        string         `json:"email"`        // 邮箱
	Phone        string         `json:"phone"`        // 手机号
	Status       int64          `json:"status"`       // 状态（0=禁用,1=启用）
	CreateTime   sql.NullTime   `json:"createTime"`   // 创建时间
	UpdateTime   sql.NullTime   `json:"updateTime"`   // 更新时间
}

// SelectByUsername 按用户名查系统用户（登录用）
//
// 对齐 Java SysUserMapper.selectByUsername
func (m *AdminUserMapper) SelectByUsername(username string) (*AdminUser, error) {
	query := `SELECT id, username, password, nickname, avatar, email, phone, status, create_time, update_time FROM sys_user WHERE username = ? LIMIT 1`
	row := m.db.QueryRow(query, username)

	u := &AdminUser{}
	var nickname, avatar, email, phone sql.NullString
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&nickname,
		&avatar,
		&email,
		&phone,
		&u.Status,
		&u.CreateTime,
		&u.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询系统用户失败: %w", err)
	}
	u.Nickname = nickname.String
	u.Avatar = avatar.String
	u.Email = email.String
	u.Phone = phone.String
	return u, nil
}

// SelectByID 按主键查系统用户
func (m *AdminUserMapper) SelectByID(id int64) (*AdminUser, error) {
	query := `SELECT id, username, password, nickname, avatar, email, phone, status, create_time, update_time FROM sys_user WHERE id = ?`
	row := m.db.QueryRow(query, id)

	u := &AdminUser{}
	var nickname, avatar, email, phone sql.NullString
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&nickname,
		&avatar,
		&email,
		&phone,
		&u.Status,
		&u.CreateTime,
		&u.UpdateTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询系统用户失败: %w", err)
	}
	u.Nickname = nickname.String
	u.Avatar = avatar.String
	u.Email = email.String
	u.Phone = phone.String
	return u, nil
}

// SelectAll 查询所有系统用户（分页）
func (m *AdminUserMapper) SelectAll(offset, limit int) ([]*AdminUser, error) {
	query := `SELECT id, username, password, nickname, avatar, email, phone, status, create_time, update_time FROM sys_user ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := m.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询系统用户列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminUser
	for rows.Next() {
		u := &AdminUser{}
		var nickname, avatar, email, phone sql.NullString
		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Password,
			&nickname,
			&avatar,
			&email,
			&phone,
			&u.Status,
			&u.CreateTime,
			&u.UpdateTime,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描系统用户失败: %w", err)
		}
		u.Nickname = nickname.String
		u.Avatar = avatar.String
		u.Email = email.String
		u.Phone = phone.String
		list = append(list, u)
	}
	return list, nil
}

// Count 总用户数
func (m *AdminUserMapper) Count() (int64, error) {
	query := `SELECT COUNT(1) FROM sys_user`
	var count int64
	err := m.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计用户数失败: %w", err)
	}
	return count, nil
}

// Insert 新增系统用户
func (m *AdminUserMapper) Insert(u *AdminUser) (int64, error) {
	query := `INSERT INTO sys_user (username, password, nickname, avatar, email, phone, status, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, u.Username, u.Password, u.Nickname, u.Avatar, u.Email, u.Phone, u.Status)
	if err != nil {
		return 0, fmt.Errorf("新增系统用户失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取用户ID失败: %w", err)
	}
	return id, nil
}

// UpdatePassword 更新密码
func (m *AdminUserMapper) UpdatePassword(id int64, hashedPassword string) error {
	query := `UPDATE sys_user SET password = ?, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, hashedPassword, id)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return nil
}
