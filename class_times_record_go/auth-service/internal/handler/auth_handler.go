// Package handler auth-service HTTP 处理层
//
// 对齐 Java com.shiroko.controller.AuthController
//
// 所有接口路径前缀 /auth（经 Gateway StripPrefix=1 后实际路径为 /auth/**）
// 公开接口（免 JWT）：login_no_pwd, login_by_pwd, login_by_token, get_open_id, register, refresh,
//                   get_bind_info, get_bind_info_by_code, check_bind_status, confirm_bind,
//                   record_subscribe, get_subscribe_status, bind_by_code
package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/auth-service/internal/service"
)

// AuthHandler 认证 HTTP 处理器
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRoutes 注册路由（对齐 Java @RequestMapping("/auth")）
//
// 路由前缀 /auth，与 Gateway PUBLIC_PATHS 中的 /auth/auth/* 对齐
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	// 微信登录相关
	mux.HandleFunc("/auth/get_open_id", h.GetOpenID)
	mux.HandleFunc("/auth/login_no_pwd", h.LoginNoPwd)
	mux.HandleFunc("/auth/login_by_pwd", h.LoginByPwd)
	mux.HandleFunc("/auth/login_by_token", h.LoginByToken)
	mux.HandleFunc("/auth/logout", h.Logout)
	mux.HandleFunc("/auth/refresh", h.RefreshToken)
	mux.HandleFunc("/auth/register", h.Register)

	// 用户信息
	mux.HandleFunc("/auth/get_user_auth_info_by_teacher_id", h.GetUserAuthInfoByTeacherID)

	// 订阅相关
	mux.HandleFunc("/auth/record_subscribe", h.RecordSubscribe)
	mux.HandleFunc("/auth/get_subscribe_status", h.GetSubscribeStatus)
}

// ============================================================
// 请求/响应辅助
// ============================================================

// readBody 读取请求体并解析 JSON
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

// writeResponse 写入响应
func writeResponse(w http.ResponseWriter, resp *response.ResponseDTO) {
	response.WriteJSON(w, resp)
}

// ============================================================
// 微信登录相关 Handler
// ============================================================

// GetOpenID 获取微信 openId
//
// 对齐 Java AuthController.getOpenId
// POST /auth/get_open_id
func (h *AuthHandler) GetOpenID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 调用 service 获取 openId
	resp := h.authService.WxLogin(req.Code)
	writeResponse(w, resp)
}

// LoginNoPwd 微信免密登录
//
// 对齐 Java AuthController.loginNoPwd
// POST /auth/login_no_pwd
func (h *AuthHandler) LoginNoPwd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.WxLogin(req.Code)
	writeResponse(w, resp)
}

// LoginByPwd 账号密码登录
//
// 对齐 Java AuthController.loginByPwd
// POST /auth/login_by_pwd
func (h *AuthHandler) LoginByPwd(w http.ResponseWriter, r *http.Request) {
	var req service.LoginByPwdRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.LoginByPwd(&req)
	writeResponse(w, resp)
}

// LoginByToken Token 续登
//
// 对齐 Java AuthController.loginByToken
// POST /auth/login_by_token
func (h *AuthHandler) LoginByToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.LoginByToken(req.Token)
	writeResponse(w, resp)
}

// Logout 登出
//
// 对齐 Java AuthController.logout
// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.Logout(req.Token, req.RefreshToken)
	writeResponse(w, resp)
}

// RefreshToken 刷新 Access Token
//
// 对齐 Java AuthController.refresh
// POST /auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.RefreshAccessToken(req.RefreshToken)
	writeResponse(w, resp)
}

// Register 注册
//
// 对齐 Java AuthController.register
// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.Register(&req)
	writeResponse(w, resp)
}

// ============================================================
// 用户信息相关 Handler
// ============================================================

// GetUserAuthInfoByTeacherID 按教师ID查认证信息
//
// 对齐 Java AuthController.getUserAuthInfoByTeacherId
// POST /auth/get_user_auth_info_by_teacher_id
func (h *AuthHandler) GetUserAuthInfoByTeacherID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeacherID int64 `json:"teacherId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.GetUserAuthByTeacherID(req.TeacherID)
	writeResponse(w, resp)
}

// ============================================================
// 订阅相关 Handler
// ============================================================

// RecordSubscribe 记录订阅授权
//
// 对齐 Java AuthController.recordSubscribe
// POST /auth/record_subscribe
func (h *AuthHandler) RecordSubscribe(w http.ResponseWriter, r *http.Request) {
	var req service.RecordSubscribeRequest
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.RecordSubscribe(&req)
	writeResponse(w, resp)
}

// GetSubscribeStatus 查询订阅状态
//
// 对齐 Java AuthController.getSubscribeStatus
// GET /auth/get_subscribe_status?code=&templateId=&studentId=
func (h *AuthHandler) GetSubscribeStatus(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	code := query.Get("code")
	templateID := query.Get("templateId")

	var studentID int64
	if s := query.Get("studentId"); s != "" {
		// 解析 studentId
		for _, c := range s {
			if c < '0' || c > '9' {
				break
			}
			studentID = studentID*10 + int64(c-'0')
		}
	}

	resp := h.authService.GetSubscribeStatus(code, templateID, studentID)
	writeResponse(w, resp)
}
