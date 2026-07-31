// Package mapper 系统菜单表操作（对齐 Java SysMenuMapper）
//
// 包含：
//   - SysMenu 实体定义（对齐 Java SysMenu.java，匹配 sys_menu 表实际字段）
//   - SysMenuMapper：菜单 CRUD + 树形结构查询辅助
//
// 注意：实体字段与 common/entity/entity.go 中的 SysMenu 不同
//   - entity.go 的 SysMenu 字段（SortOrder/IsVisible/Permission）与实际 DB schema（sort/status/perms）不匹配
//   - 此处定义的 SysMenu 与 DB schema 完全对齐，与 Java SysMenu.java 一致
package mapper

import (
	"database/sql"
	"fmt"
)

// ============================================================
// SysMenu 系统菜单实体（对齐 Java SysMenu.java，匹配 sys_menu 表）
// ============================================================

// SysMenu 系统菜单实体
//
// 对齐 Java com.shiroko.repository.entity.SysMenu
// 表 sys_menu 字段：id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, status, create_time, update_time
//
// menuType 说明（对齐 DB 注释）：
//   - M: 目录（Directory）
//   - C: 菜单（Menu）
//   - F: 按钮/权限点（Button）
//
// status 说明：0=隐藏, 1=显示
type SysMenu struct {
	ID         int64        `json:"id"`         // 菜单ID
	ParentID   int64        `json:"parentId"`   // 父菜单ID（0=顶级菜单）
	MenuName   string       `json:"menuName"`   // 菜单名称
	MenuType   string       `json:"menuType"`   // 菜单类型（M=目录, C=菜单, F=按钮）
	Path       string       `json:"path"`       // 路由地址
	Component  string       `json:"component"`  // 组件路径
	Perms      string       `json:"perms"`      // 权限标识（如 schedule:adjust）
	Icon       string       `json:"icon"`       // 菜单图标
	Sort       int64        `json:"sort"`       // 显示顺序（越小越靠前）
	Status     int64        `json:"status"`     // 状态（0=隐藏, 1=显示）
	CreateTime sql.NullTime `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime `json:"updateTime"` // 更新时间
}

// ============================================================
// SysMenuMapper 系统菜单表操作
// ============================================================

// SysMenuMapper 系统菜单表 sys_menu 的 Mapper
//
// 对齐 Java SysMenuMapper（MyBatis-Plus BaseMapper 自动生成 CRUD）
type SysMenuMapper struct {
	db *sql.DB
}

// NewSysMenuMapper 创建 SysMenuMapper
func NewSysMenuMapper(db *sql.DB) *SysMenuMapper {
	return &SysMenuMapper{db: db}
}

// scanMenu 通用菜单扫描函数（复用 sql.Row 和 sql.Rows）
//
// 直接返回原始 error（不包装），便于调用方判断 sql.ErrNoRows
func scanMenu(scanner interface {
	Scan(dest ...interface{}) error
}) (*SysMenu, error) {
	m := &SysMenu{}
	var menuName, menuType, path, component, perms, icon sql.NullString
	var parentID, sort, status sql.NullInt64
	err := scanner.Scan(
		&m.ID,
		&parentID,
		&menuName,
		&menuType,
		&path,
		&component,
		&perms,
		&icon,
		&sort,
		&status,
		&m.CreateTime,
		&m.UpdateTime,
	)
	if err != nil {
		return nil, err
	}
	m.MenuName = menuName.String
	m.MenuType = menuType.String
	m.Path = path.String
	m.Component = component.String
	m.Perms = perms.String
	m.Icon = icon.String
	if parentID.Valid {
		m.ParentID = parentID.Int64
	}
	if sort.Valid {
		m.Sort = sort.Int64
	}
	if status.Valid {
		m.Status = status.Int64
	}
	return m, nil
}

// SelectByID 按主键查菜单
//
// 对齐 Java SysMenuMapper.selectById
//
// 参数：
//   - id: 菜单ID
//
// 返回：菜单实体指针（不存在返回 nil）
func (m *SysMenuMapper) SelectByID(id int64) (*SysMenu, error) {
	query := `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, status, create_time, update_time
	          FROM sys_menu WHERE id = ?`
	row := m.db.QueryRow(query, id)
	menu, err := scanMenu(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}
	return menu, nil
}

// SelectList 按条件查询菜单列表
//
// 对齐 Java SysMenuServiceImpl.listMenus 的查询逻辑
// 筛选条件：
//   - menuName: 菜单名称模糊查询（LIKE '%menuName%'），空字符串不筛选
//   - menuType: 菜单类型精确匹配，空字符串不筛选
//   - status: 状态筛选（0=不筛选,1=显示,2=隐藏）
//
// 排序：按 sort 升序（对齐 Java orderByAsc(SysMenu::getSort)）
//
// 参数：
//   - menuName: 菜单名称筛选
//   - menuType: 菜单类型筛选
//   - status: 状态筛选
//
// 返回：菜单列表（全量，不分页，菜单数据量小）
func (m *SysMenuMapper) SelectList(menuName, menuType string, status int64) ([]*SysMenu, error) {
	query := `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, status, create_time, update_time
	          FROM sys_menu WHERE 1=1`
	args := []interface{}{}
	if menuName != "" {
		query += ` AND menu_name LIKE ?`
		args = append(args, "%"+menuName+"%")
	}
	if menuType != "" {
		query += ` AND menu_type = ?`
		args = append(args, menuType)
	}
	if status > 0 {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY sort ASC`

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询菜单列表失败: %w", err)
	}
	defer rows.Close()

	var list []*SysMenu
	for rows.Next() {
		menu, err := scanMenu(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描菜单记录失败: %w", err)
		}
		list = append(list, menu)
	}
	return list, nil
}

// SelectAll 查询所有菜单（按 sort 升序）
//
// 对齐 Java SysMenuServiceImpl.getMenuTree 中的 sysMenuMapper.selectList(null)
// 用于构建完整菜单树
func (m *SysMenuMapper) SelectAll() ([]*SysMenu, error) {
	query := `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, status, create_time, update_time
	          FROM sys_menu ORDER BY sort ASC`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询所有菜单失败: %w", err)
	}
	defer rows.Close()

	var list []*SysMenu
	for rows.Next() {
		menu, err := scanMenu(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描菜单记录失败: %w", err)
		}
		list = append(list, menu)
	}
	return list, nil
}

// SelectByIDs 按ID列表批量查询菜单
//
// 对齐 Java SysMenuMapper.selectBatchIds
// 用于根据角色关联的 menuIds 查询菜单实体
//
// 参数：
//   - menuIDs: 菜单ID列表
//
// 返回：菜单列表
func (m *SysMenuMapper) SelectByIDs(menuIDs []int64) ([]*SysMenu, error) {
	if len(menuIDs) == 0 {
		return []*SysMenu{}, nil
	}
	// 动态构造 IN 占位符
	placeholders := ""
	args := make([]interface{}, len(menuIDs))
	for i, id := range menuIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}
	query := `SELECT id, parent_id, menu_name, menu_type, path, component, perms, icon, sort, status, create_time, update_time
	          FROM sys_menu WHERE id IN (` + placeholders + `) ORDER BY sort ASC`
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("按ID批量查询菜单失败: %w", err)
	}
	defer rows.Close()

	var list []*SysMenu
	for rows.Next() {
		menu, err := scanMenu(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描菜单记录失败: %w", err)
		}
		list = append(list, menu)
	}
	return list, nil
}

// SelectByRoleIDs 按角色ID列表查询关联的菜单（去重）
//
// 对齐 Java SysMenuServiceImpl.getUserMenuTree 中的查询逻辑：
//  1. JOIN sys_role_menu 按角色ID列表查询 menuId
//  2. 按 status=1（显示）和 menuType IN ('M','C','F') 过滤
//  3. 按 sort 升序排序
//
// 参数：
//   - roleIDs: 角色ID列表
//
// 返回：菜单列表（已去重，仅显示状态）
func (m *SysMenuMapper) SelectByRoleIDs(roleIDs []int64) ([]*SysMenu, error) {
	if len(roleIDs) == 0 {
		return []*SysMenu{}, nil
	}
	// 动态构造 IN 占位符
	placeholders := ""
	args := make([]interface{}, len(roleIDs))
	for i, id := range roleIDs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = id
	}
	// JOIN sys_role_menu 按角色ID查询关联菜单，DISTINCT 去重（多角色可能共享同一菜单）
	query := `SELECT DISTINCT m.id, m.parent_id, m.menu_name, m.menu_type, m.path, m.component, m.perms, m.icon, m.sort, m.status, m.create_time, m.update_time
	          FROM sys_menu m
	          INNER JOIN sys_role_menu rm ON m.id = rm.menu_id
	          WHERE rm.role_id IN (` + placeholders + `)
	          AND m.status = 1
	          AND m.menu_type IN ('M', 'C', 'F')
	          ORDER BY m.sort ASC`
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("按角色查询菜单失败: %w", err)
	}
	defer rows.Close()

	var list []*SysMenu
	for rows.Next() {
		menu, err := scanMenu(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描菜单记录失败: %w", err)
		}
		list = append(list, menu)
	}
	return list, nil
}

// CountByParentID 统计子菜单数量
//
// 对齐 Java SysMenuServiceImpl.deleteMenu 中的子菜单检查
// 用于删除菜单前校验是否存在子菜单
//
// 参数：
//   - parentID: 父菜单ID
//
// 返回：子菜单数量
func (m *SysMenuMapper) CountByParentID(parentID int64) (int64, error) {
	query := `SELECT COUNT(1) FROM sys_menu WHERE parent_id = ?`
	var count int64
	err := m.db.QueryRow(query, parentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计子菜单数量失败: %w", err)
	}
	return count, nil
}

// Insert 新增菜单
//
// 对齐 Java SysMenuMapper.insert
//
// 参数：
//   - menu: 菜单实体
//
// 返回：新菜单ID
func (m *SysMenuMapper) Insert(menu *SysMenu) (int64, error) {
	query := `INSERT INTO sys_menu (parent_id, menu_name, menu_type, path, component, perms, icon, sort, status, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, menu.ParentID, menu.MenuName, menu.MenuType, menu.Path, menu.Component, menu.Perms, menu.Icon, menu.Sort, menu.Status)
	if err != nil {
		return 0, fmt.Errorf("新增菜单失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取菜单ID失败: %w", err)
	}
	return id, nil
}

// Update 更新菜单
//
// 对齐 Java SysMenuMapper.updateById
// 全字段更新（除 id、create_time、update_time）
//
// 参数：
//   - menu: 菜单实体（ID 必填）
func (m *SysMenuMapper) Update(menu *SysMenu) error {
	query := `UPDATE sys_menu SET parent_id = ?, menu_name = ?, menu_type = ?, path = ?, component = ?, perms = ?, icon = ?, sort = ?, status = ?, update_time = NOW()
	          WHERE id = ?`
	_, err := m.db.Exec(query, menu.ParentID, menu.MenuName, menu.MenuType, menu.Path, menu.Component, menu.Perms, menu.Icon, menu.Sort, menu.Status, menu.ID)
	if err != nil {
		return fmt.Errorf("更新菜单失败: %w", err)
	}
	return nil
}

// Delete 删除菜单（物理删除）
//
// 对齐 Java SysMenuMapper.deleteById
// 注意：调用方需先校验是否存在子菜单，并清理 sys_role_menu 关联
//
// 参数：
//   - id: 菜单ID
func (m *SysMenuMapper) Delete(id int64) error {
	query := `DELETE FROM sys_menu WHERE id = ?`
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除菜单失败: %w", err)
	}
	return nil
}
