// Package handler auth-service HTTP 处理层
//
// 对齐 Java com.shiroko.controller.AuthController
//
// 所有接口路径前缀 /auth（经 Gateway StripPrefix=1 后实际路径为 /auth/**）
// 公开接口（免 JWT）：login_no_pwd, login_by_pwd, login_by_token, get_open_id, register, refresh,
//
//	get_bind_info, get_bind_info_by_code, check_bind_status, confirm_bind,
//	record_subscribe, get_subscribe_status, bind_by_code
package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kurimula-airi/course_record_go/auth-service/internal/service"
	"github.com/kurimula-airi/course_record_go/common/response"
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

	// 家长绑定与订阅流程（对齐 Java AuthController 绑定相关接口）
	// 以下接口均无需登录（排除在 JwtInterceptor 之外）
	mux.HandleFunc("/auth/generate_bind_qrcode", h.GenerateBindQrcode)
	mux.HandleFunc("/auth/generate_subscribe_qrcode", h.GenerateSubscribeQrcode)
	mux.HandleFunc("/auth/get_bind_info", h.GetBindInfo)
	mux.HandleFunc("/auth/get_bind_info_by_code", h.GetBindInfoByCode)
	mux.HandleFunc("/auth/check_bind_status", h.CheckBindStatus)
	mux.HandleFunc("/auth/confirm_bind", h.ConfirmBind)
	mux.HandleFunc("/auth/bind_by_code", h.BindByCode)
	mux.HandleFunc("/auth/test_send_subscribe", h.TestSendSubscribe)
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)

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
//
// 仅返回 openId，不签发 Token，不查询用户信息
// 前端获取 openId 后用于后续登录流程
func (h *AuthHandler) GetOpenID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// 调用 GetOpenId 方法（仅获取 openId，不签发 Token，不查用户信息）
	openId, err := h.authService.GetOpenId(req.Code)
	if err != nil {
		writeResponse(w, response.Fail("获取 openId 失败: "+err.Error()))
		return
	}

	// 返回 LoginVO（仅 openId 有值，其他字段为空，对齐 Java getOpenId 只返回 openId 的行为）
	writeResponse(w, response.Success(&service.LoginVO{
		AccessToken:  "",
		RefreshToken: "",
		OpenID:       openId,
		User:         nil,
	}))
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
//
// 请求字段说明：
//   - Java 后端 LoginDTO 中字段名为 token（refreshToken 字段仅用于登出）
//   - 小程序前端传入 { token: refreshToken }（见 src/api/auth/index.ts refreshAccessToken）
//   - 因此这里解析 token 字段，传给 service 作为 refreshToken 参数
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.RefreshAccessToken(req.Token)
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

// ============================================================
// 家长绑定与订阅流程 Handler（对齐 Java AuthController 绑定相关接口）
// ============================================================

// GenerateBindQrcode 生成绑定二维码
//
// 对齐 Java AuthController.generateBindQrcode
// POST /auth/generate_bind_qrcode
//
// 请求体：{ studentId: int64, relation: string, isPrimary: boolean }
// 返回：{ qrcode: string, token: string, bindCode: string }
func (h *AuthHandler) GenerateBindQrcode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int64  `json:"studentId"`
		Relation  string `json:"relation"`
		IsPrimary *bool  `json:"isPrimary"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// isPrimary 默认为 true（主联系人）
	isPrimary := true
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	resp := h.authService.GenerateBindQrcode(req.StudentID, req.Relation, isPrimary)
	writeResponse(w, resp)
}

// GenerateSubscribeQrcode 生成订阅专用二维码
//
// 对齐 Java AuthController.generateSubscribeQrcode
// POST /auth/generate_subscribe_qrcode
//
// 请求体：{ studentId: int64, relation: string, isPrimary: boolean }
// 返回：{ qrcode: string, token: string, bindCode: string }
func (h *AuthHandler) GenerateSubscribeQrcode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StudentID int64  `json:"studentId"`
		Relation  string `json:"relation"`
		IsPrimary *bool  `json:"isPrimary"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	// isPrimary 默认为 true（主联系人）
	isPrimary := true
	if req.IsPrimary != nil {
		isPrimary = *req.IsPrimary
	}

	resp := h.authService.GenerateSubscribeQrcode(req.StudentID, req.Relation, isPrimary)
	writeResponse(w, resp)
}

// GetBindInfo 按 token 查绑定信息（无需登录）
//
// 对齐 Java AuthController.getBindInfo
// GET /auth/get_bind_info?token=xxx
//
// 查询参数：token（绑定 token，扫码后从二维码内容中获取）
// 返回：学生信息（studentId, studentName, sex, institutionName, isSubscribe）
func (h *AuthHandler) GetBindInfo(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	token := query.Get("token")

	resp := h.authService.GetBindInfo(token)
	writeResponse(w, resp)
}

// GetBindInfoByCode 按 6 位绑定码查学生信息（无需登录）
//
// 对齐 Java AuthController.getBindInfoByCode
// GET /auth/get_bind_info_by_code?code=xxx
//
// 查询参数：code（6 位绑定码）
// 返回：学生信息（不执行绑定）
func (h *AuthHandler) GetBindInfoByCode(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	code := query.Get("code")

	resp := h.authService.GetBindInfoByCode(code)
	writeResponse(w, resp)
}

// CheckBindStatus 检查绑定状态（无需登录）
//
// 对齐 Java AuthController.checkBindStatus
// GET /auth/check_bind_status?token=xxx&code=xxx
//
// 查询参数：token（绑定 token）, code（微信登录 code，用于换取 openId）
// 返回：{ alreadyBound: bool, hasAccount: bool }
func (h *AuthHandler) CheckBindStatus(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	token := query.Get("token")
	code := query.Get("code")

	resp := h.authService.CheckBindStatus(token, code)
	writeResponse(w, resp)
}

// ConfirmBind 确认绑定（无需登录）
//
// 对齐 Java AuthController.confirmBind
// POST /auth/confirm_bind
//
// 请求体：{ token: string, openId: string }
// 返回：绑定结果消息
func (h *AuthHandler) ConfirmBind(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token  string `json:"token"`
		OpenID string `json:"openId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.ConfirmBind(req.Token, req.OpenID)
	writeResponse(w, resp)
}

// BindByCode 按绑定码直接绑定（无需登录）
//
// 对齐 Java AuthController.bindByCode
// POST /auth/bind_by_code
//
// 请求体：{ code: string, openId: string }
// 返回：{ message: string, login: { accessToken, refreshToken, openId, user } }
func (h *AuthHandler) BindByCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code   string `json:"code"`
		OpenID string `json:"openId"`
	}
	if err := readBody(r, &req); err != nil {
		writeResponse(w, response.Fail("请求参数解析失败"))
		return
	}

	resp := h.authService.BindByCode(req.Code, req.OpenID)
	writeResponse(w, resp)
}

// TestSendSubscribe 测试发送订阅消息（无需登录）
//
// 对齐 Java AuthController.testSendSubscribe
// GET /auth/test_send_subscribe?openId=xxx
//
// 查询参数：openId（微信 openId）
// 返回：发送结果消息
func (h *AuthHandler) TestSendSubscribe(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	openId := query.Get("openId")

	resp := h.authService.TestSendSubscribe(openId)
	writeResponse(w, resp)
}
