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
// 菜单实体
// ============================================================

// Menu 菜单实体（表 c_menu，对齐 Java Menu.java）
//
// 按 role（c_role_menu.permission_id）关联角色，控制小程序端不同角色可见的菜单项
// iconType: 0=uni-app内置图标（icon 填图标名）, 1=独立图标路径（icon 填图片src）
type Menu struct {
	ID         sql.NullInt64  `json:"id"`         // 菜单ID
	MenuName   sql.NullString `json:"menuName"`   // 菜单名称
	Icon       sql.NullString `json:"icon"`       // 图标（名称或路径，取决于 iconType）
	IconType   sql.NullInt64  `json:"iconType"`   // 图标类型（0=内置, 1=路径）
	BgColor    sql.NullString `json:"bgColor"`    // 图标背景色（Hex）
	Path       sql.NullString `json:"path"`       // 跳转路由路径
	SortOrder  sql.NullInt64  `json:"sortOrder"`  // 排序权值（越小越靠前）
	IsVisible  sql.NullBool   `json:"isVisible"`  // 是否显示（1=显示, 0=隐藏）
	CreateTime sql.NullTime   `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime   `json:"updateTime"` // 更新时间
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
// 班级与课程相关实体
// ============================================================

// Class 班级实体（表 c_class，对齐 Java Class.java）
//
// Java 中使用 clazz 包名避开关键字；Go 中 Class 无冲突
// 一个班级关联一个课程（courseId），通过 c_class_teacher/c_class_student 关联表关联教师和学生
type Class struct {
	ID              sql.NullInt64  `json:"id"`              // 主键
	CourseID        sql.NullInt64  `json:"courseId"`        // 班级对应的课程ID（c_course.id）
	ClassName       sql.NullString `json:"className"`       // 班级名称
	Status          sql.NullInt64  `json:"status"`          // 班级状态
	StudentCount    sql.NullInt64  `json:"studentCount"`    // 班级学生人数
	StudentMaxCount sql.NullInt64  `json:"studentMaxCount"` // 班级最大人数
	CreateTime      sql.NullTime   `json:"createTime"`      // 创建时间
	UpdateTime      sql.NullTime   `json:"updateTime"`      // 更新时间
}

// ClassSchedule 班级课表实体（表 c_class_schedule，对齐 Java ClassSchedule.java）
//
// 定义班级的上课时间段，支持重复排课（dayOfWeek 1-7 对应周一到周日）
type ClassSchedule struct {
	ID         sql.NullInt64  `json:"id"`         // 主键
	ClassID    sql.NullInt64  `json:"classId"`    // 关联的班级ID（c_class.id）
	StartDate  sql.NullTime   `json:"startDate"`  // 时间段开始日期
	EndDate    sql.NullTime   `json:"endDate"`    // 时间段结束日期
	DayOfWeek  sql.NullInt64  `json:"dayOfWeek"`  // 上课时间（1-7代表星期一到星期日）
	StartTime  sql.NullTime   `json:"startTime"`  // 开始上课时间
	EndTime    sql.NullTime   `json:"endTime"`    // 结束上课时间
	Remark     sql.NullString `json:"remark"`     // 备注
	CreateTime sql.NullTime   `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime   `json:"updateTime"` // 更新时间
}

// Course 课程实体（表 c_course，对齐 Java Course.java）
//
// 课程归属机构，courseType 区分按次/按天
type Course struct {
	ID            sql.NullInt64  `json:"id"`            // 主键
	CourseName    sql.NullString `json:"courseName"`    // 课程名称
	CourseType    sql.NullInt64  `json:"courseType"`    // 课程类型（1=按次, 2=按天）
	InstitutionID sql.NullInt64  `json:"institutionId"` // 机构ID（c_institution.id）
	IsAvailable   sql.NullBool   `json:"isAvailable"`   // 是否可用
	CreateTime    sql.NullTime   `json:"createTime"`    // 创建时间
	UpdateTime    sql.NullTime   `json:"updateTime"`    // 更新时间
}

// CourseRecord 课卡记录实体（表 c_course_record，对齐 Java CourseRecord.java）
//
// 记录学生在某课程的课时持有情况，扣课时双重校验（Java 层 + SQL 层）
type CourseRecord struct {
	ID                sql.NullInt64  `json:"id"`                // 主键
	StudentID         sql.NullInt64  `json:"studentId"`         // 学生ID（c_student.id）
	CourseID          sql.NullInt64  `json:"courseId"`          // 课程ID（c_course.id）
	CourseTotalTime   sql.NullInt64  `json:"courseTotalTime"`   // 课时总数
	CourseRestTime    sql.NullInt64  `json:"courseRestTime"`    // 课程剩余次数
	CourseStatus      sql.NullInt64  `json:"courseStatus"`      // 课程状态（0=默认,1=未完成,2=已完成）
	CourseLastTime    sql.NullTime   `json:"courseLastTime"`    // 上次上课时间
	ExpireTime        sql.NullTime   `json:"expireTime"`        // 过期时间（NULL=永久有效，过期则禁止扣课时）
	CourseOwnerUserID sql.NullInt64  `json:"courseOwnerUserId"` // 课程归属人（c_user.id）
	CourseRemark      sql.NullString `json:"courseRemark"`      // 课程备注
	IsDelete          sql.NullBool   `json:"isDelete"`          // 逻辑删除标识（0=未删除,1=已删除）
	CreateTime        sql.NullTime   `json:"createTime"`        // 创建时间
	UpdateTime        sql.NullTime   `json:"updateTime"`        // 更新时间
}

// Record 上课记录实体（表 c_record，对齐 Java Record.java）
//
// 每次上课/扣课时的明细记录，关联课卡记录、学生、教师、班级
type Record struct {
	ID                   sql.NullInt64  `json:"id"`                   // 主键
	CourseRecordID       sql.NullInt64  `json:"courseRecordId"`       // 课卡记录ID（c_course_record.id）
	RecordTime           sql.NullTime   `json:"recordTime"`           // 记录时间
	RecordRemark         sql.NullString `json:"recordRemark"`         // 备注
	RecordType           sql.NullInt64  `json:"recordType"`           // 记录类型（1=增加, 2=减少）
	RecordChange         sql.NullInt64  `json:"recordChange"`         // 课时变更数量
	RestTimeAfterDeduct  sql.NullInt64  `json:"restTimeAfterDeduct"`  // 扣费后剩余课时（快照）
	DeductMode           sql.NullString `json:"deductMode"`           // 扣费模式（BY_STUDENT/BY_COURSE/BY_CLASS）
	ClassID              sql.NullInt64  `json:"classId"`              // 班级ID（按班级扣费时有值）
	OperateTeacherID     sql.NullInt64  `json:"operateTeacherId"`     // 操作人ID（c_teacher.id）
	CreateTime           sql.NullTime   `json:"createTime"`           // 创建时间
	UpdateTime           sql.NullTime   `json:"updateTime"`           // 更新时间
}

// ClassTeacher 班级-教师关联实体（表 c_class_teacher，对齐 Java ClassTeacher.java）
//
// 多对多关联：一个班级可有多个教师，一个教师可带多个班级
type ClassTeacher struct {
	ID         sql.NullInt64 `json:"id"`         // 主键
	ClassID    sql.NullInt64 `json:"classId"`    // 班级ID（c_class.id）
	TeacherID  sql.NullInt64 `json:"teacherId"`  // 教师ID（c_teacher.id）
	CreateTime sql.NullTime  `json:"createTime"` // 创建时间
}

// ClassStudent 班级-学生关联实体（表 c_class_student，对齐 Java ClassStudent.java）
//
// 多对多关联：一个班级可有多个学生，一个学生可属多个班级
type ClassStudent struct {
	ID         sql.NullInt64 `json:"id"`         // 主键
	ClassID    sql.NullInt64 `json:"classId"`    // 班级ID（c_class.id）
	StudentID  sql.NullInt64 `json:"studentId"`  // 学生ID（c_student.id）
	CreateTime sql.NullTime  `json:"createTime"` // 创建时间
}

// ============================================================
// 系统管理实体（sys_ 前缀表，对齐 Java sys_ 系列）
// ============================================================

// SysRole 系统角色实体（表 sys_role，对齐 Java SysRole.java）
//
// 管理端角色定义，通过 sys_user_role 关联用户，通过 sys_role_menu 关联菜单
type SysRole struct {
	ID          sql.NullInt64  `json:"id"`          // 主键
	RoleName    sql.NullString `json:"roleName"`    // 角色名称
	RoleCode    sql.NullString `json:"roleCode"`    // 角色编码（唯一标识）
	Description sql.NullString `json:"description"` // 角色描述
	Status      sql.NullInt64  `json:"status"`      // 状态（0=禁用,1=启用）
	CreateTime  sql.NullTime   `json:"createTime"`  // 创建时间
	UpdateTime  sql.NullTime   `json:"updateTime"`  // 更新时间
}

// SysMenu 系统菜单实体（表 sys_menu，对齐 Java SysMenu.java）
//
// 管理端菜单/权限定义，menuType 区分目录/菜单/按钮
type SysMenu struct {
	ID         sql.NullInt64  `json:"id"`         // 主键
	ParentID   sql.NullInt64  `json:"parentId"`   // 父菜单ID（0=根菜单）
	MenuName   sql.NullString `json:"menuName"`   // 菜单名称
	MenuType   sql.NullString `json:"menuType"`   // 菜单类型（directory=目录, menu=菜单, button=按钮）
	Path       sql.NullString `json:"path"`       // 路由路径
	Component  sql.NullString `json:"component"`  // 前端组件路径
	Icon       sql.NullString `json:"icon"`       // 菜单图标
	SortOrder  sql.NullInt64  `json:"sortOrder"`  // 排序权值（越小越靠前）
	IsVisible  sql.NullBool   `json:"isVisible"`  // 是否可见（true=显示, false=隐藏）
	Status     sql.NullInt64  `json:"status"`     // 状态（0=禁用,1=启用）
	Permission sql.NullString `json:"permission"` // 权限标识（如 user:list, user:add）
	CreateTime sql.NullTime   `json:"createTime"` // 创建时间
	UpdateTime sql.NullTime   `json:"updateTime"` // 更新时间
}

// SysOperationLog 操作日志实体（表 sys_operation_log，对齐 Java SysOperationLog.java）
//
// 记录管理端用户的操作行为，用于审计
type SysOperationLog struct {
	ID         sql.NullInt64  `json:"id"`         // 主键
	UserID     sql.NullInt64  `json:"userId"`     // 操作用户ID（sys_user.id）
	Username   sql.NullString `json:"username"`   // 操作用户名
	Operation  sql.NullString `json:"operation"`  // 操作描述
	Method     sql.NullString `json:"method"`     // 请求方法（Controller 方法名）
	Params     sql.NullString `json:"params"`     // 请求参数（JSON）
	IP         sql.NullString `json:"ip"`         // 请求IP
	Location   sql.NullString `json:"location"`   // 操作地点（IP 解析）
	Status     sql.NullInt64  `json:"status"`     // 操作状态（0=失败,1=成功）
	ErrorMsg   sql.NullString `json:"errorMsg"`   // 错误信息（失败时）
	CostTime   sql.NullInt64  `json:"costTime"`   // 耗时（毫秒）
	CreateTime sql.NullTime   `json:"createTime"` // 创建时间
}

// SysConfig 系统配置实体（表 sys_config，对齐 Java SysConfig.java）
//
// 存储系统运行时可动态调整的参数，修改后通过 Redis 缓存失效实时生效
type SysConfig struct {
	ID          sql.NullInt64  `json:"id"`          // 主键
	ConfigName  sql.NullString `json:"configName"`  // 配置名称（中文描述）
	ConfigKey   sql.NullString `json:"configKey"`   // 配置键（唯一标识）
	ConfigValue sql.NullString `json:"configValue"` // 配置值
	ConfigType  sql.NullString `json:"configType"`  // 配置类型（如 STRING/INTEGER/LONG/BOOLEAN）
	Remark      sql.NullString `json:"remark"`      // 备注说明
	CreateTime  sql.NullTime   `json:"createTime"`  // 创建时间
	UpdateTime  sql.NullTime   `json:"updateTime"`  // 更新时间
}

// SysRoleMenu 角色-菜单关联实体（表 sys_role_menu，对齐 Java SysRoleMenu.java）
//
// 多对多关联：一个角色拥有多个菜单/权限，一个菜单可被多个角色使用
// 注意：本表无独立主键 id，(roleId, menuId) 为联合唯一键
type SysRoleMenu struct {
	RoleID sql.NullInt64 `json:"roleId"` // 角色ID（sys_role.id）
	MenuID sql.NullInt64 `json:"menuId"` // 菜单ID（sys_menu.id）
}

// SysUserRole 用户-角色关联实体（表 sys_user_role，对齐 Java SysUserRole.java）
//
// 多对多关联：一个用户拥有多个角色，一个角色可被多个用户使用
// 注意：本表无独立主键 id，(userId, roleId) 为联合唯一键
type SysUserRole struct {
	UserID sql.NullInt64 `json:"userId"` // 用户ID（sys_user.id）
	RoleID sql.NullInt64 `json:"roleId"` // 角色ID（sys_role.id）
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
