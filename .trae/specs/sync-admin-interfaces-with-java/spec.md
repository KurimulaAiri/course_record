# Admin 接口与 Java 端同步修复 Spec

## Why

Go 后端 admin-service 的部分接口逻辑与 Java 端不一致，导致管理端功能异常：系统配置页"更新时间"列为空、操作日志无法追溯操作人、管理员无法通过 update 接口禁用用户。需对齐 Java 端逻辑，确保功能正确性。

## What Changes

- 修复系统配置 VO 的 `updateTime` 字段名（`updateTimeStr` → `updateTime`），对齐前端类型定义和表格列绑定
- 修复 `UpdateUser` 接口的 status=0 歧义问题，将 `Status` 字段改为指针类型（`*int64`），区分"不更新"和"禁用"
- 修复操作日志缺失用户信息问题，在 Handler 层提取当前登录用户 ID 和用户名，通过参数传递给 Service 层的 `recordLog` 方法

## Impact

- Affected specs: `go-backend-interface-completion`（阶段七响应结构修复相关）
- Affected code:
  - `admin-service/internal/mapper/sys_config_mapper.go`（SysConfigRow 字段名）
  - `admin-service/internal/service/admin_service.go`（UpdateUser 请求结构体、recordLog 签名、所有调用方）
  - `admin-service/internal/service/sys_role_service.go`（recordLog 签名、所有调用方）
  - `admin-service/internal/service/sys_menu_service.go`（recordLog 签名、所有调用方）
  - `admin-service/internal/service/sys_config_service.go`（recordLog 签名、所有调用方）
  - `admin-service/internal/service/teacher_auth_service.go`（recordLog 签名、所有调用方）
  - `admin-service/internal/service/admin_business_service.go`（recordLog 调用方传入实际用户信息）
  - `admin-service/internal/handler/admin_handler.go`（Handler 提取用户信息传入 Service）
  - `admin-service/internal/handler/business_handler.go`（同上）

## ADDED Requirements

### Requirement: 操作日志用户信息记录

系统在记录操作日志时，SHALL 自动从请求上下文中提取当前登录用户 ID 和用户名，写入 `sys_operation_log.user_id` 和 `sys_operation_log.username` 字段。

#### Scenario: 管理员执行写操作

- **WHEN** 管理员（userID=1, username="admin"）调用 `/admin/user/insert` 创建新用户
- **THEN** `sys_operation_log` 表新增记录，`user_id=1`、`username="admin"`
- **AND** 操作类型为"新增系统用户"，参数 JSON 包含新用户信息

#### Scenario: 未登录调用写操作

- **WHEN** 未登录用户调用写操作接口
- **THEN** JWT 中间件返回 401，不会进入 Service 层，不记录操作日志

## MODIFIED Requirements

### Requirement: 系统配置 VO 响应结构

系统配置列表/新增/更新接口返回的 VO SHALL 包含 `updateTime` 字段（String 类型，格式化时间），对齐前端 `SysConfigResponse` 类型定义和表格列绑定 `prop="updateTime"`。

**修改前**：`SysConfigRow` 输出 JSON 字段 `updateTimeStr`
**修改后**：`SysConfigRow` 输出 JSON 字段 `updateTime`（同时保留 `createTime` 替代 `createTimeStr`，对齐前端类型）

### Requirement: 用户更新接口 Status 字段语义

`POST /admin/user/update` 接口的 `status` 字段 SHALL 使用可空类型（指针），区分"不更新状态"（null/不传）和"禁用用户"（0）。

#### Scenario: 不更新状态

- **WHEN** 前端调用 update 接口，请求体不包含 `status` 字段或 `status=null`
- **THEN** 用户状态保持不变

#### Scenario: 禁用用户

- **WHEN** 前端调用 update 接口，请求体 `status=0`
- **THEN** 用户状态更新为 0（禁用）

#### Scenario: 启用用户

- **WHEN** 前端调用 update 接口，请求体 `status=1`
- **THEN** 用户状态更新为 1（启用）

## REMOVED Requirements

无
