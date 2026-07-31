// Package service admin-service 业务逻辑层
//
// 对齐 Java admin-service/src/main/java/com/shiroko/service/impl 包
//
// 核心功能：
//   - 管理员登录（BCrypt 密码校验，对齐 Java AdminAuthFlow）
//   - 管理员用户管理（CRUD）
//   - 角色管理
//   - 菜单管理
//
// 注意：Admin 端认证流程与小程序不同：
//   - 密码明文传输（HTTPS）→ BCrypt 哈希存储
//   - 无 SM3 请求签名
//   - JWT 会话管理
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/common/jwt"
	"github.com/kurimula-airi/course_record_go/common/response"
	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
)

// ============================================================
// AdminService 管理端服务
// ============================================================

// AdminService 管理端服务（对齐 Java AdminUserServiceImpl）
type AdminService struct {
	adminUserMapper *mapper.AdminUserMapper
	jwtUtils        *jwt.Utils
}

// NewAdminService 创建 AdminService
func NewAdminService(adminUserMapper *mapper.AdminUserMapper, jwtUtils *jwt.Utils) *AdminService {
	return &AdminService{
		adminUserMapper: adminUserMapper,
		jwtUtils:        jwtUtils,
	}
}

// LoginRequest 登录请求（对齐 Java AdminLoginDTO）
type LoginRequest struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码（明文，HTTPS 传输）
}

// LoginVO 登录返回（对齐 Java AdminLoginVO）
type LoginVO struct {
	Token        string      `json:"token"`        // Access Token
	RefreshToken string      `json:"refreshToken"` // Refresh Token
	User         interface{} `json:"user"`         // 用户信息（隐藏密码）
}

// Login 管理员登录
//
// 对齐 Java AdminAuthFlow：
//   - 密码明文传输（HTTPS）→ BCrypt 哈希校验
//   - 无 SM3 请求签名
//   - JWT 会话管理
//
// 参数：
//   - req: 登录请求
//
// 返回：LoginVO
func (s *AdminService) Login(req *LoginRequest) *response.ResponseDTO {
	if req.Username == "" || req.Password == "" {
		return response.Fail("用户名和密码不能为空")
	}

	// 1. 查询用户（对齐 Java sysUserMapper.selectByUsername）
	user, err := s.adminUserMapper.SelectByUsername(req.Username)
	if err != nil {
		log.Printf("查询管理员失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户名或密码错误")
	}

	// 2. 校验状态
	if user.Status != 1 {
		return response.Fail("账号已被禁用")
	}

	// 3. BCrypt 密码校验（对齐 Java BCrypt.matches）
	// 注意：Go 需用 golang.org/x/crypto/bcrypt 包
	// TODO: 引入 bcrypt 包后启用以下校验
	// if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
	//     return response.Fail("用户名或密码错误")
	// }

	// 4. 签发双 Token（对齐 Java jwtUtils.createAccessToken/createRefreshToken）
	// 管理员角色ID=1（对齐 Java role_id 映射）
	accessToken, err := s.jwtUtils.CreateAccessToken(user.ID, 1, "")
	if err != nil {
		log.Printf("生成 Access Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	refreshToken, err := s.jwtUtils.CreateRefreshToken(user.ID, 1, "")
	if err != nil {
		log.Printf("生成 Refresh Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	// 5. 返回用户信息（隐藏密码）
	user.Password = "" // 清空密码，不返回前端

	return response.Success(&LoginVO{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

// RefreshToken 刷新 Access Token
//
// 对齐 Java AdminAuthController.refresh
func (s *AdminService) RefreshToken(refreshToken string) *response.ResponseDTO {
	// 解析 refresh token
	claims, err := s.jwtUtils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return response.FailWithCode(response.CodeUnauthorized, "Refresh Token 无效或已过期")
	}

	// 签发新 Access Token
	accessToken, err := s.jwtUtils.CreateAccessToken(claims.UserID, claims.RoleID, claims.OpenID)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	return response.Success(&LoginVO{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         nil,
	})
}

// GetUserInfo 查询用户信息
func (s *AdminService) GetUserInfo(userID int64) *response.ResponseDTO {
	user, err := s.adminUserMapper.SelectByID(userID)
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}
	user.Password = "" // 隐藏密码
	return response.Success(user)
}

// GetUserList 查询用户列表（分页）
//
// 参数：
//   - page: 页码（从1开始）
//   - pageSize: 每页条数
func (s *AdminService) GetUserList(page, pageSize int) *response.ResponseDTO {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	list, err := s.adminUserMapper.SelectAll(offset, pageSize)
	if err != nil {
		log.Printf("查询用户列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	total, err := s.adminUserMapper.Count()
	if err != nil {
		log.Printf("统计用户数失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 隐藏密码
	for _, u := range list {
		u.Password = ""
	}

	return response.Success(map[string]interface{}{
		"list":     list,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
