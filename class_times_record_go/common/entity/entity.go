// Package entity 数据库实体定义
//
// 对齐 Java com.shiroko.repository.entity 包
//
// 表命名规范：
//   - 业务表前缀 c_（如 c_user, c_parent, c_student）
//   - 系统/管理表前缀 sys_（如 sys_user, sys_role）
//
// 实体继承体系（对齐 Java）：
//   - BaseEntity（空基类）
//   - RoleBaseEntity（继承 BaseEntity，含 userId/isAvailable/username）
//   - Parent / Teacher（继承 RoleBaseEntity）
//   - Student（继承 BaseEntity）
//   - User / UserAuth / UserPlatform / Institution（直接实现）
package entity

import (
	"database/sql"
	"time"
)

// ============================================================
// 基础实体（对齐 Java BaseEntity / RoleBaseEntity）
// ============================================================

// BaseEntity 基础实体（对齐 Java BaseEntity，空基类）
//
// Java 中是抽象类，Go 用空 struct 表示
type BaseEntity struct{}

// RoleBaseEntity 角色基础实体（对齐 Java RoleBaseEntity）
//
// 含用户关联字段，Parent 和 Teacher 继承此类
type RoleBaseEntity struct {
	UserID      sql.NullInt64  `json:"userId"`      // 关联 c_user.id
	IsAvailable sql.NullBool   `json:"isAvailable"` // 是否可用
	Username    sql.NullString `json:"username"`    // 用户名
}

// ============================================================
// 用户相关实体
// ============================================================

// User 用户实体（表 c_user，对齐 Java User.java）
//
// 一个 User 对应一个机构下的用户记录，不同机构的家长是不同 User
type User struct {
	ID            int64          `json:"id"`            // 主键
	InstitutionID sql.NullInt64  `json:"institutionId"` // 机构ID（c_institution.id）
	CreateTime    sql.NullTime   `json:"createTime"`    // 创建时间
	UpdateTime    sql.NullTime   `json:"updateTime"`    // 更新时间
}

// UserAuth 用户认证实体（表 c_user_auth，对齐 Java UserAuth.java）
//
// 存储账号密码，role_id 实际关联 c_permission.id（1=admin,3=parent,4=teacher,5=student）
type UserAuth struct {
	ID           int64          `json:"id"`           // 主键
	UserID       sql.NullInt64  `json:"userId"`       // 关联 c_user.id
	RoleID       sql.NullInt64  `json:"roleId"`       // 角色ID（关联 c_permission.id）
	Account      sql.NullString `json:"account"`      // 账号（手机号或用户名）
	Password     sql.NullString `json:"password"`     // 密码（SM3 加盐哈希）
	Salt         sql.NullString `json:"salt"`         // 盐值（32 位 UUID 去横杠）
	LastLoginTime sql.NullTime  `json:"lastLoginTime"` // 最后登录时间
}

// UserPlatform 用户平台实体（表 c_user_platform，对齐 Java UserPlatform.java）
//
// 多设备登录表：一个 User 可有多个平台记录（每个 openId 一条）
// lastLoginRole 用于区分同 openId 不同角色（如既是教师又是家长）
type UserPlatform struct {
	ID            int64          `json:"id"`            // 主键
	UserID        sql.NullInt64  `json:"userId"`        // 关联 c_user.id
	OpenID        sql.NullString `json:"openId"`        // 平台用户唯一ID（微信 openId）
	UnionID       sql.NullString `json:"unionId"`       // 跨应用唯一ID（微信 unionId）
	LastLoginTime sql.NullTime   `json:"lastLoginTime"` // 最后登录时间
	LastLoginRole sql.NullInt64  `json:"lastLoginRole"` // 最后登录角色（3=parent, 4=teacher）
	Platform      sql.NullString `json:"platform"`      // 平台标识（"WEIXIN"）
	IsAvailable   sql.NullBool   `json:"isAvailable"`   // 是否可用
	CreateTime    sql.NullTime   `json:"createTime"`    // 创建时间
}

// ============================================================
// 角色实体
// ============================================================

// Parent 家长实体（表 c_parent，对齐 Java Parent.java）
//
// 继承 RoleBaseEntity（含 userId/isAvailable/username）
// isBound=false 表示占位记录（教师创建联系人时生成，待家长绑定）
type Parent struct {
	RoleBaseEntity                                                                  // 继承：userId/isAvailable/username
	ParentID    sql.NullInt64 `json:"parentId"`                                     // 主键（注意字段名是 parentId，对应表 id 列）
	Phone       sql.NullString `json:"phone"`                                       // 手机号
	IsBound     sql.NullBool   `json:"isBound"`                                     // 是否已绑定微信用户（false=占位符）
	CreateTime  sql.NullTime   `json:"createTime"`                                  // 创建时间
	UpdateTime  sql.NullTime   `json:"updateTime"`                                  // 更新时间
}

// Teacher 教师实体（表 c_teacher，对齐 Java Teacher.java）
//
// 继承 RoleBaseEntity（含 userId/isAvailable/username）
// isInstitutionAdmin 标识机构管理员（区别于系统管理员 sys_user）
type Teacher struct {
	RoleBaseEntity                                                                  // 继承：userId/isAvailable/username
	TeacherID         sql.NullInt64  `json:"teacherId"`                             // 主键（注意字段名是 teacherId，对应表 id 列）
	InstitutionID     sql.NullInt64  `json:"institutionId"`                         // 机构ID
	IsInstitutionAdmin sql.NullBool  `json:"isInstitutionAdmin"`                    // 是否机构管理员（0=否,1=是）
	Phone             sql.NullString `json:"phone"`                                 // 手机号
}

// Student 学生实体（表 c_student，对齐 Java Student.java）
//
// 继承 BaseEntity（不继承 RoleBaseEntity，因学生不直接登录）
type Student struct {
	BaseEntity                                                                       // 继承空基类
	ID            sql.NullInt64  `json:"id"`                                         // 主键
	Avatar        sql.NullString `json:"avatar"`                                     // 头像URL
	StudentName   sql.NullString `json:"studentName"`                                // 学生姓名
	InstitutionID sql.NullInt64  `json:"institutionId"`                              // 机构ID
	Sex           sql.NullInt64  `json:"sex"`                                        // 性别（0=未知,1=男,2=女）
	Birth         sql.NullTime   `json:"birth"`                                      // 出生日期
	School        sql.NullString `json:"school"`                                     // 学校
	Address       sql.NullString `json:"address"`                                    // 地址
	CreateTime    sql.NullTime   `json:"createTime"`                                 // 创建时间
	UpdateTime    sql.NullTime   `json:"updateTime"`                                 // 更新时间
}

// ============================================================
// 机构实体
// ============================================================

// Institution 机构实体（表 c_institution，对齐 Java Institution.java）
//
// expireTime 为空表示永久有效；非空且已过期则禁止登录和绑定
type Institution struct {
	ID                   sql.NullInt64  `json:"id"`                   // 主键
	InstitutionName      sql.NullString `json:"institutionName"`      // 机构名称
	InstitutionAddress   sql.NullString `json:"institutionAddress"`   // 机构地址
	InstitutionCode      sql.NullString `json:"institutionCode"`      // 机构编码（用于登录）
	Status               sql.NullInt64  `json:"status"`               // 状态（0=待审核,1=启用,2=禁用）
	ExpireTime           sql.NullTime   `json:"expireTime"`           // 过期时间（NULL=永久有效）
	SubscriptionPlanID   sql.NullInt64  `json:"subscriptionPlanId"`   // 订阅计划ID
	SubscriptionPlanName sql.NullString `json:"subscriptionPlanName"` // 订阅计划名（JOIN 获取，非数据库字段）
	CreateTime           sql.NullTime   `json:"createTime"`           // 创建时间
	UpdateTime           sql.NullTime   `json:"updateTime"`           // 更新时间
}

// ============================================================
// 微信订阅相关实体
// ============================================================

// WxSubscribeRecord 微信订阅记录（表 c_wx_subscribe_record，对齐 Java WxSubscribeRecord.java）
//
// 按 (openId, templateId) 维度跟踪剩余推送次数
// 唯一约束：open_id + template_id
type WxSubscribeRecord struct {
	ID             sql.NullInt64  `json:"id"`             // 主键
	OpenID         sql.NullString `json:"openId"`         // 维度1：微信 openId
	TemplateID     sql.NullString `json:"templateId"`     // 维度2：模板ID
	SubscribeCount sql.NullInt64  `json:"subscribeCount"` // 剩余推送次数（授权+1，推送-1）
	IsPermanent    sql.NullBool   `json:"isPermanent"`    // 永久订阅标记
	CreateTime     sql.NullTime   `json:"createTime"`     // 创建时间
	UpdateTime     sql.NullTime   `json:"updateTime"`     // 更新时间
}

// WxStudentSubscribe 学生订阅关系（表 c_wx_student_subscribe，对齐 Java WxStudentSubscribe.java）
//
// 按 (openId, studentId) 维度跟踪订阅关系
// 与 c_parent_student（按 parent_id 维度）解耦，便于按 openId 查询
type WxStudentSubscribe struct {
	ID         sql.NullInt64  `json:"id"`         // 主键
	OpenID     sql.NullString `json:"openId"`     // 维度1：微信 openId
	StudentID  sql.NullInt64  `json:"studentId"`  // 维度2：学生ID
	IsPrimary  sql.NullBool   `json:"isPrimary"`  // 1=主联系人,0=次联系人
	BindMode   sql.NullString `json:"bindMode"`   // "subscribe"=仅订阅, "full"=绑定账号并订阅
	CreateTime sql.NullTime   `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime   `json:"updateTime"` // 更新时间
}

// ParentStudent 家长-学生关联（表 c_parent_student）
//
// 一个家长可关联多个学生，一个学生可有多个家长（主/次联系人）
// 唯一约束：(parent_id, student_id) 和 (student_id, is_primary)
type ParentStudent struct {
	ID         sql.NullInt64  `json:"id"`         // 主键
	ParentID   sql.NullInt64  `json:"parentId"`   // 家长ID（c_parent.id）
	StudentID  sql.NullInt64  `json:"studentId"`  // 学生ID（c_student.id）
	IsPrimary  sql.NullBool   `json:"isPrimary"`  // 是否主联系人（1=是,0=否）
	Relation   sql.NullString `json:"relation"`   // 关系（如"父亲"、"母亲"）
	CreateTime sql.NullTime   `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime   `json:"updateTime"` // 更新时间
}

// ============================================================
// 时间处理辅助
// ============================================================

// FormatTime 格式化时间（对齐 Java DateTimeFormatter "yyyy-MM-dd HH:mm:ss"）
//
// 参数：
//   - t: sql.NullTime，有效时格式化
//
// 返回：格式化字符串，无效返回空字符串
func FormatTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

// NowTime 返回当前时间（用于 createTime/updateTime 赋值）
func NowTime() time.Time {
	return time.Now()
}
