// Package mapper admin-service 系统配置 Mapper
//
// 对齐 Java admin-service SysConfigServiceImpl
// 操作 sys_config 表，提供系统配置的 CRUD 操作
//
// 注意：sys_config 表字段对齐 Java SysConfig 实体：
//   - id, config_key, config_value, config_name, config_group, value_type, remark, create_time, update_time
package mapper

import (
	"database/sql"
	"fmt"
	"strings"
)

// ============================================================
// VO/Row 定义（对齐 Java SysConfigVO）
// ============================================================

// SysConfigRow 系统配置查询行（对齐 Java SysConfigVO）
//
// 使用普通类型而非 sql.NullXxx，便于 JSON 序列化
type SysConfigRow struct {
	ID           int64  `json:"id"`           // 配置ID
	ConfigKey    string `json:"configKey"`    // 配置键（唯一标识）
	ConfigValue  string `json:"configValue"`  // 配置值
	ConfigName   string `json:"configName"`   // 配置名称（中文描述）
	ConfigGroup  string `json:"configGroup"`  // 配置分组（如 jwt/auth/cache）
	ValueType    string `json:"valueType"`    // 值类型（STRING/INTEGER/LONG/BOOLEAN）
	Remark       string `json:"remark"`       // 备注说明
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串（yyyy-MM-dd HH:mm:ss）
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
}

// scanSysConfig 通用系统配置行扫描函数
//
// 复用 sql.Row 和 sql.Rows 的 Scan 方法
func scanSysConfig(scanner interface {
	Scan(dest ...interface{}) error
}) (*SysConfigRow, error) {
	row := &SysConfigRow{}
	var (
		configKey, configValue, configName, configGroup, valueType, remark sql.NullString
		createTime, updateTime                                             sql.NullTime
	)
	err := scanner.Scan(
		&row.ID, &configKey, &configValue, &configName, &configGroup, &valueType, &remark, &createTime, &updateTime,
	)
	if err != nil {
		return nil, err
	}
	row.ConfigKey = configKey.String
	row.ConfigValue = configValue.String
	row.ConfigName = configName.String
	row.ConfigGroup = configGroup.String
	row.ValueType = valueType.String
	row.Remark = remark.String
	row.CreateTimeStr = formatTimeSQL(createTime)
	row.UpdateTimeStr = formatTimeSQL(updateTime)
	return row, nil
}

// ============================================================
// SysConfigMapper 系统配置 Mapper
// ============================================================

// SysConfigMapper 系统配置 Mapper
//
// 对齐 Java SysConfigMapper（MyBatis-Plus BaseMapper）
type SysConfigMapper struct {
	db *sql.DB
}

// NewSysConfigMapper 创建 SysConfigMapper
func NewSysConfigMapper(db *sql.DB) *SysConfigMapper {
	return &SysConfigMapper{db: db}
}

// SelectList 查询系统配置列表（带筛选条件）
//
// 对齐 Java listConfigs：按 config_key / config_name / config_group 过滤
// 按 config_group ASC, id ASC 排序
//
// 参数：
//   - configKey: 配置键模糊匹配（空不过滤）
//   - configName: 配置名称模糊匹配（空不过滤）
//   - configGroup: 配置分组精确匹配（空不过滤）
//
// 返回：配置列表
func (m *SysConfigMapper) SelectList(configKey, configName, configGroup string) ([]*SysConfigRow, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if configKey != "" {
		where += " AND config_key LIKE ?"
		args = append(args, "%"+configKey+"%")
	}
	if configName != "" {
		where += " AND config_name LIKE ?"
		args = append(args, "%"+configName+"%")
	}
	if configGroup != "" {
		where += " AND config_group = ?"
		args = append(args, configGroup)
	}

	query := fmt.Sprintf(`
		SELECT id, config_key, config_value, config_name, config_group, value_type, remark, create_time, update_time
		FROM sys_config
		%s
		ORDER BY config_group ASC, id ASC
	`, where)
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询系统配置列表失败: %w", err)
	}
	defer rows.Close()

	var list []*SysConfigRow
	for rows.Next() {
		row, err := scanSysConfig(rows)
		if err != nil {
			return nil, fmt.Errorf("扫描系统配置记录失败: %w", err)
		}
		list = append(list, row)
	}
	return list, nil
}

// SelectByID 按ID查询系统配置
func (m *SysConfigMapper) SelectByID(id int64) (*SysConfigRow, error) {
	query := `SELECT id, config_key, config_value, config_name, config_group, value_type, remark, create_time, update_time FROM sys_config WHERE id = ?`
	row := m.db.QueryRow(query, id)
	config, err := scanSysConfig(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询系统配置失败: %w", err)
	}
	return config, nil
}

// SelectByKey 按配置键查询（用于唯一性校验和取值）
func (m *SysConfigMapper) SelectByKey(configKey string) (*SysConfigRow, error) {
	query := `SELECT id, config_key, config_value, config_name, config_group, value_type, remark, create_time, update_time FROM sys_config WHERE config_key = ?`
	row := m.db.QueryRow(query, configKey)
	config, err := scanSysConfig(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("按 key 查询系统配置失败: %w", err)
	}
	return config, nil
}

// Insert 新增系统配置
//
// 对齐 Java insertConfig
//
// 参数：
//   - configKey: 配置键（必填，唯一）
//   - configValue: 配置值
//   - configName: 配置名称
//   - configGroup: 配置分组（空默认 "system"）
//   - valueType: 值类型（空默认 "STRING"）
//   - remark: 备注
//
// 返回：新配置ID
func (m *SysConfigMapper) Insert(configKey, configValue, configName, configGroup, valueType, remark string) (int64, error) {
	// 默认值处理（对齐 Java dto.getConfigGroup() != null ? : "system"）
	if configGroup == "" {
		configGroup = "system"
	}
	if valueType == "" {
		valueType = "STRING"
	}
	query := `INSERT INTO sys_config (config_key, config_value, config_name, config_group, value_type, remark, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, configKey, configValue, configName, configGroup, valueType, remark)
	if err != nil {
		return 0, fmt.Errorf("新增系统配置失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取系统配置ID失败: %w", err)
	}
	return id, nil
}

// Update 更新系统配置（动态更新非空字段）
//
// 对齐 Java updateConfig
//
// 参数：
//   - id: 配置ID
//   - configValue: 配置值（必填）
//   - configName: 配置名称（空不更新）
//   - remark: 备注（空不更新）
func (m *SysConfigMapper) Update(id int64, configValue, configName, remark string) error {
	setParts := []string{"config_value = ?", "update_time = NOW()"}
	args := []interface{}{configValue}
	// configName/remark 为可选更新字段（对齐 Java dto.getXxx() != null 判断）
	if configName != "" {
		setParts = append(setParts, "config_name = ?")
		args = append(args, configName)
	}
	if remark != "" {
		setParts = append(setParts, "remark = ?")
		args = append(args, remark)
	}

	query := fmt.Sprintf("UPDATE sys_config SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新系统配置失败: %w", err)
	}
	return nil
}

// DeleteByID 按ID删除系统配置
func (m *SysConfigMapper) DeleteByID(id int64) error {
	if _, err := m.db.Exec(`DELETE FROM sys_config WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除系统配置失败: %w", err)
	}
	return nil
}
