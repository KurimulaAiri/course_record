// Package mapper 管理员表操作（对齐 Java AdminMapper/AdminService）
package mapper

import (
	"database/sql"
	"fmt"
)

// ============================================================
// AdminEntity 管理员实体（表 c_admin，对齐 Java Admin.java）
// ============================================================

// AdminEntity 管理员实体
//
// 注意：这是 auth-service 本地定义的实体，因为 common/entity 中没有 Admin 实体
// 字段对应 c_admin 表：id, user_id, is_available, username, create_time, update_time
type AdminEntity struct {
	ID          sql.NullInt64  `json:"adminId"`     // 主键（c_admin.id）
	UserID      sql.NullInt64  `json:"userId"`      // 关联 c_user.id
	IsAvailable sql.NullBool   `json:"isAvailable"` // 是否可用
	Username    sql.NullString `json:"username"`    // 用户名
	CreateTime  sql.NullTime   `json:"createTime"`  // 创建时间
	UpdateTime  sql.NullTime   `json:"updateTime"`  // 更新时间
}

// ============================================================
// AdminMapper 管理员表操作（对齐 Java AdminMapper/AdminService）
// ============================================================

// AdminMapper 管理员表 c_admin 的 Mapper
//
// 对齐 Java adminService.getOne(eq("user_id", userId))
// 用途：登录后查询管理员信息，构造 UserVO.Admin
type AdminMapper struct {
	db *sql.DB
}

// NewAdminMapper 创建 AdminMapper
//
// 参数：
//   - db: 数据库连接
func NewAdminMapper(db *sql.DB) *AdminMapper {
	return &AdminMapper{db: db}
}

// SelectByUserID 按用户ID查管理员
//
// 对齐 Java adminService.getOne(eq("user_id", userId))
// 查询 c_admin 表 WHERE user_id = ?
// 如果用户不是管理员，返回 nil, nil（不报错，属于正常情况）
//
// 参数：
//   - userID: 用户ID（c_user.id）
//
// 返回：
//   - *AdminEntity: 管理员实体，未找到返回 nil
//   - error: 查询错误
func (m *AdminMapper) SelectByUserID(userID int64) (*AdminEntity, error) {
	query := `
		SELECT id, user_id, is_available, username, create_time, update_time
		FROM c_admin
		WHERE user_id = ?
		LIMIT 1
	`
	row := m.db.QueryRow(query, userID)

	admin := &AdminEntity{}
	err := row.Scan(
		&admin.ID,
		&admin.UserID,
		&admin.IsAvailable,
		&admin.Username,
		&admin.CreateTime,
		&admin.UpdateTime,
	)
	if err != nil {
		// 用户不是管理员，正常情况，返回 nil 不报错
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询管理员失败: %w", err)
	}
	return admin, nil
}
