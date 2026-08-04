# 审计 Java 接口并补齐缺失 VO/DTO/功能 Spec

## Why

对 Java master 分支全部 22 个控制器（auth-service 4 + business-service 9 + admin-service 9）与 Go `feat/go-migration-poc` 分支实现进行系统性对比审计后发现：虽然前序 spec（`go-backend-interface-completion`、`fix-vo-frontend-consistency`、`align-go-java-vo-features`）已补全接口路径和主要 VO 字段，但仍存在 **3 个 P0 级功能阻断 bug**、**4 个 P0 级功能缺失**、**5 个 P1 级重要缺失**、**6 个 P2 级次要问题**。

核心问题：
1. **business-service 扣课按班级接口完全不可用** — `deduct_by_class_id` handler 硬编码 `courseID=0`，service 层立即返回"课程ID不能为空"
2. **business-service 课卡新增 ownerUserID 恒为 0** — `insert` 接口未从 UserContext 读取登录教师 ID，导致 `courseOwnerUserId` 字段失真
3. **admin-service 操作日志机制完全失效** — `RecordLog` 工具方法已实现但所有写操作均未调用，`sys_operation_log` 表始终为空
4. **admin-service 系统配置无缓存无通用读取** — `SysConfigService` 缺失 Cache-Aside 缓存和 `getConfigValue/AsLong/AsInt/AsBoolean` 通用方法，JWT 过期时间等运行时参数无法热更新

## What Changes

### P0 阻塞级修复（7 项）

#### 1. business-service `deduct_by_class_id` 接口修复
- **BREAKING**：无（修复 bug，接口契约不变）
- handler 解析前端 `courseId` 字段（`FastDeductRequest` 已含 `courseId`），或 service 层从 `c_class.course_id` 查询课程 ID
- 当前 handler.go:1117 硬编码 `courseID=0` 传入 service，导致恒返回"课程ID不能为空"

#### 2. business-service `course_record/insert` ownerUserID 修复
- **BREAKING**：无
- handler 从请求 context 读取 `X-User-Id`（`commonctxMiddleware` 已注入），传入 service 的 `ownerUserID` 参数
- 当前 handler.go:1003 硬编码 `0`，导致所有新建课卡记录 `courseOwnerUserId` 恒为 0

#### 3. business-service `/biz/record/delete` 路由注册
- `RecordMapper.DeleteByID` 已实现（硬删除），仅缺 handler 方法和路由注册
- 对齐 Java `RecordController.delete`

#### 4. business-service `/biz/course_record/delete` 逻辑删除
- 新增 `CourseRecordMapper.DeleteByID` 方法（UPDATE `is_delete = 1`）
- 新增 handler 和路由注册
- 对齐 Java `CourseRecordController.delete`（逻辑删除）

#### 5. admin-service 操作日志自动记录
- 在 admin-service 所有写操作（用户/角色/菜单/配置/业务透传/教师账号）service 方法中调用 `logService.RecordLog`
- 对齐 Java `@OperationLog` 注解 + AOP 切面
- 日志字段：operation、username、params、createTimeStr、ip（可从 X-Real-IP 头获取）

#### 6. admin-service SysConfigService Cache-Aside 缓存
- `SysConfigService` 注入 Redis 客户端
- 读路径：先查 Redis（key: `sys:config:{configKey}`），未命中查库并回填（TTL 30min）
- 写路径：`UpdateConfig`/`DeleteConfig` 后删除对应缓存键
- 对齐 Java `SysConfigServiceImpl` 缓存策略

#### 7. admin-service SysConfigService 通用读取方法
- 新增 `GetConfigValue(key string) string`
- 新增 `GetConfigValueAsLong(key string, default int64) int64`
- 新增 `GetConfigValueAsInt(key string, default int) int`
- 新增 `GetConfigValueAsBoolean(key string, default bool) bool`
- 供其他服务（如 JWT 工具）动态读取配置值

### P1 重要级修复（5 项）

#### 8. business-service `UpdateInstitution` 请求体补字段
- handler.go:227 请求 struct 补充 `Contact string` 和 `Phone string` 字段
- 当前注释声明前端传 contact/phone，但 struct 未声明，字段被静默丢弃

#### 9. admin-service 按钮级权限校验（perms）
- 新增 `PermsMiddleware` 中间件，校验当前用户角色是否具备接口所需 perms
- perms 配置来源：`sys_menu.perms` 字段，按角色聚合
- 对齐 Java `@RequirePerms` 注解 + AOP

#### 10. auth-service 菜单缓存
- `menu_service.go` 注入 Redis 客户端
- `GetMenuByRole` 应用 Cache-Aside（key: `menu:role:{roleId}`，TTL 30min）
- 写操作（菜单 CRUD）时删除缓存
- 当前注释明确写"Go 版暂未实现缓存"

#### 11. auth-service 绑定/注册事务管理
- `doBind`、`Register` 等多表写入操作用 `sql.Tx` 包裹
- 对齐 Java `@Transactional`，避免部分失败产生孤立记录

#### 12. business-service 操作日志记录
- business-service 写操作（insert/update/delete/deduct）记录到 `sys_operation_log`
- 复用 admin-service 的 `RecordLog` 模式（或提取到 common 包）

### P2 次要级修复（6 项）

#### 13. admin-service SysRoleVO/SysMenuVO 时间字段去重
- 移除 `SysRoleVO.CreateTime`/`UpdateTime`（保留 `CreateTimeStr`/`UpdateTimeStr`）
- 移除 `SysMenuVO.CreateTime`/`UpdateTime`（保留 `CreateTimeStr`/`UpdateTimeStr`）
- 对齐 Java VO（仅 `createTimeStr`/`updateTimeStr`）

#### 14. business-service `CourseRecordVO.permissionType` 确认
- 核对 Java `CourseRecordVO` 是否含 `permissionType` 字段
- 若有，Go 补回（即便 DB 无列也可返回默认值 0）

#### 15. business-service 旧版端点评估
- 评估 `/course_record/get`、`/course_record/add`、`/record/get` 是否有前端调用
- 若无调用，记录为"已废弃"不迁移；若有调用，补齐实现

#### 16. admin-service SysMenuService 缓存失效范围
- `SaveRoleMenus` 时失效所有 `menu:user:*` 缓存（当前仅失效 `menu:role:*`）
- 角色菜单授权变更后，已登录用户菜单最长 30min 才刷新

#### 17. auth-service 缺失接口评估
- 评估 Java `UserController`（/user/info、/user/password、/user/profile）是否有前端调用
- 评估 Java `PermissionController`（/permission/by-user、/permission/by-role）是否有前端调用
- 若无调用，记录为"不迁移"；若有调用，补齐

#### 18. business-service 时间字段命名统一
- VO 时间字段统一为 `xxxTime`（非 `xxxTimeStr`），对齐前端类型定义规范
- 影响范围：`RecordVO`、`CourseRecordVO`、`InstitutionVO`、`StudentVO` 等

## Impact

- **Affected specs**：
  - `go-backend-interface-completion`（接口路径已补全，本 spec 修复功能 bug 和补齐缺失端点）
  - `align-go-java-vo-features`（跨服务功能已对齐，本 spec 补齐 admin-service 操作日志、配置缓存等遗漏）
  - `fix-vo-frontend-consistency`（VO 字段已对齐，本 spec 修复 P2 级时间字段冗余）
- **Affected code**：
  - `business-service/internal/handler/handler.go` — 修复 deduct_by_class_id、insert ownerUserID、补 record/delete 和 course_record/delete 路由、UpdateInstitution 字段
  - `business-service/internal/service/course_record_service.go` — 修复 DeductByClassID 课程 ID 获取逻辑
  - `business-service/internal/mapper/course_record_mapper.go` — 新增 DeleteByID 逻辑删除方法
  - `admin-service/internal/service/*.go` — 所有写操作补充 RecordLog 调用
  - `admin-service/internal/service/sys_config_service.go` — 注入 Redis、实现缓存和通用读取方法
  - `admin-service/internal/middleware/` — 新增 PermsMiddleware
  - `auth-service/internal/service/menu_service.go` — 注入 Redis、实现菜单缓存
  - `auth-service/internal/service/auth_service.go` — doBind/Register 事务化
- **Affected deployment**：无新增环境变量依赖

## ADDED Requirements

### Requirement: 扣课按班级接口可用性

系统 SHALL 在 `POST /biz/course_record/deduct_by_class_id` 接口正确获取课程 ID，而非硬编码 0。

#### Scenario: 前端传入 courseId
- **WHEN** 前端调用 `deduct_by_class_id` 传入 `classId` 和 `courseId`
- **THEN** 系统使用前端传入的 `courseId` 执行扣课

#### Scenario: 前端未传 courseId
- **WHEN** 前端仅传入 `classId`
- **THEN** 系统从 `c_class.course_id` 查询该班级的课程 ID
- **AND** 使用查询到的课程 ID 执行扣课

### Requirement: 课卡新增记录操作人

系统 SHALL 在 `POST /biz/course_record/insert` 接口记录创建教师 ID 到 `courseOwnerUserId` 字段。

#### Scenario: 教师登录态创建课卡
- **WHEN** 教师登录后调用 `insert` 接口
- **THEN** 系统从请求 context 的 `X-User-Id` 读取教师 ID
- **AND** 写入 `c_course_record.course_owner_user_id` 字段

### Requirement: 上课记录删除接口

系统 SHALL 提供 `POST /biz/record/delete` 接口，按记录 ID 删除上课记录。

#### Scenario: 删除上课记录
- **WHEN** 教师调用 `delete` 传入 `recordId`
- **THEN** 系统从 `c_record` 表删除该记录
- **AND** 返回操作结果

### Requirement: 课卡记录逻辑删除

系统 SHALL 提供 `POST /biz/course_record/delete` 接口，按课卡 ID 逻辑删除课卡记录。

#### Scenario: 逻辑删除课卡
- **WHEN** 管理员调用 `delete` 传入 `courseRecordId`
- **THEN** 系统更新 `c_course_record.is_delete = 1`
- **AND** 后续查询不返回该记录

### Requirement: Admin 操作日志自动记录

系统 SHALL 在 admin-service 所有写操作（insert/update/delete）成功后自动记录操作日志到 `sys_operation_log` 表。

#### Scenario: 管理员执行写操作
- **WHEN** 管理员调用任何写操作接口（如 `/admin/role/insert`）
- **AND** 操作成功
- **THEN** 系统在 `sys_operation_log` 表插入一条记录
- **AND** 记录包含：操作类型、操作人用户名、请求参数、操作时间、操作 IP

#### Scenario: 操作失败不记录
- **WHEN** 管理员调用写操作但操作失败（如校验失败、数据库异常）
- **THEN** 不记录操作日志

### Requirement: 系统配置 Cache-Aside 缓存

系统 SHALL 对 `SysConfigService` 的配置读取应用 Cache-Aside 模式。

#### Scenario: 首次读取配置
- **WHEN** 系统首次读取配置键 `jwt_expire_time`
- **THEN** 查询数据库获取值
- **AND** 写入 Redis 缓存（key: `sys:config:jwt_expire_time`，TTL 30min）

#### Scenario: 缓存命中
- **WHEN** 系统再次读取同一配置键
- **AND** Redis 缓存命中
- **THEN** 直接返回缓存值，不查库

#### Scenario: 配置更新失效缓存
- **WHEN** 管理员调用 `UpdateConfig` 更新配置
- **THEN** 删除对应 Redis 缓存键
- **AND** 下次读取重新从数据库加载

### Requirement: 系统配置通用读取方法

系统 SHALL 在 `SysConfigService` 提供通用配置读取方法，供其他服务动态获取配置值。

#### Scenario: 字符串读取
- **WHEN** 调用 `GetConfigValue("jwt_expire_time")`
- **THEN** 返回配置值的字符串形式

#### Scenario: 整数读取（带默认值）
- **WHEN** 调用 `GetConfigValueAsInt("jwt_expire_time", 300)`
- **AND** 配置值为 "600"
- **THEN** 返回整数 600

#### Scenario: 配置不存在
- **WHEN** 调用 `GetConfigValueAsInt("nonexistent", 300)`
- **AND** 配置键不存在
- **THEN** 返回默认值 300

### Requirement: 按钮级权限校验

系统 SHALL 对 admin-service 接口执行按钮级权限校验（基于 `sys_menu.perms` 字段）。

#### Scenario: 用户具备所需权限
- **WHEN** 用户调用接口
- **AND** 用户角色关联的菜单含所需 perms
- **THEN** 请求通过校验

#### Scenario: 用户无权限
- **WHEN** 用户调用接口
- **AND** 用户角色无所需 perms
- **THEN** 返回 403 错误，拒绝请求

### Requirement: 菜单数据缓存

系统 SHALL 对 auth-service 的 `GetMenuByRole` 应用 Cache-Aside 缓存。

#### Scenario: 首次查询菜单
- **WHEN** 首次按角色查询菜单
- **THEN** 查询数据库，结果写入 Redis（key: `menu:role:{roleId}`，TTL 30min）

#### Scenario: 菜单变更失效缓存
- **WHEN** 菜单 CRUD 操作
- **THEN** 删除相关菜单缓存键

## MODIFIED Requirements

### Requirement: 绑定/注册事务一致性
系统 SHALL 在 `doBind`、`Register` 等多表写入操作中使用事务，保证原子性。

**修改点**：用 `sql.Tx` 包裹 user/user_auth/parent/parent_student 多表写入。

### Requirement: 机构更新请求字段
系统 SHALL 在 `UpdateInstitution` 接口解析 `contact` 和 `phone` 字段。

**修改点**：handler 请求 struct 补充 `Contact`/`Phone` 字段声明。

## REMOVED Requirements

无移除需求。
