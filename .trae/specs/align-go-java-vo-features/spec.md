# 对齐 Go 与 Java VO/DTO 及功能逻辑 Spec

## Why

Go 后端迁移已实现核心查询接口，但与 Java master 分支系统性对比后发现：**4 个 P0 级阻塞问题**（扣课微信通知完全缺失、微信配置硬编码假值、签名校验中间件未应用、Nonce 防重放未启用）和 **5 个 P1 级重要问题**（事务管理缺失、SM2 私钥不一致、Redis/DB 配置硬编码、缓存机制缺失、admin-service 防御纵深缺失）。

这些问题导致：
1. **家长端收不到扣课通知** — 扣课成功后不发送微信订阅消息
2. **微信登录和绑定完全不可用** — wxAppID/wxAppSecret 是占位假值
3. **安全漏洞** — 签名校验和防重放机制未启用，请求可被伪造和重放
4. **数据一致性风险** — 扣课两步操作（扣课时+插流水）无事务包裹

已通过 `fix-vo-frontend-consistency` spec 修复的 11 项 VO 字段问题不再重复。

## What Changes

### P0 阻塞级修复（4 项）

#### 1. 扣课微信通知补全
- **BREAKING**：无（新增功能，不改变现有接口契约）
- 在 `business-service` 提取微信 API 服务到 common 包，供 auth-service 和 business-service 共用
- 补齐 `WxStudentSubscribeMapper.SelectByStudentID` 方法（查询订阅该学生的所有 openId）
- 补齐 `WxSubscribeRecordMapper.SelectByOpenIDsAndTemplate` 方法（批量查授权记录）
- 补齐 `WxSubscribeRecordMapper.DecrementCount` 方法（推送成功后扣减次数）
- `CourseRecordService` 注入新依赖：`StudentMapper`、`CourseMapper`、`WxStudentSubscribeMapper`、`WxSubscribeRecordMapper`、`WeChatApiService`
- 在 `deductOne` 方法扣课成功后调用通知逻辑（Go 无 AOP，直接调用），对齐 Java `DeductNotifyAspect.sendSingleNotification`：
  - 查学生、课程、课卡记录
  - 构建模板数据（thing1=课程名、thing8=学生名、number4=剩余课时、number10=扣减课时、number11=剩余课时、time5=到期时间）
  - 查订阅 openId → 查授权记录 → 发送 → 成功扣减次数（永久订阅不扣）
- 通知失败不阻塞主流程（仅记录日志），对齐 Java 行为

#### 2. 微信配置从环境变量加载
- **BREAKING**：无（配置加载方式变更，运行时需提供环境变量）
- 移除 `auth-service/internal/service/auth_service.go` 中 `wxAppID`/`wxAppSecret` 硬编码常量
- 改为从环境变量 `WX_APP_ID`/`WX_APP_SECRET`/`WX_ENV_VERSION` 加载，提供合理默认值
- `WeChatApiService` 提取到 common 包后，配置由 common 统一管理

#### 3. SignInterceptor 签名校验中间件
- **BREAKING**：无（前端已发送签名头，后端原本应校验但未实现）
- 在 `common/sign/sign.go` 的 `VerifyRequest` 基础上实现 HTTP 中间件
- 在 auth-service 和 business-service 的 `main.go` 中注册中间件（在 `commonctxMiddleware` 之前）
- 中间件逻辑：校验 `x-sign`/`x-timestamp`/`x-nonce` 头，时间戳有效期（5分钟），nonce 防重放（Redis SETNX 60s TTL）
- 公开路径（登录、绑定等）跳过校验，对齐 Java `SignInterceptor` 的排除路径

#### 4. Nonce 防重放机制应用
- **BREAKING**：无
- 在 SignInterceptor 中间件内调用 `common/redis/redis.go` 的 `SetNonceIfAbsent` 方法
- nonce 存入 Redis，TTL 60s，SETNX 失败表示重放，拒绝请求

### P1 重要级修复（5 项）

#### 5. business-service 事务管理
- **BREAKING**：无
- 在 `deductOne` 方法中用 `sql.Tx` 包裹 UPDATE（扣课时）+ INSERT（插流水）两步操作
- 批量扣课（`DeductByClassID` 遍历多学生）每个学生独立事务，单学生失败不阻塞其他
- 对齐 admin-service `sys_role_service.go` 已有的事务模式

#### 6. SM2 私钥统一并从配置加载
- **BREAKING**：无（需确保三个服务使用同一私钥）
- 移除三处硬编码的 SM2 私钥（auth-service、admin-service、business-service）
- 改为从环境变量 `SM2_PRIVATE_KEY` 加载
- 确保三处使用相同私钥值（对齐 Java Nacos `crypto.sm2.private-key`）

#### 7. Redis/DB 配置从环境变量加载
- **BREAKING**：无（已有环境变量覆盖机制，需移除硬编码默认值的敏感凭证）
- `common/redis/redis.go` 的 `DefaultConfig()` 移除硬编码密码，改为必填环境变量或空字符串
- `common/db/mysql.go` 的 `DefaultConfig()` 移除硬编码密码，改为必填环境变量或空字符串
- 部署时通过环境变量注入凭证（Docker Compose 或 systemd）

#### 8. 缓存机制（Cache-Aside）
- **BREAKING**：无
- 在 `common/redis/redis.go` 添加 Cache-Aside 辅助函数：`GetOrSet(key, ttl, loader)`
- 应用于高频查询：用户信息（`getFullUserInfo`，30min TTL）、菜单数据（`getMenuByRole`，30min TTL）
- 写操作（update/delete）时删除对应缓存键

#### 9. admin-service AdminJwtInterceptor
- **BREAKING**：无
- 在 admin-service `main.go` 添加 JWT 二次校验中间件
- 校验 `Authorization` 头的 JWT 有效性（签名、过期时间）
- 公开路径（登录）跳过校验
- 作为 Gateway 认证的防御纵深

### P2 次要级修复（3 项）

#### 10. GenerateSalt 改用标准实现
- 修复 `common/crypto/sm.go` 的 `randomString` 函数 bug（`big.NewInt(int64(len(charset))).Int64()%int64(len(charset))` 每次生成相同字符）
- 改用 `crypto/rand` 生成 32 位十六进制盐值，对齐 Java `UUID.randomUUID().toString().replace("-", "")`

#### 11. Gateway 支持 lb 负载均衡
- **BREAKING**：无（dev 环境继续用 localhost，prod 可选 lb）
- `gateway/internal/config.go` 支持从环境变量加载服务 URI
- 支持 `lb://{service-name}` 格式（需服务发现，当前可保留 localhost 直连）

#### 12. 确认 c_student 软删除处理
- 查询 Java `StudentMapper.xml` 确认是否对 c_student 做软删除
- 若 Java 有软删除逻辑，Go 需补齐 `WHERE is_delete = 0` 条件
- 若无软删除，确认 Go 当前实现正确

## Impact

- **Affected specs**：
  - `fix-vo-frontend-consistency`（VO 字段已修复，本 spec 不重复）
  - `go-backend-interface-completion`（接口已补全，本 spec 聚焦功能逻辑和配置）
- **Affected code**：
  - `common/` — 提取 WeChatApiService、添加缓存辅助、修复 GenerateSalt、配置加载
  - `auth-service/` — 微信配置加载、SignInterceptor、Nonce 防重放
  - `business-service/` — 扣课通知、事务管理、SignInterceptor、SM2 私钥
  - `admin-service/` — AdminJwtInterceptor、SM2 私钥
  - `gateway/` — 配置加载（可选 lb）
- **Affected deployment**：部署时需提供环境变量 `WX_APP_ID`/`WX_APP_SECRET`/`SM2_PRIVATE_KEY`/`REDIS_PASSWORD`/`DB_PASSWORD`

## ADDED Requirements

### Requirement: 扣课微信订阅消息通知

系统 SHALL 在扣课成功后向订阅该学生的家长微信发送订阅消息通知。

#### Scenario: 扣课成功且家长已订阅
- **WHEN** 教师执行扣课操作且成功
- **AND** 该学生有家长在 `c_wx_student_subscribe` 表中订阅
- **AND** 订阅记录 `c_wx_subscribe_record` 中 `subscribe_count > 0` 或 `is_permanent = 1`
- **THEN** 系统向每个订阅 openId 发送微信订阅消息
- **AND** 消息模板包含：课程名、学生名、剩余课时、扣减课时、到期时间
- **AND** 点击消息跳转到 `pages/main/parent/deduct-detail/index?recordId={recordId}`
- **AND** 非永久订阅推送成功后 `subscribe_count - 1`
- **AND** 永久订阅不扣减次数
- **AND** 单条通知失败不阻塞其他通知和主流程

#### Scenario: 扣课成功但家长未订阅
- **WHEN** 扣课成功
- **AND** 该学生无家长订阅（`c_wx_student_subscribe` 为空）
- **THEN** 跳过通知发送，扣课正常返回成功

#### Scenario: 通知发送失败
- **WHEN** 微信 API 返回错误
- **THEN** 记录错误日志
- **AND** 不扣减订阅次数
- **AND** 不影响已成功的扣课结果

### Requirement: 请求签名校验

系统 SHALL 对 auth-service 和 business-service 的非公开路径请求校验 SM3 签名。

#### Scenario: 合法签名请求
- **WHEN** 请求携带有效的 `x-sign`/`x-timestamp`/`x-nonce` 头
- **AND** 时间戳在 5 分钟有效期内
- **AND** nonce 未被使用过
- **THEN** 请求通过校验，正常处理

#### Scenario: 签名缺失或无效
- **WHEN** 请求缺少签名头，或签名不匹配
- **THEN** 返回 401 错误，拒绝请求

#### Scenario: 时间戳过期
- **WHEN** 请求时间戳超出 5 分钟有效期
- **THEN** 返回 401 错误，提示"时间戳过期"

#### Scenario: Nonce 重放
- **WHEN** 请求 nonce 已在 Redis 中存在（60s 内重复使用）
- **THEN** 返回 401 错误，提示"请求已过期"

#### Scenario: 公开路径豁免
- **WHEN** 请求路径在公开路径列表中（如 `/auth/login`、`/auth/generate_bind_qrcode`）
- **THEN** 跳过签名校验

### Requirement: 微信配置从环境变量加载

系统 SHALL 从环境变量加载微信小程序配置，而非硬编码。

#### Scenario: 环境变量已设置
- **WHEN** `WX_APP_ID` 和 `WX_APP_SECRET` 环境变量已设置
- **THEN** 系统使用环境变量值调用微信 API

#### Scenario: 环境变量未设置
- **WHEN** 环境变量未设置
- **THEN** 系统启动时记录警告日志
- **AND** 微信相关功能返回错误（不使用假值调用 API）

### Requirement: 扣课事务一致性

系统 SHALL 保证扣课操作的原子性。

#### Scenario: 扣课两步操作成功
- **WHEN** 扣课时（UPDATE c_course_record）和插流水（INSERT c_record）都成功
- **THEN** 事务提交，两步操作都生效

#### Scenario: 插流水失败
- **WHEN** 扣课时成功但插流水失败
- **THEN** 事务回滚，扣课时恢复
- **AND** 记录错误日志，返回系统错误

### Requirement: 配置凭证从环境变量加载

系统 SHALL 从环境变量加载敏感配置凭证（Redis 密码、DB 密码、SM2 私钥、JWT 密钥）。

#### Scenario: 凭证环境变量已设置
- **WHEN** `REDIS_PASSWORD`/`DB_PASSWORD`/`SM2_PRIVATE_KEY` 等环境变量已设置
- **THEN** 系统使用环境变量值连接服务

#### Scenario: 凭证环境变量未设置
- **WHEN** 凭证环境变量未设置
- **THEN** 系统启动失败并明确提示缺失的环境变量名

## MODIFIED Requirements

### Requirement: 扣课双重校验（已实现，补充事务）
系统 SHALL 对扣课操作执行双重校验（Service 层过期校验 + SQL 层余额校验），并在事务中执行扣课时更新和流水插入。

**修改点**：在 `deductOne` 方法中添加事务包裹，保证两步操作原子性。

### Requirement: 微信 API 服务共用（重构）
系统 SHALL 在 common 包提供 `WeChatApiService`，供 auth-service 和 business-service 共用，避免代码重复。

**修改点**：从 auth-service 的 `AuthService` 私有方法提取到 common 包独立服务。

## REMOVED Requirements

### Requirement: 硬编码微信配置
**Reason**: 微信配置硬编码为假值导致微信功能完全不可用
**Migration**: 改为从环境变量 `WX_APP_ID`/`WX_APP_SECRET`/`WX_ENV_VERSION` 加载

### Requirement: 硬编码敏感凭证
**Reason**: Redis/DB 密码、SM2 私钥硬编码在代码中有泄露风险
**Migration**: 改为从环境变量加载，部署时通过 Docker Compose 或 systemd 注入
