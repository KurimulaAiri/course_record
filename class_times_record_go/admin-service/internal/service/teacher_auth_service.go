// Package service admin-service 教师账号管理业务逻辑层
//
// 对齐 Java admin-service TeacherAuthServiceImpl
// 操作 c_teacher 表的账号相关字段（username, password, is_institution_admin）
//
// 注意：教师账号管理（teacher_auth）与系统管理员(admin表)是不同身份
//   - 教师账号：c_user_auth 表（role_id=4），密码使用 SM3+salt 哈希
//   - 系统管理员：sys_user 表，密码使用 BCrypt 哈希
//
// 涵盖接口：
//   - /teacher_auth/get：查询教师账号信息
//   - /teacher_auth/update_account：更新教师登录账号
//   - /teacher_auth/update_password：修改教师密码
//   - /teacher_auth/toggle_institution_admin：切换机构管理员身份
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/crypto"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// DTO 定义
// ============================================================

// TeacherAuthRequest 教师账号查询请求
type TeacherAuthRequest struct {
	TeacherID int64 `json:"teacherId"` // 教师ID（必填）
}

// UpdateTeacherAccountRequest 更新教师账号请求（对齐 admin 前端 UpdateTeacherAccountRequest）
type UpdateTeacherAccountRequest struct {
	TeacherID int64  `json:"teacherId"` // 教师ID（必填）
	Account   string `json:"account"`   // 新账号（必填）
}

// UpdateTeacherPasswordRequest 更新教师密码请求（对齐 admin 前端 UpdateTeacherPasswordRequest）
type UpdateTeacherPasswordRequest struct {
	TeacherID int64  `json:"teacherId"` // 教师ID（必填）
	Password  string `json:"password"`  // 新密码（SM2 加密密文，必填）
}

// ToggleInstitutionAdminRequest 切换机构管理员身份请求
type ToggleInstitutionAdminRequest struct {
	TeacherID          int64 `json:"teacherId"`          // 教师ID（必填）
	IsInstitutionAdmin bool  `json:"isInstitutionAdmin"` // true=设为机构管理员, false=取消
}

// ============================================================
// TeacherAuthService 教师账号管理服务
// ============================================================

// TeacherAuthService 教师账号管理服务（对齐 Java TeacherAuthServiceImpl）
//
// 注入：
//   - bizMapper：业务管理 Mapper（含教师账号相关查询方法）
//   - sm2PrivateKey：SM2 私钥（用于解密前端加密的新密码）
//   - logService：操作日志服务
type TeacherAuthService struct {
	bizMapper    *mapper.AdminBusinessMapper
	sm2PrivateKey string // SM2 私钥（hex 编码，用于解密前端加密的密码）
	logService   *SysOperationLogService
}

// NewTeacherAuthService 创建 TeacherAuthService
//
// 参数：
//   - bizMapper: 业务管理 Mapper
//   - sm2PrivateKey: SM2 私钥（hex 编码）
//   - logService: 操作日志服务
func NewTeacherAuthService(bizMapper *mapper.AdminBusinessMapper, sm2PrivateKey string, logService *SysOperationLogService) *TeacherAuthService {
	return &TeacherAuthService{
		bizMapper:     bizMapper,
		sm2PrivateKey: sm2PrivateKey,
		logService:    logService,
	}
}

// GetTeacherAuth 查询教师账号信息
//
// 对齐 Java getTeacherAuth
// 返回 user_auth 表中的账号、用户ID、最后登录时间
//
// 参数：
//   - teacherID: 教师ID
//
// 返回：{ id, userId, account, lastLoginTime }
func (s *TeacherAuthService) GetTeacherAuth(teacherID int64) *response.ResponseDTO {
	if teacherID == 0 {
		return response.Fail("教师ID不能为空")
	}
	auth, err := s.bizMapper.SelectUserAuthByTeacherID(teacherID)
	if err != nil {
		log.Printf("查询教师账号信息失败: teacherID=%d, err=%v", teacherID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if auth == nil {
		return response.Fail("未找到该教师的账号信息")
	}
	return response.Success(map[string]interface{}{
		"id":            auth.ID,
		"userId":        auth.UserID,
		"account":       auth.Account,
		"lastLoginTime": auth.LastLoginTime,
	})
}

// UpdateTeacherAccount 更新教师登录账号
//
// 对齐 Java updateTeacherAccount
// 流程：
//   1. 查询教师当前账号信息
//   2. 如果账号发生变化，校验同机构同角色下账号唯一性
//   3. 更新 c_user_auth.account
//
// 参数：
//   - req: 更新账号请求
func (s *TeacherAuthService) UpdateTeacherAccount(req *UpdateTeacherAccountRequest) *response.ResponseDTO {
	if req.TeacherID == 0 {
		return response.Fail("教师ID不能为空")
	}
	if req.Account == "" {
		return response.Fail("新账号不能为空")
	}
	// 1. 查询当前账号信息
	auth, err := s.bizMapper.SelectUserAuthByTeacherID(req.TeacherID)
	if err != nil {
		log.Printf("查询教师账号信息失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if auth == nil {
		return response.Fail("未找到该教师的账号信息")
	}
	// 2. 账号变化时校验唯一性
	if auth.Account != req.Account {
		// 查询用户所属机构ID
		institutionID, err := s.bizMapper.SelectUserInstitutionID(auth.UserID)
		if err != nil {
			log.Printf("查询用户机构ID失败: %v", err)
		}
		if institutionID != 0 {
			// 校验同机构同角色（role_id=4 教师）下账号是否已存在
			exists, err := s.bizMapper.ExistsUserAuthByInstitutionAndAccount(institutionID, req.Account, 4)
			if err != nil {
				log.Printf("校验账号唯一性失败: %v", err)
				return response.FailWithCode(response.CodeServerError, "系统异常")
			}
			if exists {
				return response.Fail("该机构下已存在相同账号的教师")
			}
		}
	}
	// 3. 更新账号
	if err := s.bizMapper.UpdateUserAuthAccount(auth.ID, req.Account); err != nil {
		log.Printf("更新教师账号失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "更新账号失败")
	}
	return response.Success("账号修改成功")
}

// UpdateTeacherPassword 修改教师密码
//
// 对齐 Java updateTeacherPassword
// 流程：
//   1. 查询教师当前账号信息
//   2. SM2 解密前端传来的新密码
//   3. 生成随机盐值，SM3+salt 哈希密码
//   4. 更新 c_user_auth.password 和 salt
//
// 参数：
//   - req: 更新密码请求（Password 为 SM2 加密密文）
func (s *TeacherAuthService) UpdateTeacherPassword(req *UpdateTeacherPasswordRequest) *response.ResponseDTO {
	if req.TeacherID == 0 {
		return response.Fail("教师ID不能为空")
	}
	if req.Password == "" {
		return response.Fail("新密码不能为空")
	}
	// 1. 查询当前账号信息
	auth, err := s.bizMapper.SelectUserAuthByTeacherID(req.TeacherID)
	if err != nil {
		log.Printf("查询教师账号信息失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if auth == nil {
		return response.Fail("未找到该教师的账号信息")
	}
	// 2. SM2 解密前端传来的新密码（对齐 Java SM2Util.decrypt）
	rawPassword, err := crypto.SM2Decrypt(req.Password, s.sm2PrivateKey)
	if err != nil {
		log.Printf("SM2 解密密码失败: %v", err)
		return response.Fail("密码解密失败，请检查加密参数")
	}
	// 3. 生成随机盐值，SM3+salt 哈希密码（对齐 Java SM3Util.digestWithSalt）
	salt := crypto.GenerateSalt()
	hashedPassword := crypto.SM3DigestWithSalt(rawPassword, salt)
	// 4. 更新密码和盐值
	if err := s.bizMapper.UpdateUserAuthPassword(auth.ID, hashedPassword, salt); err != nil {
		log.Printf("更新教师密码失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "密码修改失败")
	}
	return response.Success("密码修改成功")
}

// ToggleInstitutionAdmin 切换教师机构管理员身份
//
// 对齐 Java toggleTeacherAdmin
// 仅修改 c_teacher 表的 is_institution_admin 字段
// 机构管理员与系统管理员(admin表)是不同身份，不涉及 admin 表和 user_auth 表
//
// 参数：
//   - req: 切换请求
func (s *TeacherAuthService) ToggleInstitutionAdmin(req *ToggleInstitutionAdminRequest) *response.ResponseDTO {
	if req.TeacherID == 0 {
		return response.Fail("教师ID不能为空")
	}
	// 1. 查询教师当前状态
	teacher, err := s.bizMapper.SelectTeacherByID(req.TeacherID)
	if err != nil {
		log.Printf("查询教师失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if teacher == nil {
		return response.Fail("未找到该教师")
	}
	// 2. 校验当前状态是否与目标状态一致
	if teacher.IsInstitutionAdmin == req.IsInstitutionAdmin {
		if req.IsInstitutionAdmin {
			return response.Fail("该教师已是机构管理员")
		}
		return response.Fail("该教师不是机构管理员")
	}
	// 3. 更新机构管理员标识
	if err := s.bizMapper.UpdateTeacherInstitutionAdmin(req.TeacherID, req.IsInstitutionAdmin); err != nil {
		log.Printf("更新机构管理员标识失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "切换失败")
	}
	// 返回成功消息
	if req.IsInstitutionAdmin {
		return response.Success("已设为机构管理员")
	}
	return response.Success("已取消机构管理员")
}
