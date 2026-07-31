// Package mapper 菜单表操作（对齐 Java MenuMapper）
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
)

// ============================================================
// MenuMapper 菜单表操作（对齐 Java MenuMapper）
// ============================================================

// MenuMapper 菜单表 c_menu 的 Mapper
//
// 对齐 Java MenuMapper.getMenuByRole：LEFT JOIN c_role_menu 按 roleId 查询可见菜单
type MenuMapper struct {
	db *sql.DB
}

// NewMenuMapper 创建 MenuMapper
//
// 参数：
//   - db: 数据库连接
func NewMenuMapper(db *sql.DB) *MenuMapper {
	return &MenuMapper{db: db}
}

// SelectByRole 按角色ID分页查询菜单列表
//
// 对齐 Java MenuMapper.getMenuByRole(IPage, QueryMenuDTO)
// SQL 对齐 MenuMapper.xml：
//
//	SELECT m.id, m.menu_name, m.icon, m.icon_type, m.bg_color,
//	       m.path, m.sort_order, m.is_visible, m.create_time, m.update_time
//	FROM c_menu m
//	LEFT JOIN c_role_menu rm ON m.id = rm.menu_id
//	WHERE rm.permission_id = ?
//
// 参数：
//   - roleID: 角色ID（c_role_menu.permission_id，3=家长, 4=教师）
//   - offset: 分页偏移量（=(currentPage-1)*pageSize）
//   - pageSize: 每页条数
//
// 返回：
//   - []*entity.Menu: 菜单列表（按 sort_order 升序）
//   - int64: 总记录数（用于分页）
//   - error: 查询错误
func (m *MenuMapper) SelectByRole(roleID int64, offset, pageSize int64) ([]*entity.Menu, int64, error) {
	// 1. 查询总记录数（对齐 Java page.getTotal()）
	countQuery := `SELECT COUNT(1) FROM c_menu m LEFT JOIN c_role_menu rm ON m.id = rm.menu_id WHERE rm.permission_id = ?`
	var total int64
	err := m.db.QueryRow(countQuery, roleID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("统计菜单总数失败: %w", err)
	}

	// 2. 分页查询菜单列表（按 sort_order 升序，对齐前端预期排序）
	listQuery := `
		SELECT m.id, m.menu_name, m.icon, m.icon_type, m.bg_color,
		       m.path, m.sort_order, m.is_visible, m.create_time, m.update_time
		FROM c_menu m
		LEFT JOIN c_role_menu rm ON m.id = rm.menu_id
		WHERE rm.permission_id = ?
		ORDER BY m.sort_order ASC
		LIMIT ? OFFSET ?
	`
	rows, err := m.db.Query(listQuery, roleID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("查询菜单列表失败: %w", err)
	}
	defer rows.Close()

	// 3. 扫描结果集
	var list []*entity.Menu
	for rows.Next() {
		menu := &entity.Menu{}
		err := rows.Scan(
			&menu.ID,
			&menu.MenuName,
			&menu.Icon,
			&menu.IconType,
			&menu.BgColor,
			&menu.Path,
			&menu.SortOrder,
			&menu.IsVisible,
			&menu.CreateTime,
			&menu.UpdateTime,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描菜单记录失败: %w", err)
		}
		list = append(list, menu)
	}

	// 确保返回非 nil 空切片（便于 JSON 序列化为 [] 而非 null）
	if list == nil {
		list = []*entity.Menu{}
	}
	return list, total, nil
}
