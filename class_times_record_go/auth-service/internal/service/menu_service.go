// Package service 菜单业务逻辑层（对齐 Java MenuServiceImpl）
//
// 职责：按角色查询小程序端菜单列表（分页）
// 缓存策略：Java 版用 Redis Cache-Aside 缓存 30min，Go 版暂未实现缓存（可直接查库，菜单数据量小）
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/auth-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// DTO / VO 定义（对齐 Java QueryMenuDTO / MenuVO / QueryMenuVO）
// ============================================================

// QueryMenuDTO 查询菜单请求（对齐 Java QueryMenuDTO）
//
// 前端类型定义（src/types/menu.d.ts GetMenuListRequest）：
//   - roleId: 角色ID（实际由后端从 UserContext 覆盖，前端值忽略）
//   - currentPage: 当前页码
//   - pageSize: 每页条数
type QueryMenuDTO struct {
	RoleID      int64 `json:"roleId"`      // 角色ID（后端从上下文覆盖，确保安全）
	CurrentPage int64 `json:"currentPage"` // 当前页码（从1开始）
	PageSize    int64 `json:"pageSize"`    // 每页条数
}

// MenuVO 菜单视图对象（对齐 Java MenuVO）
//
// 前端类型定义（src/types/menu.d.ts MenuResponse）：
//   - id / menuName / icon / iconType / bgColor / path / sortOrder / isVisible
//   - createTimeStr / updateTimeStr（格式化字符串，非时间戳）
//
// 注意：使用普通类型而非 sql.NullXxx，避免 JSON 序列化为对象格式
type MenuVO struct {
	ID            int64  `json:"id"`            // 菜单ID
	MenuName      string `json:"menuName"`      // 菜单名称
	Icon          string `json:"icon"`          // 图标（名称或路径）
	IconType      int64  `json:"iconType"`      // 图标类型（0=内置, 1=路径）
	BgColor       string `json:"bgColor"`       // 图标背景色（Hex）
	Path          string `json:"path"`          // 跳转路由路径
	SortOrder     int64  `json:"sortOrder"`     // 排序权值（越小越靠前）
	IsVisible     bool   `json:"isVisible"`     // 是否显示
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
}

// QueryMenuVO 菜单查询响应包装（对齐 Java QueryMenuVO）
//
// 前端类型定义（src/types/menu.d.ts MenuListResponse）：
//   - total: 总记录数
//   - menus: 菜单列表
type QueryMenuVO struct {
	Menus []*MenuVO `json:"menus"` // 菜单列表
	Total int64     `json:"total"` // 总记录数
}

// ============================================================
// MenuService 菜单服务
// ============================================================

// MenuService 菜单服务（对齐 Java MenuServiceImpl）
//
// 注入：MenuMapper
type MenuService struct {
	menuMapper *mapper.MenuMapper
}

// NewMenuService 创建 MenuService
//
// 参数：
//   - menuMapper: 菜单表 Mapper
func NewMenuService(menuMapper *mapper.MenuMapper) *MenuService {
	return &MenuService{menuMapper: menuMapper}
}

// GetMenuByRole 按角色查询菜单列表
//
// 对齐 Java MenuServiceImpl.getMenuByRole
//
// 流程：
//  1. 从 UserContext 获取 roleID（覆盖请求中的 roleId，确保安全，对齐 Java UserContext.getUser().getRoleId()）
//  2. 计算分页偏移量 offset = (currentPage - 1) * pageSize
//  3. 查询数据库（分页 + 总数）
//  4. 转换为 MenuVO 列表（格式化时间字段）
//  5. 包装为 QueryMenuVO 返回
//
// 参数：
//   - req: 查询请求（roleId 字段会被上下文角色覆盖）
//   - roleID: 从上下文获取的角色ID（由 handler 传入）
//
// 返回：ResponseDTO<QueryMenuVO>
func (s *MenuService) GetMenuByRole(req *QueryMenuDTO, roleID int64) *response.ResponseDTO {
	// 1. 用上下文角色覆盖请求角色（对齐 Java queryMenuDTO.setRoleId(UserContext.getUser().getRoleId())）
	req.RoleID = roleID

	// 2. 参数校验与默认值处理（对齐 Java Page 构造）
	currentPage := req.CurrentPage
	if currentPage < 1 {
		currentPage = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (currentPage - 1) * pageSize

	// 3. 查询数据库（分页查询 + 总数统计）
	list, total, err := s.menuMapper.SelectByRole(roleID, offset, pageSize)
	if err != nil {
		log.Printf("查询菜单列表失败: roleID=%d, err=%v", roleID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 4. 转换为 MenuVO 列表（对齐 Java MenuConverter.pojoListToVOList）
	voList := make([]*MenuVO, 0, len(list))
	for _, menu := range list {
		voList = append(voList, toMenuVO(menu))
	}

	// 5. 包装为 QueryMenuVO 返回（对齐 Java new QueryMenuVO(menuVOs, page.getTotal())）
	return response.Success(&QueryMenuVO{
		Menus: voList,
		Total: total,
	})
}

// toMenuVO 将 Menu 实体转换为 MenuVO（对齐 Java MenuConverter.pojoToVO）
//
// 转换要点：
//   - sql.NullXxx → 普通类型（无效值转为零值）
//   - createTime/updateTime → 格式化字符串（对齐 Java @BaseDateTimeToString "yyyy-MM-dd HH:mm:ss"）
func toMenuVO(m *entity.Menu) *MenuVO {
	vo := &MenuVO{}

	// 主键
	if m.ID.Valid {
		vo.ID = m.ID.Int64
	}

	// 字符串字段
	if m.MenuName.Valid {
		vo.MenuName = m.MenuName.String
	}
	if m.Icon.Valid {
		vo.Icon = m.Icon.String
	}
	if m.BgColor.Valid {
		vo.BgColor = m.BgColor.String
	}
	if m.Path.Valid {
		vo.Path = m.Path.String
	}

	// 数值字段
	if m.IconType.Valid {
		vo.IconType = m.IconType.Int64
	}
	if m.SortOrder.Valid {
		vo.SortOrder = m.SortOrder.Int64
	}

	// 布尔字段
	if m.IsVisible.Valid {
		vo.IsVisible = m.IsVisible.Bool
	}

	// 时间字段格式化（对齐 Java DateTransformUtils + @BaseDateTimeToString）
	vo.CreateTimeStr = entity.FormatTime(m.CreateTime)
	vo.UpdateTimeStr = entity.FormatTime(m.UpdateTime)

	return vo
}
