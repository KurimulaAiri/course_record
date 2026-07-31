// Package mapper 用户平台表操作（对齐 Java UserPlatformMapper）
//
// Java 中无自定义方法，全靠 LambdaQueryWrapper，Go 这里翻译为等价 SQL
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/pkg/errors"
)

// UserPlatformMapper 用户平台表 c_user_platform 的 Mapper
type UserPlatformMapper struct {
	db *sql.DB
}

// NewUserPlatformMapper 创建 UserPlatformMapper
func NewUserPlatformMapper(db *sql.DB) *UserPlatformMapper {
	return &UserPlatformMapper{db: db}
}

// SelectByOpenIdAndPlatform 按 openId+平台查（不限角色）
//
// 用途：微信免密登录时查找平台记录
//
// 参数：
//   - openId: 微信 openId
//   - platform: 平台标识（"WEIXIN"）
//
// 返回：平台记录，未找到返回 nil
func (m *UserPlatformMapper) SelectByOpenIdAndPlatform(openId, platform string) (*entity.UserPlatform, error) {
	query := `
			SELECT id, user_id, open_id, union_id, last_login_time, last_login_role, platform, is_available, create_time
			FROM c_user_platform
			WHERE open_id = ? AND platform = ? AND is_available = 1
			ORDER BY id
			LIMIT 1
		`
	row := m.db.QueryRow(query, openId, platform)

	p := &entity.UserPlatform{}
	err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.OpenID,
		&p.UnionID,
		&p.LastLoginTime,
		&p.LastLoginRole,
		&p.Platform,
		&p.IsAvailable,
		&p.CreateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询平台记录失败: %w", err)
	}
	return p, nil
}

// SelectByOpenIdPlatformAndRole 按 openId+平台+角色查（对齐 Java lastLoginRole=3 过滤）
//
// 用途：recordSubscribe 时按家长角色查找用户
//
// 参数：
//   - openId: 微信 openId
//   - platform: 平台标识
//   - lastLoginRole: 最后登录角色（3=parent）
//
// 返回：平台记录，未找到返回 nil
func (m *UserPlatformMapper) SelectByOpenIdPlatformAndRole(openId, platform string, lastLoginRole int64) (*entity.UserPlatform, error) {
	query := `
			SELECT id, user_id, open_id, union_id, last_login_time, last_login_role, platform, is_available, create_time
			FROM c_user_platform
			WHERE open_id = ? AND platform = ? AND is_available = 1 AND last_login_role = ?
			ORDER BY id
			LIMIT 1
		`
	row := m.db.QueryRow(query, openId, platform, lastLoginRole)

	p := &entity.UserPlatform{}
	err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.OpenID,
		&p.UnionID,
		&p.LastLoginTime,
		&p.LastLoginRole,
		&p.Platform,
		&p.IsAvailable,
		&p.CreateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询平台记录失败: %w", err)
	}
	return p, nil
}

// SelectByUserIDAndPlatform 按用户ID+平台查
func (m *UserPlatformMapper) SelectByUserIDAndPlatform(userID int64, platform string) (*entity.UserPlatform, error) {
	query := `
		SELECT id, user_id, open_id, union_id, last_login_time, last_login_role, platform, is_available, create_time
		FROM c_user_platform
		WHERE user_id = ? AND platform = ? AND is_available = 1
		ORDER BY id DESC
		LIMIT 1
	`
	row := m.db.QueryRow(query, userID, platform)

	p := &entity.UserPlatform{}
	err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.OpenID,
		&p.UnionID,
		&p.LastLoginTime,
		&p.LastLoginRole,
		&p.Platform,
		&p.IsAvailable,
		&p.CreateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询平台记录失败: %w", err)
	}
	return p, nil
}

// Insert 新增平台记录
func (m *UserPlatformMapper) Insert(p *entity.UserPlatform) (int64, error) {
	query := `INSERT INTO c_user_platform (user_id, open_id, union_id, last_login_time, last_login_role, platform, is_available, create_time)
	          VALUES (?, ?, ?, NOW(), ?, ?, 1, NOW())`
	result, err := m.db.Exec(query,
		p.UserID,
		p.OpenID,
		p.UnionID,
		p.LastLoginRole,
		p.Platform,
	)
	if err != nil {
		return 0, fmt.Errorf("新增平台记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取平台记录ID失败: %w", err)
	}
	return id, nil
}

// UpdateLastLogin 更新最后登录时间和角色
//
// 参数：
//   - id: 平台记录ID
//   - lastLoginRole: 最后登录角色
func (m *UserPlatformMapper) UpdateLastLogin(id int64, lastLoginRole int64) error {
	query := `UPDATE c_user_platform SET last_login_time = NOW(), last_login_role = ? WHERE id = ?`
	_, err := m.db.Exec(query, lastLoginRole, id)
	if err != nil {
		return fmt.Errorf("更新登录信息失败: %w", err)
	}
	return nil
}

// UpdateAvailable 更新可用状态
func (m *UserPlatformMapper) UpdateAvailable(id int64, isAvailable bool) error {
	query := `UPDATE c_user_platform SET is_available = ? WHERE id = ?`
	_, err := m.db.Exec(query, isAvailable, id)
	if err != nil {
		return fmt.Errorf("更新可用状态失败: %w", err)
	}
	return nil
}
