// Package mapper 微信订阅相关表操作（对齐 Java WxSubscribeRecordMapper / WxStudentSubscribeMapper）
package mapper

import (
	"database/sql"
	"fmt"

	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/pkg/errors"
)

// ============================================================
// WxSubscribeRecordMapper 微信订阅记录表操作
// ============================================================

// WxSubscribeRecordMapper 微信订阅记录表 c_wx_subscribe_record 的 Mapper
type WxSubscribeRecordMapper struct {
	db *sql.DB
}

// NewWxSubscribeRecordMapper 创建 WxSubscribeRecordMapper
func NewWxSubscribeRecordMapper(db *sql.DB) *WxSubscribeRecordMapper {
	return &WxSubscribeRecordMapper{db: db}
}

// SelectByOpenIdAndTemplate 按 openId+模板ID查订阅记录
//
// 用途：查询剩余推送次数
func (m *WxSubscribeRecordMapper) SelectByOpenIdAndTemplate(openID, templateID string) (*entity.WxSubscribeRecord, error) {
	query := `SELECT id, open_id, template_id, subscribe_count, is_permanent, create_time, update_time FROM c_wx_subscribe_record WHERE open_id = ? AND template_id = ? LIMIT 1`
	row := m.db.QueryRow(query, openID, templateID)

	r := &entity.WxSubscribeRecord{}
	err := row.Scan(
		&r.ID,
		&r.OpenID,
		&r.TemplateID,
		&r.SubscribeCount,
		&r.IsPermanent,
		&r.CreateTime,
		&r.UpdateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询订阅记录失败: %w", err)
	}
	return r, nil
}

// Insert 新增订阅记录
func (m *WxSubscribeRecordMapper) Insert(r *entity.WxSubscribeRecord) (int64, error) {
	query := `INSERT INTO c_wx_subscribe_record (open_id, template_id, subscribe_count, is_permanent, create_time, update_time)
	          VALUES (?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query,
		r.OpenID,
		r.TemplateID,
		r.SubscribeCount,
		r.IsPermanent,
	)
	if err != nil {
		return 0, fmt.Errorf("新增订阅记录失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取订阅记录ID失败: %w", err)
	}
	return id, nil
}

// IncrementCount 订阅次数+1（对齐 Java count+1 逻辑）
//
// 参数：
//   - id: 订阅记录ID
//   - isPermanent: 是否永久订阅
func (m *WxSubscribeRecordMapper) IncrementCount(id int64, isPermanent bool) error {
	if isPermanent {
		// 永久订阅只标记，不加次数
		query := `UPDATE c_wx_subscribe_record SET is_permanent = 1, update_time = NOW() WHERE id = ?`
		_, err := m.db.Exec(query, id)
		if err != nil {
			return fmt.Errorf("更新永久订阅标记失败: %w", err)
		}
		return nil
	}

	query := `UPDATE c_wx_subscribe_record SET subscribe_count = subscribe_count + 1, update_time = NOW() WHERE id = ?`
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("订阅次数+1失败: %w", err)
	}
	return nil
}

// DecrementCount 推送后订阅次数-1（对齐 Java 推送成功 count-1）
//
// 永久订阅不扣减
func (m *WxSubscribeRecordMapper) DecrementCount(id int64) error {
	query := `UPDATE c_wx_subscribe_record SET subscribe_count = subscribe_count - 1, update_time = NOW()
	          WHERE id = ? AND is_permanent = 0 AND subscribe_count > 0`
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("订阅次数-1失败: %w", err)
	}
	return nil
}

// ============================================================
// WxStudentSubscribeMapper 学生订阅关系表操作
// ============================================================

// WxStudentSubscribeMapper 学生订阅关系表 c_wx_student_subscribe 的 Mapper
type WxStudentSubscribeMapper struct {
	db *sql.DB
}

// NewWxStudentSubscribeMapper 创建 WxStudentSubscribeMapper
func NewWxStudentSubscribeMapper(db *sql.DB) *WxStudentSubscribeMapper {
	return &WxStudentSubscribeMapper{db: db}
}

// SelectByOpenIdAndStudent 按 openId+学生ID查订阅关系
func (m *WxStudentSubscribeMapper) SelectByOpenIdAndStudent(openID string, studentID int64) (*entity.WxStudentSubscribe, error) {
	query := `SELECT id, open_id, student_id, is_primary, bind_mode, create_time, update_time FROM c_wx_student_subscribe WHERE open_id = ? AND student_id = ? LIMIT 1`
	row := m.db.QueryRow(query, openID, studentID)

	s := &entity.WxStudentSubscribe{}
	err := row.Scan(
		&s.ID,
		&s.OpenID,
		&s.StudentID,
		&s.IsPrimary,
		&s.BindMode,
		&s.CreateTime,
		&s.UpdateTime,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询学生订阅关系失败: %w", err)
	}
	return s, nil
}

// Insert 新增学生订阅关系
func (m *WxStudentSubscribeMapper) Insert(s *entity.WxStudentSubscribe) (int64, error) {
	query := `INSERT INTO c_wx_student_subscribe (open_id, student_id, is_primary, bind_mode, create_time, update_time)
	          VALUES (?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query,
		s.OpenID,
		s.StudentID,
		s.IsPrimary,
		s.BindMode,
	)
	if err != nil {
		return 0, fmt.Errorf("新增学生订阅关系失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取订阅关系ID失败: %w", err)
	}
	return id, nil
}

// DeleteByOpenIdAndStudent 删除订阅关系（取消订阅时用）
func (m *WxStudentSubscribeMapper) DeleteByOpenIdAndStudent(openID string, studentID int64) error {
	query := `DELETE FROM c_wx_student_subscribe WHERE open_id = ? AND student_id = ?`
	_, err := m.db.Exec(query, openID, studentID)
	if err != nil {
		return fmt.Errorf("删除学生订阅关系失败: %w", err)
	}
	return nil
}

// UpdateByOpenIdAndStudent 按 openId+学生ID 更新订阅关系（isPrimary 和 bindMode）
//
// 用途：绑定时若已有订阅记录，更新 isPrimary 和 bindMode（可能从仅订阅升级为绑定账号）
//
// 参数：
//   - openID: 微信 openId
//   - studentID: 学生ID
//   - isPrimary: 是否主要联系人
//   - bindMode: 绑定模式（"subscribe"=仅订阅, "full"=绑定账号并订阅）
func (m *WxStudentSubscribeMapper) UpdateByOpenIdAndStudent(openID string, studentID int64, isPrimary bool, bindMode string) error {
	query := `UPDATE c_wx_student_subscribe SET is_primary = ?, bind_mode = ?, update_time = NOW() WHERE open_id = ? AND student_id = ?`
	_, err := m.db.Exec(query, isPrimary, bindMode, openID, studentID)
	if err != nil {
		return fmt.Errorf("更新学生订阅关系失败: %w", err)
	}
	return nil
}

// CountByOpenIdAndStudent 按 openId+学生ID 统计订阅数
//
// 用途：getSubscribeStatus 判断 wechatSubscribed
func (m *WxStudentSubscribeMapper) CountByOpenIdAndStudent(openID string, studentID int64) (int64, error) {
	query := `SELECT COUNT(1) FROM c_wx_student_subscribe WHERE open_id = ? AND student_id = ?`
	var count int64
	err := m.db.QueryRow(query, openID, studentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计学生订阅关系失败: %w", err)
	}
	return count, nil
}
