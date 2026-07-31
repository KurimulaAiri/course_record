// Package jwt JWT 工具包
//
// 与 Java JwtUtils.java 完全对齐，保证 Go↔Java Token 互通
//
// Java 侧实现要点：
//   - 算法：HS256 (HMAC-SHA256)
//   - 密钥：shiroko_project_secret_key_at_least_32_chars_long（硬编码）
//   - Claims: sub="c_user_auth", userId, roleId, tokenType, openId(可选)
//   - Access Token 过期：5 分钟
//   - Refresh Token 过期：7 天
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// ============================================================
// Claims 定义
// ============================================================

// CustomClaims 自定义 JWT Claims
// 与 Java JwtUtils 的 Claims 结构完全对齐
//
// Java 侧 claims：
//   - sub: "c_user_auth"（固定值）
//   - userId: Long（用户 ID）
//   - roleId: Long（角色 ID）
//   - tokenType: "access" | "refresh"
//   - openId: String（可选，微信 openId）
type CustomClaims struct {
	UserID    int64  `json:"userId"`              // 用户 ID（Java Long 类型）
	RoleID    int64  `json:"roleId"`              // 角色 ID（1=admin, 3=parent, 4=teacher, 5=student）
	TokenType string `json:"tokenType"`           // Token 类型：access 或 refresh
	OpenID    string `json:"openId,omitempty"`    // 微信 openId（可选，账号密码登录时不存在）
	jwtv5.RegisteredClaims                      // 标准字段：sub, iat, exp
}

// Token 类型常量（与 Java JwtUtils.TOKEN_TYPE_* 一致）
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// ============================================================
// JWT 工具
// ============================================================

// Utils JWT 工具实例
type Utils struct {
	secretKey        []byte            // HMAC-SHA256 密钥
	accessExpiration time.Duration     // Access Token 过期时间
	refreshExpiration time.Duration    // Refresh Token 过期时间
}

// NewUtils 创建 JWT 工具实例
//
// 参数：
//   - secretKey: HMAC-SHA256 密钥（与 Java SECRET_KEY 一致）
//   - accessExpirationMs: Access Token 过期时间（毫秒）
//   - refreshExpirationMs: Refresh Token 过期时间（毫秒）
func NewUtils(secretKey string, accessExpirationMs, refreshExpirationMs int64) *Utils {
	return &Utils{
		secretKey:         []byte(secretKey),
		accessExpiration:  time.Duration(accessExpirationMs) * time.Millisecond,
		refreshExpiration: time.Duration(refreshExpirationMs) * time.Millisecond,
	}
}

// CreateAccessToken 生成 Access Token
// 与 Java JwtUtils.createAccessToken 对齐
//
// 参数：
//   - userId: 用户 ID
//   - roleId: 角色 ID
//   - openId: 微信 openId（可为空）
//
// 返回：JWT token 字符串
func (u *Utils) CreateAccessToken(userId, roleId int64, openId string) (string, error) {
	return u.createToken(userId, roleId, openId, TokenTypeAccess, u.accessExpiration)
}

// CreateRefreshToken 生成 Refresh Token
// 与 Java JwtUtils.createRefreshToken 对齐
//
// 参数：
//   - userId: 用户 ID
//   - roleId: 角色 ID
//   - openId: 微信 openId（可为空）
//
// 返回：JWT token 字符串
func (u *Utils) CreateRefreshToken(userId, roleId int64, openId string) (string, error) {
	return u.createToken(userId, roleId, openId, TokenTypeRefresh, u.refreshExpiration)
}

// createToken 内部 Token 生成方法
// 对齐 Java Jwts.builder() 的参数：
//   - subject: "c_user_auth"
//   - claim: userId, roleId, tokenType, openId(可选)
//   - issuedAt: 当前时间
//   - expiration: 当前时间 + expiration
//   - signWith: HS256
func (u *Utils) createToken(userId, roleId int64, openId, tokenType string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := CustomClaims{
		UserID:    userId,
		RoleID:    roleId,
		TokenType: tokenType,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   "c_user_auth", // 与 Java 侧 sub 固定值一致
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(expiration)),
		},
	}
	// openId 非空时才加入（对齐 Java 侧 .claim("openId", openId) 当 openId!=null 时才调用）
	if openId != "" {
		claims.OpenID = openId
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString(u.secretKey)
}

// ParseClaims 解析 Token（仅解析，不校验 tokenType）
// 对齐 Java JwtUtils.parseClaims
//
// 参数：
//   - token: JWT token 字符串
//
// 返回：Claims 指针，错误信息
func (u *Utils) ParseClaims(token string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	_, err := jwtv5.ParseWithClaims(token, claims, func(t *jwtv5.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return u.secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ValidateAccessToken 校验 Access Token
// 对齐 Java JwtUtils.validateAccessToken：验签 + 验过期 + 校验 tokenType=="access"
func (u *Utils) ValidateAccessToken(token string) (*CustomClaims, error) {
	claims, err := u.ParseClaims(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, errors.New("invalid token type: expected access")
	}
	return claims, nil
}

// ValidateRefreshToken 校验 Refresh Token
// 对齐 Java JwtUtils.validateRefreshToken：验签 + 验过期 + 校验 tokenType=="refresh"
func (u *Utils) ValidateRefreshToken(token string) (*CustomClaims, error) {
	claims, err := u.ParseClaims(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, errors.New("invalid token type: expected refresh")
	}
	return claims, nil
}

// GetUserInfoFromToken 从 Token 中提取用户信息
// 对齐 Java JwtUtils.getUserInfoFromToken，返回 map 便于兼容
//
// 返回的 map 包含：userId, roleId, tokenType, openId（如果存在）
func (u *Utils) GetUserInfoFromToken(token string) (map[string]interface{}, error) {
	claims, err := u.ParseClaims(token)
	if err != nil {
		return nil, err
	}
	info := map[string]interface{}{
		"userId":    claims.UserID,
		"roleId":    claims.RoleID,
		"tokenType": claims.TokenType,
	}
	if claims.OpenID != "" {
		info["openId"] = claims.OpenID
	}
	return info, nil
}
