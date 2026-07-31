// Package context 用户上下文
//
// 对齐 Java com.shiroko.context.UserContext + UserDTO + GatewayUserFilter
//
// 链路：Gateway 解析 JWT → 注入 X-User-Id / X-User-Role / X-User-OpenId header
//       → 微服务中间件读 header → 写入 UserContext → Handler/Service 通过 UserContext 获取用户信息
//
// Go 实现说明：
//   - Java 用 ThreadLocal（线程本地变量）
//   - Go 用 context.Context 传递（更符合 Go 并发模型）
//   - 提供 WithUser/SetUser/GetUser 等方法，封装 context.Context 操作
package context

import (
	"context"
	"strconv"
)

// UserDTO 用户信息（对齐 Java com.shiroko.repository.dto.UserDTO）
//
// 从 Gateway 注入的 X-User-* header 解析得到
type UserDTO struct {
	UserID         int64  `json:"userId"`         // 用户ID（c_user.id）
	RoleID         int64  `json:"roleId"`         // 角色ID（c_permission.id: 1=admin,3=parent,4=teacher,5=student）
	InstitutionID  int64  `json:"institutionId"`  // 机构ID（c_institution.id）
	Username       string `json:"username"`       // 用户名
	OpenID         string `json:"openId"`         // 当前会话 openId（从 JWT 透传，可能为空）
}

// contextKey 自定义类型，避免 context.Value key 冲突
type contextKey struct{}

// userKey 用户上下文的 key
var userKey contextKey

// WithUser 将用户信息写入 context（对齐 Java GatewayUserFilter.doFilterInternal）
//
// 用法：ctx = context.WithUser(ctx, user)
//
// 参数：
//   - ctx: 原 context
//   - user: 用户信息
//
// 返回：包含用户信息的新 context
func WithUser(ctx context.Context, user *UserDTO) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// GetUser 从 context 获取用户信息（对齐 Java UserContext.getUser）
//
// 若未设置返回 nil
func GetUser(ctx context.Context) *UserDTO {
	if v, ok := ctx.Value(userKey).(*UserDTO); ok {
		return v
	}
	return nil
}

// GetUserID 从 context 获取用户ID（对齐 Java UserContext.getUserId）
//
// 若未设置返回 0
func GetUserID(ctx context.Context) int64 {
	if u := GetUser(ctx); u != nil {
		return u.UserID
	}
	return 0
}

// GetRoleID 从 context 获取角色ID
func GetRoleID(ctx context.Context) int64 {
	if u := GetUser(ctx); u != nil {
		return u.RoleID
	}
	return 0
}

// GetOpenID 从 context 获取 openId
func GetOpenID(ctx context.Context) string {
	if u := GetUser(ctx); u != nil {
		return u.OpenID
	}
	return ""
}

// FromHeaders 从 HTTP header 解析用户信息（对齐 Java GatewayUserFilter 读 header 逻辑）
//
// 参数：
//   - userIDHeader: X-User-Id header 值
//   - roleIDHeader: X-User-Role header 值
//   - openIDHeader: X-User-OpenId header 值
//
// 返回：用户信息指针（userIDHeader 为空时返回 nil）
func FromHeaders(userIDHeader, roleIDHeader, openIDHeader string) *UserDTO {
	if userIDHeader == "" {
		return nil
	}

	userID, err := strconv.ParseInt(userIDHeader, 10, 64)
	if err != nil {
		return nil
	}

	user := &UserDTO{
		UserID: userID,
	}

	// roleID 可选（对齐 Java roleHeader != null 判断）
	if roleIDHeader != "" {
		if roleID, err := strconv.ParseInt(roleIDHeader, 10, 64); err == nil {
			user.RoleID = roleID
		}
	}

	// openId 可选（对齐 Java openIdHeader != null && !openIdHeader.isEmpty() 判断）
	if openIDHeader != "" {
		user.OpenID = openIDHeader
	}

	return user
}

// 角色常量（对齐 Java ROLE 枚举：1=admin, 3=parent, 4=teacher, 5=student）
const (
	RoleAdmin   int64 = 1 // 系统管理员（→ sys_user.id）
	RoleParent  int64 = 3 // 家长（→ c_parent.id）
	RoleTeacher int64 = 4 // 教师（→ c_teacher.teacher_id）
	RoleStudent int64 = 5 // 学生（→ c_student.id）
)

// RoleName 角色名映射（对齐 Java identityService.checkAvailable 用的 roleName）
//
// 参数：
//   - roleID: 角色ID
//
// 返回：角色名字符串（用于 identityService.createIdentity/checkAvailable）
func RoleName(roleID int64) string {
	switch roleID {
	case RoleAdmin:
		return "admin"
	case RoleParent:
		return "parent"
	case RoleTeacher:
		return "teacher"
	case RoleStudent:
		return "student"
	default:
		return ""
	}
}
