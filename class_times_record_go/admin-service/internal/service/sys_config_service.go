// Package service admin-service 系统配置业务逻辑层
//
// 对齐 Java admin-service SysConfigServiceImpl
// 操作 sys_config 表，管理系统运行时可动态调整的参数
//
// 注意：Java 端使用 Redis Cache-Aside 模式缓存配置值，Go 端暂未实现缓存
// （Redis 缓存可后续在 common 包统一实现，不影响接口功能）
//
// 涵盖接口：
//   - /config/list：查询系统配置列表（支持筛选）
//   - /config/insert：新增系统配置（key 唯一性校验）
//   - /config/update：更新系统配置
//   - /config/delete：删除系统配置
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// DTO 定义（对齐 Java QuerySysConfigDTO / InsertSysConfigDTO / UpdateSysConfigDTO）
// ============================================================

// QuerySysConfigRequest 系统配置列表查询请求
type QuerySysConfigRequest struct {
	ConfigKey   string `json:"configKey"`   // 配置键模糊匹配（空不过滤）
	ConfigName  string `json:"configName"`  // 配置名称模糊匹配（空不过滤）
	ConfigGroup string `json:"configGroup"` // 配置分组精确匹配（空不过滤）
}

// InsertSysConfigRequest 新增系统配置请求
type InsertSysConfigRequest struct {
	ConfigKey   string `json:"configKey"`   // 配置键（必填，唯一）
	ConfigValue string `json:"configValue"` // 配置值
	ConfigName  string `json:"configName"`  // 配置名称
	ConfigGroup string `json:"configGroup"` // 配置分组（空默认 "system"）
	ValueType   string `json:"valueType"`   // 值类型（空默认 "STRING"）
	Remark      string `json:"remark"`      // 备注
}

// UpdateSysConfigRequest 更新系统配置请求
type UpdateSysConfigRequest struct {
	ID          int64  `json:"id"`          // 配置ID（必填）
	ConfigValue string `json:"configValue"` // 配置值（必填）
	ConfigName  string `json:"configName"`  // 配置名称（空不更新）
	Remark      string `json:"remark"`      // 备注（空不更新）
}

// ============================================================
// SysConfigService 系统配置服务
// ============================================================

// SysConfigService 系统配置服务（对齐 Java SysConfigServiceImpl）
//
// 注入：
//   - configMapper：系统配置 Mapper
//   - logService：操作日志服务
//
// 注意：Go 端暂未实现 Redis 缓存（Java 端使用 Cache-Aside 模式）
type SysConfigService struct {
	configMapper *mapper.SysConfigMapper
	logService   *SysOperationLogService
}

// NewSysConfigService 创建 SysConfigService
//
// 参数：
//   - configMapper: 系统配置 Mapper
//   - logService: 操作日志服务
func NewSysConfigService(configMapper *mapper.SysConfigMapper, logService *SysOperationLogService) *SysConfigService {
	return &SysConfigService{
		configMapper: configMapper,
		logService:   logService,
	}
}

// ListConfigs 查询系统配置列表
//
// 对齐 Java listConfigs
// 按 config_key / config_name / config_group 过滤
// 按 config_group ASC, id ASC 排序
//
// 参数：
//   - req: 查询请求
//
// 返回：SysConfigVO[]
func (s *SysConfigService) ListConfigs(req *QuerySysConfigRequest) *response.ResponseDTO {
	if req == nil {
		req = &QuerySysConfigRequest{}
	}
	list, err := s.configMapper.SelectList(req.ConfigKey, req.ConfigName, req.ConfigGroup)
	if err != nil {
		log.Printf("查询系统配置列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.SysConfigRow{}
	}
	return response.Success(list)
}

// InsertConfig 新增系统配置
//
// 对齐 Java insertConfig
// 流程：
//   1. 校验配置键唯一性
//   2. 插入新配置（默认 configGroup="system", valueType="STRING"）
//
// 参数：
//   - req: 新增请求
//
// 返回：新配置 VO
func (s *SysConfigService) InsertConfig(req *InsertSysConfigRequest) *response.ResponseDTO {
	if req.ConfigKey == "" {
		return response.Fail("配置键不能为空")
	}
	// 1. 校验配置键唯一性（对齐 Java sysConfigMapper.selectOne eq configKey）
	existing, err := s.configMapper.SelectByKey(req.ConfigKey)
	if err != nil {
		log.Printf("校验配置键唯一性失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if existing != nil {
		return response.Fail("配置键已存在: " + req.ConfigKey)
	}
	// 2. 插入新配置
	id, err := s.configMapper.Insert(req.ConfigKey, req.ConfigValue, req.ConfigName, req.ConfigGroup, req.ValueType, req.Remark)
	if err != nil {
		log.Printf("新增系统配置失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增配置失败")
	}
	// 3. 查询返回完整配置信息
	config, err := s.configMapper.SelectByID(id)
	if err != nil || config == nil {
		return response.Success(map[string]interface{}{
			"id":        id,
			"configKey": req.ConfigKey,
		})
	}
	return response.Success(config)
}

// UpdateConfig 更新系统配置
//
// 对齐 Java updateConfig
// 流程：
//   1. 校验配置存在
//   2. 更新配置值（同时更新 configName/remark 如提供）
//
// 注意：Java 端更新后会清除 Redis 缓存使修改实时生效，Go 端暂未实现缓存
//
// 参数：
//   - req: 更新请求
//
// 返回：更新后的配置 VO
func (s *SysConfigService) UpdateConfig(req *UpdateSysConfigRequest) *response.ResponseDTO {
	if req.ID == 0 {
		return response.Fail("配置ID不能为空")
	}
	if req.ConfigValue == "" {
		return response.Fail("配置值不能为空")
	}
	// 1. 校验配置存在
	config, err := s.configMapper.SelectByID(req.ID)
	if err != nil {
		log.Printf("查询系统配置失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if config == nil {
		return response.Fail("配置不存在")
	}
	// 2. 更新配置
	if err := s.configMapper.Update(req.ID, req.ConfigValue, req.ConfigName, req.Remark); err != nil {
		log.Printf("更新系统配置失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新配置失败")
	}
	// 3. 查询返回更新后的配置
	updated, err := s.configMapper.SelectByID(req.ID)
	if err != nil || updated == nil {
		return response.Success(map[string]interface{}{"id": req.ID})
	}
	return response.Success(updated)
}

// DeleteConfig 删除系统配置
//
// 对齐 Java deleteConfig
// 注意：Java 端删除后会清除 Redis 缓存，Go 端暂未实现缓存
//
// 参数：
//   - id: 配置ID
//
// 返回：操作结果消息
func (s *SysConfigService) DeleteConfig(id int64) *response.ResponseDTO {
	if id == 0 {
		return response.Fail("配置ID不能为空")
	}
	// 校验配置存在
	config, err := s.configMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询系统配置失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if config == nil {
		return response.Fail("配置不存在")
	}
	// 删除配置
	if err := s.configMapper.DeleteByID(id); err != nil {
		log.Printf("删除系统配置失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "删除配置失败")
	}
	return response.Success("删除成功")
}
