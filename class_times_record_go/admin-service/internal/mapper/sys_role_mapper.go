// Package mapper 系统角色表操作（对齐 Java SysRoleMapper / SysRoleMenuMapper）
//
// 包含：
//   - SysRole 实体定义（对齐 Java SysRole.java，匹配 sys_role 表实际字段）
//   - SysRoleMapper：角色 CRUD
//   - SysRoleMenuMapper：角色-菜单关联表操作（查询/保存）
//
// 注意：实体字段与 common/entity/entity.go 中的 SysRole 不同
//   - entity.go 的 SysRole 字段（RoleCode/Description）与实际 DB schema（role_key/remark）不匹配
//   - 此处定义的 SysRole 与 DB schema 完全对齐，与 Java SysRole.java 一致
package mapper

import (
	"database/sql"
	"fmt"
)

// ============================================================
// SysRole 系统角色实体（对齐 Java SysRole.java，匹配 sys_role 表）
// ============================================================

// SysRole 系统角色实体
//
// 对齐 Java com.shiroko.repository.entity.SysRole
// 表 sys_role 字段：id, role_name, role_key, sort, status, is_deleted, create_time, update_time, remark
//
// 注意：使用普通类型而非 sql.NullXxx，便于 JSON 序列化和业务处理
type SysRole struct {
	ID         int64        `json:"id"`         // 主键
	RoleName   string       `json:"roleName"`   // 角色名称（如：教务主管）
	RoleKey    string       `json:"roleKey"`    // 角色权限字符串（如：academic_admin，唯一）
	Sort       int64        `json:"sort"`       // 显示顺序
	Status     int64        `json:"status"`     // 状态（0=停用, 1=正常）
	IsDeleted  int64        `json:"isDeleted"`  // 逻辑删除（0=存在, 1=删除）
	CreateTime sql.NullTime `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime `json:"updateTime"` // 更新时间
	Remark     string       `json:"remark"`     // 备注
}

// ============================================================
// SysRoleMapper 系统角色表操作
// ============================================================

// SysRoleMapper 系统角色表 sys_role 的 Mapper
//
// 对齐 Java SysRoleMapper（MyBatis-Plus BaseMapper 自动生成 CRUD）
type SysRoleMapper struct {
	db *sql.DB
}

// NewSysRoleMapper 创建 SysRoleMapper
//
// 参数：
//   - db: 数据库连接
func NewSysRoleMapper(db *sql.DB) *SysRoleMapper {
	return &SysRoleMapper{db: db}
}

// scanRole 通用角色扫描函数（复用 sql.Row 和 sql.Rows）
//
// 使用 scanner 抽象 sql.Row 和 sql.Rows 的 Scan 方法
// 避免在 QueryRow 和 Query 两个路径重复扫描代码
//
// 注意：直接返回原始 error（不包装），便于调用方判断 sql.ErrNoRows
//
// 参数：
//   - scanner: 实现 Scan(dest ...interface{}) error 的对象（*sql.Row 或 *sql.Rows）
//
// 返回：角色实体指针
func scanRole(scanner interface {
	Scan(dest ...interface{}) error
}) (*SysRole, error) {
	r := &SysRole{}
	var roleName, roleKey, remark sql.NullString
	var sort, status, isDeleted sql.NullInt64
	err := scanner.Scan(
		&r.ID,
		&roleName,
		&roleKey,
		&sort,
		&status,
		&isDeleted,
		&r.CreateTime,
		&r.UpdateTime,
		&remark,
	)
	if err != nil {
		// 直接返回原始 error（可能是 sql.ErrNoRows），由调用方判断
		return nil, err
	}
	r.RoleName = roleName.String
	r.RoleKey = roleKey.String
	r.Remark = remark.String
	if sort.Valid {
		r.Sort = sort.Int64
	}
	if status.Valid {
		r.Status = status.Int64
	}
	if isDeleted.Valid {
		r.IsDeleted = isDeleted.Int64
	}
	return r, nil
}

// SelectByID 按主键查角色
//
// 对齐 Java SysRoleMapper.selectById
// 只查询未逻辑删除的记录（is_deleted=0）
//
// 参数：
//   - id: 角色ID
//
// 返回：角色实体指针（不存在返回 nil）
func (m *SysRoleMapper) SelectByID(id int64) (*SysRole, error) {
	query := `SELECT id, role_name, role_key, sort, status, is_deleted, create_time, update_time, remark
	          FROM sys_role WHERE id = ? AND is_deleted = 0`
	row := m.db.QueryRow(query, id)
	role, err := scanRole(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return role, nil
}

// SelectList 按条件查询角色列表（分页）
//
// 对齐 Java SysRoleServiceImpl.listRoles 的查询逻辑
// 筛选条件：
//   - roleName: 角色名称模糊查询（LIKE '%roleName%'），空字符串不筛选
//   - roleKey: 角色标识模糊查询（LIKE '%roleKey%'），空字符串不筛选
//   - status: 状态筛选（0=不筛选,1=正常,2=停用）
//   - offset/limit: 分页参数
//
// 排序：按 sort 升序（对齐 Java orderByAsc(SysRole::getSort)）
// 只查询未逻辑删除的记录
//
// 参数：
//   - roleName: 角色名称筛选
//   - roleKey: 角色标识筛选
//   - status: 状态筛选
//   - offset: 分页偏移量
//   - limit: 每页条数
//
// 返回：角色列表
func (m *SysRoleMapper) SelectList(roleName, roleKey string, status int64, offset, limit int) ([]*SysRole, error) {
	// 动态构造 WHERE 条件
	query := `SELECT id, role_name, role_key, sort, status, is_deleted, create_time, update_time, remark
	          FROM sys_role WHERE is_deleted = 0`
	args := []interface{}{}
	if roleName != "" {
		query += ` AND role_name LIKE ?`
		args = append(args, "%"+roleName+"%")
	}
	if roleKey != "" {
		query += ` AND role_key LIKE ?`
		args = append(args, "%"+roleKey+"%")
	}
	if status > 0 {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY sort ASC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
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

// CountWithFilter 按条件统计角色数
//
// 筛选条件与 SelectList 保持一致
// 只统计未逻辑删除的记录
func (m *SysRoleMapper) CountWithFilter(roleName, roleKey string, status int64) (int64, error) {
	query := `SELECT COUNT(1) FROM sys_role WHERE is_deleted = 0`
	args := []interface{}{}
	if roleName != "" {
		query += ` AND role_name LIKE ?`
		args = append(args, "%"+roleName+"%")
	}
	if roleKey != "" {
		query += ` AND role_key LIKE ?`
		args = append(args, "%"+roleKey+"%")
	}
	if status > 0 {
		query += ` AND status = ?`
		args = append(args, status)
	}

	var count int64
	err := m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计角色数失败: %w", err)
	}
	return count, nil
}

// CountByKey 按角色标识统计角色数（用于唯一性校验）
//
// 对齐 Java SysRoleServiceImpl.insertRole/updateRole 中的 roleKey 唯一性检查
// 只统计未逻辑删除的记录
//
// 参数：
//   - roleKey: 角色标识
//   - excludeID: 排除的ID（更新时排除自身，新增时传 0）
//
// 返回：匹配的角色数
func (m *SysRoleMapper) CountByKey(roleKey string, excludeID int64) (int64, error) {
	query := `SELECT COUNT(1) FROM sys_role WHERE role_key = ? AND is_deleted = 0 AND id != ?`
	var count int64
	err := m.db.QueryRow(query, roleKey, excludeID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计角色标识数量失败: %w", err)
	}
	return count, nil
}

// Insert 新增角色
//
// 对齐 Java SysRoleMapper.insert
//
// 参数：
//   - r: 角色实体
//
// 返回：新角色ID
func (m *SysRoleMapper) Insert(r *SysRole) (int64, error) {
	query := `INSERT INTO sys_role (role_name, role_key, sort, status, is_deleted, create_time, update_time, remark)
	          VALUES (?, ?, ?, ?, 0, NOW(), NOW(), ?)`
	result, err := m.db.Exec(query, r.RoleName, r.RoleKey, r.Sort, r.Status, r.Remark)
	if err != nil {
		return 0, fmt.Errorf("新增角色失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取角色ID失败: %w", err)
	}
	return id, nil
}

// Update 更新角色
//
// 对齐 Java SysRoleMapper.updateById
// 全字段更新（除 id、is_deleted、create_time）
//
// 参数：
//   - r: 角色实体（ID 必填）
func (m *SysRoleMapper) Update(r *SysRole) error {
	query := `UPDATE sys_role SET role_name = ?, role_key = ?, sort = ?, status = ?, remark = ?, update_time = NOW()
	          WHERE id = ? AND is_deleted = 0`
	_, err := m.db.Exec(query, r.RoleName, r.RoleKey, r.Sort, r.Status, r.Remark, r.ID)
	if err != nil {
		return fmt.Errorf("更新角色失败: %w", err)
	}
	return nil
}

// Delete 逻辑删除角色
//
// 对齐 Java SysRoleMapper.deleteById（@TableLogic 逻辑删除）
//
// 参数：
//   - id: 角色ID
func (m *SysRoleMapper) Delete(id int64) error {
	query := `UPDATE sys_role SET is_deleted = 1, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}
	return nil
}

// ============================================================
// SysRoleMenuMapper 角色-菜单关联表操作
// ============================================================

// SysRoleMenuMapper 角色-菜单关联表 sys_role_menu 的 Mapper
//
// 对齐 Java SysRoleMenuMapper
// 用于角色菜单授权（save_menus）和查询角色已分配菜单（get_menus）
type SysRoleMenuMapper struct {
	db *sql.DB
}

// NewSysRoleMenuMapper 创建 SysRoleMenuMapper
func NewSysRoleMenuMapper(db *sql.DB) *SysRoleMenuMapper {
	return &SysRoleMenuMapper{db: db}
}

// SelectMenuIDsByRoleID 查询角色已分配的菜单ID列表
//
// 对齐 Java SysRoleMenuMapper.selectList(eq roleId)
//
// 参数：
//   - roleID: 角色ID
//
// 返回：菜单ID列表（无关联时返回空切片）
func (m *SysRoleMenuMapper) SelectMenuIDsByRoleID(roleID int64) ([]int64, error) {
	query := `SELECT menu_id FROM sys_role_menu WHERE role_id = ?`
	rows, err := m.db.Query(query, roleID)
	if err != nil {
		return nil, fmt.Errorf("查询角色菜单ID失败: %w", err)
	}
	defer rows.Close()

	var menuIds []int64
	for rows.Next() {
		var menuId int64
		if err := rows.Scan(&menuId); err != nil {
			return nil, fmt.Errorf("扫描菜单ID失败: %w", err)
		}
		menuIds = append(menuIds, menuId)
	}
	if menuIds == nil {
		menuIds = []int64{}
	}
	return menuIds, nil
}

// DeleteByRoleID 删除角色的所有菜单关联
//
// 对齐 Java SysRoleMenuMapper.delete(eq roleId)
// 用于 save_menus 先删旧关联再插新关联
//
// 参数：
//   - roleID: 角色ID
func (m *SysRoleMenuMapper) DeleteByRoleID(roleID int64) error {
	query := `DELETE FROM sys_role_menu WHERE role_id = ?`
	_, err := m.db.Exec(query, roleID)
	if err != nil {
		return fmt.Errorf("删除角色菜单关联失败: %w", err)
	}
	return nil
}

// InsertBatch 批量插入角色-菜单关联
//
// 对齐 Java SysRoleMenuMapper.insert（循环插入）
//
// 参数：
//   - roleID: 角色ID
//   - menuIDs: 菜单ID列表
func (m *SysRoleMenuMapper) InsertBatch(roleID int64, menuIDs []int64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	// 使用事务批量插入，避免部分失败
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("预处理插入语句失败: %w", err)
	}
	defer stmt.Close()

	for _, menuID := range menuIDs {
		if _, err := stmt.Exec(roleID, menuID); err != nil {
			tx.Rollback()
			return fmt.Errorf("插入角色菜单关联失败: roleID=%d, menuID=%d, err=%w", roleID, menuID, err)
		}
	}
	return tx.Commit()
}

// DeleteByMenuID 删除菜单的所有角色关联
//
// 对齐 Java SysRoleMenuMapper.delete(eq menuId)
// 用于删除菜单时清理关联
//
// 参数：
//   - menuID: 菜单ID
func (m *SysRoleMenuMapper) DeleteByMenuID(menuID int64) error {
	query := `DELETE FROM sys_role_menu WHERE menu_id = ?`
	_, err := m.db.Exec(query, menuID)
	if err != nil {
		return fmt.Errorf("删除菜单角色关联失败: %w", err)
	}
	return nil
}
