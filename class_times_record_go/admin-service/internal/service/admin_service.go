// Package service admin-service 业务逻辑层
//
// 对齐 Java admin-service/src/main/java/com/shiroko/service/impl 包
//
// 核心功能：
//   - 管理员登录（SM2 解密 + BCrypt 密码校验，对齐 Java AdminAuthFlow）
//   - 管理员用户管理（CRUD）
//   - 角色管理
//   - 菜单管理
//
// 注意：Admin 端认证流程：
//   - 前端用 SM2 公钥加密密码 → 后端 SM2 私钥解密 → BCrypt 哈希校验
//   - 无 SM3 请求签名
//   - JWT 会话管理
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/crypto"
	"github.com/kurimula-airi/course_record_go/common/jwt"
	"github.com/kurimula-airi/course_record_go/common/response"
	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// VO 定义（对齐 Java VO 类）
// ============================================================

// LoginVO 登录返回（对齐 Java LoginSysUserVO）
//
// 字段命名与 Java 后端保持一致：
//   - accessToken / refreshToken / userInfo
//   - admin 前端依赖这些字段名（见 src/types/admin.d.ts LoginResponse）
type LoginVO struct {
	AccessToken  string      `json:"accessToken"`  // Access Token
	RefreshToken string      `json:"refreshToken"` // Refresh Token
	UserInfo     interface{} `json:"userInfo"`     // 用户信息（隐藏密码）
}

// SysUserVO 系统用户视图对象（对齐 Java SysUserVO）
//
// 使用普通类型而非 sql.NullXxx，避免 JSON 序列化输出对象格式
// 字段命名对齐 admin 前端 SysUserResponse 类型（见 src/types/admin.d.ts）
type SysUserVO struct {
	ID           int64  `json:"id"`           // 主键
	Username     string `json:"username"`     // 用户名
	Nickname     string `json:"nickname"`     // 昵称
	Phone        string `json:"phone"`        // 手机号
	Email        string `json:"email"`        // 邮箱
	Avatar       string `json:"avatar"`       // 头像
	Status       int64  `json:"status"`       // 状态（0=禁用,1=启用）
	IsDeleted    int64  `json:"isDeleted"`    // 是否删除（0=未删除,1=已删除）
	Remark       string `json:"remark"`       // 备注
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串（yyyy-MM-dd HH:mm:ss）
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
	RoleIDs      []int64 `json:"roleIds"`     // 角色ID列表
}

// ToSysUserVO 实体转 VO
//
// 将 AdminUser 转换为 SysUserVO，避免 sql.NullTime 序列化为对象
//
// 参数：
//   - u: AdminUser 实体
//   - roleIds: 角色ID列表（从 sys_user_role 表查询）
func ToSysUserVO(u *mapper.AdminUser, roleIds []int64) *SysUserVO {
	if u == nil {
		return nil
	}
	// 确保返回非 nil 的空切片（而非 nil），以便 JSON 序列化为 [] 而非 null
	if roleIds == nil {
		roleIds = []int64{}
	}
	vo := &SysUserVO{
		ID:            u.ID,
		Username:      u.Username,
		Nickname:      u.Nickname,
		Phone:         u.Phone,
		Email:         u.Email,
		Avatar:        u.Avatar,
		Status:        u.Status,
		IsDeleted:     u.IsDeleted,    // 逻辑删除标记（0=未删除,1=已删除）
		Remark:        u.Remark,       // 备注
		RoleIDs:       roleIds,
	}
	// 时间格式化（对齐 Java DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")）
	if u.CreateTime.Valid {
		vo.CreateTimeStr = u.CreateTime.Time.Format("2006-01-02 15:04:05")
	}
	if u.UpdateTime.Valid {
		vo.UpdateTimeStr = u.UpdateTime.Time.Format("2006-01-02 15:04:05")
	}
	return vo
}

// ============================================================
// AdminService 管理端服务
// ============================================================

// AdminService 管理端服务（对齐 Java AdminUserServiceImpl）
type AdminService struct {
	adminUserMapper *mapper.AdminUserMapper
	jwtUtils        *jwt.Utils
	sm2PrivateKey   string // SM2 私钥（hex，用于解密前端加密的密码，对齐 Java Nacos cr-admin-service.yaml）
}

// NewAdminService 创建 AdminService
//
// 参数：
//   - adminUserMapper: 用户 Mapper
//   - jwtUtils: JWT 工具
//   - sm2PrivateKey: SM2 私钥（hex 编码，来自 Nacos 配置，对齐 Java SM2Util.decrypt）
func NewAdminService(adminUserMapper *mapper.AdminUserMapper, jwtUtils *jwt.Utils, sm2PrivateKey string) *AdminService {
	return &AdminService{
		adminUserMapper: adminUserMapper,
		jwtUtils:        jwtUtils,
		sm2PrivateKey:   sm2PrivateKey,
	}
}

// LoginRequest 登录请求（对齐 Java AdminLoginDTO）
type LoginRequest struct {
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 密码（前端用 SM2 公钥加密后的密文）
}

// Login 管理员登录
//
// 对齐 Java AdminAuthFlow：
//   - 前端用 SM2 公钥加密密码
//   - 后端 SM2 私钥解密
//   - BCrypt 哈希校验
//   - JWT 会话管理
//
// 参数：
//   - req: 登录请求（Password 字段为 SM2 加密后的密文）
//
// 返回：LoginVO（含 accessToken/refreshToken/userInfo）
func (s *AdminService) Login(req *LoginRequest) *response.ResponseDTO {
	if req.Username == "" || req.Password == "" {
		return response.Fail("用户名和密码不能为空")
	}

	// 1. SM2 解密密码（对齐 Java SM2Util.decrypt）
	// 前端使用 SM2 公钥加密密码，后端用私钥解密得到明文
	rawPassword, err := crypto.SM2Decrypt(req.Password, s.sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密失败: %v", err)
		return response.Fail("密码解密失败，请检查加密参数")
	}

	// 2. 查询用户（对齐 Java sysUserMapper.selectByUsername）
	user, err := s.adminUserMapper.SelectByUsername(req.Username)
	if err != nil {
		log.Printf("查询管理员失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户名或密码错误")
	}

	// 3. 校验状态
	if user.Status != 1 {
		return response.Fail("账号已被禁用")
	}

	// 4. BCrypt 密码校验（对齐 Java BCrypt.matches）
	// 使用 golang.org/x/crypto/bcrypt 与 Java spring-security BCrypt 互通
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(rawPassword)); err != nil {
		return response.Fail("用户名或密码错误")
	}

	// 5. 签发双 Token（对齐 Java jwtUtils.createAccessToken/createRefreshToken）
	// 管理员角色ID=0（对齐 Java SysUserServiceImpl.login 中的 0L）
	accessToken, err := s.jwtUtils.CreateAccessToken(user.ID, 0, "")
	if err != nil {
		log.Printf("生成 Access Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	refreshToken, err := s.jwtUtils.CreateRefreshToken(user.ID, 0, "")
	if err != nil {
		log.Printf("生成 Refresh Token 失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	// 6. 查询角色ID（对齐 Java getRoleIdsByUserId）
	// 查询失败不阻断登录，降级为空数组
	roleIds, err := s.adminUserMapper.SelectRoleIDsByUserID(user.ID)
	if err != nil {
		log.Printf("查询角色ID失败: %v", err)
		roleIds = []int64{}
	}

	// 7. 返回用户信息（含 roleIds，隐藏密码，转换为 VO 避免时间序列化问题）
	userInfo := ToSysUserVO(user, roleIds)

	return response.Success(&LoginVO{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserInfo:     userInfo,
	})
}

// RefreshToken 刷新 Access Token
//
// 对齐 Java SysUserServiceImpl.refreshToken
//
// 请求字段：{ refreshToken: string }（admin 前端 src/api/auth/index.ts refreshTokenApi）
//
// 修改点（vs 原实现）：
//   1. 重新查询用户信息构造 SysUserVO（非 nil）
//   2. 签发新的 refreshToken（非返回原 token）
//   3. 校验用户状态（status != 1 拒绝刷新）
//   4. 查询 roleIds 填充到 userInfo
//
// 参数：
//   - refreshToken: Refresh Token
//
// 返回：LoginVO（含新 accessToken/refreshToken/userInfo）
func (s *AdminService) RefreshToken(refreshToken string) *response.ResponseDTO {
	// 1. 解析 refresh token
	claims, err := s.jwtUtils.ValidateRefreshToken(refreshToken)
	if err != nil {
		return response.FailWithCode(response.CodeUnauthorized, "Refresh Token 无效或已过期")
	}

	// 2. 重新查询用户信息（对齐 Java sysUserMapper.selectById）
	user, err := s.adminUserMapper.SelectByID(claims.UserID)
	if err != nil {
		log.Printf("刷新Token时查询用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.FailWithCode(response.CodeUnauthorized, "用户不存在")
	}

	// 3. 校验用户状态（对齐 Java sysUser.getStatus() != 1）
	if user.Status != 1 {
		return response.FailWithCode(response.CodeUnauthorized, "账号已被禁用")
	}

	// 4. 签发新的双 Token（对齐 Java 签发新 refreshToken，而非返回原 token）
	accessToken, err := s.jwtUtils.CreateAccessToken(claims.UserID, claims.RoleID, claims.OpenID)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	newRefreshToken, err := s.jwtUtils.CreateRefreshToken(claims.UserID, claims.RoleID, claims.OpenID)
	if err != nil {
		return response.FailWithCode(response.CodeServerError, "生成 Token 失败")
	}

	// 5. 查询角色ID（对齐 Java getRoleIdsByUserId）
	// 查询失败不阻断刷新，降级为空数组
	roleIds, err := s.adminUserMapper.SelectRoleIDsByUserID(user.ID)
	if err != nil {
		log.Printf("查询角色ID失败: %v", err)
		roleIds = []int64{}
	}

	// 6. 构造完整 userInfo（非 nil，对齐 Java 返回完整用户信息）
	userInfo := ToSysUserVO(user, roleIds)

	return response.Success(&LoginVO{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		UserInfo:     userInfo,
	})
}

// GetUserInfo 查询用户信息
//
// 返回 SysUserVO（避免 sql.NullTime 序列化为对象，含 roleIds）
func (s *AdminService) GetUserInfo(userID int64) *response.ResponseDTO {
	user, err := s.adminUserMapper.SelectByID(userID)
	if err != nil {
		log.Printf("查询用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}
	// 查询角色ID（对齐 Java getRoleIdsByUserId），查询失败降级为空数组
	roleIds, err := s.adminUserMapper.SelectRoleIDsByUserID(user.ID)
	if err != nil {
		log.Printf("查询角色ID失败: %v", err)
		roleIds = []int64{}
	}
	// 转换为 VO，隐藏密码并格式化时间
	return response.Success(ToSysUserVO(user, roleIds))
}

// GetUserListRequest 用户列表查询请求（对齐 admin 前端 GetUserListRequest）
//
// admin 前端 src/api/user/index.ts getUserList 请求类型
type GetUserListRequest struct {
	Username    string `json:"username"`    // 用户名（模糊查询，可选）
	Phone       string `json:"phone"`       // 手机号（模糊查询，可选）
	Status      int64  `json:"status"`      // 状态（0=禁用,1=启用，0表示不筛选）
	CurrentPage int    `json:"currentPage"` // 当前页码（从1开始）
	PageSize    int    `json:"pageSize"`    // 每页条数
}

// GetUserList 查询用户列表（分页）
//
// 对齐 admin 前端 src/api/user/index.ts getUserList
// 路径：POST /admin/user/list
//
// 参数：
//   - req: 查询请求（含筛选条件+分页）
//
// 返回：{ list: SysUserVO[], total: int64 }
func (s *AdminService) GetUserList(req *GetUserListRequest) *response.ResponseDTO {
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	// 调用 Mapper 查询（带筛选条件）
	list, err := s.adminUserMapper.SelectList(req.Username, req.Phone, req.Status, offset, req.PageSize)
	if err != nil {
		log.Printf("查询用户列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	total, err := s.adminUserMapper.CountWithFilter(req.Username, req.Phone, req.Status)
	if err != nil {
		log.Printf("统计用户数失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 转换为 VO 列表（隐藏密码、格式化时间，每个用户单独查询 roleIds）
	voList := make([]*SysUserVO, 0, len(list))
	for _, u := range list {
		// 查询每个用户的角色ID（对齐 Java getRoleIdsByUserId），查询失败降级为空数组
		roleIds, err := s.adminUserMapper.SelectRoleIDsByUserID(u.ID)
		if err != nil {
			log.Printf("查询用户(ID=%d)角色ID失败: %v", u.ID, err)
			roleIds = []int64{}
		}
		if vo := ToSysUserVO(u, roleIds); vo != nil {
			voList = append(voList, vo)
		}
	}

	// 响应格式对齐 admin 前端 PageData<SysUserResponse>
	return response.Success(map[string]interface{}{
		"list":  voList,
		"total": total,
	})
}

// ============================================================
// 用户管理写操作（对齐 Java SysUserServiceImpl 的 CRUD 方法）
// ============================================================

// InsertUserRequest 新增用户请求（对齐 admin 前端 InsertUserRequest）
//
// admin 前端 src/types/admin.d.ts InsertUserRequest
// password 字段为前端 SM2 公钥加密后的密文（对齐 Java InsertSysUserDTO.password）
type InsertUserRequest struct {
	Username string  `json:"username"` // 用户名（必填，唯一）
	Nickname string  `json:"nickname"` // 昵称（必填）
	Password string  `json:"password"` // 密码（SM2 加密密文，必填）
	Phone    string  `json:"phone"`    // 手机号（可选）
	Email    string  `json:"email"`    // 邮箱（可选）
	Status   int64   `json:"status"`   // 状态（0=禁用,1=启用，默认1）
	Remark   string  `json:"remark"`   // 备注（可选）
	RoleIDs  []int64 `json:"roleIds"`  // 角色ID列表（可选）
}

// UpdateUserRequest 更新用户请求（对齐 admin 前端 UpdateUserRequest）
//
// admin 前端 src/types/admin.d.ts UpdateUserRequest
// 仅更新提供的字段（对齐 Java UpdateSysUserDTO 的动态更新）
type UpdateUserRequest struct {
	ID       int64   `json:"id"`       // 用户ID（必填）
	Nickname string  `json:"nickname"` // 昵称（可选）
	Phone    string  `json:"phone"`    // 手机号（可选）
	Email    string  `json:"email"`    // 邮箱（可选）
	Status   int64   `json:"status"`   // 状态（可选）
	Remark   string  `json:"remark"`   // 备注（可选）
	RoleIDs  []int64 `json:"roleIds"`  // 角色ID列表（可选，提供时覆盖原有关联）
}

// ResetPasswordRequest 重置密码请求（对齐 admin 前端 ResetPasswordRequest）
//
// admin 前端 src/types/admin.d.ts ResetPasswordRequest
// password 字段为前端 SM2 公钥加密后的密文
// 注意：Java 后端使用 newPassword 字段名，前端使用 password，此处兼容两者
type ResetPasswordRequest struct {
	ID          int64  `json:"id"`          // 用户ID（必填）
	Password    string `json:"password"`    // 新密码（SM2 加密密文，前端字段名）
	NewPassword string `json:"newPassword"` // 新密码（SM2 加密密文，Java 后端字段名，兼容）
}

// GetUserByID 按 ID 查系统用户
//
// 对齐 Java SysUserServiceImpl.getUserById
//
// 参数：
//   - id: 用户ID
//
// 返回：SysUserVO
func (s *AdminService) GetUserByID(id int64) *response.ResponseDTO {
	user, err := s.adminUserMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询用户失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}
	// 查询角色ID（对齐 Java getRoleIdsByUserId），查询失败降级为空数组
	roleIds, err := s.adminUserMapper.SelectRoleIDsByUserID(user.ID)
	if err != nil {
		log.Printf("查询角色ID失败: %v", err)
		roleIds = []int64{}
	}
	return response.Success(ToSysUserVO(user, roleIds))
}

// InsertUser 新增系统用户
//
// 对齐 Java SysUserServiceImpl.insertUser
//
// 流程：
//  1. 校验用户名唯一性
//  2. SM2 解密前端加密的密码
//  3. BCrypt 哈希密码（Go 端使用 BCrypt，Java 端使用 SM3+salt）
//  4. 插入用户记录
//  5. 保存用户角色关联（如有）
//
// 参数：
//   - req: 新增用户请求
//
// 返回：SysUserVO（含新用户ID）
func (s *AdminService) InsertUser(req *InsertUserRequest) *response.ResponseDTO {
	// 1. 参数校验
	if req.Username == "" || req.Nickname == "" || req.Password == "" {
		return response.Fail("用户名、昵称和密码不能为空")
	}

	// 2. 校验用户名唯一性（对齐 Java sysUserMapper.selectCount eq username）
	count, err := s.adminUserMapper.CountByUsername(req.Username)
	if err != nil {
		log.Printf("查询用户名唯一性失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if count > 0 {
		return response.Fail("用户名已存在")
	}

	// 3. SM2 解密密码（对齐 Java SM2Util.decrypt）
	// 前端使用 SM2 公钥加密密码，后端用私钥解密得到明文
	rawPassword, err := crypto.SM2Decrypt(req.Password, s.sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密密码失败: %v", err)
		return response.Fail("密码解密失败，请检查加密参数")
	}

	// 4. BCrypt 哈希密码（Go 端使用 BCrypt，对齐 Java spring-security BCrypt 互通）
	// bcrypt.DefaultCost=10，与 Java BCryptPasswordEncoder 默认强度一致
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("BCrypt 哈希密码失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "密码加密失败")
	}

	// 5. 构造用户实体并插入（状态默认1=启用）
	status := req.Status
	if status == 0 {
		status = 1 // 默认启用（对齐 Java dto.getStatus() != null ? dto.getStatus() : 1）
	}
	user := &mapper.AdminUser{
		Username: req.Username,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Email:    req.Email,
		Status:   status,
		Remark:   req.Remark,
	}
	userID, err := s.adminUserMapper.Insert(user)
	if err != nil {
		log.Printf("新增用户失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增用户失败")
	}
	user.ID = userID

	// 6. 保存用户角色关联（对齐 Java saveUserRoles）
	if len(req.RoleIDs) > 0 {
		for _, roleID := range req.RoleIDs {
			if err := s.adminUserMapper.InsertUserRole(userID, roleID); err != nil {
				// 角色关联失败不阻断主流程，记录日志
				log.Printf("保存用户角色关联失败: userID=%d, roleID=%d, err=%v", userID, roleID, err)
			}
		}
	}

	// 7. 返回新用户 VO（对齐 Java sysUserConverter.pojoToVO）
	roleIds := req.RoleIDs
	if roleIds == nil {
		roleIds = []int64{}
	}
	return response.Success(ToSysUserVO(user, roleIds))
}

// UpdateUser 更新系统用户
//
// 对齐 Java SysUserServiceImpl.updateUser
//
// 流程：
//  1. 校验用户存在
//  2. 更新用户字段（nickname/phone/email/status/remark）
//  3. 如提供 roleIDs，先删除旧关联再插入新关联
//
// 参数：
//   - req: 更新用户请求
//
// 返回：SysUserVO
func (s *AdminService) UpdateUser(req *UpdateUserRequest) *response.ResponseDTO {
	// 1. 校验用户存在（对齐 Java sysUserMapper.selectById）
	user, err := s.adminUserMapper.SelectByID(req.ID)
	if err != nil {
		log.Printf("查询用户失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}

	// 2. 更新字段（仅更新非空字段，对齐 Java dto.getXxx() != null 判断）
	// 注意：Go 端无法区分"未提供"和"零值"，这里按前端约定：
	//   - 字符串字段：空字符串表示不更新（前端表单不会清空必填字段）
	//   - 数值字段：0表示不更新（status=0=禁用是有效值，但前端更新时会显式传1或2）
	// 为简化处理，全字段更新（与 Java updateById 行为一致）
	updateUser := &mapper.AdminUser{
		ID:       req.ID,
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Email:    req.Email,
		Status:   req.Status,
		Remark:   req.Remark,
	}
	// 如果 status 为 0 且原用户状态非 0，保留原状态（避免误禁用）
	// 对齐 Java dto.getStatus() != null 判断（Go 无法区分 nil 和 0，此处保留原值）
	if req.Status == 0 {
		updateUser.Status = user.Status
	}
	if err := s.adminUserMapper.Update(updateUser); err != nil {
		log.Printf("更新用户失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "更新用户失败")
	}

	// 3. 重新分配角色（对齐 Java sysUserRoleMapper.delete + saveUserRoles）
	// 仅当 req.RoleIDs 非 nil 时更新（nil 表示不修改角色）
	if req.RoleIDs != nil {
		// 先删除旧关联
		if err := s.adminUserMapper.DeleteUserRolesByUserID(req.ID); err != nil {
			log.Printf("删除用户旧角色关联失败: userID=%d, err=%v", req.ID, err)
		}
		// 再插入新关联
		for _, roleID := range req.RoleIDs {
			if err := s.adminUserMapper.InsertUserRole(req.ID, roleID); err != nil {
				log.Printf("保存用户角色关联失败: userID=%d, roleID=%d, err=%v", req.ID, roleID, err)
			}
		}
	}

	// 4. 重新查询用户信息返回（确保返回最新数据）
	updatedUser, err := s.adminUserMapper.SelectByID(req.ID)
	if err != nil || updatedUser == nil {
		// 查询失败时使用 updateUser 构造 VO
		updatedUser = user
	}
	roleIds := req.RoleIDs
	if roleIds == nil {
		// 未更新角色时查询现有角色
		roleIds, err = s.adminUserMapper.SelectRoleIDsByUserID(req.ID)
		if err != nil {
			roleIds = []int64{}
		}
	}
	return response.Success(ToSysUserVO(updatedUser, roleIds))
}

// DeleteUser 删除系统用户
//
// 对齐 Java SysUserServiceImpl.deleteUser
//
// 流程：
//  1. 校验用户存在
//  2. 逻辑删除用户（is_deleted=1）
//  3. 删除用户角色关联
//
// 参数：
//   - id: 用户ID
//
// 返回：操作结果消息
func (s *AdminService) DeleteUser(id int64) *response.ResponseDTO {
	// 1. 校验用户存在
	user, err := s.adminUserMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询用户失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}

	// 2. 逻辑删除用户（对齐 Java sysUserMapper.deleteById @TableLogic）
	if err := s.adminUserMapper.Delete(id); err != nil {
		log.Printf("删除用户失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "删除用户失败")
	}

	// 3. 删除用户角色关联（对齐 Java sysUserRoleMapper.delete eq userId）
	// 注意：Java 代码未显式删除关联，但为避免脏数据，Go 端主动清理
	if err := s.adminUserMapper.DeleteUserRolesByUserID(id); err != nil {
		// 关联清理失败不阻断主流程，记录日志
		log.Printf("删除用户角色关联失败: userID=%d, err=%v", id, err)
	}

	return response.Success("删除成功")
}

// ResetPassword 重置用户密码
//
// 对齐 Java SysUserServiceImpl.resetPassword
//
// 流程：
//  1. 校验用户存在
//  2. SM2 解密前端加密的新密码
//  3. BCrypt 哈希新密码
//  4. 更新密码
//
// 参数：
//   - req: 重置密码请求（兼容 password 和 newPassword 字段名）
//
// 返回：操作结果消息
func (s *AdminService) ResetPassword(req *ResetPasswordRequest) *response.ResponseDTO {
	// 1. 校验用户存在
	user, err := s.adminUserMapper.SelectByID(req.ID)
	if err != nil {
		log.Printf("查询用户失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if user == nil {
		return response.Fail("用户不存在")
	}

	// 2. 获取密码（兼容 password 和 newPassword 字段名，对齐前端与 Java 后端差异）
	encryptedPassword := req.Password
	if encryptedPassword == "" {
		encryptedPassword = req.NewPassword
	}
	if encryptedPassword == "" {
		return response.Fail("新密码不能为空")
	}

	// 3. SM2 解密新密码（对齐 Java SM2Util.decrypt）
	rawPassword, err := crypto.SM2Decrypt(encryptedPassword, s.sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密密码失败: %v", err)
		return response.Fail("密码解密失败，请检查加密参数")
	}

	// 4. BCrypt 哈希新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("BCrypt 哈希密码失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "密码加密失败")
	}

	// 5. 更新密码（对齐 Java sysUserMapper.updateById）
	if err := s.adminUserMapper.ResetPassword(req.ID, string(hashedPassword)); err != nil {
		log.Printf("重置密码失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "重置密码失败")
	}

	return response.Success("密码重置成功")
}

// GetUserRoles 查询用户角色列表
//
// 对齐 Java SysUserServiceImpl.getUserRoles
//
// 注意：前端实际使用方式为 res.data.map(r => r.id) 提取 roleIds
// 因此前端期望返回角色对象数组（List<SysRole>），而非 { roleIds: [...] }
// 与 Java 后端行为一致，返回角色实体列表
//
// 参数：
//   - userID: 用户ID
//
// 返回：角色 VO 列表
func (s *AdminService) GetUserRoles(userID int64) *response.ResponseDTO {
	roles, err := s.adminUserMapper.SelectRolesByUserID(userID)
	if err != nil {
		log.Printf("查询用户角色失败: userID=%d, err=%v", userID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	// 转换为角色 VO 列表（含 menuIds）
	voList := make([]*SysRoleVO, 0, len(roles))
	for _, role := range roles {
		voList = append(voList, ToSysRoleVO(role, nil))
	}
	return response.Success(voList)
}
