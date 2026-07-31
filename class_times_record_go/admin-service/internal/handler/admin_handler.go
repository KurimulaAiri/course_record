// Package handler admin-service HTTP 处理层
//
// 对齐 Java admin-service/src/main/java/com/shiroko/controller 包
//
// 所有接口路径前缀 /admin（经 Gateway StripPrefix=1 后实际路径为 /{module}/**）
// 公开接口（免 JWT）：user/login, user/refresh, crypto/public_key
package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/admin-service/internal/service"
)

// AdminHandler 管理端 HTTP 处理器
type AdminHandler struct {
	adminService *service.AdminService
}

// NewAdminHandler 创建 AdminHandler
func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

// RegisterRoutes 注册路由（对齐 Java 各 Controller 的 @RequestMapping）
//
// 路由前缀说明：
//   - Gateway 转发 /admin/** 到 admin-service，StripPrefix=1 去除 /admin
//   - 所以 admin-service 收到的路径是 /user/**, /role/**, /menu/**
func (h *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	// 用户相关（对齐 Java AdminUserController @RequestMapping("/user")）
	mux.HandleFunc("/user/login", h.Login)
	mux.HandleFunc("/user/refresh", h.RefreshToken)
	mux.HandleFunc("/user/info", h.GetUserInfo)
	mux.HandleFunc("/user/list", h.GetUserList)
}

// 请求/响应辅助
func readBody(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

func writeResponse(w http.ResponseWriter, resp *response.ResponseDTO) {
	response.WriteJSON(w, resp)
}

// Login 管理员登录
// POST /user/login
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.Login(&req))
}

// RefreshToken 刷新 Token
// POST /user/refresh
func (h *AdminHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.RefreshToken(req.RefreshToken))
}

// GetUserInfo 查询用户信息
// POST /user/info
func (h *AdminHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int64 `json:"userId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}
	writeResponse(w, h.adminService.GetUserInfo(req.UserID))
}

// GetUserList 查询用户列表
// GET /user/list?page=1&pageSize=10
func (h *AdminHandler) GetUserList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("pageSize"))
	writeResponse(w, h.adminService.GetUserList(page, pageSize))
}
