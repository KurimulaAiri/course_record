// Package mapper 系统操作日志表操作（对齐 Java SysOperationLogMapper）
//
// 包含：
//   - SysOperationLog 实体定义（对齐 Java SysOperationLog.java，匹配 sys_operation_log 表实际字段）
//   - SysOperationLogMapper：日志 CRUD
//
// 注意：实体字段与 common/entity/entity.go 中的 SysOperationLog 不同
//   - entity.go 的 SysOperationLog 字段（CostTime/Location/ErrorMsg/Status）与实际 DB schema（duration）不匹配
//   - 此处定义的 SysOperationLog 与 DB schema 完全对齐，与 Java SysOperationLog.java 一致
//   - 前端 SysOperationLogResponse 类型也使用 duration 字段（见 admin 前端 src/types/admin.d.ts）
package mapper

import (
	"database/sql"
	"fmt"
)

// ============================================================
// SysOperationLog 系统操作日志实体（对齐 Java SysOperationLog.java）
// ============================================================

// SysOperationLog 系统操作日志实体
//
// 对齐 Java com.shiroko.repository.entity.SysOperationLog
// 表 sys_operation_log 字段：id, user_id, username, operation, method, params, ip, duration, create_time
//
// duration 字段说明：操作耗时（毫秒），对齐前端 SysOperationLogResponse.duration
type SysOperationLog struct {
	ID         int64        `json:"id"`         // 主键
	UserID     int64        `json:"userId"`     // 操作用户ID（sys_user.id）
	Username   string       `json:"username"`   // 操作用户名
	Operation  string       `json:"operation"`  // 操作描述（如 "新增用户"、"删除角色"）
	Method     string       `json:"method"`     // 请求方法（Controller/Service 方法名）
	Params     string       `json:"params"`     // 请求参数（JSON 字符串）
	IP         string       `json:"ip"`         // 请求IP
	Duration   int64        `json:"duration"`   // 耗时（毫秒）
	CreateTime sql.NullTime `json:"createTime"` // 创建时间
}

// ============================================================
// SysOperationLogMapper 系统操作日志表操作
// ============================================================

// SysOperationLogMapper 系统操作日志表 sys_operation_log 的 Mapper
//
// 对齐 Java SysOperationLogMapper（MyBatis-Plus BaseMapper 自动生成 CRUD）
type SysOperationLogMapper struct {
	db *sql.DB
}

// NewSysOperationLogMapper 创建 SysOperationLogMapper
func NewSysOperationLogMapper(db *sql.DB) *SysOperationLogMapper {
	return &SysOperationLogMapper{db: db}
}

// scanLog 通用日志扫描函数（复用 sql.Row 和 sql.Rows）
//
// 直接返回原始 error（不包装），便于调用方判断 sql.ErrNoRows
func scanLog(scanner interface {
	Scan(dest ...interface{}) error
}) (*SysOperationLog, error) {
	l := &SysOperationLog{}
	var username, operation, method, params, ip sql.NullString
	var userID, duration sql.NullInt64
	err := scanner.Scan(
		&l.ID,
		&userID,
		&username,
		&operation,
		&method,
		&params,
		&ip,
		&duration,
		&l.CreateTime,
	)
	if err != nil {
		return nil, err
	}
	l.Username = username.String
	l.Operation = operation.String
	l.Method = method.String
	l.Params = params.String
	l.IP = ip.String
	if userID.Valid {
		l.UserID = userID.Int64
	}
	if duration.Valid {
		l.Duration = duration.Int64
	}
	return l, nil
}

// SelectList 按条件查询日志列表（分页）
//
// 对齐 Java SysOperationLogController.getLogList 的查询逻辑
// 筛选条件：
//   - operation: 操作描述模糊查询（LIKE '%operation%'），空字符串不筛选
//   - username: 用户名模糊查询（LIKE '%username%'），空字符串不筛选
//
// 排序：按 create_time 降序（对齐 Java orderByDesc(SysOperationLog::getCreateTime)）
//
// 参数：
//   - operation: 操作描述筛选
//   - username: 用户名筛选
//   - offset: 分页偏移量
//   - limit: 每页条数
//
// 返回：日志列表
func (m *SysOperationLogMapper) SelectList(operation, username string, offset, limit int) ([]*SysOperationLog, error) {
	query := `SELECT id, user_id, username, operation, method, params, ip, duration, create_time
	          FROM sys_operation_log WHERE 1=1`
	args := []interface{}{}
	if operation != "" {
		query += ` AND operation LIKE ?`
		args = append(args, "%"+operation+"%")
	}
	if username != "" {
		query += ` AND username LIKE ?`
		args = append(args, "%"+username+"%")
	}
	query += ` ORDER BY create_time DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询操作日志列表失败: %w", err)
	}
	defer rows.Close()

	var list []*SysOperationLog
	for rows.Next() {
		log, err := scanLog(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描操作日志记录失败: %w", err)
		}
		list = append(list, log)
	}
	return list, nil
}

// CountWithFilter 按条件统计日志数
//
// 筛选条件与 SelectList 保持一致
func (m *SysOperationLogMapper) CountWithFilter(operation, username string) (int64, error) {
	query := `SELECT COUNT(1) FROM sys_operation_log WHERE 1=1`
	args := []interface{}{}
	if operation != "" {
		query += ` AND operation LIKE ?`
		args = append(args, "%"+operation+"%")
	}
	if username != "" {
		query += ` AND username LIKE ?`
		args = append(args, "%"+username+"%")
	}

	var count int64
	err := m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计操作日志数失败: %w", err)
	}
	return count, nil
}

// Insert 新增操作日志
//
// 对齐 Java SysOperationLogMapper.insert
// 用于记录管理端用户的操作行为（审计日志）
//
// 参数：
//   - log: 日志实体
//
// 返回：新日志ID
func (m *SysOperationLogMapper) Insert(log *SysOperationLog) (int64, error) {
	query := `INSERT INTO sys_operation_log (user_id, username, operation, method, params, ip, duration, create_time)
	          VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`
	result, err := m.db.Exec(query, log.UserID, log.Username, log.Operation, log.Method, log.Params, log.IP, log.Duration)
	if err != nil {
		return 0, fmt.Errorf("新增操作日志失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取日志ID失败: %w", err)
	}
	return id, nil
}

// DeleteByID 删除单条日志
//
// 对齐 Java SysOperationLogMapper.deleteById
//
// 参数：
//   - id: 日志ID
func (m *SysOperationLogMapper) DeleteByID(id int64) error {
	query := `DELETE FROM sys_operation_log WHERE id = ?`
	_, err := m.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除操作日志失败: %w", err)
	}
	return nil
}

// DeleteAll 清空全部日志
//
// 对齐 Java SysOperationLogMapper.delete(null)（MyBatis-Plus 传 null 条件即全表删除）
//
// 注意：TRUNCATE 比 DELETE 更快，但 MCP DB 工具禁止 TRUNCATE
// 这里使用 DELETE FROM（不带 WHERE）清空表
func (m *SysOperationLogMapper) DeleteAll() error {
	query := `DELETE FROM sys_operation_log`
	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("清空操作日志失败: %w", err)
	}
	return nil
}
