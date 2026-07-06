# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Course recording system (课时记录系统) for educational training institutions. Four modules in a monorepo:

| Module | Directory | Stack | Purpose |
|--------|-----------|-------|---------|
| Admin Frontend | `admin-frontend/` | Vue 3 + Element Plus + Vite 8 + pnpm | Management dashboard for system admins |
| Mini Program Frontend | `frontend/uni-app/` | uni-app (Vue 3) + Vite 5 + pnpm | WeChat mini-program for teachers & parents |
| Backend | `backend/` | Spring Cloud Alibaba (Java 21, Maven multi-module) | Microservices: gateway, auth, business, admin |
| MCP Server | `mcp-server/` | Node.js + TypeScript + @modelcontextprotocol/sdk | Local ops API for Jenkins/Nacos/Sentinel/Docker/MySQL |

Detailed conventions and architecture for each module are documented in the sections below (originally in each module's `AGENTS.md`, now consolidated here).

## Backend Architecture (Spring Cloud Alibaba)

Maven parent POM at `backend/pom.xml` builds 5 modules in order:
- `common` — shared entities, DTOs, VOs, converters (MapStruct), utils (SM2/SM3/JWT), service interfaces
- `gateway` — Spring Cloud Gateway (port 9999, service `cr-gateway`), JWT auth filter, routing
- `auth-service` — auth/menu/permissions (port 10002, `cr-auth-service`)
- `business-service` — core business CRUD (port 10001, `cr-business-service`)
- `admin-service` — admin user/role/menu management (port 10003, `cr-admin-service`)

Gateway routes (defined in `gateway/src/main/resources/application-dev.yml` for local dev, and Nacos `cr-gateway.yaml` for production):
- `/auth/**` → StripPrefix=1 → `lb://cr-auth-service`
- `/biz/**` → StripPrefix=1 → `lb://cr-business-service`
- `/admin/**` → StripPrefix=1 → `lb://cr-admin-service`

All services use Nacos config center (namespace `course-record`). Config files are managed in Nacos (Data IDs: `common-db.yaml`, `common-sentinel.yaml`, `cr-gateway.yaml`, `cr-auth-service.yaml`, `cr-business-service.yaml`, `cr-admin-service.yaml`). Database is MySQL `class_times_record` at `121.196.229.10:3306`.

Call chain: **Controller → Service (interface, in common) → ServiceImpl (in microservice) → Mapper**

Infrastructure (already running on server, not containerized):
- Nacos: `nacos.kurimula-airi.top:8848`
- Sentinel: `sentinel.kurimula-airi.top:7819`
- MySQL: `121.196.229.10:3306`
- Nginx: `121.196.229.10:9080` (reverse proxy → Gateway :9999)

## Building & Running

### Backend

```bash
cd backend

# Set JDK 21
$env:JAVA_HOME = "D:\JAVA\jdk\jdk21"   # PowerShell
export JAVA_HOME="D:\JAVA\jdk\jdk21"    # Bash

# Build all modules (skip tests)
mvn clean package -DskipTests

# Start services (in order: gateway → auth → business → admin)
# Each in its own terminal:
java -jar gateway/target/gateway-1.0-SNAPSHOT.jar
java -jar auth-service/target/auth-service-1.0-SNAPSHOT.jar
java -jar business-service/target/business-service-1.0-SNAPSHOT.jar
java -jar admin-service/target/admin-service-1.0-SNAPSHOT.jar
```

Tests are skipped by default (`maven-surefire-plugin` config). Nacos and MySQL must be reachable.

### Admin Frontend

```bash
cd admin-frontend
pnpm install
pnpm dev              # Vite dev server (proxies /admin, /auth, /biz → localhost:9999)
pnpm build            # Production build
pnpm type-check       # vue-tsc --build
pnpm lint             # oxlint + eslint
pnpm test:unit        # Vitest
pnpm test:e2e         # Playwright
```

### Mini Program Frontend

```bash
cd frontend/uni-app
pnpm install
pnpm dev:mp-weixin    # Dev → WeChat mini-program (auto-opens WeChat dev tools)
pnpm build:mp-weixin  # Production build
pnpm type-check       # vue-tsc --noEmit
```

## Security: Crypto & Auth

Two distinct auth flows:

**Mini Program (teachers/parents):**
- Password encrypted with SM2 (C1C3C2, cipherMode=1, "04" prefix) → backend decrypts with SM2 private key → hashed with SM3 + salt for storage
- JWT tokens (5-min expiry) with HMAC-SHA256
- Every request signed with SM3 (`x-sign`, `x-timestamp`, `x-nonce` headers)
- 401 triggers silent refresh via refresh token

**Admin Dashboard:**
- Password transmitted in plaintext (HTTPS) → backend hashes with BCrypt
- JWT tokens for session
- No SM3 request signing (uses different interceptor chain)

## Key Conventions

- **DTO naming**: `{Action}{Entity}DTO` for request bodies (e.g., `InsertStudentDTO`), VO for responses (e.g., `StudentVO`). Admin-specific DTOs prefixed with `Admin`.
- **API paths**: Must include Gateway prefix (`/admin/`, `/biz/`, `/auth/`). Never hardcode without prefix.
- **Java entities**: Inherit from `BaseEntity`. Role-based entities (Teacher, Student, Parent) extend `RoleBaseEntity`. `Class` entity uses `clazz` package name (Java keyword conflict).
- **Temp files**: All temporary files go in project root `.temp/` directory (gitignored).
- **Vue components**: `<script setup lang="ts">` + Composition API. Admin uses separate `.scss` files (not inline styles). Mini program uses SCSS with global variables auto-injected.
- **Frontend types**: Global `.d.ts` files in `types/` — no import needed.
- **WeChat Subscribe Message**: 微信订阅消息为一次性授权模型。前端在用户 tap 同步调用栈中调用 `wx.requestSubscribeMessage`（微信要求必须在 tap 同步栈中调用），获取用户授权后，通过 `/auth/record_subscribe` 接口记录授权次数。按 `(parent_id, open_id, template_id)` 跟踪授权次数，同一家长多设备各自独立计数。教师扣课时，business-service 查询学生家长的 open_id，按 openId 去重后推送扣课通知（同一 openId 只发一次），发送成功则对应记录授权次数 -1。查询订阅状态时只查当前 openId 的记录，不聚合其他设备。WeChatApiService 位于 common 包，auth-service 和 business-service 共用。

## Docker Deployment

Docker Compose file (`backend/docker-compose.yml`) and Jenkins CI/CD pipeline (`backend/pipeline/Jenkinsfile`) are in the local repo. Gateway runs as a plain JAR on the host, not in a container.

---

# Backend Detailed Conventions

> 原文件：`backend/AGENTS.md`（已合并到根目录）

## 项目结构

```
backend/
├── common/                # 共享代码库：Entity、DTO、VO、Converter、Service 接口、工具类
│   └── src/main/java/com/shiroko/
│       ├── annotation/    # 自定义注解（BaseDateTimeToString、UpdateStudentCount 等）
│       ├── common/enums/  # 通用枚举（ResultCode）
│       ├── config/        # 通用配置（MyBatisPlusConfig、OpenApiConfig）
│       ├── context/       # 线程上下文（UserContext — ThreadLocal 用户信息）
│       ├── converter/     # MapStruct 转换器接口（@Mapper(componentModel="spring")）
│       ├── exception/     # 异常定义 + 全局异常处理器（GlobalExceptionHandler）
│       ├── filter/        # Servlet 过滤器（GatewayUserFilter、RequestCachingFilter）
│       ├── interceptor/   # 通用拦截器（SignInterceptor 签名校验、UserInterceptor 上下文清理）
│       ├── repository/
│       │   ├── dto/       # 数据传输对象（按业务分子目录：admin/、auth/、student/ 等）
│       │   ├── entity/    # MyBatis-Plus 实体类（继承 BaseEntity，@TableName 指定表名）
│       │   │   └── common/  # 基础实体（BaseEntity、RoleBaseEntity）
│       │   └── vo/        # 视图对象（按业务分子目录，返回给前端）
│       ├── mapper/        # 公用 Mapper 接口（InstitutionMapper、StudentMapper 等）
│       ├── service/       # 业务 Service 接口（跨模块共享）
│       └── util/          # 工具类
│           ├── JwtUtils.java           # JWT 令牌工具（AccessToken + RefreshToken）
│           ├── SM2Util.java            # 国密 SM2 非对称加解密
│           ├── SM3Util.java            # 国密 SM3 哈希摘要（带盐值）
│           ├── SM2KeyGenerator.java    # SM2 密钥对生成器
│           ├── DateTransformUtils.java # 日期格式转换
│           ├── InstitutionCodeUtil.java# 机构编码生成
│           └── WeChatApiService.java   # 微信API服务（access_token缓存、小程序码生成、订阅消息推送）
├── gateway/               # Spring Cloud Gateway（端口 9999，服务名 cr-gateway）
│   └── src/main/java/com/shiroko/gateway/
│       └── filter/        # JwtAuthFilter（JWT 校验 + 公开路径放行）
├── auth-service/          # 认证授权微服务（端口 10002，服务名 cr-auth-service）
│   └── src/main/java/com/shiroko/
│       ├── config/        # AuthWebConfig（注册 JwtInterceptor + SignInterceptor + UserInterceptor）
│       ├── controller/    # AuthController、MenuController、PermissionRecordController
│       ├── interceptor/   # JwtInterceptor（小程序 JWT 鉴权，依赖 UserService + UserConverter）
│       ├── mapper/        # 服务专属 Mapper（UserMapper、UserAuthMapper 等，公用 Mapper 在 common 包）
│       └── service/       # AuthService、UserService、IdentityService 等 + impl
├── business-service/      # 核心业务微服务（端口 10001，服务名 cr-business-service）
│   └── src/main/java/com/shiroko/
│       ├── aspect/        # AOP 切面（StudentCountAspect）
│       ├── controller/    # Institution、Student、Teacher、Course、Class 等业务 Controller
│       ├── mapper/        # 服务专属 Mapper（AdminMapper、UserMapper、UserAuthMapper 等，公用 Mapper 在 common 包）
│       └── service/       # 各业务 Service 接口 + impl
├── admin-service/         # 管理后台微服务（端口 10003，服务名 cr-admin-service）
│   └── src/main/java/com/shiroko/
│       ├── aspect/        # AOP 切面（OperationLogAspect）
│       ├── config/        # AdminWebConfig（注册 AdminJwtInterceptor + UserInterceptor）
│       ├── controller/    # SysUserController、SysRoleController、SysMenuController、SysDashboardController、SysOperationLogController、AdminBusinessController、CryptoController、TeacherAuthController
│       ├── interceptor/   # AdminJwtInterceptor（管理后台 JWT 鉴权，依赖 SysUserService）
│       ├── mapper/        # 服务专属 Mapper（SysUserMapper、SysRoleMapper、SysMenuMapper 等，公用 Mapper 在 common 包）
│       └── service/       # AdminBusinessService、TeacherAuthService + impl（SysUser/SysRole/SysMenu/SysDashboard 仅有 impl 无接口）
├── docker-compose.yml     # 生产环境容器编排（network_mode: host）
├── pipeline/Jenkinsfile   # Jenkins CI/CD 流水线
├── docs/                  # 文档和初始化 SQL
└── README.md              # 后端项目说明
```

> **注意**：`nacos-config/` 目录不在本地仓库，Nacos 配置文件由部署服务器管理，需上传至 Nacos 命名空间 `course-record`。

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 语言 | Java | 21 |
| 框架 | Spring Boot | 4.0.4 |
| 微服务 | Spring Cloud | 2025.1.1 |
| 微服务治理 | Spring Cloud Alibaba | 2025.1.0.0 |
| 网关 | Spring Cloud Gateway (WebFlux) | 5.0.1 |
| 服务注册/配置 | Nacos | 2.x |
| 流量控制 | Sentinel | 1.8.x |
| 数据访问 | MyBatis-Plus | 3.5.16 |
| 数据库 | MySQL | 8.x |
| 对象映射 | MapStruct | 1.5.5.Final |
| 密码编码 | Spring Security Crypto (BCrypt) | 7.0.4 |
| 国密算法 | Bouncy Castle (SM2/SM3) | — |
| 构建工具 | Maven 多模块 | — |
| 容器化 | Docker + Docker Compose | — |
| CI/CD | Jenkins (Docker) | — |

## Nacos 配置中心

### 命名空间

- 命名空间 ID：`course-record`
- Group：`DEFAULT_GROUP`

### Nacos API

Nacos API 已集成到 MCP Server（`mcp-server/server.ts`），可通过 MCP 工具直接调用：
- `list_nacos_services` — 列出注册服务
- `list_nacos_configs` — 列出配置文件
- `get_nacos_config` — 获取配置内容
- `update_nacos_config` — 更新配置
- `get_nacos_service_instances` — 获取服务实例
- `list_nacos_ai_mcp` — 列出 Nacos AI MCP 服务
- `get_nacos_ai_mcp` — 获取 Nacos AI MCP 服务详情
- `list_nacos_ai_prompt` — 列出 Nacos AI Prompt 模板
- `list_nacos_ai_agent` — 列出 Nacos AI A2A Agent
- `list_nacos_ai_skill` — 列出 Nacos AI Skill

### 数据库操作

数据库操作已集成到 MCP Server，可通过 MCP 工具直接执行 SQL：

| 工具 | 用途 | 安全限制 |
|------|------|----------|
| `get_db_config` | 获取数据库连接信息 | 仅返回连接串，不返回密码 |
| `execute_db_query` | 执行 SELECT/SHOW/DESC/EXPLAIN 查询 | 禁止写操作，自动加 LIMIT |
| `execute_db_update` | 执行 INSERT/UPDATE/DELETE 写操作，支持 DDL（需 allow_ddl=true） | 默认禁止 DDL，设置 allow_ddl=true 允许 ALTER/CREATE，始终禁止 DROP/TRUNCATE/GRANT/REVOKE |

**写操作规范**：
- 插入或修改数据时必须使用 `utf8mb4` 编码（工具已内置 `SET NAMES utf8mb4`）
- 推荐使用参数化查询（`params` 参数）防止 SQL 注入
- DDL 操作（ALTER/CREATE）需设置 `allow_ddl=true` 确认后执行，始终禁止 DROP/TRUNCATE/GRANT/REVOKE

### 配置文件上传

Nacos 配置文件由部署服务器管理（不在本地仓库），需上传至 Nacos 命名空间 `course-record`：

| Data ID | 文件 | 说明 |
|---------|------|------|
| `common-db.yaml` | 数据库 + MyBatis-Plus | auth/business/admin 共用 |
| `common-sentinel.yaml` | Sentinel 流控 | 所有服务共用 |
| `cr-gateway.yaml` | Gateway 路由 + JWT + CORS | Gateway 专属 |
| `cr-auth-service.yaml` | 端口 + JWT + 微信 + SM2 | Auth 专属 |
| `cr-business-service.yaml` | 端口 + JWT + SM2 | Business 专属 |
| `cr-admin-service.yaml` | 端口 + JWT | Admin 专属 |

### 本地 application.yml

各服务的 `application.yml` 仅保留 Nacos 连接信息 + `spring.config.import` 引用，业务配置全部由 Nacos 下发：

```yaml
spring:
  application:
    name: cr-xxx
  config:
    import:
      - optional:nacos:cr-xxx.yaml?group=DEFAULT_GROUP&refresh=true
      - optional:nacos:common-db.yaml?group=DEFAULT_GROUP&refresh=true
      - optional:nacos:common-sentinel.yaml?group=DEFAULT_GROUP&refresh=true
  cloud:
    nacos:
      server-addr: ${NACOS_SERVER_ADDR:nacos.kurimula-airi.top}
      discovery:
        namespace: ${NACOS_NAMESPACE:course-record}
      config:
        namespace: ${NACOS_NAMESPACE:course-record}
        file-extension: yaml
```

## common 包工具类

### 加密工具

| 工具类 | 算法 | 用途 | 使用场景 |
|--------|------|------|----------|
| `SM2Util` | 国密 SM2 非对称加密 | 前端密码加密传输 → 后端解密 | 登录/注册时密码解密 |
| `SM3Util` | 国密 SM3 哈希摘要 | 密码存储（带盐值） | `SM3Util.digestWithSalt(password, salt)` |
| `SM2KeyGenerator` | SM2 密钥对生成 | 生成 SM2 公私钥对 | 密钥初始化 |
| `JwtUtils` | HMAC-SHA256 | Access Token / Refresh Token 生成与校验 | 全链路鉴权 |
| BCrypt (Spring Security Crypto) | BCrypt | 管理后台密码存储 | admin-service SysUser 登录 |

**密码流程**：
- 小程序端：前端 SM2 公钥加密 → 后端 `SM2Util.decrypt()` 解密 → `SM3Util.digestWithSalt()` 哈希存储
- 管理后台端：前端明文传输（HTTPS）→ 后端 `BCryptPasswordEncoder.encode()` 哈希存储

### 其他工具

| 工具类 | 用途 |
|--------|------|
| `DateTransformUtils` | 日期格式转换 |
| `InstitutionCodeUtil` | 机构邀请码生成 |
| `UserContext` | ThreadLocal 用户上下文（请求级） |

## 跨模块约定

### DTO / VO / Entity / Converter

- `repository/dto/` 下的 DTO 类使用 `@Data` 注解，按业务分子目录
- `repository/vo/` 下的 VO 类使用 `@Data` 注解，返回前端
- `repository/entity/` 下的实体类继承 `BaseEntity`（含 id、创建/更新时间、逻辑删除），使用 `@TableName` 指定表名
- `converter/` 下的 Converter 接口使用 `@Mapper(componentModel = "spring")`

**目录规范**：
- **一个数据库表一个文件夹**：每个实体（表）对应的 DTO 和 VO 必须放在该实体的专属子目录中
  - DTO：`dto/{entity}/`，如 `dto/student/`、`dto/course/`、`dto/admin/student/`
  - VO：`vo/{entity}/`，如 `vo/student/`、`vo/course/`、`vo/admin/student/`
- 同一实体的所有 DTO（Insert/Update/Query）放在同一个 `dto/{entity}/` 目录
- 同一实体的所有 VO（详情/列表）放在同一个 `vo/{entity}/` 目录
- 若 Java 实体类名与保留字冲突（如 `Class`），目录名使用别名（如 `clazz`）

**命名规范**：
- **接收报文体（请求体）**必须以 `DTO` 结尾，如 `InsertStudentDTO`、`UpdateCourseRecordDTO`、`QueryInstitutionDTO`
- **返回报文体（响应体）**必须以 `VO` 结尾，如 `StudentVO`、`AdminInstitutionVO`、`QueryClassVO`
- Controller 入参统一使用 `@RequestBody XxxDTO`，出参统一使用 `ResponseDTO<XxxVO>`
- 若同一实体在 admin 端和业务端有不同的 DTO，admin 端 DTO 以 `Admin` 为前缀区分，如 `AdminInsertStudentDTO`，放于 `dto/admin/{entity}/` 包下

### Mapper

- **一个实体对应一个 Mapper**：每个数据库表/实体类只能有一个对应的 Mapper，命名规则为 `实体类名 + Mapper`（如 `Student` → `StudentMapper`、`Menu` → `MenuMapper`）
- **公用 Mapper**（被多个服务使用的）必须放在 `common/src/main/java/com/shiroko/mapper/` 下，对应的 XML 文件放在 `common/src/main/resources/com/shiroko/mapper/` 下
- **服务专属 Mapper**（仅单个服务使用的）放在对应微服务的 `mapper/` 包下
- **Class 实体特殊命名**：由于 `Class` 是 Java 保留字，对应 Mapper 命名为 `ClazzMapper`（泛型仍为 `BaseMapper<Class>`）
- **禁止重复**：同一实体的 Mapper 不得在多个服务中重复定义，必须提取到 common 包共享

### 调用链规范

严格遵循分层调用链：**Controller → Service（接口） → ServiceImpl（实现） → Mapper**

- **Controller 不得直接注入 Mapper**，只能调用 Service 接口方法
- Service 接口定义在 `common/src/.../service/` 中
- ServiceImpl 实现在各微服务模块中，使用 `@Service` 注解
- ServiceImpl 注入 Mapper 完成数据访问

### Controller

- 使用 `@RestController` + `@RequestMapping`
- 统一返回 `ResponseDTO<T>` 封装
- Auth Service 路径前缀 `/auth/auth/`，Business Service 路径前缀 `/biz/`，Admin Service 路径前缀 `/admin/`

### Gateway 路由

路由采用 **YAML 配置**（无 Java 编程式路由）：

**YAML 配置 — 生产 / DEV 直连本地**

Nacos `cr-gateway.yaml`（生产）和 `gateway/src/main/resources/application-dev.yml`（本地开发）使用相同格式，路径为 `spring.cloud.gateway.server.webflux.routes`：

```yaml
spring:
  cloud:
    gateway:
      server:
        webflux:
          routes:
            - id: cr-auth-service
              uri: lb://cr-auth-service          # 生产用 lb://，DEV 用 http://localhost:10002
              predicates:
                - Path=/auth/**
              filters:
                - StripPrefix=1
            - id: cr-business-service
              uri: lb://cr-business-service      # 生产用 lb://，DEV 用 http://localhost:10001
              predicates:
                - Path=/biz/**
              filters:
                - StripPrefix=1
            - id: cr-admin-service
              uri: lb://cr-admin-service         # 生产用 lb://，DEV 用 http://localhost:10003
              predicates:
                - Path=/admin/**
              filters:
                - StripPrefix=1
```

**uri 规范：**

| 环境 | 格式 | 示例 |
|------|------|------|
| 生产 | `lb://{service-name}` | `lb://cr-business-service` |
| DEV 本地 | `http://localhost:{port}` | `http://localhost:10001` |

Gateway 依赖 `spring-cloud-starter-loadbalancer` 实现 `lb://` 服务发现。

### 安全

- `JwtAuthFilter` (Gateway) — 统一 JWT 校验，公开路径放行
- `JwtInterceptor` (auth-service) — 小程序鉴权，依赖 UserService + UserConverter
- `AdminJwtInterceptor` (admin-service) — 管理后台鉴权，依赖 SysUserService
- `SignInterceptor` (common) — API 签名校验（auth-service、business-service 使用）
- `UserInterceptor` (common) — 请求结束清理 UserContext（所有服务使用）
- `GatewayUserFilter` (common) — 读取网关转发的 X-User-Id/X-User-Role 头设置 UserContext

### 各服务 WebConfig

- auth-service: `AuthWebConfig` — 注册 JwtInterceptor + SignInterceptor + UserInterceptor（排除 common 的 WebConfig）
- admin-service: `AdminWebConfig` — 注册 AdminJwtInterceptor + UserInterceptor（排除 SignInterceptor + WebConfig）
- business-service: 使用 common 的 `WebConfig`（注册 SignInterceptor + UserInterceptor）

### IdentityService（auth-service 内部）

auth-service 直接操作 Teacher/Parent 表查询身份信息，不再通过 Feign 调用 business-service：
- `IdentityService.getByUserId(roleName, userId)` — 查询角色记录
- `IdentityService.checkAvailable(roleName, userId)` — 检查身份是否可用
- `IdentityService.createIdentity(roleName, userId)` — 创建身份记录

## Adding a New Feature

1. **Determine domain**: auth（认证授权）、business（核心业务）、admin（管理后台）
2. **Add entity** in `common/src/.../repository/entity/`
3. **Add DTOs/VOs** in `common/src/.../repository/dto/` 和 `vo/`（按业务分子目录）
4. **Add converter** in `common/src/.../converter/`
5. **Add Mapper interface + XML** in `common/src/.../mapper/`（公用）或对应微服务（专属）
6. **Add Service interface** in `common/src/.../service/`
7. **Add ServiceImpl** in 对应微服务模块
8. **Add Controller** in 对应微服务模块（Controller 只调用 Service，不直接注入 Mapper）
9. **Add route** in `gateway/src/main/resources/application-dev.yml` or Nacos `cr-gateway.yaml`（如需新路径前缀）
10. **Add Nacos config**（如需新配置项，上传至 Nacos 命名空间 course-record，配置文件由部署服务器管理）

## 数据库关系说明

数据库名：`class_times_record`，字符集 utf8mb4。

### 核心业务表

| 表名 | 实体类 | 说明 | 关键字段 |
|------|--------|------|----------|
| `institution` | Institution | 机构 | id, institution_name, institution_address, institution_code, status(0待审核/1启用/2禁用) |
| `teacher` | Teacher | 教师（继承 RoleBaseEntity） | id, institution_id, user_id, is_available, username |
| `student` | Student | 学生（继承 BaseEntity） | id, student_name, institution_id, sex(0女/1男), birth, school, address |
| `parent` | Parent | 家长（继承 RoleBaseEntity） | id, phone, username, is_available, is_bound(false=占位符/true=已绑定), user_id(NULL=未绑定), create_time, update_time; 唯一约束 uk_user_id(user_id) |
| `course` | Course | 课程 | id, course_name, course_type, institution_id, is_available |
| `class` | Class | 班级 | id, course_id, class_name, student_count, student_max_count, status |
| `class_schedule` | ClassSchedule | 排课 | id, class_id, start_date, end_date, day_of_week, start_time, end_time |
| `course_record` | CourseRecord | 课程记录 | id, student_id, course_id, course_total_time, course_rest_time, course_status |
| `record` | Record | 课时变动记录 | id, course_record_id, record_time, record_type, record_change, operate_teacher_id |

### 关联表

| 表名 | 实体类 | 说明 | 关键字段 |
|------|--------|------|----------|
| `user_auth` | UserAuth | 用户认证（账号密码） | id, user_id, role_id, account, password(SM3), salt; 唯一约束 uk_user_role(user_id, role_id) |
| `user` | User | 小程序用户 | id, institution_id |
| `user_platform` | UserPlatform | 用户平台关联 | id, user_id, open_id, union_id, platform, is_available, last_login_time, last_login_role |
| `class_teacher` | ClassTeacher | 班级-教师关联（多对多） | id, class_id, teacher_id |
| `class_student` | ClassStudent | 班级-学生关联（多对多） | id, class_id, student_id |
| `parent_student` | ParentStudent | 家长-学生关联（多对多） | id, parent_id, student_id, is_primary, relation, create_time, update_time; 唯一约束 uk_student_isprimary(student_id, is_primary), uk_parent_student(parent_id, student_id) |
| `subscribe_record` | SubscribeRecord | 微信订阅消息授权记录 | id, parent_id, open_id, template_id, subscribe_count, create_time, update_time; 唯一约束 uk_parent_open_template(parent_id, open_id, template_id) |
| `permission_record` | PermissionRecord | 权限记录 | id, user_id, permission_id, record_id |
| `permission` | Permission | 权限 | id, permission_name, permission_type |
| `menu` | Menu | 小程序菜单 | id, menu_name, menu_url, menu_type |

### 管理后台表

| 表名 | 实体类 | 说明 | 关键字段 |
|------|--------|------|----------|
| `sys_user` | SysUser | 管理员用户 | id, username, nickname, password(BCrypt), salt(未使用), phone, email, avatar, status, is_deleted(逻辑删除) |
| `sys_role` | SysRole | 角色 | id, role_name, role_key, sort, status |
| `sys_menu` | SysMenu | 菜单 | id, parent_id, menu_name, menu_type(M目录/C菜单/F按钮), path, component, perms, icon, sort |
| `sys_user_role` | SysUserRole | 用户-角色关联 | id, user_id, role_id |
| `sys_role_menu` | SysRoleMenu | 角色-菜单关联 | id, role_id, menu_id |
| `sys_operation_log` | SysOperationLog | 操作日志 | id, user_id, username, operation, method, params, ip, duration |

### 实体继承关系

```
BaseEntity (空基类，无字段)
├── RoleBaseEntity (user_id / is_available / username)
│   ├── Teacher
│   └── Parent
└── Student (直接继承 BaseEntity，含 institution_id 等字段)
```

### 关键关系图

```
institution 1──N teacher        (机构拥有多个教师)
institution 1──N student        (机构拥有多个学生)
institution 1──N course         (机构拥有多个课程)
course      1──N class          (课程对应多个班级)
class       N──N teacher        (class_teacher 关联表，班级有多个教师)
class       N──N student        (class_student 关联表，班级有多个学生)
class       1──N class_schedule (班级有多个排课)
student     1──N course_record  (学生有多个课程记录)
course      1──N course_record  (课程有多个课程记录)
course_record 1──N record       (课程记录有多个课时变动)
parent      N──N student        (parent_student 关联表，含 is_primary/relation)
user        1──1 parent         (parent.user_id → user.id，uk_user_id 唯一约束)
user        1──N user_platform  (用户多设备登录，每个设备一个 open_id)
parent      1──N subscribe_record (家长多设备订阅，按 open_id 独立跟踪授权)
user_auth   1──1 teacher/student/parent  (user_id + role_id 关联，uk_user_role 唯一约束)

sys_user    N──N sys_role       (sys_user_role 关联表)
sys_role    N──N sys_menu       (sys_role_menu 关联表)
```

### user_auth 角色映射

`user_auth.role_id` 实际对应 `permission.id`（permission 表的 permission_name 字段标识角色名）：

| role_id | permission_name | 角色 | user_id 对应 |
|---------|-----------------|------|-------------|
| 1 | admin | 管理员 | sys_user.id |
| 2 | guest | 访客 | — |
| 3 | parent | 家长 | parent.id |
| 4 | teacher | 教师 | teacher.teacher_id |
| 5 | student | 学生 | student.id |

## 临时文件

所有临时文件（SQL 脚本、调试脚本、中间产物等）必须放在项目根目录的 `.temp/` 文件夹中，禁止在项目其他位置创建临时文件。`.temp/` 目录应加入 `.gitignore`。

## 相关文档

- [后端架构设计文档](backend/docs/architecture.md) — 系统架构、Gateway 过滤器、认证流程、数据库设计
- [后端 README](backend/README.md) — 项目介绍、功能列表、技术栈、构建说明

## Building & Running

### 部署架构

服务器上已有以下基础设施（非 Docker 容器）：

| 基础设施 | 地址 | 反向代理 |
|----------|------|----------|
| Nacos | `121.196.229.10:8848` | `nacos.kurimula-airi.top` |
| Sentinel | `121.196.229.10:7819` | `sentinel.kurimula-airi.top` |
| MySQL | `121.196.229.10:3306` | — |
| Nginx | `121.196.229.10:9080` | 反向代理到 Gateway |

只需将 4 个微服务打包为 Docker 镜像并在宿主机运行：

| 微服务 | 服务名 | 端口 | 容器名 |
|--------|--------|------|--------|
| Gateway | cr-gateway | 9999 | cr-gateway |
| Auth Service | cr-auth-service | 10002 | cr-auth-service |
| Business Service | cr-business-service | 10001 | cr-business-service |
| Admin Service | cr-admin-service | 10003 | cr-admin-service |

所有微服务使用 `network_mode: host`，直接访问宿主机网络。

### Jenkins CI/CD

Jenkins 运行在 Docker 中，通过 `pipeline/Jenkinsfile` 自动完成编译、构建镜像、部署。

**环境变量**：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `NACOS_ADDR` | `nacos.kurimula-airi.top` | Nacos 地址 |
| `NACOS_NAMESPACE` | `course-record` | Nacos 命名空间 |
| `SENTINEL_ADDR` | `sentinel.kurimula-airi.top` | Sentinel 地址 |
| `DB_ADDR` | `121.196.229.10` | MySQL 地址 |

### 数据库连接信息

数据库连接信息已迁移至 MCP Server，通过 `get_db_config` 工具获取。敏感凭据通过 MCP 环境变量注入，不再写入文档。

### Docker Compose（手动部署）

```bash
mvn clean package -DskipTests
docker compose up -d --build
docker compose ps
docker compose logs -f
```

### 本地开发

**前置条件**：
- JDK 21（路径 `D:\JAVA\jdk\jdk21`）
- 确保 Nacos 配置中心已上传配置文件（由部署服务器管理，不在本地仓库）
- 确保 Nacos 日志目录可写：`C:\Users\{user}\logs\nacos\` 和 `C:\Users\{user}\nacos\config\`

**方式一：直接运行 JAR（推荐）**

```powershell
# 设置 JAVA_HOME
$env:JAVA_HOME = "D:\JAVA\jdk\jdk21"

# 编译打包
mvn clean package -DskipTests

# 启动顺序：gateway → auth-service → business-service → admin-service
# 每个服务在独立终端运行
& "D:\JAVA\jdk\jdk21\bin\java" -jar gateway/target/gateway-1.0-SNAPSHOT.jar           # port 9999
& "D:\JAVA\jdk\jdk21\bin\java" -jar auth-service/target/auth-service-1.0-SNAPSHOT.jar # port 10002
& "D:\JAVA\jdk\jdk21\bin\java" -jar business-service/target/business-service-1.0-SNAPSHOT.jar # port 10001
& "D:\JAVA\jdk\jdk21\bin\java" -jar admin-service/target/admin-service-1.0-SNAPSHOT.jar # port 10003
```

**方式二：Maven 插件运行**

```powershell
$env:JAVA_HOME = "D:\JAVA\jdk\jdk21"
mvn clean package -DskipTests

# 启动顺序：gateway → auth-service → business-service → admin-service
cd gateway && mvn spring-boot:run           # port 9999
cd auth-service && mvn spring-boot:run      # port 10002
cd business-service && mvn spring-boot:run  # port 10001
cd admin-service && mvn spring-boot:run     # port 10003
```

**注意**：
- 不需要启动 Docker，直接运行 JAR 或 Maven 即可
- 服务依赖 Nacos 配置中心，需确保网络可访问 `nacos.kurimula-airi.top`
- 服务依赖远程 MySQL `121.196.229.10:3306`
- 如果 Nacos 日志写入失败（`拒绝访问`），需手动创建并授权 `C:\Users\{user}\logs\nacos\` 目录

---

# Admin Frontend Detailed Conventions

> 原文件：`admin-frontend/AGENTS.md`（已合并到根目录）

## 项目结构

```
admin-frontend/
├── src/
│   ├── api/                # API 接口层（按业务模块分目录）
│   ├── components/         # 共享组件和工具（图标映射等）
│   ├── config/             # 配置（帮助内容等）
│   ├── router/             # Vue Router 路由配置
│   ├── stores/             # Pinia 状态管理
│   ├── styles/             # 全局样式（SCSS）
│   ├── types/              # 全局类型声明（.d.ts，无需 import）
│   ├── utils/              # 工具函数（请求封装、格式化等）
│   └── views/              # 页面组件
├── pipeline/
│   └── Jenkinsfile         # Jenkins CI/CD 流水线
└── README.md               # 项目说明
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 框架 | Vue 3 (Composition API + `<script setup>`) |
| 语言 | TypeScript (strict) |
| 构建 | Vite 8 |
| UI 库 | Element Plus 2.x + @element-plus/icons-vue |
| 状态管理 | Pinia (Composition Store 风格) |
| 路由 | Vue Router 5 (history 模式) |
| HTTP | Axios |
| 样式 | SCSS |
| 包管理 | pnpm |

## 目录约定

### api/

按业务模块分目录，每个模块一个 `index.ts`：

| 目录 | Gateway 前缀 | 目标服务 | 说明 |
|------|------------|---------|------|
| `auth/` | `/admin/user/` | admin-service | 管理员登录、token续签 |
| `user/` | `/admin/user/` | admin-service | 系统用户 CRUD |
| `role/` | `/admin/role/` | admin-service | 角色 CRUD |
| `menu/` | `/admin/menu/` | admin-service | 菜单 CRUD |
| `dashboard/` | `/admin/dashboard/` | admin-service | 仪表盘数据 |
| `log/` | `/admin/operation_log/` | admin-service | 操作日志 |
| `institution/` | `/admin/business/institution/` | admin-service | 机构管理（管理员通道） |
| `student/` | `/admin/business/student/` | admin-service | 学生管理（管理员通道） |
| `teacher/` | `/admin/business/teacher/` | admin-service | 教师管理（管理员通道） |
| `teacher-auth/` | `/admin/teacher_auth/` | admin-service | 教师认证审核 |
| `course/` | `/admin/business/course/` | admin-service | 课程管理（管理员通道） |
| `class/` | `/admin/business/class/` | admin-service | 班级管理（管理员通道） |
| `class-schedule/` | `/admin/business/class_schedule/` | admin-service | 排课管理（管理员通道） |
| `course-record/` | `/admin/business/course_record/` | admin-service | 学生课程管理（管理员通道） |
| `record/` | `/admin/business/record/` | admin-service | 课时记录（管理员通道） |
| `mini-menu/` | `/admin/business/mini_menu/` | admin-service | 小程序菜单管理（管理员通道） |

API 函数签名统一使用 `post<T>(url, data)` 形式，返回 `Promise<ApiResponse<T>>`。

### components/

共享组件和工具，供多个页面复用：

| 文件 | 内容 |
|------|------|
| `icons.ts` | Element Plus 图标映射（`iconMap`、`iconOptions`、`getIconComponent`） |
| `page-help/index.vue` | 页面帮助组件 |

**规则**：跨页面复用的组件类型、工具函数必须放在 `src/components/` 目录，禁止在多个页面中重复定义相同的图标映射或工具函数。

### types/

全局类型声明（.d.ts），tsconfig 自动 include，无需 import：

| 文件 | 内容 |
|------|------|
| `http.d.ts` | `ApiResponse<T>`（统一响应包装）、`PageData<T>`（分页数据） |
| `admin.d.ts` | `SysUser`/`SysUserVO`、`SysRole`/`SysRoleVO`、`SysMenu`/`SysMenuVO` 及其 DTO、`LoginSysUserVO`、`DashboardVO` |
| `business.d.ts` | `Institution`、`Student`/`InsertStudentDTO`/`UpdateStudentDTO`、`Teacher`、`Course`、`ClassInfo`、`ClassSchedule`、`CourseRecord`、`Record` 及其 DTO |
| `sm-crypto.d.ts` | sm-crypto 库类型声明 |

### views/

每个页面目录包含三个文件：

```
views/{module}/{page}/
├── index.vue      # Vue 组件（<script setup lang="ts">）
├── index.scss     # 页面样式（scoped SCSS）
└── index.d.ts     # 页面局部类型（查询表单、编辑表单等）
```

页面列表：

| 路由 | 目录 | 说明 |
|------|------|------|
| `/login` | `views/login/` | 登录页 |
| `/` | `views/layout/` | 侧边栏布局 |
| `/dashboard` | `views/dashboard/` | 仪表盘 |
| `/profile` | `views/profile/` | 个人信息 |
| `/system/user` | `views/system/user/` | 用户管理 |
| `/system/role` | `views/system/role/` | 角色管理 |
| `/system/menu` | `views/system/menu/` | 菜单管理 |
| `/system/permission` | `views/system/permission/` | 权限管理 |
| `/system/log` | `views/system/log/` | 操作日志 |
| `/business/institution` | `views/business/institution/` | 机构管理 |
| `/business/student` | `views/business/student/` | 学生管理 |
| `/business/teacher` | `views/business/teacher/` | 教师管理 |
| `/business/course` | `views/business/course/` | 课程管理 |
| `/business/class` | `views/business/class/` | 班级管理 |
| `/business/class-schedule` | `views/business/class-schedule/` | 排课管理 |
| `/business/course-record` | `views/business/course-record/` | 学生课程管理 |
| `/business/record` | `views/business/record/` | 课时记录 |
| `/business/mini-menu` | `views/business/mini-menu/` | 小程序菜单管理 |

### utils/

- `request.ts` — Axios 请求封装
  - baseURL 为空（通过 Vite proxy 转发）
  - 请求拦截器：自动附加 `Authorization: Bearer {token}` + SM3签名头
  - 响应拦截器：401 时自动尝试 refreshToken 续签，续签失败清除 token 并跳转 `/login`
  - 并发请求续签去重（`isRefreshing` + `refreshSubscribers`）
  - 导出方法：`get`、`post`、`put`、`del`
- `format.ts` — 格式化工具
  - `formatEmpty(val, placeholder)` — 空值显示"未设定"等占位文字
  - `formatTime(val)` — 时间格式化
  - `formatDate(val)` — 日期格式化
- `sm2.ts` — 国密加密工具
  - SM2 加密（C1C3C2 模式，cipherMode=1，加密结果拼接"04"前缀）
  - SM3 签名生成（`generateSign`）

### stores/

- `user.ts` — 管理员用户状态（token、refreshToken、userInfo、roles、menus），token 持久化到 localStorage `admin_token` 和 `admin_refresh_token`

### styles/

- `index.scss` — 全局样式，包含：
  - CSS 变量（暗色侧边栏 `#1a1f2e`、琥珀色强调 `#e8a838`、浅灰内容区 `#f5f6fa`）
  - Google Fonts（DM Sans + Sora）
  - Element Plus 主题覆盖
  - NProgress 琥珀色进度条
  - 全局工具类（`.page-container`、`.page-header`、`.search-bar`、`.pagination-wrapper`）

## 跨模块约定

### API 路径

前端 API 路径 = Gateway 前缀 + Controller 路径：

```
/admin/user/login       → Gateway 去掉 /admin → admin-service /user/login
/admin/role/list        → Gateway 去掉 /admin → admin-service /role/list
/biz/student/insert     → Gateway 去掉 /biz   → business-service /student/insert
```

**禁止**使用不带 `/admin/` 或 `/biz/` 前缀的路径。

### Token 格式

- 请求头：`Authorization: Bearer {accessToken}`
- 存储：`localStorage.getItem('admin_token')` 和 `localStorage.getItem('admin_refresh_token')`
- 401 时自动续签（refreshToken → 新双Token），续签失败清除并跳转 `/login`
- 续签接口：`POST /admin/user/refresh`，参数 `{ refreshToken }`

### 类型命名

- 实体：`SysUser`、`Student`、`Course` 等
- 视图对象：`{Entity}VO`（如 `SysUserVO`，含格式化时间字符串和关联 ID 列表）
- 请求 DTO：`Insert{Entity}DTO`、`Update{Entity}DTO`、`Query{Entity}DTO`、`Login{Entity}DTO`
- 响应包装：`ApiResponse<T>`、`PageData<T>`
- 页面局部类型：定义在 `views/{module}/{page}/index.d.ts`（如 `UserQueryForm`、`UserForm`）

### Vue 组件

- `<script setup lang="ts">` + Composition API
- 样式使用 SCSS，通过 `<style lang="scss" scoped src="./index.scss" />` 引入
- 禁止在 `<style>` 块内直接写样式，必须提取到 `index.scss`
- 表格使用 `v-loading` 加载状态
- 操作成功/失败使用 `ElMessage.success/error` 提示
- 删除操作使用 `ElMessageBox.confirm` 或 `el-popconfirm` 确认

### 文件命名

- 目录：kebab-case（`class-schedule/`、`course-record/`）
- API 模块目录：kebab-case（与 views 目录对应）
- 类型文件：kebab-case（`business.d.ts`、`http.d.ts`）

## 设计系统

| 变量 | 值 | 用途 |
|------|-----|------|
| `--color-primary` | `#e8a838` | 琥珀色强调（按钮、链接、选中态） |
| `--color-sidebar` | `#1a1f2e` | 侧边栏深色背景 |
| `--color-bg` | `#f5f6fa` | 内容区浅灰背景 |
| `--color-dark` | `#0f1419` | 登录页近黑背景 |
| `--color-text` | `#1a1f2e` | 主文本色 |
| `--color-text-secondary` | `#6b7280` | 次要文本色 |
| `--color-success` | `#10b981` | 成功/启用 |
| `--color-danger` | `#ef4444` | 危险/删除 |
| `--color-warning` | `#f59e0b` | 警告 |
| `--color-border` | `#e5e7eb` | 边框色 |

字体：DM Sans（正文）+ Sora（标题），通过 Google Fonts 加载。

## Adding a New Feature

1. **定义类型**：在 `src/types/` 对应 `.d.ts` 中定义实体/DTO/VO 类型；页面局部类型放在 `views/{module}/{page}/index.d.ts`
2. **添加 API**：在 `src/api/{module}/index.ts` 中添加函数，路径加 `/admin/` 或 `/biz/` 前缀
3. **创建页面**：在 `src/views/{module}/{page}/` 下创建 `index.vue` + `index.scss` + `index.d.ts`
4. **注册路由**：在 `src/router/index.ts` 的 layout children 中添加路由配置
5. **添加菜单**：在 `src/views/layout/index.vue` 的 el-menu 中添加菜单项

## 临时文件

所有临时文件（调试脚本、中间产物等）必须放在项目根目录的 `.temp/` 文件夹中，禁止在项目其他位置创建临时文件。`.temp/` 目录应加入 `.gitignore`。

## 相关文档

- [管理前端 README](admin-frontend/README.md) — 功能概览、技术栈、构建说明

## Building & Running

```bash
# 安装依赖
pnpm install

# 开发模式（Vite dev server，自动代理到 Gateway）
pnpm dev

# 类型检查
npx vue-tsc --noEmit

# 生产构建
pnpm build

# 预览构建产物
pnpm preview
```

### Vite 代理配置

开发模式下，Vite 自动将 API 请求代理到 Gateway：

| 前缀 | 代理目标 |
|------|---------|
| `/admin` | `http://localhost:9999` |
| `/auth` | `http://localhost:9999` |
| `/biz` | `http://localhost:9999` |

生产环境需通过 Nginx 反向代理到 Gateway（`http://localhost:9080`）。

## 注意事项

1. **API 路径必须带 Gateway 前缀**：`/admin/` 或 `/biz/` 或 `/auth/`
2. **Token 格式**：`Authorization: Bearer xxx`，存储在 `localStorage` 的 `admin_token` 和 `admin_refresh_token`
3. **类型文件无需 import**：`src/types/*.d.ts` 和 `views/**/index.d.ts` 全局可用
4. **样式必须分离**：页面样式写在 `index.scss`，通过 `src` 属性引入
5. **Element Plus 图标**：使用 `@/components/icons` 中的 `iconMap`/`getIconComponent`，禁止在页面中重复定义图标映射
6. **严格类型检查**：禁止使用 `any`，优先使用 `index.d.ts` 中定义的类型
7. **空字段展示**：使用 `@/utils/format` 中的 `formatEmpty` 显示"未设定"等占位文字
8. **组件复用**：跨页面复用的组件类型、工具函数必须放在 `src/components/` 目录

## 数据库信息

数据库连接信息通过 MCP Server 的 `get_db_config` 工具获取，敏感凭据不再写入文档。

---

# Mini Program Frontend Detailed Conventions

> 原文件：`frontend/uni-app/AGENTS.md`（已合并到根目录）

基于 uni-app + Vue 3 + TypeScript 的微信小程序前端，面向教育培训机构的教师和家长。

## 项目结构

```
frontend/uni-app/
├── src/
│   ├── api/                # API 接口层（按业务模块分目录）
│   ├── components/         # 通用组件
│   ├── config/             # 路由常量、数据映射
│   ├── pages/              # 页面（主包 + 分包）
│   ├── stores/             # Pinia 状态管理
│   ├── types/              # 全局类型声明（.d.ts，无需 import）
│   ├── utils/              # 工具函数
│   └── static/             # 静态资源
├── docs/                   # 文档
│   └── architecture.md     # 架构设计文档
└── README.md               # 项目说明
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 框架 | uni-app (Vue 3 Composition API) |
| 语言 | TypeScript (strict, noImplicitAny: false) |
| 构建 | Vite 5 + @dcloudio/vite-plugin-uni |
| 状态管理 | Pinia (Composition Store 风格) |
| 加密 | sm-crypto (SM2 + SM3 国密算法) |
| 包管理 | pnpm |
| 目标平台 | 微信小程序 (mp-weixin) 为主，兼容 H5 |

## 目录约定

- `src/api/{module}/index.ts` — API 接口函数，路径必须以 `/auth/` 或 `/biz/` 开头
- `src/types/*.d.ts` — 全局类型声明，tsconfig 自动 include，无需 import
- `src/stores/*.ts` — Pinia Composition Store（`defineStore('name', () => { ... })`）
- `src/utils/request/index.ts` — HTTP 请求封装（Token、签名、401 刷新）
- `src/utils/crypto/index.ts` — SM2 加密 + SM3 签名
- `src/utils/common/index.ts` — 通用工具（jump、showToast、parseData、usePageData）
- `src/utils/share/index.ts` — 分享工具
- `src/config/routes.ts` — 页面路由常量（ROUTES 对象），跳转必须使用
- `src/config/common.ts` — 通用配置常量
- `src/components/` — 通用组件（form-group、form-page、search-filter-bar 等）

## 核心模块

### api/

按业务模块分目录，每个模块一个 `index.ts`：

| 目录 | Gateway 前缀 | 目标服务 | 说明 |
|------|------------|---------|------|
| `auth/` | `/auth/auth/` | auth-service | 登录/登出/Token 刷新 |
| `menu/` | `/auth/menu/` | auth-service | 菜单查询 |
| `bind/` | `/auth/auth/` | auth-service | 家长绑定学生 |
| `student/` | `/biz/student/` | business-service | 学生 CRUD |
| `teacher/` | `/biz/teacher/` | business-service | 教师 CRUD |
| `class/` | `/biz/class/` | business-service | 班级管理 |
| `course/` | `/biz/course/` | business-service | 课程 CRUD |
| `course-record/` | `/biz/course_record/` | business-service | 课卡记录 + 扣课 |
| `class-schedule/` | `/biz/class_schedule/` | business-service | 班级课表 |
| `institution/` | `/biz/institution/` | business-service | 机构/校区 |
| `record/` | `/biz/record/` | business-service | 扣课记录 |

### utils/request

HTTP 请求封装，核心机制：
- **baseUrl**：development → `http://localhost:9080`，production → `https://api.kurimula-airi.top`
- **认证头**：`Authorization: Bearer {accessToken}`
- **签名头**：`x-sign`、`x-timestamp`、`x-nonce`（SM3 签名防篡改）
- **401 处理**：自动使用 refreshToken 静默刷新，刷新期间其他请求排队等待
- **刷新 URL**：`/auth/auth/refresh`

导出方法：`get`、`post`、`put`、`del`

### utils/crypto

- `encryptPassword(data)` — SM2 公钥加密，返回 "04" + 密文（cipherMode=1）
- `generateSign(params)` — SM3 签名，返回 `{ sign, timestamp, nonce }`

### utils/common

- `jump(path, data?, type?, useEventChannel?)` — 页面跳转（自动校验路径有效性）
- `showToast(msgOrOptions)` — 消息提示
- `parseData<T>(dataStr)` — 解析 URL JSON 参数
- `usePageData<T>(callback?)` — 接收页面参数（兼容 EventChannel + URL 兜底）
- `switchUser(role)` — 切换账号

### stores

- **user store**：`userInfo`（UserResponse 联合类型，roleId 区分角色）、`setUserInfo` / `clearUserInfo`
- **student store**：`studentList`、`studentInfo`、`setStudentInfo` / `clearAll`

### types/

全局类型声明（.d.ts），无需 import：

- `http.d.ts` — `ApiResponse<T>`（统一响应包装）、`LoginResponse`
- `user.d.ts` — `UserResponse`（联合类型，roleId=3 家长 / roleId=4 教师）
- `auth.d.ts` — `LoginByPwdRequest`、`RefreshTokenRequest`
- 各业务模块类型文件

### pages/

| 分包 | root | 页面数 | 说明 |
|------|------|--------|------|
| 主包 | - | 4 | 启动页、登录页、隐私政策、用户协议 |
| main | `pages/main` | 43 | 教师端 40 + 家长端 3 |
| class-record | `pages/class-record` | 5 | 家长端课时记录 |

## 跨模块约定

### API 路径

前端 API 路径 = Gateway 前缀 + Controller 类路径 + 方法路径：

```
/auth/auth/login_by_pwd    → Gateway 去掉 /auth → auth-service /auth/login_by_pwd
/biz/student/get_by_id     → Gateway 去掉 /biz  → business-service /student/get_by_id
```

**禁止**使用不带 `/auth/` 或 `/biz/` 前缀的路径。

### 类型命名

- Request：`{Action}{Entity}Request`（如 `InsertStudentRequest`）
- Response：`{Action}{Entity}Response`（如 `StudentListResponse`）
- 列表响应：含 `list` + `total` 字段

### Vue 组件

- `<script setup lang="ts">` + Composition API
- 样式使用 SCSS，全局变量自动注入
- 页面参数用 `usePageData<T>()` 接收，禁止手动解析 `options.data`
- 路由跳转用 `jump(ROUTES.XXX)` 或 `ROUTES.XXX`

### 页面参数传递规范

- **传递参数**：使用 `jump(ROUTES.XXX, data, type)` 传递，`data` 为任意对象
- **接收参数**：使用 `usePageData<T>(callback)` 接收，兼容 EventChannel 与 URL 兜底
- **禁止**在 `onLoad` 中手动调用 `parseData(options.data)` 解析 jump 传参，必须使用 `usePageData`
- **例外**：微信小程序码扫码进入时通过 `options.scene` 获取场景值，可在 `onLoad` 中直接处理

### 优先使用组件（Component-First）

**禁止**在页面中手写原生 `<picker>`、`<input>` 等基础控件，必须使用以下封装组件构建表单：

| 组件 | 路径 | 用途 |
|------|------|------|
| `FormPage` | `@/components/form-page/index.vue` | 表单页面容器，通过 `groups` 配置渲染 |
| `FormGroup` | `@/components/form-group/index.vue` | 表单项渲染器，支持 `type` 驱动 |
| `PageFooter` | `@/components/page-footer/index.vue` | 页面底部操作按钮 |
| `SearchFilterBar` | `@/components/search-filter-bar/index.vue` | 搜索筛选栏 |
| `FloatingActionButton` | `@/components/floating-action-button/index.vue` | 悬浮操作按钮 |

**FormPage / FormGroup 支持的 type 类型：**

| type | 渲染控件 | 说明 |
|------|---------|------|
| `input` | `<input>` | 文本输入 |
| `textarea` | `<textarea>` | 多行文本 |
| `radio` | 单选按钮组 | 需配置 `options` |
| `select` | `<picker mode="selector">` | 下拉选择，需配置 `options` |
| `date` | `<picker mode="date">` | 日期选择，可选 `column: true` 纵向布局 |
| `time` | `<picker mode="time">` | 时间选择 |
| `text` | 纯文本 | `mode: "display"` 展示模式 |
| `number` | 数字输入 | 数字键盘 |
| `switch` | 开关 | 需配置 `options` |

**示例 — date 类型（到期时间）：**
```ts
{
    label: "课程到期时间",
    key: "expireTime",
    type: "date",           // 使用 FormGroup 内置 picker，不要单独写 uni-datetime-picker
    column: true,           // 纵向布局（日历图标 + 边框）
    placeholder: "请选择到期时间",
}
// 返回值格式: "YYYY-MM-DD"，后端负责拼接 " 23:59:59"
```

**规则**：表单字段必须通过 FormPage + FormGroup 的 `type` 参数渲染，禁止在页面中手写 `<picker>`、`<uni-datetime-picker>` 等控件。日期用 `type: "date"`，时间用 `type: "time"`。

### 文件命名

- 目录/文件：kebab-case（`class-manage/`、`course-record.d.ts`）
- 路由常量：UPPER_SNAKE_CASE（`STUDENT_DETAIL`）

## Adding a New Feature

1. **定义类型**：在 `src/types/` 对应 .d.ts 中定义 Request/Response 类型
2. **添加 API**：在 `src/api/{module}/index.ts` 中添加函数，路径加 `/auth/` 或 `/biz/` 前缀
3. **创建页面**：在 `src/pages/main/teacher/` 或 `parent/` 下创建目录 + `index.vue`
4. **注册页面**：在 `pages.json` 的 subPackages 中注册
5. **添加路由**：在 `src/config/routes.ts` 中添加 ROUTES 常量

## 临时文件

所有临时文件（调试脚本、中间产物等）必须放在项目根目录的 `.temp/` 文件夹中，禁止在项目其他位置创建临时文件。`.temp/` 目录应加入 `.gitignore`。

## Building & Running

```bash
# 安装依赖
pnpm install

# 开发模式 - 微信小程序
pnpm dev:mp-weixin

# 开发模式 - H5
pnpm dev:h5

# 生产构建
pnpm build:mp-weixin

# 类型检查
pnpm type-check
```

## 注意事项

1. **API 路径必须带 Gateway 前缀**：`/auth/` 或 `/biz/`
2. **Token 格式**：`Authorization: Bearer xxx`
3. **SM2 cipherMode**：前后端统一使用 cipherMode=1（C1C3C2）
4. **类型文件无需 import**：`src/types/*.d.ts` 全局可用
5. **路由跳转用 ROUTES 常量**：禁止硬编码页面路径
6. **分包加载**：主包 ≤ 2MB，业务页面放 subPackages
7. **签名一致性**：前端 `stableStringify()` 模拟后端 Jackson 排序行为

## 相关文档

- [小程序前端架构设计文档](frontend/uni-app/docs/architecture.md) — 系统架构、核心机制、安全设计、页面路由、编码规范
- [小程序前端 README](frontend/uni-app/README.md) — 系统架构概览、Gateway 路由规则、技术栈

---

# MCP Server Detailed Conventions

> 运维 API MCP 服务器，统一封装 Jenkins / Nacos / Sentinel / Docker / MySQL 运维操作。

## 项目结构

```
mcp-server/
├── server.ts           # MCP Server 主入口（单文件实现，所有工具注册）
├── package.json        # 依赖与启动脚本
├── tsconfig.json       # TypeScript 配置（ES2022 + Node16 模块）
├── pnpm-lock.yaml      # pnpm 锁定文件
├── package-lock.json   # npm 锁定文件（兼容）
├── .gitignore          # Git 忽略配置
└── .vscode/
    └── settings.json   # VSCode 拼写检查配置
```

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 运行时 | Node.js | 22+ |
| 语言 | TypeScript | 5.7+ |
| MCP SDK | @modelcontextprotocol/sdk | ^1.29.0 |
| 数据库驱动 | mysql2 | ^3.22.5 |
| 参数校验 | zod | ^3.25.0 |
| 传输协议 | StdioServerTransport | MCP 标准输入输出 |
| 包管理 | pnpm / npm | — |

## 运行方式

```bash
cd mcp-server

# 安装依赖
pnpm install   # 或 npm install

# 启动 MCP Server（通过 stdio 与 MCP 客户端通信）
pnpm start     # 等价于 npx tsx server.ts
```

MCP Server 通过 **stdio** 与 MCP 客户端（如 TRAE IDE）通信，不暴露 HTTP 端口。由 TRAE MCP 配置注入环境变量启动。

## 环境变量

所有敏感信息通过 MCP 客户端环境变量注入，不在代码中硬编码：

### Jenkins

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `JENKINS_URL` | `https://jenkins.kurimula-airi.top` | Jenkins 地址 |
| `JENKINS_USER` | `KurimulaAiri` | Jenkins 用户名 |
| `JENKINS_TOKEN` | （空） | Jenkins API Token（必填） |

### Nacos

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `NACOS_URL` | `https://nacos.kurimula-airi.top` | Nacos 地址 |
| `NACOS_USER` | `nacos` | Nacos 用户名 |
| `NACOS_PASSWORD` | （内置） | Nacos 密码 |
| `NACOS_NAMESPACE` | `course-record` | Nacos 命名空间 |

### Sentinel

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SENTINEL_URL` | `https://sentinel.kurimula-airi.top` | Sentinel Dashboard 地址 |
| `SENTINEL_USER` | （空） | Sentinel 用户名 |
| `SENTINEL_PASSWORD` | （空） | Sentinel 密码 |

### Docker

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DOCKER_URL` | `https://docker.kurimula-airi.top` | Docker API 地址 |

### MySQL

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DB_HOST` | `121.196.229.10` | 数据库地址 |
| `DB_PORT` | `3306` | 数据库端口 |
| `DB_NAME` | `class_times_record` | 数据库名 |
| `DB_USER` | `class_times_record` | 数据库用户 |
| `DB_PASSWORD` | （空） | 数据库密码（必填） |

## 工具清单

MCP Server 注册了以下工具，按类别分组：

### 数据库工具（MySQL）

| 工具 | 用途 | 安全限制 |
|------|------|----------|
| `get_db_config` | 获取数据库连接信息 | 仅返回连接串，不返回密码 |
| `execute_db_query` | 执行 SELECT/SHOW/DESC/EXPLAIN 查询 | 禁止写操作，自动加 LIMIT |
| `execute_db_update` | 执行 INSERT/UPDATE/DELETE 写操作，支持 DDL（需 allow_ddl=true） | 默认禁止 DDL，设置 allow_ddl=true 允许 ALTER/CREATE，始终禁止 DROP/TRUNCATE/GRANT/REVOKE |

**写操作规范**：
- 插入或修改数据时必须使用 `utf8mb4` 编码（工具已内置 `SET NAMES utf8mb4`）
- 推荐使用参数化查询（`params` 参数）防止 SQL 注入
- DDL 操作（ALTER/CREATE）需设置 `allow_ddl=true` 确认后执行，始终禁止 DROP/TRUNCATE/GRANT/REVOKE

### Jenkins 工具

| 工具 | 用途 |
|------|------|
| `trigger_jenkins_job` | 触发构建任务（支持分支、部署范围、跳过构建、回滚参数） |
| `list_jenkins_jobs` | 列出所有任务及状态 |
| `get_jenkins_builds` | 获取任务构建历史 |
| `get_jenkins_build_log` | 获取构建日志（尾部 N 行） |
| `get_jenkins_build_status` | 获取指定构建状态详情（含参数） |
| `get_jenkins_queue` | 获取当前构建队列 |

**Jenkins CSRF 处理**：`jenkinsPostWithCrumb` 在同一 HTTP session 中先获取 CRUMB + session cookie，再发送 POST 请求。

### Nacos 工具

| 工具 | 用途 |
|------|------|
| `list_nacos_services` | 列出注册服务 |
| `list_nacos_configs` | 列出配置文件 |
| `get_nacos_config` | 获取配置内容 |
| `update_nacos_config` | 更新配置内容 |
| `get_nacos_service_instances` | 获取服务实例列表 |

**Nacos 认证**：使用 Bearer Token 认证，`ensureNacosToken` 自动登录并缓存 token，401 时自动重新登录。

### Nacos AI 工具

| 工具 | 用途 |
|------|------|
| `list_nacos_ai_mcp` | 列出 Nacos AI 注册中心的 MCP 服务 |
| `get_nacos_ai_mcp` | 获取 MCP 服务详情（含工具列表） |
| `list_nacos_ai_prompt` | 列出 Prompt 模板 |
| `list_nacos_ai_agent` | 列出 A2A Agent |
| `list_nacos_ai_skill` | 列出 Skill |

### Sentinel 工具

| 工具 | 用途 |
|------|------|
| `list_sentinel_apps` | 列出所有应用及机器状态 |
| `get_sentinel_machines` | 获取指定应用的机器列表 |
| `get_sentinel_flow_rules` | 获取流控规则 |
| `get_sentinel_degrade_rules` | 获取熔断降级规则 |
| `remove_sentinel_machine` | 移除失效机器 |
| `set_sentinel_flow_rule` | 创建或更新流控规则 |
| `delete_sentinel_flow_rule` | 删除流控规则 |
| `set_sentinel_degrade_rule` | 创建或更新熔断降级规则 |
| `delete_sentinel_degrade_rule` | 删除熔断降级规则 |

**Sentinel 认证**：使用 cookie 认证（`sentinel_dashboard_cookie`），`sentinelApi` 自动登录并缓存 cookie，401 时自动重新登录。

### Docker 工具

| 工具 | 用途 |
|------|------|
| `list_docker_containers` | 列出容器（支持过滤名称、包含已停止） |
| `get_docker_container_info` | 获取容器详情（状态、网络、环境变量） |
| `docker_container_action` | 容器操作（start/stop/restart） |
| `get_docker_container_logs` | 获取容器日志（尾部 N 行） |
| `list_docker_images` | 列出镜像（支持过滤、悬空镜像） |
| `get_docker_system_info` | 获取 Docker 系统信息（版本、容器数、磁盘使用） |
| `remove_docker_image` | 删除镜像 |
| `prune_docker_images` | 清理悬空镜像 |

## 架构设计

### 单文件实现

`server.ts` 是单文件实现，包含：
1. **环境变量读取**（顶部）
2. **HTTP Helper**（`httpFetch`、`basicAuth`）
3. **各服务认证 Helper**（`jenkinsPostWithCrumb`、`nacosLogin`、`sentinelLogin`、`dockerApi`）
4. **MCP Server 实例**（`new McpServer({ name: "Local-Ops-API", version: "1.0.0" })`）
5. **工具注册**（`server.registerTool`）
6. **启动入口**（`main()` 通过 `StdioServerTransport` 连接）

### 认证机制

| 服务 | 认证方式 | 缓存策略 |
|------|---------|----------|
| Jenkins | Basic Auth（user:token）+ CSRF CRUMB | 每次请求获取 CRUMB（session-based） |
| Nacos | Bearer Token（`/nacos/v1/auth/login`） | 内存缓存 `nacosToken`，401 自动重新登录 |
| Sentinel | Cookie（`/auth/login`） | 内存缓存 `sentinelCookie`，401 自动重新登录 |
| Docker | 无认证（通过反代限制访问） | — |
| MySQL | 用户名密码 | 每次请求创建新连接 |

### 安全限制

- **数据库查询**：仅允许 SELECT/SHOW/DESC/EXPLAIN，自动加 LIMIT
- **数据库写操作**：仅允许 INSERT/UPDATE/DELETE，禁止 DDL 和权限操作
- **敏感信息**：密码、Token 等通过环境变量注入，不在代码中硬编码
- **超时控制**：所有 HTTP 请求设置 10-15 秒超时（`AbortSignal.timeout`）

## 添加新工具

1. 在 `server.ts` 中使用 `server.registerTool(name, { description, inputSchema }, handler)` 注册
2. 使用 `zod` 定义参数 schema
3. 返回 `{ content: [{ type: "text", text: "..." }] }` 格式
4. 如需认证，复用现有的 `jenkinsAuthHeader`、`nacosHeaders`、`sentinelApi`、`dockerApi` 等 helper

**示例**：

```typescript
server.registerTool("my_new_tool", {
  description: "工具描述",
  inputSchema: {
    param1: z.string().describe("参数1说明"),
  },
}, async ({ param1 }) => {
  // 业务逻辑
  return { content: [{ type: "text", text: `结果: ${param1}` }] };
});
```

## 注意事项

1. **单文件实现**：所有工具都在 `server.ts` 中注册，不拆分文件
2. **环境变量必填**：`JENKINS_TOKEN` 和 `DB_PASSWORD` 必须配置，否则对应工具会返回错误
3. **Stdio 传输**：MCP Server 通过 stdio 通信，不暴露 HTTP 端口，由 MCP 客户端管理生命周期
4. **Nacos v3 API**：使用 `/nacos/v3/admin/` API 路径，兼容 Nacos 3.x
5. **Docker 日志解析**：`get_docker_container_logs` 处理 Docker stream 格式（8 字节头 + 数据）
6. **错误处理**：所有工具捕获异常并返回错误文本，不抛出异常中断 MCP 连接
