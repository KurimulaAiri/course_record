// Package handler 菜单 HTTP 处理层（对齐 Java MenuController）
//
// 对齐 Java com.shiroko.controller.MenuController
//
// 路由前缀 /menu（注意：不是 /auth，与 Java @RequestMapping("/menu") 一致）
// 经 Gateway StripPrefix=1 后实际路径为 /menu/**
//
// 接口：
//   - POST /menu/get_menu_by_role：按角色查询菜单列表（需 JWT）
package handler

import (
	"net/http"

	commonctx "github.com/kurimula-airi/course_record_go/common/context"
	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/auth-service/internal/service"
)

// ============================================================
// MenuHandler 菜单 HTTP 处理器
// ============================================================

// MenuHandler 菜单处理器（对齐 Java MenuController）
type MenuHandler struct {
	menuService *service.MenuService
}

// NewMenuHandler 创建 MenuHandler
//
// 参数：
//   - menuService: 菜单服务
func NewMenuHandler(menuService *service.MenuService) *MenuHandler {
	return &MenuHandler{menuService: menuService}
}

// RegisterRoutes 注册菜单路由（对齐 Java @RequestMapping("/menu")）
//
// 路由前缀 /menu，与 Java MenuController 一致
// 注意：此路径为 Gateway StripPrefix=1 之后的路径
// 前端请求 /auth/menu/get_menu_by_role → Gateway 转发 /menu/get_menu_by_role
func (h *MenuHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/menu/get_menu_by_role", h.GetMenuByRole)
}

// GetMenuByRole 按角色获取菜单列表
//
// 对齐 Java MenuController.getMenuByRole
// POST /menu/get_menu_by_role
//
// 请求体：QueryMenuDTO { roleId, currentPage, pageSize }
// 注意：roleId 会被上下文角色覆盖（安全考虑，对齐 Java UserContext.getUser().getRoleId()）
//
// 响应：ResponseDTO<QueryMenuVO> { menus: [...], total: N }
func (h *MenuHandler) GetMenuByRole(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求体
	var req service.QueryMenuDTO
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 2. 从上下文获取用户角色ID（对齐 Java UserContext.getUser().getRoleId()）
	//    Gateway 已通过 JWT 解析并注入 X-User-Role header，中间件写入 context
	//    此处覆盖请求中的 roleId，确保用户无法越权查询其他角色菜单
	ctx := r.Context()
	user := commonctx.GetUser(ctx)
	if user == nil {
		writeResponse(w, response.FailWithCode(response.CodeUnauthorized, "未授权"))
		return
	}

	// 3. 调用 service 查询菜单（传入上下文角色，覆盖请求角色）
	resp := h.menuService.GetMenuByRole(&req, user.RoleID)
	writeResponse(w, resp)
}
