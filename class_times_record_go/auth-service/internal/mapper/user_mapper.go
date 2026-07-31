// Package mapper auth-service 数据访问层
//
// 对齐 Java auth-service/src/main/java/com/shiroko/mapper 包
//
// 包含：
//   - UserMapper：用户表 c_user 操作
//   - UserAuthMapper：用户认证表 c_user_auth 操作
//   - UserPlatformMapper：用户平台表 c_user_platform 操作
//   - ParentMapper：家长表 c_parent 操作
//   - InstitutionMapper：机构表 c_institution 操作
//   - StudentMapper：学生表 c_student 操作（绑定流程需要）
//   - ParentStudentMapper：家长-学生关联表 c_parent_student 操作
//   - WxSubscribeMapper：微信订阅相关表操作
//
// 与 Java MyBatis-Plus 的区别：
//   - Java 用 BaseMapper + LambdaQueryWrapper
//   - Go 直接用 database/sql + 手写 SQL
//   - Go 用 sql.NullXxx 处理 NULL 值
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/pkg/errors"
)

// ============================================================
// UserMapper 用户表操作（对齐 Java UserMapper）
// ============================================================

// UserMapper 用户表 c_user 的 Mapper
type UserMapper struct {
	db *sql.DB
}

// NewUserMapper 创建 UserMapper
func NewUserMapper(db *sql.DB) *UserMapper {
	return &UserMapper{db: db}
}

// SelectUserByPlatformOpenid 按平台+openId 查用户（跨机构）
//
// 对齐 Java UserMapper.selectUserByPlatformOpenid
// 用途：微信免密登录时，跨机构查找已绑定该 openId 的用户
//
// 参数：
//   - platform: 平台标识（"WEIXIN"）
//   - openId: 微信 openId
//
// 返回：用户实体指针，未找到返回 nil
func (m *UserMapper) SelectUserByPlatformOpenid(platform, openId string) (*entity.User, error) {
	query := `
			SELECT u.id, u.institution_id, u.create_time, u.update_time
			FROM c_user u
			INNER JOIN c_user_platform up ON u.id = up.user_id
			WHERE up.platform = ? AND up.open_id = ? AND up.is_available = 1
			ORDER BY u.id
			LIMIT 1
		`
	row := m.db.QueryRow(query, platform, openId)

	user := &entity.User{}
	err := row.Scan(
		&user.ID,
		&user.InstitutionID,
		&user.CreateTime,
		&user.UpdateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return user, nil
}

// SelectUserByPlatformOpenidAndInstitution 按平台+openId+机构查用户
//
// 对齐 Java UserMapper.selectUserByPlatformOpenidAndInstitution
// 用途：绑定时，在同一机构内查找已绑定该 openId 的用户（避免跨机构复用）
//
// 参数：
//   - platform: 平台标识
//   - openId: 微信 openId
//   - institutionId: 机构ID
//
// 返回：用户实体指针，未找到返回 nil
func (m *UserMapper) SelectUserByPlatformOpenidAndInstitution(platform, openId string, institutionId int64) (*entity.User, error) {
	query := `
			SELECT u.id, u.institution_id, u.create_time, u.update_time
			FROM c_user u
			INNER JOIN c_user_platform up ON u.id = up.user_id
			WHERE up.platform = ? AND up.open_id = ? AND up.is_available = 1
			  AND u.institution_id = ?
			ORDER BY u.id
			LIMIT 1
		`
	row := m.db.QueryRow(query, platform, openId, institutionId)

	user := &entity.User{}
	err := row.Scan(
		&user.ID,
		&user.InstitutionID,
		&user.CreateTime,
		&user.UpdateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return user, nil
}

// SelectByID 按主键查用户
func (m *UserMapper) SelectByID(id int64) (*entity.User, error) {
	query := `SELECT id, institution_id, create_time, update_time FROM c_user WHERE id = ?`
	row := m.db.QueryRow(query, id)

	user := &entity.User{}
	err := row.Scan(
		&user.ID,
		&user.InstitutionID,
		&user.CreateTime,
		&user.UpdateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return user, nil
}

// Insert 新增用户
//
// 返回：新用户ID
func (m *UserMapper) Insert(user *entity.User) (int64, error) {
	query := `INSERT INTO c_user (institution_id, create_time, update_time) VALUES (?, NOW(), NOW())`
	result, err := m.db.Exec(query, user.InstitutionID)
	if err != nil {
		return 0, fmt.Errorf("新增用户失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取用户ID失败: %w", err)
	}
	return id, nil
}

// UpdateInstitutionID 更新用户机构
func (m *UserMapper) UpdateInstitutionID(userID, institutionID int64) error {
	query := `UPDATE c_user SET institution_id = ?, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, institutionID, userID)
	if err != nil {
		return fmt.Errorf("更新用户机构失败: %w", err)
	}
	return nil
}
