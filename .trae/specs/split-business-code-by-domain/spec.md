# 按业务对象拆分 Go 后端大文件 Spec

## Why
Go 后端多个业务文件行数过多（最大 2169 行），导致维护困难、代码定位效率低、合并冲突频繁。需按业务对象拆分到独立文件，提升可维护性。

## What Changes
- 将 8 个超 600 行的 Go 文件按业务对象拆分到独立文件
- 每个业务对象一个文件（如 `institution_mapper.go`、`student_service.go`）
- 公共部分（结构体定义、构造函数、辅助函数、路由注册）保留在主文件
- **BREAKING**：仅文件结构变更，不修改任何业务逻辑、方法签名、API 行为

## Impact
- Affected specs: 无（纯重构，不影响功能规范）
- Affected code:
  - `admin-service/internal/mapper/admin_business_mapper.go` (2169 行 → 拆分 10 个文件)
  - `auth-service/internal/service/auth_service.go` (1895 行 → 拆分 5 个文件)
  - `business-service/internal/mapper/mapper.go` (1534 行 → 拆分 9 个文件)
  - `business-service/internal/service/service.go` (1181 行 → 拆分 3 个文件 + vo.go)
  - `business-service/internal/handler/handler.go` (1146 行 → 拆分 8 个文件)
  - `admin-service/internal/service/admin_business_service.go` (1103 行 → 拆分 9 个文件)
  - `admin-service/internal/handler/admin_handler.go` (697 行 → 拆分 4 个文件)
  - `admin-service/internal/handler/business_handler.go` (686 行 → 拆分 12 个文件)

## 拆分原则

### 1. 文件命名规范
- Mapper 层：`{domain}_mapper.go`（如 `institution_mapper.go`）
- Service 层：`{domain}_service.go`（如 `institution_service.go`）
- Handler 层：`{domain}_handler.go`（如 `institution_handler.go`）
- admin-service 业务透传 Handler 加 `business_` 前缀避免与系统管理冲突（如 `business_institution_handler.go`）

### 2. 公共部分保留规则
主文件保留以下内容：
- 包声明和 import
- 核心聚合结构体定义（如 `AdminBusinessMapper`、`AuthService`、`BusinessHandler`）
- 构造函数（如 `NewAdminBusinessMapper`、`NewAuthService`）
- 辅助函数（如 `readBody`、`writeResponse`、`recordLog`、`scanXxx`）
- 路由注册函数（`RegisterRoutes`）
- 跨模块共享的 VO/DTO 类型（如 `UpdateResultVO`）

### 3. 业务对象文件包含内容
- 该业务对象的所有方法（挂在聚合结构体上）
- 该业务对象专属的 Request/VO/Row 类型定义
- 该业务对象专属的辅助函数

### 4. 跨文件依赖处理
- 导出类型（大写开头）随业务对象文件迁移，其他文件通过包名访问
- 未导出类型（小写开头）需评估使用范围：单文件使用→随业务对象；多文件使用→保留主文件
- 包级函数（如 `generateBindCode`）随业务对象迁移

## ADDED Requirements

### Requirement: 按业务对象拆分文件
系统 SHALL 将超 600 行的业务代码文件按业务对象拆分到独立文件，每个文件聚焦单一业务对象。

#### Scenario: 拆分后文件行数
- **WHEN** 拆分完成
- **THEN** 每个新生成的业务对象文件不超过 500 行
- **AND** 主文件仅保留公共部分，不超过 300 行

#### Scenario: 编译验证
- **WHEN** 拆分完成
- **THEN** `go build ./...` 编译通过
- **AND** `go vet ./...` 无警告
- **AND** 所有 API 行为不变

#### Scenario: 跨文件类型访问
- **WHEN** 业务对象 A 的方法引用业务对象 B 的类型
- **THEN** 通过包名访问（如 `mapper.AdminStudentRow`），无需 import 同包文件

## MODIFIED Requirements

### Requirement: 文件组织结构
原：8 个大文件（600-2169 行）集中所有业务逻辑
改：按业务对象拆分到 60+ 个独立文件，每个文件聚焦单一业务对象

## 拆分范围与目标文件

### P0：超 1500 行文件（3 个，优先拆分）

#### 1. admin_business_mapper.go (2169 行) → 拆分 10 个文件
- `admin_business_mapper.go`（主文件）：结构体、构造函数、`scanInstitution`、`formatXxxSQL` 辅助函数
- `institution_mapper.go`、`student_mapper.go`、`teacher_mapper.go`、`course_mapper.go`
- `class_mapper.go`、`class_schedule_mapper.go`、`course_record_mapper.go`、`record_mapper.go`
- `mini_menu_mapper.go`、`user_auth_mapper.go`

#### 2. auth_service.go (1895 行) → 拆分 5 个文件
- `auth_service.go`（主文件）：结构体、构造函数、VO 定义、常量
- `auth_service_login.go`：微信登录 + 账号密码登录 + Token 管理
- `auth_service_register.go`：注册流程
- `auth_service_bind.go`：绑定流程（最大模块，约 810 行）
- `auth_service_subscribe.go`：订阅 + 用户信息查询

#### 3. business-service/mapper.go (1534 行) → 拆分 9 个文件
- `mapper.go`（主文件）：包声明、import（如无公共结构体则仅保留 import）
- `institution_mapper.go`、`student_mapper.go`、`teacher_mapper.go`
- `parent_mapper.go`、`parent_student_mapper.go`
- `user_mapper.go`、`user_auth_mapper.go`、`user_platform_mapper.go`
- `wx_subscribe_mapper.go`

### P1：1000-1500 行文件（3 个）

#### 4. business-service/service.go (1181 行) → 拆分 4 个文件
- `service.go`（主文件）：VO 转换函数、`UpdateResultVO`、QueryXxxVO 包装类型
- `institution_service.go`、`student_service.go`、`teacher_service.go`

#### 5. business-service/handler.go (1146 行) → 拆分 8 个文件
- `handler.go`（主文件）：`BusinessHandler` 结构体、构造函数、`RegisterRoutes`、辅助函数
- `institution_handler.go`、`student_handler.go`、`teacher_handler.go`
- `class_handler.go`、`class_schedule_handler.go`、`course_handler.go`
- `course_record_handler.go`、`record_handler.go`

#### 6. admin_business_service.go (1103 行) → 拆分 9 个文件
- `admin_business_service.go`（主文件）：结构体、构造函数、`recordLog`、所有 Request DTO
- `institution_service.go`、`student_service.go`、`teacher_service.go`、`course_service.go`
- `class_service.go`、`class_schedule_service.go`、`course_record_service.go`
- `record_service.go`、`mini_menu_service.go`

### P2：600-1000 行文件（2 个，可选拆分）

#### 7. admin_handler.go (697 行) → 拆分 4 个文件
- `admin_handler.go`（主文件）：结构体、构造函数、`RegisterRoutes`、辅助函数、auth、crypto
- `user_handler.go`、`role_handler.go`、`menu_handler.go`、`operation_log_handler.go`

#### 8. business_handler.go (686 行) → 拆分 12 个文件
- `business_handler.go`（主文件）：仅保留 import（复用 admin_handler 公共部分）
- 按业务对象拆分：`business_institution_handler.go` 等 12 个文件
- **可选**：因单文件仅 686 行且每个业务对象较小（35-108 行），可合并相近模块为 4-6 个文件

## REMOVED Requirements
无（纯重构，不删除功能）
