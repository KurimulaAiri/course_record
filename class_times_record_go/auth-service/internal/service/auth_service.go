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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	wxAppID     = "wx1234567890abcdef" // TODO: 从配置加载
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
type LoginVO struct {
	Token        string      `json:"token"`        // Access Token
	RefreshToken string      `json:"refreshToken"` // Refresh Token
	OpenID       string      `json:"openId"`       // 微信 openId
	User         interface{} `json:"user"`         // 用户信息（登录后返回，免密登录可能为 null）
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
//      InstitutionMapper, StudentMapper, ParentStudentMapper,
//      WxSubscribeRecordMapper, WxStudentSubscribeMapper, JwtUtils, RedisClient
type AuthService struct {
	userMapper             *mapper.UserMapper
	userAuthMapper         *mapper.UserAuthMapper
	userPlatformMapper     *mapper.UserPlatformMapper
	parentMapper           *mapper.ParentMapper
	institutionMapper      *mapper.InstitutionMapper
	studentMapper          *mapper.StudentMapper
	parentStudentMapper    *mapper.ParentStudentMapper
	wxSubscribeRecordMapper *mapper.WxSubscribeRecordMapper
	wxStudentSubscribeMapper *mapper.WxStudentSubscribeMapper
	jwtUtils               *jwt.Utils
	redisClient            *rediscli.Client
}

// NewAuthService 创建 AuthService
func NewAuthService(
	userMapper *mapper.UserMapper,
	userAuthMapper *mapper.UserAuthMapper,
	userPlatformMapper *mapper.UserPlatformMapper,
	parentMapper *mapper.ParentMapper,
	institutionMapper *mapper.InstitutionMapper,
	studentMapper *mapper.StudentMapper,
	parentStudentMapper *mapper.ParentStudentMapper,
	wxSubscribeRecordMapper *mapper.WxSubscribeRecordMapper,
	wxStudentSubscribeMapper *mapper.WxStudentSubscribeMapper,
	jwtUtils *jwt.Utils,
	redisClient *rediscli.Client,
) *AuthService {
	return &AuthService{
		userMapper:              userMapper,
		userAuthMapper:          userAuthMapper,
		userPlatformMapper:      userPlatformMapper,
		parentMapper:            parentMapper,
		institutionMapper:       institutionMapper,
		studentMapper:           studentMapper,
		parentStudentMapper:     parentStudentMapper,
		wxSubscribeRecordMapper: wxSubscribeRecordMapper,
		wxStudentSubscribeMapper: wxStudentSubscribeMapper,
		jwtUtils:                jwtUtils,
		redisClient:             redisClient,
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
//      ?appid=APPID&secret=SECRET&js_code=JSCODE&grant_type=authorization_code
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
	defer resp.Body.Close()

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
		Token:        accessToken,
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

	// 8. 查询用户信息返回（对齐 Java getFullUserInfoByUserId）
	user, _ := s.userMapper.SelectByID(userID)

	return response.Success(&LoginVO{
		Token:        accessToken,
		RefreshToken: refreshToken,
		OpenID:       req.OpenID,
		User:         user,
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

	// 2. 查询用户信息（对齐 Java getFullUserInfoByUserId）
	user, err := s.userMapper.SelectByID(userID)
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
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
		Token:        accessToken,
		RefreshToken: refreshToken,
		OpenID:       openId,
		User:         user,
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
	accessToken, err := s.jwtUtils.CreateAccessToken(claims.UserID, claims.RoleID, claims.OpenID)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	return response.Success(&LoginVO{
		Token:        accessToken,
		RefreshToken: refreshToken, // 返回原 refresh token
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

	return response.Success("已登出")
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
		UserID:  sql.NullInt64{Int64: userID, Valid: true},
		RoleID:  sql.NullInt64{Int64: req.Role, Valid: true},
		Account: sql.NullString{String: req.Account, Valid: true},
		Password: sql.NullString{String: hashedPassword, Valid: true},
		Salt:    sql.NullString{String: salt, Valid: true},
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

// GetUserAuthByTeacherID 按教师ID查认证信息
//
// 对齐 Java AuthServiceImpl.getUserAuthByTeacherId
//
// 参数：
//   - teacherID: 教师ID
//
// 返回：认证信息
func (s *AuthService) GetUserAuthByTeacherID(teacherID int64) *response.ResponseDTO {
	auth, err := s.userAuthMapper.SelectAuthByTeacherId(teacherID)
	if err != nil {
		log.Printf("查询教师认证信息失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if auth == nil {
		return response.Fail("教师认证信息不存在")
	}

	// 返回时隐藏密码和盐值
	return response.Success(map[string]interface{}{
		"id":           auth.ID,
		"userId":       auth.UserID.Int64,
		"roleId":       auth.RoleID.Int64,
		"account":      auth.Account.String,
		"lastLoginTime": entity.FormatTime(auth.LastLoginTime),
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
	Code         string `json:"code"`         // 微信登录凭证（用于获取 openId）
	TemplateID   string `json:"templateId"`   // 模板ID
	IsPermanent  bool   `json:"isPermanent"`  // 是否永久订阅
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
		"subscribed":      false,
		"subscribeCount":  0,
		"isPermanent":     false,
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
