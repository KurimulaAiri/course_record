# CLAUDE.md

Course recording system (课时记录系统) — four modules in a monorepo:

| Module | Directory | Stack | Purpose |
|--------|-----------|-------|---------|
| Admin Frontend | `class_record_admin_front/` | Vue 3 + Element Plus + Vite 8 + pnpm | Management dashboard |
| Mini Program | `class_times_record/` | uni-app (Vue 3) + Vite 5 + pnpm | WeChat mini-program for teachers & parents |
| Backend | `class_times_record_back/` | Spring Cloud Alibaba (Java 21, Maven) | Microservices: gateway, auth, business, admin |
| MCP Server | `course_record_mcp_server/` | Node.js + TypeScript + MCP SDK | Ops API (Jenkins/Nacos/Sentinel/Docker/MySQL) |

---

## Global Rules

- **API paths must include Gateway prefix**: `/admin/`, `/biz/`, or `/auth/`. Never hardcode without prefix.
- **Temp files**: All temporary files go in project root `.temp/` (gitignored). No temp files elsewhere.
- **Vue components**: `<script setup lang="ts">` + Composition API. Admin uses separate `.scss` files. Mini program uses SCSS with global variables auto-injected.
- **Frontend types**: Global `.d.ts` files in `types/` — no import needed.
- **DTO naming**: `{Action}{Entity}DTO` for request, `VO` for response. Admin-specific DTOs prefixed with `Admin`.
- **Java entities**: Inherit from `BaseEntity`. Teacher/Parent extend `RoleBaseEntity`. `Class` entity uses `clazz` package name (Java keyword conflict).

---

## Backend

### Architecture

Maven multi-module (parent POM at `class_times_record_back/pom.xml`):

| Module | Port | Service Name | Role |
|--------|------|-------------|------|
| `common` | — | — | Shared: entities, DTOs, VOs, converters (MapStruct), utils (SM2/SM3/JWT), service interfaces |
| `gateway` | 9999 | cr-gateway | JWT auth filter + routing |
| `auth-service` | 10002 | cr-auth-service | Auth, menu, permissions |
| `business-service` | 10001 | cr-business-service | Core business CRUD |
| `admin-service` | 10003 | cr-admin-service | Admin user/role/menu management |

**Call chain**: Controller → Service (interface, in common) → ServiceImpl (in microservice) → Mapper. **Controller must not inject Mapper directly.**

### Gateway Routes

All requests enter via Gateway (`/auth/**` → auth-service, `/biz/**` → business-service, `/admin/**` → admin-service), StripPrefix=1 strips the prefix.

| Prefix | Target Service |
|--------|---------------|
| `/auth/**` | `cr-auth-service` |
| `/biz/**` | `cr-business-service` |
| `/admin/**` | `cr-admin-service` |

Production: `lb://{service-name}`, DEV: `http://localhost:{port}`. Config in Nacos `cr-gateway.yaml` (prod) and `gateway/src/main/resources/application-dev.yml` (local).

### Security

**Mini Program auth flow**: SM2 encrypt password (cipherMode=1, "04" prefix) → backend SM2 decrypt → SM3+salt hash for storage. JWT (5-min expiry, HMAC-SHA256). Every request signed with SM3 (`x-sign`, `x-timestamp`, `x-nonce` headers). 401 triggers silent refresh. AuthServiceImpl 中 `userPlatformMapper.selectOne` 按 `lastLoginRole=3` 限定家长角色查询。

**Admin auth flow**: Password plaintext (HTTPS) → BCrypt hash for storage. JWT for session. No SM3 request signing.

### Interceptors per Service

- auth-service: `JwtInterceptor` + `SignInterceptor` + `UserInterceptor` (AuthWebConfig)
- admin-service: `AdminJwtInterceptor` + `UserInterceptor` (AdminWebConfig)
- business-service: `SignInterceptor` + `UserInterceptor` (common WebConfig)
- gateway: `JwtAuthFilter` (unified JWT check, public paths bypassed) + `GatewayUserFilter` (X-User-Id/X-User-Role → UserContext)

### Nacos Config

Namespace: `course-record`. All business config from Nacos; local `application.yml` only has Nacos connection info. Config files: `common-db.yaml`, `common-sentinel.yaml`, `cr-gateway.yaml`, `cr-{service}.yaml`. Managed on deployment server, not in local repo.

Nacos API available via MCP tools (`list_nacos_services`, `get_nacos_config`, `update_nacos_config`, etc.).

### Database

Database: `class_times_record` (utf8mb4) on `121.196.229.10:3306`. Access via MCP tools (`execute_db_query`, `execute_db_update`, `get_db_config`). DDL requires `allow_ddl=true`; DROP/TRUNCATE/GRANT/REVOKE always forbidden.

**Table naming convention**: Business tables use `c_` prefix (e.g., `c_institution`, `c_student`, `c_teacher`, `c_course`, `c_class`, `c_user`, `c_user_auth`, `c_user_platform`, `c_parent`, `c_parent_student`, `c_class_student`, `c_class_teacher`, `c_class_schedule`, `c_course_record`, `c_record`, `c_permission`, `c_permission_record`, `c_menu`, `c_admin`, `c_wx_subscribe_record`, `c_wx_student_subscribe`, `c_subscription_plan`). Admin/system tables use `sys_` prefix (e.g., `sys_user`, `sys_role`, `sys_menu`, `sys_role_menu`, `sys_user_role`, `sys_operation_log`, `sys_config`).

### Backend Conventions

**DTO/VO directory**: One subfolder per entity — `dto/{entity}/`, `vo/{entity}/`. Admin DTOs: `dto/admin/{entity}/`, prefixed `Admin`.

**Mapper**: One per entity, `EntityNameMapper`. Shared mappers in `common/src/.../mapper/`. Service-specific mappers in microservice `mapper/` package. **No duplicate mappers across services** — extract to common. `Class` → `ClazzMapper`.

**Entity inheritance**: `BaseEntity` (empty base) → `RoleBaseEntity` (user_id/is_available/username) → Teacher (+isInstitutionAdmin, replacing admin table association), Parent. Student inherits BaseEntity directly. SubscriptionPlan implements Serializable directly.

**user_auth.role_id mapping** (actually `permission.id`):

| role_id | Role | user_id maps to |
|---------|------|----------------|
| 1 | admin | sys_user.id |
| 3 | parent | c_parent.id |
| 4 | teacher | c_teacher.teacher_id |
| 5 | student | c_student.id |

### Database Relationships

```
c_institution 1──N c_teacher        c_institution 1──N c_course
c_institution 1──N c_student        c_course 1──N c_class
c_class N──N c_teacher (c_class_teacher)   c_class N──N c_student (c_class_student)
c_class 1──N c_class_schedule            c_student 1──N c_course_record
c_course 1──N c_course_record            c_course_record 1──N c_record
c_parent N──N c_student (c_parent_student, with is_primary/relation)
c_user 1──1 c_parent (c_parent.user_id→c_user.id, uk_user_id)
c_user 1──N c_user_platform (multi-device, each device has open_id)
c_parent 1──N c_wx_subscribe_record (per open_id independent tracking)
c_institution 1──1 c_subscription_plan (subscription_plan_id)

sys_user N──N sys_role (sys_user_role)   sys_role N──N sys_menu (sys_role_menu)
```

**Core tables**: c_institution, c_teacher, c_student, c_parent, c_course, c_class, c_class_schedule, c_course_record, c_record, c_subscription_plan
**Link tables**: c_user_auth, c_user, c_user_platform, c_class_teacher, c_class_student, c_parent_student, c_wx_subscribe_record, c_wx_student_subscribe, c_permission_record, c_permission, c_menu
**Admin tables**: sys_user, sys_role, sys_menu, sys_user_role, sys_role_menu, sys_operation_log, sys_config

### Adding a Backend Feature

1. Determine domain (auth/business/admin)
2. Entity → `common/src/.../repository/entity/`
3. DTO/VO → `common/src/.../repository/dto/{entity}/` and `vo/{entity}/`
4. Converter → `common/src/.../converter/`
5. Mapper (shared → common; service-specific → microservice)
6. Service interface → `common/src/.../service/`
7. ServiceImpl → corresponding microservice
8. Controller → corresponding microservice (calls Service only, never Mapper)
9. Route → `application-dev.yml` / Nacos `cr-gateway.yaml` (if new prefix needed)

### Notable API Additions

| Endpoint | Method | Service | Description |
|----------|--------|---------|-------------|
| `/biz/teacher/delete` | POST | business-service | 删除教师及其关联 user_auth、user 记录 |
| `/biz/student/cancel_subscribe` | POST | business-service | 取消家长对学生的微信订阅通知 |
| `/admin/teacher_auth/toggle_institution_admin` | POST | admin-service | 切换教师机构管理员身份 |
| `/admin/teacher/delete` | POST | admin-service | 管理端删除教师 |

---

## Admin Frontend

**Stack**: Vue 3 + Element Plus + Vite 8 + TypeScript (strict) + Pinia + Vue Router 5 + Axios + SCSS + pnpm

### Key Conventions

- **Views structure**: `views/{module}/{page}/index.vue` + `index.scss` + `index.d.ts`
- **Styles**: Must be in `index.scss`, imported via `<style lang="scss" scoped src="./index.scss" />`. No inline styles.
- **Icons**: Use `@/components/icons` (`iconMap`/`getIconComponent`). No duplicate icon maps in pages.
- **Empty fields**: Use `formatEmpty` from `@/utils/format`.
- **No `any`**: Use types from `index.d.ts`.
- **Reused components/tools**: Must go in `src/components/`.
- **Token**: `Authorization: Bearer {accessToken}`, stored in localStorage `admin_token` / `admin_refresh_token`. 401 → auto refresh via `POST /admin/user/refresh`.
- **Login verification**: Slider CAPTCHA dialog (`vue3-slide-verify`) pops up after form validation passes, before sending login request.

### API Modules

Each module: `src/api/{module}/index.ts`, all paths start with `/admin/` or `/biz/`. Modules: auth, user, role, menu, dashboard, log, institution, student, teacher, teacher-auth, course, class, class-schedule, course-record, record, mini-menu.

**Admin teacher-auth module**: `/admin/teacher_auth/` — 教师账号管理（get/update_account/update_password）+ 机构管理员身份切换（`toggle_institution_admin`）。机构管理员标识存储在 `c_teacher.is_institution_admin` 字段，与系统管理员(admin表)是不同身份。

### Design System

Primary: `#e8a838` (amber), Sidebar: `#1a1f2e`, BG: `#f5f6fa`, Dark: `#0f1419`. Fonts: DM Sans (body) + Sora (titles).

### Adding an Admin Feature

1. Types → `src/types/*.d.ts` (global) + `views/{module}/{page}/index.d.ts` (page-local)
2. API → `src/api/{module}/index.ts` (path with `/admin/` prefix)
3. Page → `views/{module}/{page}/index.vue` + `index.scss` + `index.d.ts`
4. Route → `src/router/index.ts` (layout children)
5. Menu → layout `el-menu`

---

## Mini Program Frontend

**Stack**: uni-app (Vue 3) + Vite 5 + TypeScript + Pinia + sm-crypto (SM2+SM3) + pnpm. Target: WeChat mini-program (mp-weixin).

### Key Conventions

- **Page params**: Use `usePageData<T>()` to receive, `jump(ROUTES.XXX, data)` to send. **Never** manually parse `options.data` in `onLoad`.
- **Route jumps**: Use `ROUTES` constants from `src/config/routes.ts`. No hardcoded paths.
- **SubPackages**: Main package ≤ 2MB, business pages in subPackages.
- **SM2 cipherMode**: cipherMode=1 (C1C3C2), "04" prefix.
- **Sign consistency**: Frontend `stableStringify()` mimics backend Jackson sorting.
- **Token**: `Authorization: Bearer {accessToken}`. 401 → silent refresh via `/auth/auth/refresh`.

### Component-First (Critical)

**Forbidden**: Hand-written `<picker>`, `<input>`, `<uni-datetime-picker>` in pages. Must use:

| Component | Path | Purpose |
|-----------|------|---------|
| FormPage | `@/components/form-page/index.vue` | Form container, renders via `groups` config |
| FormGroup | `@/components/form-group/index.vue` | Form item renderer, `type`-driven |
| PageFooter | `@/components/page-footer/index.vue` | Bottom action buttons |
| SearchFilterBar | `@/components/search-filter-bar/index.vue` | Search & filter |
| FloatingActionButton | `@/components/floating-action-button/index.vue` | Floating action |
| EmptyState | `@/components/empty-state/index.vue` | Empty data placeholder (text + optional tip) |

FormGroup types: `input`, `textarea`, `radio` (needs `options`), `select` (needs `options`), `date` (optional `column: true`), `time`, `text` (`mode: "display"`), `number`, `switch` (needs `options`).

**Constants**: `ROLE` enum from `@/config/common` — `{ ADMIN: 1, PARENT: 3, TEACHER: 4, STUDENT: 5 }`. Use instead of magic numbers.

**EventChannel**: `jump()` 默认使用 EventChannel 传参（`useEventChannel=true`），事件名 `pageDataTransfer`。接收方使用 `usePageData<T>()`。redirect/relaunch 走 URL 传参。

### API Modules

Each module: `src/api/{module}/index.ts`, paths start with `/auth/` or `/biz/`. Modules: auth, menu, bind, student, teacher, class, course, course-record, class-schedule, institution, record.

### Adding a Mini Program Feature

1. Types → `src/types/*.d.ts`
2. API → `src/api/{module}/index.ts` (path with `/auth/` or `/biz/`)
3. Page → `src/pages/main/teacher/` or `parent/` + `index.vue`
4. Register → `pages.json` subPackages
5. Route → `src/config/routes.ts` ROUTES constant

---

## MCP Server

**Stack**: Node.js 22+ + TypeScript + @modelcontextprotocol/sdk + mysql2 + zod. Stdio transport, no HTTP port.

### Key Info

- Single file: `server.ts` — all tools registered there
- Env vars injected by MCP client (JENKINS_TOKEN, DB_PASSWORD required)
- New tool: `server.registerTool(name, { description, inputSchema }, handler)` with zod schema

### Available MCP Tools (by category)

- **MySQL**: `get_db_config`, `execute_db_query` (SELECT only, auto LIMIT), `execute_db_update` (INSERT/UPDATE/DELETE, DDL needs `allow_ddl=true`, no DROP/TRUNCATE/GRANT/REVOKE)
- **Jenkins**: `trigger_jenkins_job`, `list_jenkins_jobs`, `get_jenkins_builds`, `get_jenkins_build_log`, `get_jenkins_build_status`, `get_jenkins_queue`
- **Nacos**: `list_nacos_services`, `list_nacos_configs`, `get_nacos_config`, `update_nacos_config`, `get_nacos_service_instances`, `list_nacos_ai_mcp`, `get_nacos_ai_mcp`, `list_nacos_ai_prompt`, `list_nacos_ai_agent`, `list_nacos_ai_skill`
- **Sentinel**: `list_sentinel_apps`, `get_sentinel_machines`, `get_sentinel_flow_rules`, `get_sentinel_degrade_rules`, `remove_sentinel_machine`, `set_sentinel_flow_rule`, `delete_sentinel_flow_rule`, `set_sentinel_degrade_rule`, `delete_sentinel_degrade_rule`
- **Docker**: `list_docker_containers`, `get_docker_container_info`, `docker_container_action`, `get_docker_container_logs`, `list_docker_images`, `get_docker_system_info`, `remove_docker_image`, `prune_docker_images`

---

## Building & Running

### Backend

```bash
cd class_times_record_back
export JAVA_HOME="D:\JAVA\jdk\jdk21"
mvn clean package -DskipTests
# Start each in separate terminal (order: gateway → auth → business → admin)
java -jar gateway/target/gateway-1.0-SNAPSHOT.jar           # 9999
java -jar auth-service/target/auth-service-1.0-SNAPSHOT.jar # 10002
java -jar business-service/target/business-service-1.0-SNAPSHOT.jar # 10001
java -jar admin-service/target/admin-service-1.0-SNAPSHOT.jar # 10003
```

Requires Nacos (`nacos.kurimula-airi.top`) and MySQL (`121.196.229.10:3306`) reachable.

### Admin Frontend

```bash
cd class_record_admin_front && pnpm install && pnpm dev    # Proxies /admin, /auth, /biz → localhost:9999
```

### Mini Program

```bash
cd class_times_record && pnpm install && pnpm dev:mp-weixin
```

### Infrastructure (server, already running)

Nacos: `nacos.kurimula-airi.top:8848`, Sentinel: `sentinel.kurimula-airi.top:7819`, MySQL: `121.196.229.10:3306`, Nginx: `121.196.229.10:9080` → Gateway :9999.

### Deployment

Docker Compose (`class_times_record_back/docker-compose.yml`), all services `network_mode: host`. Jenkins CI/CD via `pipeline/Jenkinsfile`. DB credentials via MCP `get_db_config`.

---

## WeChat Subscribe Message

微信订阅消息为一次性授权模型。前端在用户 tap 同步调用栈中调用 `wx.requestSubscribeMessage`，授权后通过 `/auth/record_subscribe` 记录次数。按 `(parent_id, open_id, template_id)` 跟踪，同一家长多设备各自独立计数。教师扣课时，business-service 查家长 open_id，按 openId 去重推送（同一 openId 只发一次），发送成功则授权次数 -1。查询订阅状态只查当前 openId，不聚合其他设备。WeChatApiService 在 common 包，auth/business 共用。取消订阅接口：`POST /biz/student/cancel_subscribe`（删除 `c_wx_student_subscribe` 和 `c_wx_subscribe_record` 记录）。`c_wx_student_subscribe` 表以 `(student_id, is_primary)` 维度跟踪订阅关系，解耦 parent.userId 绑定链路。
