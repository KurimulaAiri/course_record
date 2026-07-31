// Package service auth-service 业务逻辑层
//
// 对齐 Java com.shiroko.service.impl.AuthServiceImpl
//
// 核心功能：
//   - 微信免密登录（getOpenId + 双 Token 签发）
//   - 账号密码登录（SM2 解密 + SM3 验签 + 机构过期校验）
//   - Token 续登 / 刷新 / 登出
//   - 注册（去重 + SM3 加盐哈希存储）
//   - 绑定学生（核心复杂逻辑，分"仅订阅"和"绑定账号"两种模式）
//   - 微信订阅记录与查询
package service

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kurimula-airi/course_record_go/common/context"
	"github.com/kurimula-airi/course_record_go/common/crypto"
	"github.com/kurimula-airi/course_record_go/common/entity"
	"github.com/kurimula-airi/course_record_go/common/jwt"
	"github.com/kurimula-airi/course_record_go/common/redis"
	"github.com/kurimula-airi/course_record_go/common/response"
	rediscli "github.com/redis/go-redis/v9"

	"github.com/kurimula-airi/course_record_go/auth-service/internal/mapper"
)

// ============================================================
// 常量与配置
// ============================================================

// 微信小程序配置（对齐 Java @Value("${uni-app.wx.app-id}")）
const (
	wxAppID     = "wx1234567890abcdef"               // TODO: 从配置加载
	wxAppSecret = "abcdef1234567890abcdef1234567890" // TODO: 从配置加载

	// SM2 私钥（对齐 Java @Value("${crypto.sm2.private-key}")，来自 Nacos cr-auth-service.yaml）
	// 注意：Java BigInteger 可能有 "00" 前缀，SM2Decrypt 会自动去除
	sm2PrivateKey = "b3b8e61213bbd5e7d001e0cd4e33015efc04ae68ae61f2a36da55d92903cb0eb"

	// 订阅模板ID（对齐 Java AuthServiceImpl.SUBSCRIBE_TEMPLATE_ID）
	subscribeTemplateID = "XbZ4Xnj0DMzfhCtihmBJcaQGoos010q87Xz20l7aevg"

	// 平台标识
	platformWeixin = "WEIXIN"
)

// ============================================================
// VO 定义（对齐 Java VO 类）
// ============================================================

// LoginVO 登录返回（对齐 Java LoginVO）
//
// 字段命名与 Java 后端 LoginVO 保持一致：
//   - accessToken / refreshToken / openId / user
//   - 前端（小程序+admin）依赖 accessToken 字段名
type LoginVO struct {
	AccessToken  string      `json:"accessToken"`  // Access Token
	RefreshToken string      `json:"refreshToken"` // Refresh Token
	OpenID       string      `json:"openId"`       // 微信 openId
	User         interface{} `json:"user"`         // 用户信息（登录后返回，免密登录可能为 null）
}

// UserVO 用户视图对象（对齐 Java UserVO<RoleBaseEntity>）
//
// 前端类型定义（src/types/user.d.ts）：
//   - UserResponse { userId, roleId, createTimeStr, updateTimeStr, identityInfo, admin }
//
// 用于登录响应的 user 字段，包含身份信息和管理员信息
type UserVO struct {
	UserID        int64       `json:"userId"`        // 用户ID（对应 c_user.id）
	RoleID        int64       `json:"roleId"`        // 角色ID（3=家长, 4=教师）
	IdentityInfo  interface{} `json:"identityInfo"`  // 身份信息（*ParentIdentityVO 或 *TeacherIdentityVO）
	Admin         interface{} `json:"admin"`         // 管理员信息（*AdminVO 或 nil）
	CreateTimeStr string      `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string      `json:"updateTimeStr"` // 更新时间字符串
}

// ParentIdentityVO 家长身份信息（对齐 Java Parent 实体的 JSON 输出）
//
// 前端类型定义（src/types/parent.d.ts）：
//   - ParentIdentity { userId, isAvailable, username, parentId }
type ParentIdentityVO struct {
	UserID      int64  `json:"userId"`      // 关联 c_user.id
	IsAvailable bool   `json:"isAvailable"` // 是否可用
	Username    string `json:"username"`    // 用户名
	ParentID    int64  `json:"parentId"`    // 家长ID（c_parent.id）
}

// TeacherIdentityVO 教师身份信息（对齐 Java Teacher 实体的 JSON 输出）
//
// 前端类型定义（src/types/teacher.d.ts）：
//   - TeacherIdentity { userId, isAvailable, username, institutionId, teacherId, phone?, isInstitutionAdmin }
type TeacherIdentityVO struct {
	UserID             int64  `json:"userId"`             // 关联 c_user.id
	IsAvailable        bool   `json:"isAvailable"`        // 是否可用
	Username           string `json:"username"`           // 用户名
	InstitutionID      int64  `json:"institutionId"`      // 机构ID
	TeacherID          int64  `json:"teacherId"`          // 教师ID（c_teacher.id）
	Phone              string `json:"phone"`              // 手机号
	IsInstitutionAdmin bool   `json:"isInstitutionAdmin"` // 是否机构管理员
}

// AdminVO 管理员信息（对齐 Java AdminVO）
//
// 前端类型定义（src/types/admin.d.ts）：
//   - AdminResponse { adminId, userId, username, isAvailable, createTimeStr, updateTimeStr }
type AdminVO struct {
	AdminID       int64  `json:"adminId"`       // 管理员ID（c_admin.id）
	UserID        int64  `json:"userId"`        // 关联 c_user.id
	IsAvailable   bool   `json:"isAvailable"`   // 是否可用
	Username      string `json:"username"`      // 用户名
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
}

// RegisterVO 注册返回（对齐 Java RegisterVO）
type RegisterVO struct {
	UserID  int64  `json:"userId"`  // 用户ID
	Message string `json:"message"` // 消息
}

// ============================================================
// AuthService 认证服务
// ============================================================

// AuthService 认证服务（对齐 Java AuthServiceImpl）
//
// 注入：UserMapper, UserAuthMapper, UserPlatformMapper, ParentMapper,
//
//	TeacherMapper, AdminMapper, InstitutionMapper, StudentMapper,
//	ParentStudentMapper, WxSubscribeRecordMapper, WxStudentSubscribeMapper,
//	JwtUtils, RedisClient
type AuthService struct {
	userMapper               *mapper.UserMapper
	userAuthMapper           *mapper.UserAuthMapper
	userPlatformMapper       *mapper.UserPlatformMapper
	parentMapper             *mapper.ParentMapper
	teacherMapper            *mapper.TeacherMapper
	adminMapper              *mapper.AdminMapper
	institutionMapper        *mapper.InstitutionMapper
	studentMapper            *mapper.StudentMapper
	parentStudentMapper      *mapper.ParentStudentMapper
	wxSubscribeRecordMapper  *mapper.WxSubscribeRecordMapper
	wxStudentSubscribeMapper *mapper.WxStudentSubscribeMapper
	jwtUtils                 *jwt.Utils
	redisClient              *rediscli.Client

	// 微信 access_token 缓存（对齐 Java WeChatApiService 的缓存机制）
	accessTokenMu       sync.Mutex // 保护并发刷新
	cachedAccessToken   string     // 缓存的 access_token
	accessTokenExpireAt int64      // token 过期时间戳（Unix 毫秒）
}

// NewAuthService 创建 AuthService
//
// 参数：
//   - userMapper: 用户表 Mapper
//   - userAuthMapper: 用户认证表 Mapper
//   - userPlatformMapper: 用户平台表 Mapper
//   - parentMapper: 家长表 Mapper（用于查询家长身份信息）
//   - teacherMapper: 教师表 Mapper（用于查询教师身份信息）
//   - adminMapper: 管理员表 Mapper（用于查询管理员信息）
//   - institutionMapper: 机构表 Mapper
//   - studentMapper: 学生表 Mapper
//   - parentStudentMapper: 家长-学生关联表 Mapper
//   - wxSubscribeRecordMapper: 微信订阅记录 Mapper
//   - wxStudentSubscribeMapper: 学生订阅关系 Mapper
//   - jwtUtils: JWT 工具
//   - redisClient: Redis 客户端
func NewAuthService(
	userMapper *mapper.UserMapper,
	userAuthMapper *mapper.UserAuthMapper,
	userPlatformMapper *mapper.UserPlatformMapper,
	parentMapper *mapper.ParentMapper,
	teacherMapper *mapper.TeacherMapper,
	adminMapper *mapper.AdminMapper,
	institutionMapper *mapper.InstitutionMapper,
	studentMapper *mapper.StudentMapper,
	parentStudentMapper *mapper.ParentStudentMapper,
	wxSubscribeRecordMapper *mapper.WxSubscribeRecordMapper,
	wxStudentSubscribeMapper *mapper.WxStudentSubscribeMapper,
	jwtUtils *jwt.Utils,
	redisClient *rediscli.Client,
) *AuthService {
	return &AuthService{
		userMapper:               userMapper,
		userAuthMapper:           userAuthMapper,
		userPlatformMapper:       userPlatformMapper,
		parentMapper:             parentMapper,
		teacherMapper:            teacherMapper,
		adminMapper:              adminMapper,
		institutionMapper:        institutionMapper,
		studentMapper:            studentMapper,
		parentStudentMapper:      parentStudentMapper,
		wxSubscribeRecordMapper:  wxSubscribeRecordMapper,
		wxStudentSubscribeMapper: wxStudentSubscribeMapper,
		jwtUtils:                 jwtUtils,
		redisClient:              redisClient,
	}
}

// ============================================================
// 微信登录相关
// ============================================================

// GetOpenId 调用微信 jscode2session 获取 openId
//
// 对齐 Java AuthServiceImpl.getOpenId(String code)
//
// 接口：GET https://api.weixin.qq.com/sns/jscode2session
//
//	?appid=APPID&secret=SECRET&js_code=JSCODE&grant_type=authorization_code
//
// 参数：
//   - code: 微信登录凭证（前端 wx.login 获取）
//
// 返回：
//   - openId: 微信用户唯一标识
//   - 错误信息
func (s *AuthService) GetOpenId(code string) (string, error) {
	if code == "" {
		return "", errors.New("code 不能为空")
	}

	// 构造微信接口 URL
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		wxAppID, wxAppSecret, code,
	)

	// 调用微信接口（对齐 Java RestTemplate.getForObject）
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("调用微信接口失败: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("关闭微信响应体失败: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取微信响应失败: %w", err)
	}

	// 解析响应
	var result struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析微信响应失败: %w", err)
	}

	// 错误处理
	if result.ErrCode != 0 {
		return "", fmt.Errorf("微信接口错误: code=%d, msg=%s", result.ErrCode, result.ErrMsg)
	}

	if result.OpenID == "" {
		return "", errors.New("微信返回 openId 为空")
	}

	return result.OpenID, nil
}

// WxLogin 微信免密登录
//
// 对齐 Java AuthServiceImpl.wxLogin(LoginDTO)
//
// 流程：
//  1. 通过 code 获取 openId
//  2. 查找或创建用户平台记录
//  3. 签发双 Token
//
// 参数：
//   - code: 微信登录凭证
//
// 返回：LoginVO
func (s *AuthService) WxLogin(code string) *response.ResponseDTO {
	// 1. 获取 openId
	openId, err := s.GetOpenId(code)
	if err != nil {
		return response.Fail("获取 openId 失败: " + err.Error())
	}

	// 2. 查找用户平台记录（对齐 Java userService.saveOrUpdateUser）
	platform, err := s.userPlatformMapper.SelectByOpenIdAndPlatform(openId, platformWeixin)
	if err != nil {
		log.Printf("查询用户平台记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	var userID int64
	var roleID int64
	if platform != nil {
		// 已有记录，复用 userID
		userID = platform.UserID.Int64
		roleID = platform.LastLoginRole.Int64
		// 更新最后登录时间
		_ = s.userPlatformMapper.UpdateLastLogin(platform.ID, roleID)
	} else {
		// 首次登录，无角色信息（需后续绑定或注册）
		log.Printf("首次微信登录，openId=%s，无用户记录", openId)
	}

	// 3. 签发双 Token（对齐 Java jwtUtils.createAccessToken/createRefreshToken）
	accessToken, err := s.jwtUtils.CreateAccessToken(userID, roleID, openId)
	if err != nil {
		log.Printf("生成 Access Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	refreshToken, err := s.jwtUtils.CreateRefreshToken(userID, roleID, openId)
	if err != nil {
		log.Printf("生成 Refresh Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	return response.Success(&LoginVO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		OpenID:       openId,
		User:         nil, // 免密登录不返回用户信息
	})
}

// ============================================================
// 账号密码登录
// ============================================================

// LoginByPwdRequest 密码登录请求（对齐 Java LoginDTO.LoginByPwd 分组）
type LoginByPwdRequest struct {
	Account       string `json:"account"`       // 账号（手机号或用户名）
	Password      string `json:"password"`      // 密码（SM2 加密）
	Role          int64  `json:"role"`          // 角色（3=parent, 4=teacher）
	InstitutionID int64  `json:"institutionId"` // 机构ID
	OpenID        string `json:"openId"`        // 微信 openId（可选，用于绑定平台）
	Platform      string `json:"platform"`      // 平台标识（默认 WEIXIN）
}

// LoginByPwd 账号密码登录
//
// 对齐 Java AuthServiceImpl.loginByPwd(LoginDTO)
//
// 流程：
//  1. 校验机构ID非空，查机构，校验未过期
//  2. SM2 解密密码
//  3. 按 (account + role + institutionId) 查认证记录
//  4. SM3 加盐验签
//  5. 查/建用户平台记录，更新登录信息
//  6. 签发双 Token
//
// 参数：
//   - req: 登录请求
//
// 返回：LoginVO
func (s *AuthService) LoginByPwd(req *LoginByPwdRequest) *response.ResponseDTO {
	// 1. 校验机构ID
	if req.InstitutionID == 0 {
		return response.Fail("机构ID不能为空")
	}

	// 查询机构（对齐 Java institutionMapper.selectById）
	institution, err := s.institutionMapper.SelectByID(req.InstitutionID)
	if err != nil {
		log.Printf("查询机构失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if institution == nil {
		return response.Fail("机构不存在")
	}

	// 校验机构过期时间（对齐 Java 检查 expireTime）
	if institution.ExpireTime.Valid {
		// 兼容 iOS 时间格式（对齐 Java replace("-", "/")）
		expireTime := institution.ExpireTime.Time
		if time.Now().After(expireTime) {
			return response.Fail("该机构使用期限已到期")
		}
	}

	// 2. SM2 解密密码（对齐 Java SM2Util.decrypt）
	rawPassword, err := crypto.SM2Decrypt(req.Password, sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密失败: %v", err)
		return response.Fail("密码解密失败，请检查加密参数")
	}

	// 3. 查询认证记录（对齐 Java userAuthMapper.selectAuthByAccountAndInstitution）
	auth, err := s.userAuthMapper.SelectAuthByAccountAndInstitution(req.Account, req.Role, req.InstitutionID)
	if err != nil {
		log.Printf("查询认证记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if auth == nil {
		return response.Fail("账号不存在")
	}

	// 4. SM3 加盐验签（对齐 Java SM3Util.digestWithSalt 比对）
	hashedPassword := crypto.SM3DigestWithSalt(rawPassword, auth.Salt.String)
	if !strings.EqualFold(hashedPassword, auth.Password.String) {
		return response.Fail("密码错误")
	}

	// 5. 更新最后登录时间
	_ = s.userAuthMapper.UpdateLastLoginTime(auth.ID)

	// 6. 查/建用户平台记录（对齐 Java 查/建 UserPlatform）
	userID := auth.UserID.Int64
	platform := req.Platform
	if platform == "" {
		platform = platformWeixin
	}

	var userPlatform *entity.UserPlatform
	if req.OpenID != "" {
		// 有 openId 时，查找或创建平台记录
		userPlatform, err = s.userPlatformMapper.SelectByUserIDAndPlatform(userID, platform)
		if err != nil {
			log.Printf("查询用户平台记录失败: %v", err)
		}

		if userPlatform != nil {
			// 更新登录信息
			_ = s.userPlatformMapper.UpdateLastLogin(userPlatform.ID, req.Role)
		} else {
			// 新建平台记录
			newPlatform := &entity.UserPlatform{
				UserID:        sql.NullInt64{Int64: userID, Valid: true},
				OpenID:        sql.NullString{String: req.OpenID, Valid: true},
				Platform:      sql.NullString{String: platform, Valid: true},
				LastLoginRole: sql.NullInt64{Int64: req.Role, Valid: true},
			}
			_, err = s.userPlatformMapper.Insert(newPlatform)
			if err != nil {
				log.Printf("新增用户平台记录失败: %v", err)
			}
		}
	}

	// 7. 签发双 Token
	accessToken, err := s.jwtUtils.CreateAccessToken(userID, req.Role, req.OpenID)
	if err != nil {
		log.Printf("生成 Access Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	refreshToken, err := s.jwtUtils.CreateRefreshToken(userID, req.Role, req.OpenID)
	if err != nil {
		log.Printf("生成 Refresh Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	// 8. 查询完整用户信息返回（对齐 Java getFullUserInfoByUserId）
	// 构造包含 identityInfo/admin 的 UserVO，而非仅 c_user 表的 entity.User
	// 前端依赖 userId/roleId/identityInfo/admin/createTimeStr/updateTimeStr 字段
	userVO := s.GetFullUserInfo(userID, req.Role)
	if userVO == nil {
		return response.Fail("用户账户异常")
	}

	return response.Success(&LoginVO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		OpenID:       req.OpenID,
		User:         userVO,
	})
}

// ============================================================
// Token 管理
// ============================================================

// LoginByToken Token 续登
//
// 对齐 Java AuthServiceImpl.loginByToken(LoginDTO)
//
// 流程：
//  1. 解析 Token 取 userId/roleId
//  2. 查询用户信息
//  3. 重新签发双 Token
//  4. 更新登录时间
//
// 参数：
//   - token: 旧 Access Token
//
// 返回：LoginVO
func (s *AuthService) LoginByToken(token string) *response.ResponseDTO {
	// 1. 解析 Token（对齐 Java jwtUtils.parseClaims）
	claims, err := s.jwtUtils.ParseClaims(token)
	if err != nil {
		return response.FailWithCode(response.CodeUnauthorized, "Token 无效或已过期")
	}

	userID := claims.UserID
	roleID := claims.RoleID
	openId := claims.OpenID

	// 2. 查询完整用户信息（对齐 Java getFullUserInfoByUserId）
	// 构造包含 identityInfo/admin 的 UserVO，而非仅 c_user 表的 entity.User
	userVO := s.GetFullUserInfo(userID, roleID)
	if userVO == nil {
		return response.Fail("用户不存在")
	}

	// 3. 更新登录时间
	auth, _ := s.userAuthMapper.SelectByUserID(userID)
	if auth != nil {
		_ = s.userAuthMapper.UpdateLastLoginTime(auth.ID)
	}

	// 4. 重新签发双 Token
	accessToken, err := s.jwtUtils.CreateAccessToken(userID, roleID, openId)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	refreshToken, err := s.jwtUtils.CreateRefreshToken(userID, roleID, openId)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	return response.Success(&LoginVO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		OpenID:       openId,
		User:         userVO,
	})
}

// RefreshAccessToken 刷新 Access Token
//
// 对齐 Java AuthServiceImpl.refreshAccessToken(String refreshToken)
//
// 流程：
//  1. 检查 refresh token 是否在黑名单
//  2. 解析 refresh token
//  3. 只签发新 Access Token（不刷新 refresh token）
//
// 参数：
//   - refreshToken: Refresh Token
//
// 返回：LoginVO（仅含新 Access Token）
func (s *AuthService) RefreshAccessToken(refreshToken string) *response.ResponseDTO {
	// 1. 检查黑名单（对齐 Java tokenBlacklistService.isRefreshTokenBlacklisted）
	isBlacklisted, err := redis.IsBlacklisted(s.redisClient, refreshToken)
	if err != nil {
		log.Printf("查询黑名单失败: %v", err)
		// 降级处理：查询失败时拒绝刷新，保证安全
		return response.FailWithCode(response.CodeUnauthorized, "Token 状态校验失败")
	}
	if isBlacklisted {
		return response.FailWithCode(response.CodeUnauthorized, "Token 已失效，请重新登录")
	}

	// 2. 解析 refresh token（对齐 Java jwtUtils.validateRefreshToken）
	claims, err := s.jwtUtils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return response.FailWithCode(response.CodeUnauthorized, "Refresh Token 无效或已过期")
	}

	// 3. 签发新 Access Token（对齐 Java 只签发 access，不刷新 refresh）
	// Java 行为：new LoginVO(newAccessToken, null, openId, null)
	// refreshToken 返回空字符串（对齐 Java null），前端不更新本地 refreshToken
	accessToken, err := s.jwtUtils.CreateAccessToken(claims.UserID, claims.RoleID, claims.OpenID)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	return response.Success(&LoginVO{
		AccessToken:  accessToken,
		RefreshToken: "", // 对齐 Java null 行为，不返回 refreshToken
		OpenID:       claims.OpenID,
		User:         nil,
	})
}

// Logout 登出
//
// 对齐 Java AuthServiceImpl.logout(LoginDTO)
//
// 流程：
//  1. Access Token 加入黑名单
//  2. Refresh Token 加入黑名单
//  3. 清除用户缓存
//
// 参数：
//   - accessToken: Access Token
//   - refreshToken: Refresh Token
//
// 返回：操作结果
func (s *AuthService) Logout(accessToken, refreshToken string) *response.ResponseDTO {
	// Access Token 加入黑名单（对齐 Java tokenBlacklistService.addToBlacklist）
	if accessToken != "" {
		// 计算 Access Token 剩余有效期作为黑名单 TTL
		// 对齐 Java：TTL = token 剩余有效期
		ttl := 5 * time.Minute // 默认 5 分钟（Access Token 过期时间）
		if claims, err := s.jwtUtils.ParseClaims(accessToken); err == nil {
			if claims.ExpiresAt != nil {
				remaining := time.Until(claims.ExpiresAt.Time)
				if remaining > 0 {
					ttl = remaining
				}
			}
		}
		_ = redis.AddToBlacklist(s.redisClient, accessToken, ttl)
	}

	// Refresh Token 加入黑名单
	if refreshToken != "" {
		ttl := 7 * 24 * time.Hour // 默认 7 天（Refresh Token 过期时间）
		if claims, err := s.jwtUtils.ValidateRefreshToken(refreshToken); err == nil {
			if claims.ExpiresAt != nil {
				remaining := time.Until(claims.ExpiresAt.Time)
				if remaining > 0 {
					ttl = remaining
				}
			}
		}
		_ = redis.AddToBlacklist(s.redisClient, refreshToken, ttl)
	}

	return response.Success("登出成功")
}

// ============================================================
// 注册
// ============================================================

// RegisterRequest 注册请求（对齐 Java RegisterDTO）
type RegisterRequest struct {
	Account       string `json:"account"`       // 账号（手机号）
	Password      string `json:"password"`      // 密码（SM2 加密）
	Role          int64  `json:"role"`          // 角色（3=parent, 4=teacher）
	InstitutionID int64  `json:"institutionId"` // 机构ID
	OpenID        string `json:"openId"`        // 微信 openId（可选）
	Platform      string `json:"platform"`      // 平台标识
}

// Register 注册
//
// 对齐 Java AuthServiceImpl.register(RegisterDTO)
//
// 流程：
//  1. 查重（userId+role 已存在 / account+role+institution 已存在）
//  2. SM2 解密密码
//  3. 生成盐值，SM3 加盐哈希
//  4. 创建 User + UserAuth
//
// 参数：
//   - req: 注册请求
//
// 返回：RegisterVO
func (s *AuthService) Register(req *RegisterRequest) *response.ResponseDTO {
	// 1. SM2 解密密码（对齐 Java SM2Util.decrypt）
	rawPassword, err := crypto.SM2Decrypt(req.Password, sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密失败: %v", err)
		return response.Fail("密码解密失败")
	}

	// 2. 创建 User 记录（对齐 Java userService.saveOrUpdateUser）
	user := &entity.User{
		InstitutionID: sql.NullInt64{Int64: req.InstitutionID, Valid: req.InstitutionID > 0},
	}
	userID, err := s.userMapper.Insert(user)
	if err != nil {
		log.Printf("创建用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "创建用户失败")
	}

	// 3. 查重：account + role + institution（对齐 Java existsByInstitutionAndAccountAndRole）
	exists, err := s.userAuthMapper.ExistsByInstitutionAndAccountAndRole(req.InstitutionID, req.Account, req.Role)
	if err != nil {
		log.Printf("查重失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if exists {
		return response.Fail("该账号已存在")
	}

	// 查重：userId + role（对齐 Java 已有认证记录检查）
	exists, err = s.userAuthMapper.ExistsByUserIDAndRole(userID, req.Role)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if exists {
		return response.Fail("该用户已注册此角色")
	}

	// 4. 生成盐值 + SM3 加盐哈希（对齐 Java salt = UUID去横杠, hashed = SM3Util.digestWithSalt）
	salt := crypto.GenerateSalt()
	hashedPassword := crypto.SM3DigestWithSalt(rawPassword, salt)

	// 5. 创建 UserAuth 记录
	auth := &entity.UserAuth{
		UserID:   sql.NullInt64{Int64: userID, Valid: true},
		RoleID:   sql.NullInt64{Int64: req.Role, Valid: true},
		Account:  sql.NullString{String: req.Account, Valid: true},
		Password: sql.NullString{String: hashedPassword, Valid: true},
		Salt:     sql.NullString{String: salt, Valid: true},
	}
	_, err = s.userAuthMapper.Insert(auth)
	if err != nil {
		log.Printf("创建认证记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "创建认证记录失败")
	}

	// 6. 创建平台记录（如果有 openId）
	if req.OpenID != "" {
		platform := req.Platform
		if platform == "" {
			platform = platformWeixin
		}
		newPlatform := &entity.UserPlatform{
			UserID:        sql.NullInt64{Int64: userID, Valid: true},
			OpenID:        sql.NullString{String: req.OpenID, Valid: true},
			Platform:      sql.NullString{String: platform, Valid: true},
			LastLoginRole: sql.NullInt64{Int64: req.Role, Valid: true},
		}
		_, _ = s.userPlatformMapper.Insert(newPlatform)
	}

	// 7. 如果是家长角色，创建 Parent 记录（对齐 Java saveIdentityRecord）
	if req.Role == context.RoleParent {
		parent := &entity.Parent{
			RoleBaseEntity: entity.RoleBaseEntity{
				UserID:      sql.NullInt64{Int64: userID, Valid: true},
				IsAvailable: sql.NullBool{Bool: true, Valid: true},
			},
			Phone:   sql.NullString{String: req.Account, Valid: req.Account != ""},
			IsBound: sql.NullBool{Bool: false, Valid: true}, // 占位记录，待绑定
		}
		_, _ = s.parentMapper.Insert(parent)
	}

	return response.Success(&RegisterVO{
		UserID:  userID,
		Message: "注册成功",
	})
}

// ============================================================
// 用户信息查询
// ============================================================

// GetFullUserInfo 获取完整用户信息（对齐 Java UserServiceImpl.getFullUserInfo）
//
// 根据 roleID 查询对应的身份表（c_parent/c_teacher），并查询 c_admin 表
// 构造完整的 UserVO 对象返回给前端
//
// 参数：
//   - userID: 用户ID（c_user.id）
//   - roleID: 角色ID（3=家长, 4=教师）
//
// 返回：*UserVO 或 nil（用户不存在）
func (s *AuthService) GetFullUserInfo(userID int64, roleID int64) *UserVO {
	// 1. 查询 c_user 表获取用户基本信息
	user, err := s.userMapper.SelectByID(userID)
	if err != nil || user == nil {
		return nil
	}

	vo := &UserVO{
		UserID: userID,
		RoleID: roleID,
	}

	// 格式化时间（对齐 Java @BaseDateTimeToString，使用 entity.FormatTime 统一处理 sql.NullTime）
	vo.CreateTimeStr = entity.FormatTime(user.CreateTime)
	vo.UpdateTimeStr = entity.FormatTime(user.UpdateTime)

	// 2. 根据 roleID 查询身份信息（对齐 Java IdentityServiceImpl.getByUserId）
	switch roleID {
	case context.RoleParent: // 家长（roleID=3）
		parent, err := s.parentMapper.SelectByUserID(userID)
		if err == nil && parent != nil {
			vo.IdentityInfo = &ParentIdentityVO{
				UserID:      userID,
				IsAvailable: parent.IsAvailable.Bool,
				Username:    parent.Username.String,
				ParentID:    parent.ParentID.Int64,
			}
		}
	case context.RoleTeacher: // 教师（roleID=4）
		teacher, err := s.teacherMapper.SelectByUserID(userID)
		if err == nil && teacher != nil {
			vo.IdentityInfo = &TeacherIdentityVO{
				UserID:             userID,
				IsAvailable:        teacher.IsAvailable.Bool,
				Username:           teacher.Username.String,
				InstitutionID:      teacher.InstitutionID.Int64,
				TeacherID:          teacher.TeacherID.Int64,
				Phone:              teacher.Phone.String,
				IsInstitutionAdmin: teacher.IsInstitutionAdmin.Bool,
			}
		}
	}

	// 3. 查询 c_admin 表（所有角色都查，非管理员返回 nil）
	// 对齐 Java adminService.getOne(eq("user_id", userId))
	admin, err := s.adminMapper.SelectByUserID(userID)
	if err == nil && admin != nil {
		vo.Admin = &AdminVO{
			AdminID:       admin.ID.Int64,
			UserID:        userID,
			IsAvailable:   admin.IsAvailable.Bool,
			Username:      admin.Username.String,
			CreateTimeStr: entity.FormatTime(admin.CreateTime),
			UpdateTimeStr: entity.FormatTime(admin.UpdateTime),
		}
	} else {
		vo.Admin = nil
	}

	return vo
}

// GetUserAuthByTeacherID 按教师ID查认证信息
//
// 对齐 Java AuthServiceImpl.getUserAuthByTeacherId
//
// 前端类型定义（src/types/auth.d.ts）：
//
//	interface GetUserAuthInfoByTeacherIdResponse {
//	    account: string  // 仅返回账号字段
//	}
//
// 参数：
//   - teacherID: 教师ID
//
// 返回：{ account: string }
func (s *AuthService) GetUserAuthByTeacherID(teacherID int64) *response.ResponseDTO {
	auth, err := s.userAuthMapper.SelectAuthByTeacherId(teacherID)
	if err != nil {
		log.Printf("查询教师认证信息失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if auth == nil {
		return response.Fail("教师认证信息不存在")
	}

	// 仅返回前端预期的 account 字段（隐藏密码、盐值等敏感信息）
	return response.Success(map[string]interface{}{
		"account": auth.Account.String,
	})
}

// GetUserInfoByUserID 按用户ID查用户信息
func (s *AuthService) GetUserInfoByUserID(userID int64) *response.ResponseDTO {
	user, err := s.userMapper.SelectByID(userID)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}
	return response.Success(user)
}

// ============================================================
// 微信订阅相关（对齐 Java recordSubscribe / getSubscribeStatus）
// ============================================================

// RecordSubscribeRequest 记录订阅授权请求（对齐 Java RecordSubscribeDTO）
type RecordSubscribeRequest struct {
	Code        string `json:"code"`        // 微信登录凭证（用于获取 openId）
	TemplateID  string `json:"templateId"`  // 模板ID
	IsPermanent bool   `json:"isPermanent"` // 是否永久订阅
}

// RecordSubscribe 记录订阅授权
//
// 对齐 Java AuthServiceImpl.recordSubscribe(RecordSubscribeDTO)
//
// 流程：
//  1. 通过 code 获取 openId
//  2. 按 openId + lastLoginRole=3 查用户平台记录
//  3. 查/建 WxSubscribeRecord，count+1
//
// 参数：
//   - req: 订阅请求
//
// 返回：操作结果
func (s *AuthService) RecordSubscribe(req *RecordSubscribeRequest) *response.ResponseDTO {
	// 1. 获取 openId
	openId, err := s.GetOpenId(req.Code)
	if err != nil {
		return response.Fail("获取 openId 失败: " + err.Error())
	}

	// 2. 按 openId + 家长角色查平台记录（对齐 Java lastLoginRole=3 过滤）
	platform, err := s.userPlatformMapper.SelectByOpenIdPlatformAndRole(openId, platformWeixin, context.RoleParent)
	if err != nil {
		log.Printf("查询平台记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if platform == nil {
		return response.Fail("未找到家长账号，请先登录")
	}

	// 3. 查/建订阅记录（对齐 Java 查 WxSubscribeRecord）
	templateID := req.TemplateID
	if templateID == "" {
		templateID = subscribeTemplateID // 默认扣课通知模板
	}

	record, err := s.wxSubscribeRecordMapper.SelectByOpenIdAndTemplate(openId, templateID)
	if err != nil {
		log.Printf("查询订阅记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	if record != nil {
		// 已有记录，count+1（对齐 Java count+1 逻辑）
		err = s.wxSubscribeRecordMapper.IncrementCount(record.ID.Int64, req.IsPermanent)
	} else {
		// 新建记录
		newRecord := &entity.WxSubscribeRecord{
			OpenID:         sql.NullString{String: openId, Valid: true},
			TemplateID:     sql.NullString{String: templateID, Valid: true},
			SubscribeCount: sql.NullInt64{Int64: 1, Valid: true},
			IsPermanent:    sql.NullBool{Bool: req.IsPermanent, Valid: true},
		}
		_, err = s.wxSubscribeRecordMapper.Insert(newRecord)
	}

	if err != nil {
		log.Printf("记录订阅失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "记录订阅失败")
	}

	return response.Success("订阅记录成功")
}

// GetSubscribeStatus 查询订阅状态
//
// 对齐 Java AuthServiceImpl.getSubscribeStatus
//
// 参数：
//   - code: 微信登录凭证
//   - templateID: 模板ID
//   - studentID: 学生ID（可选，用于查询学生级订阅状态）
//
// 返回：订阅状态
func (s *AuthService) GetSubscribeStatus(code, templateID string, studentID int64) *response.ResponseDTO {
	// 1. 获取 openId
	openId, err := s.GetOpenId(code)
	if err != nil {
		return response.Fail("获取 openId 失败: " + err.Error())
	}

	if templateID == "" {
		templateID = subscribeTemplateID
	}

	// 2. 查询订阅记录（对齐 Java 查 WxSubscribeRecord）
	record, err := s.wxSubscribeRecordMapper.SelectByOpenIdAndTemplate(openId, templateID)
	if err != nil {
		log.Printf("查询订阅记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 3. 构造响应
	result := map[string]interface{}{
		"subscribed":       false,
		"subscribeCount":   0,
		"isPermanent":      false,
		"wechatSubscribed": false,
	}

	if record != nil {
		result["subscribed"] = record.SubscribeCount.Int64 > 0 || record.IsPermanent.Bool
		result["subscribeCount"] = record.SubscribeCount.Int64
		result["isPermanent"] = record.IsPermanent.Bool
	}

	// 4. 查询学生级订阅状态（对齐 Java 查 WxStudentSubscribe）
	if studentID > 0 {
		count, err := s.wxStudentSubscribeMapper.CountByOpenIdAndStudent(openId, studentID)
		if err == nil && count > 0 {
			result["wechatSubscribed"] = true
		}
	}

	return response.Success(result)
}

// ============================================================
// 家长绑定与订阅流程（对齐 Java AuthServiceImpl 绑定相关方法）
// ============================================================

// 绑定流程 Redis key 前缀和 TTL（对齐 Java BindTokenCache 的 Redis 键设计）
const (
	// bindCodePrefix 绑定码 Redis key 前缀（key: bind:code:{code} → token 字符串）
	bindCodePrefix = "bind:code:"
	// bindTokenPrefix 绑定 token Redis key 前缀（key: bind:token:{token} → BindTokenInfo JSON）
	bindTokenPrefix = "bind:token:"
	// bindCodeTTL 绑定码/token 有效期（10 分钟，对齐任务要求）
	bindCodeTTL = 10 * time.Minute
	// bindCodeChars 6位绑定码字符集（排除易混淆字符 0/O, 1/I/L，对齐 Java generateBindCode）
	bindCodeChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	// bindCodeLength 绑定码长度
	bindCodeLength = 6
	// bindTokenLength 绑定 token 长度（8 字符 hex，对齐 Java UUID 前 8 位）
	bindTokenLength = 16
	// wxEnvVersion 小程序码版本（release=正式版, trial=体验版, develop=开发版）
	wxEnvVersion = "release"
)

// BindTokenInfo 绑定 token 缓存信息（对齐 Java BindTokenCache.BindTokenInfo）
//
// 存储在 Redis 中，包含学生ID、关系、是否主联系人、绑定码、订阅模式等完整绑定信息
// 通过 bind:token:{token} 键存储为 JSON，通过 bind:code:{code} → token 间接查找
type BindTokenInfo struct {
	StudentID    int64  `json:"studentId"`    // 学生ID
	Relation     string `json:"relation"`     // 与学生的关系（如"父亲"、"母亲"）
	IsPrimary    bool   `json:"isPrimary"`    // 是否主要联系人
	BindCode     string `json:"bindCode"`     // 6 位绑定码（用于家长手动输入）
	SubscribeOnly bool  `json:"subscribeOnly"` // 是否为订阅专用模式（true=家长端只能订阅不能绑定账号）
}

// BindQrcodeResponse 绑定二维码响应（对齐 Java BindQrcodeResponse）
//
// 返回给前端的二维码信息，包含二维码 base64 图片、绑定 token 和 6 位绑定码
type BindQrcodeResponse struct {
	Qrcode   string `json:"qrcode"`   // 二维码图片 base64（含 data:image/png;base64 前缀）
	Token    string `json:"token"`    // 绑定 token（存入 Redis 的 key，二维码场景参数）
	BindCode string `json:"bindCode"` // 6 位绑定码（用于家长手动输入）
}

// BindInfoResponse 绑定信息响应（对齐 Java BindInfoResponse）
//
// 家长扫码或输入绑定码后查询到的学生信息，包含关系、主联系人标识和占位家长信息
type BindInfoResponse struct {
	StudentID       int64  `json:"studentId"`                 // 学生ID
	StudentName     string `json:"studentName"`               // 学生姓名
	Sex             int64  `json:"sex"`                       // 性别（1=男, 2=女）
	InstitutionName string `json:"institutionName"`           // 机构名称
	Relation        string `json:"relation"`                  // 与学生的关系（如"父亲"、"母亲"）
	IsPrimary       bool   `json:"isPrimary"`                 // 是否为主要联系人
	SubscribeOnly   bool   `json:"subscribeOnly"`             // 是否为订阅专用模式（true=家长端只能订阅不能绑定账号）
	ParentName      string `json:"parentName,omitempty"`      // 教师/管理端预填的家长名称（占位家长）
	ParentPhone     string `json:"parentPhone,omitempty"`     // 教师/管理端预填的家长手机号（占位家长）
}

// BindStatusResponse 绑定状态响应（对齐 Java BindStatusResponse）
//
// 检查绑定状态，返回是否已绑定该学生和是否已有家长账号
type BindStatusResponse struct {
	AlreadyBound bool `json:"alreadyBound"` // 是否已绑定该学生
	HasAccount   bool `json:"hasAccount"`   // 是否已有家长账号（true=无需再设置账号密码）
}

// BindResultResponse 绑定结果响应（包含 LoginVO）
//
// bind_by_code 接口返回，包含绑定结果和登录凭证
type BindResultResponse struct {
	Message string  `json:"message"` // 绑定结果消息
	Login   *LoginVO `json:"login"`   // 登录凭证（含 token）
}

// ---- 随机码生成辅助 ----

// generateBindCode 生成 6 位绑定码（大写字母+数字，排除易混淆字符）
//
// 对齐 Java BindTokenCache.generateBindCode
// 字符集：ABCDEFGHJKMNPQRSTUVWXYZ23456789（排除 0/O, 1/I/L）
func generateBindCode() string {
	chars := []byte(bindCodeChars)
	result := make([]byte, bindCodeLength)
	// crypto/rand.Read 生成随机字节
	_, _ = rand.Read(result)
	for i := range result {
		result[i] = chars[int(result[i])%len(chars)]
	}
	return string(result)
}

// generateBindToken 生成绑定 token（16 字符 hex，对齐 Java UUID 前 8 位去横杠）
//
// 使用 crypto/rand 生成 8 字节随机数，转为 16 字符 hex 字符串
func generateBindToken() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ---- Redis 绑定码/token 存取辅助（对齐 Java BindTokenCache） ----

// setBindInfo 存储绑定信息到 Redis（token → BindTokenInfo JSON，code → token 字符串）
//
// 对齐 Java BindTokenCache.put 方法
// 使用统一的 key 前缀（不再区分绑定/订阅模式，subscribeOnly 标志存储在 JSON 中）
//
// 参数：
//   - token: 绑定 token
//   - info: 绑定信息（含 studentID, relation, isPrimary, bindCode, subscribeOnly）
func (s *AuthService) setBindInfo(token string, info *BindTokenInfo) error {
	// 序列化 BindTokenInfo 为 JSON
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("序列化绑定信息失败: %w", err)
	}
	// 存储 token → BindTokenInfo JSON
	if err := redis.SetKeyValue(s.redisClient, bindTokenPrefix+token, string(data), bindCodeTTL); err != nil {
		return err
	}
	// 存储绑定码 → token 字符串（用于通过绑定码查找 token，再查找 BindTokenInfo）
	if info.BindCode != "" {
		if err := redis.SetKeyValue(s.redisClient, bindCodePrefix+info.BindCode, token, bindCodeTTL); err != nil {
			return err
		}
	}
	return nil
}

// getBindInfoByToken 从 Redis 读取 token 对应的绑定信息
//
// 对齐 Java BindTokenCache.getIfValid 方法
//
// 参数：
//   - token: 绑定 token
//
// 返回：
//   - *BindTokenInfo: 绑定信息（nil 表示未找到或已过期）
//   - error: Redis 错误
func (s *AuthService) getBindInfoByToken(token string) (*BindTokenInfo, error) {
	val, err := redis.GetKeyValue(s.redisClient, bindTokenPrefix+token)
	if err != nil {
		if redis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, err
	}
	if val == "" {
		return nil, nil
	}
	// 反序列化 JSON
	var info BindTokenInfo
	if err := json.Unmarshal([]byte(val), &info); err != nil {
		return nil, fmt.Errorf("反序列化绑定信息失败: %w", err)
	}
	return &info, nil
}

// getBindInfoByCode 从 Redis 读取绑定码对应的绑定信息
//
// 对齐 Java BindTokenCache.getByBindCode 方法
// 先通过绑定码找到 token，再通过 token 找到 BindTokenInfo
//
// 参数：
//   - code: 6 位绑定码
//
// 返回：
//   - info: 绑定信息（nil 表示未找到或已过期）
//   - token: 绑定 token（用于删除缓存）
//   - error: Redis 错误
func (s *AuthService) getBindInfoByCode(code string) (*BindTokenInfo, string, error) {
	// 1. 通过绑定码找到 token
	token, err := redis.GetKeyValue(s.redisClient, bindCodePrefix+code)
	if err != nil {
		if redis.IsRedisNil(err) {
			return nil, "", nil
		}
		return nil, "", err
	}
	if token == "" {
		return nil, "", nil
	}
	// 2. 通过 token 找到 BindTokenInfo
	info, err := s.getBindInfoByToken(token)
	if err != nil {
		return nil, "", err
	}
	return info, token, nil
}

// deleteBindInfo 删除绑定信息（标记为已使用）
//
// 对齐 Java BindTokenCache.markUsed 方法
// 同时删除 token 和绑定码两个 key
//
// 参数：
//   - token: 绑定 token
//   - code: 6 位绑定码
func (s *AuthService) deleteBindInfo(token, code string) {
	_ = redis.DeleteKey(s.redisClient, bindTokenPrefix+token)
	if code != "" {
		_ = redis.DeleteKey(s.redisClient, bindCodePrefix+code)
	}
}

// ---- 微信 API 辅助方法（对齐 Java WeChatApiService） ----

// getAccessToken 获取微信 access_token（带缓存，提前 5 分钟刷新）
//
// 对齐 Java WeChatApiService.getAccessToken
// 使用双重检查锁避免并发重复请求
//
// 返回：access_token 字符串，失败返回空字符串
func (s *AuthService) getAccessToken() string {
	// 快速路径：检查缓存是否有效
	if s.cachedAccessToken != "" && time.Now().UnixMilli() < s.accessTokenExpireAt {
		return s.cachedAccessToken
	}

	s.accessTokenMu.Lock()
	defer s.accessTokenMu.Unlock()

	// 双重检查
	if s.cachedAccessToken != "" && time.Now().UnixMilli() < s.accessTokenExpireAt {
		return s.cachedAccessToken
	}

	// 调用微信 CGI 接口获取 access_token
	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		wxAppID, wxAppSecret,
	)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("获取微信 access_token 失败: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取微信 access_token 响应失败: %v", err)
		return ""
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("解析微信 access_token 响应失败: %v", err)
		return ""
	}

	if result.AccessToken == "" {
		log.Printf("微信 access_token 获取失败: errcode=%d, errmsg=%s", result.ErrCode, result.ErrMsg)
		return ""
	}

	// 提前 5 分钟过期，避免边界问题（对齐 Java 行为）
	s.cachedAccessToken = result.AccessToken
	s.accessTokenExpireAt = time.Now().UnixMilli() + int64(result.ExpiresIn-300)*1000
	log.Printf("微信 access_token 获取成功，有效期 %d 秒", result.ExpiresIn)
	return s.cachedAccessToken
}

// generateQrCode 调用 wxacode.getUnlimited 生成小程序码
//
// 对齐 Java WeChatApiService.generateQrCode
// 使用入口页 pages/index/index 作为 page 参数，通过 scene 参数传递绑定 token
//
// 参数：
//   - scene: 场景值（绑定 token，最大 32 字符）
//
// 返回：base64 编码的图片（含 data:image/png;base64 前缀），失败返回空字符串
func (s *AuthService) generateQrCode(scene string) string {
	accessToken := s.getAccessToken()
	if accessToken == "" {
		log.Printf("无法获取 access_token，生成二维码失败")
		return ""
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/wxa/getwxacodeunlimit?access_token=%s",
		accessToken,
	)

	// 构建请求体（对齐 Java bodyMap）
	bodyMap := map[string]interface{}{
		"scene":       scene,
		"page":        "pages/index/index",
		"check_path":  false,
		"env_version": wxEnvVersion,
	}
	jsonBody, err := json.Marshal(bodyMap)
	if err != nil {
		log.Printf("构建二维码请求体失败: %v", err)
		return ""
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("调用微信二维码 API 失败: %v", err)
		return ""
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取微信二维码响应失败: %v", err)
		return ""
	}

	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode != 200 {
		log.Printf("微信二维码 API 返回非 200 状态码: %d, body=%s", resp.StatusCode, string(bodyBytes))
		return ""
	}

	// 成功返回图片
	if strings.HasPrefix(contentType, "image/") {
		base64Str := base64.StdEncoding.EncodeToString(bodyBytes)
		log.Printf("小程序码生成成功, scene=%s", scene)
		return "data:image/png;base64," + base64Str
	}

	// API 返回了 JSON 错误信息
	log.Printf("小程序码生成失败, contentType=%s, body=%s", contentType, string(bodyBytes))
	return ""
}

// sendSubscribeMessage 发送微信订阅消息
//
// 对齐 Java WeChatApiService.sendSubscribeMessage
// 调用微信接口 POST /cgi-bin/message/subscribe/send 向指定用户推送订阅消息
//
// 参数：
//   - openId: 接收消息的用户 openid
//   - templateId: 微信订阅消息模板 ID
//   - page: 点击消息后跳转的小程序页面路径
//   - data: 模板数据，key 为模板字段名，value 为字段值
//
// 返回：发送成功返回 true
func (s *AuthService) sendSubscribeMessage(openId, templateId, page string, data map[string]string) bool {
	accessToken := s.getAccessToken()
	if accessToken == "" {
		log.Printf("无法获取 access_token，发送订阅消息失败")
		return false
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/message/subscribe/send?access_token=%s",
		accessToken,
	)

	// 构建模板数据：每个字段格式为 { "value": "xxx" }
	dataMap := make(map[string]interface{})
	for k, v := range data {
		dataMap[k] = map[string]string{"value": v}
	}

	bodyMap := map[string]interface{}{
		"touser":           openId,
		"template_id":      templateId,
		"miniprogram_state": "formal",
		"lang":             "zh_CN",
		"data":             dataMap,
	}
	if page != "" {
		bodyMap["page"] = page
	}

	jsonBody, err := json.Marshal(bodyMap)
	if err != nil {
		log.Printf("构建订阅消息请求体失败: %v", err)
		return false
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("发送微信订阅消息异常: openId=%s, templateId=%s, error=%v", openId, templateId, err)
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取微信订阅消息响应失败: %v", err)
		return false
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("解析微信订阅消息响应失败: %v", err)
		return false
	}

	if result.ErrCode == 0 {
		log.Printf("订阅消息发送成功, openId=%s, templateId=%s", openId, templateId)
		return true
	}

	log.Printf("订阅消息发送失败, openId=%s, templateId=%s, errcode=%d, errmsg=%s",
		openId, templateId, result.ErrCode, result.ErrMsg)
	return false
}

// ---- 8 个绑定/订阅业务方法 ----

// GenerateBindQrcode 生成绑定二维码
//
// 对齐 Java AuthServiceImpl.generateBindQrcode
// 流程：
//  1. 生成 6 位绑定码和绑定 token
//  2. 存入 Redis（bind:token:{token} → BindTokenInfo JSON, bind:code:{code} → token, TTL=10分钟）
//  3. 调用微信 API 生成小程序码（scene 参数为 token）
//
// 参数：
//   - studentID: 学生ID
//   - relation: 与学生的关系（如"父亲"、"母亲"）
//   - isPrimary: 是否主要联系人
//
// 返回：BindQrcodeResponse（含 qrcode, token, bindCode）
func (s *AuthService) GenerateBindQrcode(studentID int64, relation string, isPrimary bool) *response.ResponseDTO {
	// 1. 校验学生存在性
	student, err := s.studentMapper.SelectByID(studentID)
	if err != nil {
		log.Printf("查询学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if student == nil {
		return response.Fail("学生不存在")
	}

	// 2. 生成 6 位绑定码和 token
	code := generateBindCode()
	token := generateBindToken()

	// 3. 构建绑定信息并存入 Redis（绑定模式，subscribeOnly=false）
	info := &BindTokenInfo{
		StudentID:    studentID,
		Relation:     relation,
		IsPrimary:    isPrimary,
		BindCode:     code,
		SubscribeOnly: false,
	}
	if err := s.setBindInfo(token, info); err != nil {
		log.Printf("存储绑定信息失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成绑定码失败")
	}

	// 4. 调用微信 API 生成小程序码（scene 参数为 token）
	qrcode := s.generateQrCode(token)
	if qrcode == "" {
		// 二维码生成失败，但仍返回 bindCode 供手动输入绑定
		log.Printf("二维码生成失败，但绑定码可用: studentID=%d, code=%s", studentID, code)
	}

	return response.Success(&BindQrcodeResponse{
		Qrcode:   qrcode,
		Token:    token,
		BindCode: code,
	})
}

// GenerateSubscribeQrcode 生成订阅专用二维码
//
// 对齐 Java AuthServiceImpl.generateSubscribeQrcode
// 类似绑定二维码，但用于仅订阅模式（家长端只能订阅不能绑定账号）
// subscribeOnly=true 存储在 BindTokenInfo 中，Redis key 前缀统一
//
// 参数：
//   - studentID: 学生ID
//   - relation: 与学生的关系
//   - isPrimary: 是否主要联系人
//
// 返回：BindQrcodeResponse（含 qrcode, token, bindCode）
func (s *AuthService) GenerateSubscribeQrcode(studentID int64, relation string, isPrimary bool) *response.ResponseDTO {
	// 1. 校验学生存在性
	student, err := s.studentMapper.SelectByID(studentID)
	if err != nil {
		log.Printf("查询学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if student == nil {
		return response.Fail("学生不存在")
	}

	// 2. 生成 6 位绑定码和 token
	code := generateBindCode()
	token := generateBindToken()

	// 3. 构建绑定信息并存入 Redis（订阅模式，subscribeOnly=true）
	info := &BindTokenInfo{
		StudentID:    studentID,
		Relation:     relation,
		IsPrimary:    isPrimary,
		BindCode:     code,
		SubscribeOnly: true,
	}
	if err := s.setBindInfo(token, info); err != nil {
		log.Printf("存储订阅信息失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成订阅码失败")
	}

	// 4. 调用微信 API 生成小程序码
	qrcode := s.generateQrCode(token)
	if qrcode == "" {
		log.Printf("二维码生成失败，但订阅码可用: studentID=%d, code=%s", studentID, code)
	}

	return response.Success(&BindQrcodeResponse{
		Qrcode:   qrcode,
		Token:    token,
		BindCode: code,
	})
}

// GetBindInfo 按 token 查绑定信息
//
// 对齐 Java AuthServiceImpl.getBindInfo
// 从 Redis 读取 token 对应的 BindTokenInfo，查询学生信息返回（不执行绑定）
//
// 参数：
//   - token: 绑定 token（扫码后从二维码内容中获取）
//
// 返回：BindInfoResponse（学生信息，含 relation/isPrimary/parentName/parentPhone）
func (s *AuthService) GetBindInfo(token string) *response.ResponseDTO {
	// 1. 从 Redis 读取 token 对应的绑定信息
	info, err := s.getBindInfoByToken(token)
	if err != nil || info == nil {
		return response.Fail("二维码已过期或无效，请重新生成")
	}

	// 2. 查询学生信息并构建响应
	return s.buildBindInfoResponse(info)
}

// GetBindInfoByCode 按 6 位绑定码查学生信息
//
// 对齐 Java AuthServiceImpl.getBindInfoByCode
// 从 Redis 读取绑定码对应的 BindTokenInfo，查询学生信息返回（不执行绑定）
// 家长端输入绑定码后先查询展示学生信息，确认后再执行绑定
//
// 参数：
//   - code: 6 位绑定码
//
// 返回：BindInfoResponse（学生信息，含 relation/isPrimary/parentName/parentPhone）
func (s *AuthService) GetBindInfoByCode(code string) *response.ResponseDTO {
	if code == "" {
		return response.Fail("绑定码不能为空")
	}

	// 1. 从 Redis 读取绑定码对应的绑定信息
	info, _, err := s.getBindInfoByCode(code)
	if err != nil || info == nil {
		return response.Fail("绑定码无效或已过期，请重新获取")
	}

	// 2. 查询学生信息并构建响应
	return s.buildBindInfoResponse(info)
}

// buildBindInfoResponse 构建绑定信息响应（GetBindInfo 和 GetBindInfoByCode 共用）
//
// 对齐 Java AuthServiceImpl.buildBindInfoResponse
// 查询学生信息、机构名称、占位家长信息，构造响应对象
//
// 参数：
//   - info: 绑定令牌信息（含 studentID, relation, isPrimary, subscribeOnly）
//
// 返回：BindInfoResponse
func (s *AuthService) buildBindInfoResponse(info *BindTokenInfo) *response.ResponseDTO {
	// 查询学生信息
	student, err := s.studentMapper.SelectByID(info.StudentID)
	if err != nil {
		log.Printf("查询学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if student == nil {
		return response.Fail("学生信息不存在")
	}

	// 查询机构名称
	var institutionName string
	if student.InstitutionID.Valid {
		institution, err := s.institutionMapper.SelectByID(student.InstitutionID.Int64)
		if err == nil && institution != nil {
			institutionName = institution.InstitutionName.String
		}
	}
	if institutionName == "" {
		institutionName = "未知机构"
	}

	// 查询占位家长信息（教师创建学生时预填的家长名称和手机号）
	// 按 studentID + isPrimary 匹配 c_parent_student，再 JOIN c_parent 获取 name/phone
	var parentName, parentPhone string
	ps, err := s.parentStudentMapper.SelectByStudentIDAndIsPrimary(info.StudentID, info.IsPrimary)
	if err != nil {
		log.Printf("查询家长学生关联失败: %v", err)
	} else if ps != nil && ps.ParentID.Valid {
		// 查询占位家长记录
		placeholder, err := s.parentMapper.SelectByID(ps.ParentID.Int64)
		if err != nil {
			log.Printf("查询占位家长失败: %v", err)
		} else if placeholder != nil {
			parentName = placeholder.Username.String
			parentPhone = placeholder.Phone.String
		}
	}

	return response.Success(&BindInfoResponse{
		StudentID:       student.ID.Int64,
		StudentName:     student.StudentName.String,
		Sex:             student.Sex.Int64,
		InstitutionName: institutionName,
		Relation:        info.Relation,
		IsPrimary:       info.IsPrimary,
		SubscribeOnly:   info.SubscribeOnly,
		ParentName:      parentName,
		ParentPhone:     parentPhone,
	})
}

// CheckBindStatus 检查绑定状态
//
// 对齐 Java AuthServiceImpl.checkBindStatus
// 通过微信 code 获取 openId，检查当前家长是否已绑定该学生、该学生是否已有家长账号
//
// 参数：
//   - token: 绑定 token（从 Redis 获取 BindTokenInfo，含 studentID 和 isPrimary）
//   - code: 微信登录 code（前端 wx.login 获取，用于换取 openId）
//
// 返回：BindStatusResponse（alreadyBound: 是否已绑定, hasAccount: 是否已有家长账号）
func (s *AuthService) CheckBindStatus(token, code string) *response.ResponseDTO {
	// 1. 从 Redis 读取 token 对应的绑定信息
	info, err := s.getBindInfoByToken(token)
	if err != nil || info == nil {
		return response.Fail("二维码已过期或无效，请重新生成")
	}

	// 2. 通过微信 code 获取 openId
	openId, err := s.GetOpenId(code)
	if err != nil {
		return response.Fail("微信授权失败：" + err.Error())
	}

	// 3. 获取学生信息（用于确定机构）
	student, err := s.studentMapper.SelectByID(info.StudentID)
	if err != nil {
		log.Printf("查询学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if student == nil {
		return response.Fail("学生信息不存在")
	}

	// 4. 初始化响应（默认未绑定、无账号）
	statusResponse := &BindStatusResponse{
		AlreadyBound: false,
		HasAccount:   false,
	}

	// 5. 检查 HasAccount：查询该学生是否已有任意家长账号
	// 通过 c_parent_student JOIN c_parent JOIN c_user_platform 查询 last_login_role=3 的记录
	hasAccount, err := s.parentStudentMapper.HasParentWithAccount(info.StudentID)
	if err != nil {
		log.Printf("查询学生家长账号失败: %v", err)
	} else {
		statusResponse.HasAccount = hasAccount
	}

	// 6. 检查 AlreadyBound：通过 openId 查找家长，检查是否已绑定该学生
	// 6.1 通过 openId 查找已绑定的用户
	user, err := s.userMapper.SelectUserByPlatformOpenid(platformWeixin, openId)
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return response.Success(statusResponse)
	}
	if user == nil {
		// 新用户，未绑定
		return response.Success(statusResponse)
	}

	// 6.2 查找家长记录
	parent, err := s.parentMapper.SelectByUserID(user.ID)
	if err != nil {
		log.Printf("查询家长失败: %v", err)
		return response.Success(statusResponse)
	}
	if parent == nil || !parent.ParentID.Valid {
		// 无家长记录，未绑定
		return response.Success(statusResponse)
	}

	// 6.3 检查是否已有 parent_student 记录（按 parentId + studentId 查询）
	existingPS, err := s.parentStudentMapper.SelectByParentAndStudent(parent.ParentID.Int64, info.StudentID)
	if err != nil {
		log.Printf("查询家长学生关联失败: %v", err)
		return response.Success(statusResponse)
	}
	if existingPS != nil {
		// 已存在关联记录，标记为已绑定
		statusResponse.AlreadyBound = true
	}

	return response.Success(statusResponse)
}

// ConfirmBind 确认绑定（通过 token）
//
// 对齐 Java AuthServiceImpl.confirmBind
// 确认绑定：校验 token → 获取 openId → 查/建用户和家长 → 创建 parent_student 关联 → 更新 wx_student_subscribe
//
// 参数：
//   - token: 绑定 token
//   - openId: 家长微信 openId
//
// 返回：绑定结果
func (s *AuthService) ConfirmBind(token, openId string) *response.ResponseDTO {
	if token == "" || openId == "" {
		return response.Fail("token 和 openId 不能为空")
	}

	// 1. 从 Redis 读取 token 对应的绑定信息
	info, err := s.getBindInfoByToken(token)
	if err != nil || info == nil {
		return response.Fail("二维码已过期或无效，请重新生成")
	}

	// 2. 执行绑定流程
	result := s.doBind(info.StudentID, openId, info.SubscribeOnly)
	if result.Code == 200 {
		// 绑定成功，删除绑定信息（标记已使用）
		s.deleteBindInfo(token, info.BindCode)
	}
	return result
}

// BindByCode 按绑定码直接绑定
//
// 对齐 Java AuthServiceImpl.bindByCode
// 合并查询+绑定流程：校验绑定码 → 查询学生 → 创建家长账号（如不存在）→ 创建 parent_student 关联 → 更新 wx_student_subscribe
// 返回绑定结果 + LoginVO（含 token）
//
// 参数：
//   - code: 6 位绑定码
//   - openId: 家长微信 openId
//
// 返回：BindResultResponse（含 LoginVO）
func (s *AuthService) BindByCode(code, openId string) *response.ResponseDTO {
	if code == "" || openId == "" {
		return response.Fail("绑定码和 openId 不能为空")
	}

	// 1. 从 Redis 读取绑定码对应的绑定信息（同时获取 token 用于删除缓存）
	info, token, err := s.getBindInfoByCode(code)
	if err != nil || info == nil {
		return response.Fail("绑定码无效或已过期，请重新获取")
	}

	// 2. 执行绑定流程
	bindResult := s.doBind(info.StudentID, openId, info.SubscribeOnly)
	if bindResult.Code != 200 {
		return bindResult
	}

	// 3. 绑定成功，删除绑定信息（标记已使用，同时删除 token 和 code 两个 key）
	s.deleteBindInfo(token, code)

	// 4. 查询家长用户信息以签发 token
	// 通过 openId 查找用户平台记录
	platform, err := s.userPlatformMapper.SelectByOpenIdAndPlatform(openId, platformWeixin)
	if err != nil || platform == nil {
		// 查不到平台记录，仅返回绑定成功消息（不签发 token）
		log.Printf("绑定成功但查不到平台记录，无法签发 token: openId=%s, err=%v", openId, err)
		return response.Success(&BindResultResponse{
			Message: bindResult.Message,
			Login:   nil,
		})
	}

	userID := platform.UserID.Int64
	roleID := context.RoleParent // 家长角色

	// 5. 签发双 Token
	accessToken, err := s.jwtUtils.CreateAccessToken(userID, roleID, openId)
	if err != nil {
		log.Printf("生成 Access Token 失败: %v", err)
		return response.Success(&BindResultResponse{
			Message: bindResult.Message,
			Login:   nil,
		})
	}

	refreshToken, err := s.jwtUtils.CreateRefreshToken(userID, roleID, openId)
	if err != nil {
		log.Printf("生成 Refresh Token 失败: %v", err)
		return response.Success(&BindResultResponse{
			Message: bindResult.Message,
			Login:   nil,
		})
	}

	// 6. 查询完整用户信息
	userVO := s.GetFullUserInfo(userID, roleID)

	return response.Success(&BindResultResponse{
		Message: bindResult.Message,
		Login: &LoginVO{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			OpenID:       openId,
			User:         userVO,
		},
	})
}

// doBind 公共绑定逻辑（ConfirmBind 和 BindByCode 共用）
//
// 对齐 Java AuthServiceImpl.doBind（简化版，不含账号密码验证）
// 流程：
//  1. 查询学生信息（校验学生存在 + 机构未过期）
//  2. 仅订阅模式：只记录 wx_student_subscribe，不创建 user/parent
//  3. 绑定模式：查/建用户 → 查/建 parent → 创建 parent_student 关联 → 记录 wx_student_subscribe
//
// 参数：
//   - studentID: 学生ID
//   - openId: 家长微信 openId
//   - subscribeOnly: 是否仅订阅模式
//
// 返回：绑定结果
func (s *AuthService) doBind(studentID int64, openId string, subscribeOnly bool) *response.ResponseDTO {
	// 1. 查询学生信息（校验学生存在）
	student, err := s.studentMapper.SelectByID(studentID)
	if err != nil {
		log.Printf("查询学生失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if student == nil {
		return response.Fail("学生信息不存在")
	}

	// 1.1 校验机构是否过期（对齐 Java doBind 机构过期校验）
	if student.InstitutionID.Valid {
		institution, err := s.institutionMapper.SelectByID(student.InstitutionID.Int64)
		if err == nil && institution != nil {
			if institution.ExpireTime.Valid && time.Now().After(institution.ExpireTime.Time) {
				return response.Fail("该机构使用期限已到期，请联系管理员续期")
			}
		}
	}

	// 2. 仅订阅模式：只记录 wx_student_subscribe，不创建 user/parent 记录
	if subscribeOnly {
		return s.doSubscribeOnly(openId, studentID)
	}

	// 3. 绑定模式：查/建用户 → 查/建 parent → 创建 parent_student 关联
	// 3.1 通过 openId 查找已绑定的用户
	user, err := s.userMapper.SelectUserByPlatformOpenid(platformWeixin, openId)
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	var userID int64
	if user != nil {
		// 已有用户记录，复用
		userID = user.ID
	} else {
		// 3.2 新用户，创建 c_user 记录（机构ID 来自学生）
		newUser := &entity.User{
			InstitutionID: student.InstitutionID,
		}
		userID, err = s.userMapper.Insert(newUser)
		if err != nil {
			log.Printf("创建用户失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "创建用户失败")
		}

		// 3.3 创建 user_platform 记录（关联 openId 到新用户）
		newPlatform := &entity.UserPlatform{
			UserID:        sql.NullInt64{Int64: userID, Valid: true},
			OpenID:        sql.NullString{String: openId, Valid: true},
			Platform:      sql.NullString{String: platformWeixin, Valid: true},
			LastLoginRole: sql.NullInt64{Int64: context.RoleParent, Valid: true},
		}
		if _, err := s.userPlatformMapper.Insert(newPlatform); err != nil {
			log.Printf("创建用户平台记录失败: %v", err)
			// 不返回错误，绑定流程可继续（用户已创建）
		}
	}

	// 3.4 查找或创建 parent 记录
	parent, err := s.parentMapper.SelectByUserID(userID)
	if err != nil {
		log.Printf("查询家长失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	if parent == nil {
		// 创建新的 parent 记录
		newParent := &entity.Parent{
			RoleBaseEntity: entity.RoleBaseEntity{
				UserID:      sql.NullInt64{Int64: userID, Valid: true},
				IsAvailable: sql.NullBool{Bool: true, Valid: true},
			},
			IsBound: sql.NullBool{Bool: true, Valid: true},
		}
		parentID, err := s.parentMapper.Insert(newParent)
		if err != nil {
			log.Printf("创建家长记录失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "创建家长记录失败")
		}
		// 重新查询以获取完整的 parent 记录
		parent, _ = s.parentMapper.SelectByID(parentID)
		if parent == nil {
			return response.Fail("家长记录创建失败，请重试")
		}
	}

	// 3.5 检查是否已有 parent_student 记录（去重）
	existingPS, err := s.parentStudentMapper.SelectByParentAndStudent(parent.ParentID.Int64, studentID)
	if err != nil {
		log.Printf("查询家长学生关联失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if existingPS != nil {
		// 已绑定，幂等返回成功
		// 仍更新 wx_student_subscribe（确保订阅关系存在）
		s.saveWxStudentSubscribe(openId, studentID, true, "full")
		return response.Success("该学生已绑定，无需重复绑定")
	}

	// 3.6 创建 parent_student 关联记录
	newPS := &entity.ParentStudent{
		ParentID:  sql.NullInt64{Int64: parent.ParentID.Int64, Valid: true},
		StudentID: sql.NullInt64{Int64: studentID, Valid: true},
		IsPrimary: sql.NullBool{Bool: true, Valid: true}, // 默认主联系人
		Relation:  sql.NullString{String: "家长", Valid: true}, // 默认关系
	}
	if _, err := s.parentStudentMapper.Insert(newPS); err != nil {
		log.Printf("创建家长学生关联失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "绑定失败")
	}

	// 3.7 记录微信号-学生订阅关系
	s.saveWxStudentSubscribe(openId, studentID, true, "full")

	log.Printf("家长绑定成功: userID=%d, studentID=%d", userID, studentID)
	return response.Success("绑定成功")
}

// doSubscribeOnly 仅订阅模式绑定逻辑
//
// 对齐 Java doBind 中 isSubscribeOnly 分支
// 只记录 wx_student_subscribe，不创建 user/parent 记录
//
// 参数：
//   - openId: 微信 openId
//   - studentID: 学生ID
//
// 返回：订阅结果
func (s *AuthService) doSubscribeOnly(openId string, studentID int64) *response.ResponseDTO {
	// 创建或更新 wx_subscribe_record（确保扣费通知能发到此 openId）
	record, err := s.wxSubscribeRecordMapper.SelectByOpenIdAndTemplate(openId, subscribeTemplateID)
	if err != nil {
		log.Printf("查询订阅记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	if record == nil {
		// 新建订阅记录
		newRecord := &entity.WxSubscribeRecord{
			OpenID:         sql.NullString{String: openId, Valid: true},
			TemplateID:     sql.NullString{String: subscribeTemplateID, Valid: true},
			SubscribeCount: sql.NullInt64{Int64: 1, Valid: true},
		}
		if _, err := s.wxSubscribeRecordMapper.Insert(newRecord); err != nil {
			log.Printf("创建订阅记录失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "订阅失败")
		}
	} else {
		// 已有记录，次数 +1
		if err := s.wxSubscribeRecordMapper.IncrementCount(record.ID.Int64, false); err != nil {
			log.Printf("更新订阅记录失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "订阅失败")
		}
	}

	// 记录微信号-学生订阅关系
	s.saveWxStudentSubscribe(openId, studentID, true, "subscribe")

	log.Printf("仅订阅成功: openId=%s, studentID=%d", openId, studentID)
	return response.Success("订阅成功")
}

// saveWxStudentSubscribe 保存或更新微信号-学生订阅关系
//
// 对齐 Java AuthServiceImpl.saveWxStudentSubscribe
// 如果该 openId 已订阅该学生，则更新 is_primary 和 bind_mode；否则新建记录
//
// 参数：
//   - openId: 微信 openId
//   - studentID: 学生ID
//   - isPrimary: 是否主要联系人
//   - bindMode: 绑定模式（"subscribe"=仅订阅, "full"=绑定账号并订阅）
func (s *AuthService) saveWxStudentSubscribe(openId string, studentID int64, isPrimary bool, bindMode string) {
	existing, err := s.wxStudentSubscribeMapper.SelectByOpenIdAndStudent(openId, studentID)
	if err != nil {
		log.Printf("查询学生订阅关系失败: %v", err)
		return
	}

	if existing == nil {
		// 新建订阅关系
		newRecord := &entity.WxStudentSubscribe{
			OpenID:    sql.NullString{String: openId, Valid: true},
			StudentID: sql.NullInt64{Int64: studentID, Valid: true},
			IsPrimary: sql.NullBool{Bool: isPrimary, Valid: true},
			BindMode:  sql.NullString{String: bindMode, Valid: true},
		}
		if _, err := s.wxStudentSubscribeMapper.Insert(newRecord); err != nil {
			log.Printf("创建学生订阅关系失败: %v", err)
		}
	} else {
		// 已存在则更新（可能从仅订阅升级为绑定账号）
		if err := s.wxStudentSubscribeMapper.UpdateByOpenIdAndStudent(openId, studentID, isPrimary, bindMode); err != nil {
			log.Printf("更新学生订阅关系失败: %v", err)
		}
	}
}

// TestSendSubscribe 测试发送订阅消息
//
// 对齐 Java AuthServiceImpl.testSendSubscribe
// 通过 openId 查询订阅记录，向当前设备发送一条测试扣课通知
// 发送成功后授权次数 -1
//
// 参数：
//   - openId: 微信 openId
//
// 返回：发送结果
func (s *AuthService) TestSendSubscribe(openId string) *response.ResponseDTO {
	if openId == "" {
		return response.Fail("openId 不能为空")
	}

	// 1. 查询当前 openId 的订阅记录
	record, err := s.wxSubscribeRecordMapper.SelectByOpenIdAndTemplate(openId, subscribeTemplateID)
	if err != nil {
		log.Printf("查询订阅记录失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if record == nil {
		return response.Fail("当前设备无订阅记录，请先点击订阅按钮授权")
	}

	// 2. 构建测试订阅消息数据（对齐 Java templateData）
	now := time.Now().Format("2006-01-02")
	remainingCount := record.SubscribeCount.Int64 - 1
	if remainingCount < 0 {
		remainingCount = 0
	}
	templateData := map[string]string{
		"thing1":  "测试扣次项目",
		"thing8":  "测试学生",
		"number4":  strconv.FormatInt(remainingCount, 10),
		"number10": "1",
		"number11": strconv.FormatInt(remainingCount, 10),
		"time5":    now,
	}

	// 3. 调用微信接口发送订阅消息（向当前设备的 openId 发送）
	success := s.sendSubscribeMessage(
		openId,
		subscribeTemplateID,
		"pages/main/parent/my-course/index",
		templateData,
	)

	if success {
		// 4. 发送成功，授权次数 -1
		if err := s.wxSubscribeRecordMapper.DecrementCount(record.ID.Int64); err != nil {
			log.Printf("更新订阅次数失败: %v", err)
		}
		newCount := remainingCount
		log.Printf("测试订阅消息发送成功: openId=%s, 剩余次数=%d", openId, newCount)
		return response.Success("测试消息发送成功，当前设备剩余授权次数：" + strconv.FormatInt(newCount, 10))
	}

	log.Printf("测试订阅消息发送失败: openId=%s", openId)
	return response.Fail("测试消息发送失败，请检查微信后台配置")
}
