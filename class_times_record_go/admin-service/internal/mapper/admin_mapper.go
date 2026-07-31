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
	Remark       string         `json:"remark"`       // 备注
	IsDeleted    int64          `json:"isDeleted"`    // 逻辑删除（0=未删除,1=已删除）
	CreateTime   sql.NullTime   `json:"createTime"`   // 创建时间
	UpdateTime   sql.NullTime   `json:"updateTime"`   // 更新时间
}

// SelectByUsername 按用户名查系统用户（登录用）
//
// 对齐 Java SysUserMapper.selectByUsername
// 只查询未逻辑删除的记录（is_deleted=0，对齐 Java @TableLogic 自动过滤）
func (m *AdminUserMapper) SelectByUsername(username string) (*AdminUser, error) {
	query := `SELECT id, username, password, nickname, avatar, email, phone, status, remark, is_deleted, create_time, update_time FROM sys_user WHERE username = ? AND is_deleted = 0 LIMIT 1`
	row := m.db.QueryRow(query, username)

	u := &AdminUser{}
	var nickname, avatar, email, phone, remark sql.NullString
	var isDeleted sql.NullInt64
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&nickname,
		&avatar,
		&email,
		&phone,
		&u.Status,
		&remark,
		&isDeleted,
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
	u.Remark = remark.String
	if isDeleted.Valid {
		u.IsDeleted = isDeleted.Int64
	}
	return u, nil
}

// SelectByID 按主键查系统用户
//
// 只查询未逻辑删除的记录（is_deleted=0，对齐 Java @TableLogic 自动过滤）
func (m *AdminUserMapper) SelectByID(id int64) (*AdminUser, error) {
	query := `SELECT id, username, password, nickname, avatar, email, phone, status, remark, is_deleted, create_time, update_time FROM sys_user WHERE id = ? AND is_deleted = 0`
	row := m.db.QueryRow(query, id)

	u := &AdminUser{}
	var nickname, avatar, email, phone, remark sql.NullString
	var isDeleted sql.NullInt64
	err := row.Scan(
		&u.ID,
		&u.Username,
		&u.Password,
		&nickname,
		&avatar,
		&email,
		&phone,
		&u.Status,
		&remark,
		&isDeleted,
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
	u.Remark = remark.String
	if isDeleted.Valid {
		u.IsDeleted = isDeleted.Int64
	}
	return u, nil
}

// SelectAll 查询所有系统用户（分页）
//
// 兼容旧调用，等价于 SelectList("", "", 0, offset, limit)
func (m *AdminUserMapper) SelectAll(offset, limit int) ([]*AdminUser, error) {
	return m.SelectList("", "", 0, offset, limit)
}

// SelectList 按条件查询系统用户列表（分页）
//
// 对齐 admin 前端 GetUserListRequest 的筛选条件：
//   - username: 用户名模糊查询（LIKE '%username%'），空字符串表示不筛选
//   - phone: 手机号模糊查询（LIKE '%phone%'），空字符串表示不筛选
//   - status: 状态筛选（0=不筛选,1=启用,2=禁用）
//   - offset/limit: 分页参数
//
// 只查询未逻辑删除的记录（is_deleted=0，对齐 Java @TableLogic 自动过滤）
func (m *AdminUserMapper) SelectList(username, phone string, status int64, offset, limit int) ([]*AdminUser, error) {
	// 动态构造 WHERE 条件
	query := `SELECT id, username, password, nickname, avatar, email, phone, status, remark, is_deleted, create_time, update_time FROM sys_user WHERE is_deleted = 0`
	args := []interface{}{}
	if username != "" {
		query += ` AND username LIKE ?`
		args = append(args, "%"+username+"%")
	}
	if phone != "" {
		query += ` AND phone LIKE ?`
		args = append(args, "%"+phone+"%")
	}
	if status > 0 {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询系统用户列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminUser
	for rows.Next() {
		u := &AdminUser{}
		var nickname, avatar, email, phone, remark sql.NullString
		var isDeleted sql.NullInt64
		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Password,
			&nickname,
			&avatar,
			&email,
			&phone,
			&u.Status,
			&remark,
			&isDeleted,
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
		u.Remark = remark.String
		if isDeleted.Valid {
			u.IsDeleted = isDeleted.Int64
		}
		list = append(list, u)
	}
	return list, nil
}

// Count 总用户数
func (m *AdminUserMapper) Count() (int64, error) {
	return m.CountWithFilter("", "", 0)
}

// CountWithFilter 按条件统计用户数
//
// 筛选条件与 SelectList 保持一致
// 只统计未逻辑删除的记录（is_deleted=0，对齐 Java @TableLogic 自动过滤）
func (m *AdminUserMapper) CountWithFilter(username, phone string, status int64) (int64, error) {
	query := `SELECT COUNT(1) FROM sys_user WHERE is_deleted = 0`
	args := []interface{}{}
	if username != "" {
		query += ` AND username LIKE ?`
		args = append(args, "%"+username+"%")
	}
	if phone != "" {
		query += ` AND phone LIKE ?`
		args = append(args, "%"+phone+"%")
	}
	if status > 0 {
		query += ` AND status = ?`
		args = append(args, status)
	}

	var count int64
	err := m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计用户数失败: %w", err)
	}
	return count, nil
}

// SelectRoleIDsByUserID 查询用户的角色ID列表
//
// 对齐 Java SysUserServiceImpl.getRoleIdsByUserId
// 查询 sys_user_role 表 WHERE user_id = ?
//
// 参数：
//   - userID: 用户ID
//
// 返回：角色ID列表（如 [1, 2]），无角色时返回空切片
func (m *AdminUserMapper) SelectRoleIDsByUserID(userID int64) ([]int64, error) {
	// 查询 sys_user_role 关联表，对齐 Java SysUserRoleMapper
	query := `SELECT role_id FROM sys_user_role WHERE user_id = ?`
	rows, err := m.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户角色失败: %w", err)
	}
	defer rows.Close()

	var roleIds []int64
	for rows.Next() {
		var roleId int64
		if err := rows.Scan(&roleId); err != nil {
			return nil, fmt.Errorf("扫描角色ID失败: %w", err)
		}
		roleIds = append(roleIds, roleId)
	}

	// 确保返回非 nil 的空切片（而非 nil），以便 JSON 序列化为 [] 而非 null
	if roleIds == nil {
		roleIds = []int64{}
	}
	return roleIds, nil
}

// Insert 新增系统用户
//
// 对齐 Java SysUserMapper.insert
// 密码字段应为 BCrypt 哈希后的值（Go 端使用 BCrypt，Java 端使用 SM3+salt）
//
// 参数：
//   - u: 用户实体（Password 字段为 BCrypt 哈希值）
//
// 返回：新用户ID
func (m *AdminUserMapper) Insert(u *AdminUser) (int64, error) {
	query := `INSERT INTO sys_user (username, password, nickname, avatar, email, phone, status, remark, is_deleted, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, NOW(), NOW())`
	result, err := m.db.Exec(query, u.Username, u.Password, u.Nickname, u.Avatar, u.Email, u.Phone, u.Status, u.Remark)
	if err != nil {
		return 0, fmt.Errorf("新增系统用户失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取用户ID失败: %w", err)
	}
	return id, nil
}

// Update 更新系统用户（不更新密码字段）
//
// 对齐 Java SysUserMapper.updateById
// 仅更新非空字段（对齐 Java updateUser 中按字段非空判断的动态更新）
//
// 参数：
//   - u: 用户实体（ID 必填，其余字段非空时更新）
func (m *AdminUserMapper) Update(u *AdminUser) error {
	query := `UPDATE sys_user SET nickname = ?, phone = ?, email = ?, status = ?, remark = ?, update_time = NOW() WHERE id = ? AND is_deleted = 0`
	_, err := m.db.Exec(query, u.Nickname, u.Phone, u.Email, u.Status, u.Remark, u.ID)
	if err != nil {
		return fmt.Errorf("更新系统用户失败: %w", err)
	}
	return nil
}

// Delete 逻辑删除系统用户
//
// 对齐 Java SysUserMapper.deleteById（@TableLogic 逻辑删除，设置 is_deleted=1）
//
// 参数：
//   - id: 用户ID
func (m *AdminUserMapper) Delete(id int64) error {
	query := `UPDATE sys_user SET is_deleted = 1, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除系统用户失败: %w", err)
	}
	return nil
}

// UpdatePassword 更新密码（对齐 Java SysUserMapper.updateById 中密码字段更新）
//
// 参数：
//   - id: 用户ID
//   - hashedPassword: BCrypt 哈希后的密码
func (m *AdminUserMapper) UpdatePassword(id int64, hashedPassword string) error {
	query := `UPDATE sys_user SET password = ?, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, hashedPassword, id)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return nil
}

// ResetPassword 重置用户密码（对齐 Java SysUserServiceImpl.resetPassword）
//
// 等价于 UpdatePassword，语义化别名
//
// 参数：
//   - id: 用户ID
//   - hashedPassword: BCrypt 哈希后的新密码
func (m *AdminUserMapper) ResetPassword(id int64, hashedPassword string) error {
	return m.UpdatePassword(id, hashedPassword)
}

// CountByUsername 按用户名统计用户数（用于新增时唯一性校验）
//
// 对齐 Java SysUserServiceImpl.insertUser 中的用户名唯一性检查
// 只统计未逻辑删除的记录
//
// 参数：
//   - username: 用户名
//
// 返回：匹配的用户数（0=不存在，>0=已存在）
func (m *AdminUserMapper) CountByUsername(username string) (int64, error) {
	query := `SELECT COUNT(1) FROM sys_user WHERE username = ? AND is_deleted = 0`
	var count int64
	err := m.db.QueryRow(query, username).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计用户名数量失败: %w", err)
	}
	return count, nil
}

// DeleteUserRolesByUserID 删除用户的所有角色关联
//
// 对齐 Java SysUserRoleMapper.delete(LambdaQueryWrapper eq userId)
// 用于更新用户角色前先清除旧关联
//
// 参数：
//   - userID: 用户ID
func (m *AdminUserMapper) DeleteUserRolesByUserID(userID int64) error {
	query := `DELETE FROM sys_user_role WHERE user_id = ?`
	_, err := m.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("删除用户角色关联失败: %w", err)
	}
	return nil
}

// InsertUserRole 新增用户-角色关联
//
// 对齐 Java SysUserRoleMapper.insert
//
// 参数：
//   - userID: 用户ID
//   - roleID: 角色ID
func (m *AdminUserMapper) InsertUserRole(userID, roleID int64) error {
	query := `INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)`
	_, err := m.db.Exec(query, userID, roleID)
	if err != nil {
		return fmt.Errorf("新增用户角色关联失败: %w", err)
	}
	return nil
}

// SelectRolesByUserID 查询用户的角色实体列表
//
// 对齐 Java SysUserServiceImpl.getUserRoles
// JOIN sys_user_role 和 sys_role 表查询用户关联的角色信息
// 只查询未逻辑删除的角色（is_deleted=0）
//
// 参数：
//   - userID: 用户ID
//
// 返回：角色实体列表
func (m *AdminUserMapper) SelectRolesByUserID(userID int64) ([]*SysRole, error) {
	query := `SELECT r.id, r.role_name, r.role_key, r.sort, r.status, r.is_deleted, r.create_time, r.update_time, r.remark
	          FROM sys_role r
	          INNER JOIN sys_user_role ur ON r.id = ur.role_id
	          WHERE ur.user_id = ? AND r.is_deleted = 0
	          ORDER BY r.sort ASC`
	rows, err := m.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户角色实体失败: %w", err)
	}
	defer rows.Close()

	var list []*SysRole
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, role)
	}
	return list, nil
}
