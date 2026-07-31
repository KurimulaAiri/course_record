// Package service admin-service 仪表盘业务逻辑层
//
// 对齐 Java admin-service SysDashboardServiceImpl
// 提供管理端仪表盘的汇总数据、趋势数据、机构统计查询
//
// 涵盖接口：
//   - /dashboard/data：返回学生/教师/机构/课程/班级 总数
//   - /dashboard/trend：返回新增学生/教师 趋势数据（支持 range 参数）
//   - /dashboard/institution/stats：返回机构统计列表（支持 limit 参数）
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// DTO 定义
// ============================================================

// DashboardTrendRequest 趋势数据查询请求
type DashboardTrendRequest struct {
	Range string `json:"range"` // 时间范围（week/month/halfyear/year，默认 year）
}

// InstitutionStatsRequest 机构统计查询请求
type InstitutionStatsRequest struct {
	Limit int `json:"limit"` // 限制返回数量（<=0 表示不限制）
}

// ============================================================
// DashboardService 仪表盘服务
// ============================================================

// DashboardService 仪表盘服务（对齐 Java SysDashboardServiceImpl）
//
// 注入：
//   - dashboardMapper：仪表盘统计 Mapper
type DashboardService struct {
	dashboardMapper *mapper.DashboardMapper
}

// NewDashboardService 创建 DashboardService
//
// 参数：
//   - dashboardMapper: 仪表盘 Mapper
func NewDashboardService(dashboardMapper *mapper.DashboardMapper) *DashboardService {
	return &DashboardService{
		dashboardMapper: dashboardMapper,
	}
}

// GetDashboardData 获取仪表盘汇总数据
//
// 对齐 Java getDashboardData
//
// 返回：DashboardVO（studentCount/teacherCount/institutionCount/courseCount/classCount）
func (s *DashboardService) GetDashboardData() *response.ResponseDTO {
	row, err := s.dashboardMapper.GetDashboardData()
	if err != nil {
		log.Printf("查询仪表盘汇总数据失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	return response.Success(row)
}

// GetTrend 获取趋势数据
//
// 对齐 Java getTrend
// 根据时间范围决定粒度与刻度：
//   - week: 按天统计最近7天
//   - month: 按天统计最近30天
//   - halfyear: 按月统计最近6个月
//   - year（默认）: 按月统计最近12个月
//
// 参数：
//   - req: 查询请求（Range 字段，空默认 "year"）
func (s *DashboardService) GetTrend(req *DashboardTrendRequest) *response.ResponseDTO {
	rangeStr := "year" // 默认按年统计
	if req != nil && req.Range != "" {
		rangeStr = req.Range
	}
	row, err := s.dashboardMapper.GetTrend(rangeStr)
	if err != nil {
		log.Printf("查询仪表盘趋势数据失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	return response.Success(row)
}

// GetInstitutionStats 获取机构统计列表
//
// 对齐 Java getInstitutionStats
// 查询所有机构（id > 0），并统计每个机构的学生/教师/课程/班级数量
// 结果按学生数降序排序，支持 limit 限制返回数量
//
// 参数：
//   - req: 查询请求（Limit 字段，<=0 表示不限制）
func (s *DashboardService) GetInstitutionStats(req *InstitutionStatsRequest) *response.ResponseDTO {
	limit := 0
	if req != nil {
		limit = req.Limit
	}
	list, err := s.dashboardMapper.GetInstitutionStats(limit)
	if err != nil {
		log.Printf("查询机构统计数据失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if list == nil {
		list = []*mapper.InstitutionStatRow{}
	}
	return response.Success(list)
}
