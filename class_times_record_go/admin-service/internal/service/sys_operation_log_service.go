// Package service 系统操作日志业务逻辑层（对齐 Java SysOperationLogServiceImpl）
//
// 职责：
//   - 操作日志查询（list，分页+筛选）
//   - 操作日志删除（delete，按 ID）
//   - 操作日志清空（clear，全表清空）
//   - 操作日志记录工具方法（供其他 Service 调用，对齐 Java @OperationLog 注解切面）
//
// 对齐 Java com.shiroko.service.impl.SysOperationLogServiceImpl
//
// 注意：操作日志记录采用"不阻断主流程"策略，写入失败仅记录日志不抛出错误
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// VO 定义（对齐 Java SysOperationLogVO）
// ============================================================

// SysOperationLogVO 系统操作日志视图对象
//
// 对齐 admin 前端 src/types/admin.d.ts SysOperationLogResponse
//
// 字段命名与前端类型保持一致：
//   - 仅含 createTimeStr（无 createTime），对齐前端类型定义
//   - duration 为操作耗时（毫秒）
type SysOperationLogVO struct {
	ID           int64  `json:"id"`           // 主键
	UserID       int64  `json:"userId"`       // 操作用户ID（sys_user.id）
	Username     string `json:"username"`     // 操作用户名
	Operation    string `json:"operation"`    // 操作描述（如 "新增用户"）
	Method       string `json:"method"`       // 请求方法（Controller/Service 方法名）
	Params       string `json:"params"`       // 请求参数（JSON 字符串）
	IP           string `json:"ip"`           // 请求IP
	Duration     int64  `json:"duration"`     // 耗时（毫秒）
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串（yyyy-MM-dd HH:mm:ss）
}

// ToSysOperationLogVO 操作日志实体转 VO
//
// 将 SysOperationLog 转换为 SysOperationLogVO，避免 sql.NullTime 序列化为对象
//
// 参数：
//   - l: SysOperationLog 实体
func ToSysOperationLogVO(l *mapper.SysOperationLog) *SysOperationLogVO {
	if l == nil {
		return nil
	}
	vo := &SysOperationLogVO{
		ID:        l.ID,
		UserID:    l.UserID,
		Username:  l.Username,
		Operation: l.Operation,
		Method:    l.Method,
		Params:    l.Params,
		IP:        l.IP,
		Duration:  l.Duration,
	}
	// 时间格式化（对齐 Java DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")）
	vo.CreateTimeStr = formatNullTime(l.CreateTime)
	return vo
}

// ============================================================
// DTO 定义（对齐 Java QuerySysOperationLogDTO）
// ============================================================

// QueryOperationLogListRequest 操作日志列表查询请求
//
// 对齐 admin 前端 src/types/admin.d.ts GetOperationLogListRequest
type QueryOperationLogListRequest struct {
	Operation    string `json:"operation"`    // 操作描述（模糊查询，可选）
	Username     string `json:"username"`     // 用户名（模糊查询，可选）
	CurrentPage  int    `json:"currentPage"`  // 当前页码（从1开始）
	PageSize     int    `json:"pageSize"`     // 每页条数
}

// RecordOperationLogRequest 操作日志记录请求（供其他 Service 调用）
//
// 用于在用户操作（新增/更新/删除）时记录审计日志
// 对齐 Java @OperationLog 注解切面收集的信息
type RecordOperationLogRequest struct {
	UserID    int64  `json:"userId"`    // 操作用户ID
	Username  string `json:"username"`  // 操作用户名
	Operation string `json:"operation"` // 操作描述（如 "新增用户"）
	Method    string `json:"method"`    // 请求方法（如 "SysUserServiceImpl.insertUser"）
	Params    string `json:"params"`    // 请求参数（JSON 字符串）
	IP        string `json:"ip"`        // 请求IP
	Duration  int64  `json:"duration"`  // 耗时（毫秒）
}

// ============================================================
// SysOperationLogService 系统操作日志服务
// ============================================================

// SysOperationLogService 系统操作日志服务（对齐 Java SysOperationLogServiceImpl）
//
// 注入：
//   - SysOperationLogMapper：操作日志表操作
type SysOperationLogService struct {
	logMapper *mapper.SysOperationLogMapper
}

// NewSysOperationLogService 创建 SysOperationLogService
//
// 参数：
//   - logMapper: 操作日志 Mapper
func NewSysOperationLogService(logMapper *mapper.SysOperationLogMapper) *SysOperationLogService {
	return &SysOperationLogService{
		logMapper: logMapper,
	}
}

// ListLogs 操作日志列表（分页+筛选）
//
// 对齐 Java SysOperationLogServiceImpl.getLogList
//
// 流程：
//  1. 参数默认值处理
//  2. 分页查询日志列表（按 create_time 降序）
//  3. 统计总数
//  4. 转换为 VO 列表返回
//
// 参数：
//   - req: 查询请求
//
// 返回：{ list: SysOperationLogVO[], total: int64 }
func (s *SysOperationLogService) ListLogs(req *QueryOperationLogListRequest) *response.ResponseDTO {
	// 1. 参数默认值处理
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	// 2. 查询日志列表（按 create_time 降序，对齐 Java orderByDesc）
	list, err := s.logMapper.SelectList(req.Operation, req.Username, offset, req.PageSize)
	if err != nil {
		log.Printf("查询操作日志列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 3. 统计总数
	total, err := s.logMapper.CountWithFilter(req.Operation, req.Username)
	if err != nil {
		log.Printf("统计操作日志数失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 4. 转换为 VO 列表（避免 sql.NullTime 序列化为对象）
	voList := make([]*SysOperationLogVO, 0, len(list))
	for _, l := range list {
		if vo := ToSysOperationLogVO(l); vo != nil {
			voList = append(voList, vo)
		}
	}

	// 响应格式对齐 admin 前端 GetOperationLogListResponse
	return response.Success(map[string]interface{}{
		"list":  voList,
		"total": total,
	})
}

// DeleteLog 删除单条操作日志
//
// 对齐 Java SysOperationLogServiceImpl.deleteLog
//
// 参数：
//   - id: 日志ID
//
// 返回：操作结果消息
func (s *SysOperationLogService) DeleteLog(id int64) *response.ResponseDTO {
	if id <= 0 {
		return response.Fail("日志ID无效")
	}
	if err := s.logMapper.DeleteByID(id); err != nil {
		log.Printf("删除操作日志失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "删除日志失败")
	}
	return response.Success("删除成功")
}

// ClearLogs 清空全部操作日志
//
// 对齐 Java SysOperationLogServiceImpl.clearLogs
//
// 使用 DELETE FROM（不带 WHERE）清空表，对齐 Java sysOperationLogMapper.delete(null)
//
// 返回：操作结果消息
func (s *SysOperationLogService) ClearLogs() *response.ResponseDTO {
	if err := s.logMapper.DeleteAll(); err != nil {
		log.Printf("清空操作日志失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "清空日志失败")
	}
	return response.Success("清空成功")
}

// RecordLog 记录操作日志（不阻断主流程）
//
// 对齐 Java @OperationLog 注解切面，由其他 Service 在执行写操作后调用
//
// 重要：本方法为"尽力而为"模式，写入失败仅记录日志不返回错误
// 避免日志记录失败影响主业务流程
//
// 参数：
//   - req: 操作日志记录请求
func (s *SysOperationLogService) RecordLog(req *RecordOperationLogRequest) {
	if req == nil {
		return
	}
	logEntity := &mapper.SysOperationLog{
		UserID:    req.UserID,
		Username:  req.Username,
		Operation: req.Operation,
		Method:    req.Method,
		Params:    req.Params,
		IP:        req.IP,
		Duration:  req.Duration,
	}
	if _, err := s.logMapper.Insert(logEntity); err != nil {
		// 日志写入失败不阻断主流程，仅记录错误日志
		log.Printf("记录操作日志失败: operation=%s, err=%v", req.Operation, err)
	}
}
