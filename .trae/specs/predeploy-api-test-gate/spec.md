# 部署前全接口回归测试门禁 Spec

## Why

Jenkins 每次上线部署直接编译重启，**没有任何自动化测试门禁**。此前多次因 SQL 与表结构不一致（如 `get_by_course_id` 报 `Unknown column 's.is_delete'`）、接口参数解析失败等运行时错误，直到部署后才暴露。需要在每次部署前自动跑一套**覆盖所有业务接口**的回归测试，测试通过才允许部署，把问题挡在上线前。

## What Changes

- **新增部署前集成测试包** `class_times_record_back/poc/predeploy/`：通过 Gateway HTTP 调用覆盖 auth-service / business-service / admin-service **全部业务接口**（约 115 个），断言 HTTP 200 + 标准 JSON 结构（`code` 字段）+ 关键业务断言
- **deploy.sh 新增 `test` 子命令**：编译 4 个服务 → 以**独立测试端口**启动整套服务（不与生产 9999/10001/10002/10003 冲突）→ 等待就绪 → 运行 `go test ./poc/predeploy/...` → 清理测试实例 → 返回测试结果
- **Jenkinsfile 新增 `Pre-Deploy Test` 阶段**：位于 `Sync Source to Host` 之后、`Build & Deploy` 之前；测试失败即中止流水线（不进入部署）；仅在 `SKIP_BUILD=false` 时执行
- **测试账号注入**：管理端/小程序端测试账号密码通过环境变量注入（不硬编码在代码/脚本中）
- **BREAKING**: deploy.sh 新增子命令（向后兼容，原有命令不变）；Jenkinsfile 流程增加测试阶段

## Impact

- **Affected specs**: 部署流水线质量门禁、业务接口回归覆盖
- **Affected code**:
  - 新增 `class_times_record_back/poc/predeploy/`（测试包：配置/HTTP客户端/SM3签名/SM2加密/登录辅助 + 三服务接口测试）
  - `class_times_record_back/deploy/deploy.sh`（新增 `test` 子命令）
  - `class_times_record_back/pipeline/Jenkinsfile`（新增 Pre-Deploy Test 阶段）
  - `class_times_record_back/CLAUDE.md`（部署流程说明）
- **运行环境**: 测试在宿主机执行（Go 编译依赖宿主机），独立测试端口 `19999/20001/20002/20003`，复用 `config.prod.yml`（DB/Redis/SM2/JWT 与生产同源）

## ADDED Requirements

### Requirement: 部署前测试门禁

每次 Jenkins 部署流水线在正式部署前 SHALL 运行覆盖全部业务接口的集成测试；测试全部通过才进入部署阶段，任一接口测试失败则中止流水线。

#### Scenario: 测试通过放行部署
- **WHEN** 触发 Jenkins 构建（SKIP_BUILD=false），源码同步完成
- **THEN** 流水线先编译并以测试端口启动服务，运行全接口回归测试；全部通过后继续 Build & Deploy

#### Scenario: 测试失败阻止部署
- **WHEN** 任一业务接口测试失败（如 SQL 报错、参数解析失败、鉴权失败）
- **THEN** 流水线在该阶段失败中止，不执行部署，post.failure 提示查看测试日志

### Requirement: 全业务接口测试覆盖

测试包 SHALL 覆盖三个服务在 handler 中注册的全部路由（auth ~20 个、business ~35 个、admin ~60 个），每个接口至少断言：请求可路由、返回 HTTP 200、响应为标准 `{code, message, data, requestTime}` 结构，且 `code` 为 200 或接口预期的业务错误码（参数校验/鉴权错误码也视为接口工作正常）。

#### Scenario: 查询类接口
- **WHEN** 使用有效参数调用（如 `/biz/student/get_by_institution_id`）
- **THEN** 返回 `code=200` 且 `data` 结构符合预期

#### Scenario: 写操作类接口
- **WHEN** 使用专用测试参数调用（如 `/admin/user/insert`、`/biz/course_record/deduct_by_student_id`）
- **THEN** 返回 `code=200` 或接口明确的业务错误码，且不污染生产数据（测试数据在测试后清理，或用幂等方式验证）

### Requirement: 独立测试环境

测试 SHALL 使用独立端口启动服务实例（gateway=19999、business=20001、auth=20002、admin=20003），通过 `GATEWAY_PORT`/`BIZ_PORT`/`AUTH_PORT`/`ADMIN_PORT` 及 `GATEWAY_AUTH_URI`/`GATEWAY_BUSINESS_URI`/`GATEWAY_ADMIN_URI` 环境变量覆盖，测试结束后停止实例并清理。

#### Scenario: 测试实例与生产隔离
- **WHEN** 执行 `deploy.sh test`
- **THEN** 生产服务（9999/10001/10002/10003）不受影响，测试实例仅在测试期间存活，结束后进程停止、端口释放

### Requirement: 测试账号配置

管理端（`/admin/user/login`）与小程序端（`/auth/auth/login_by_pwd`，需 SM2 加密密码 + SM3 签名）测试账号 SHALL 通过环境变量注入（如 `PRE_TEST_ADMIN_USERNAME/PASSWORD`、`PRE_TEST_MINI_ACCOUNT/PASSWORD`），未配置时测试跳过登录相关用例并告警，不硬编码凭据。

#### Scenario: 未配置测试账号
- **WHEN** 部署测试环境未设置测试账号环境变量
- **THEN** 登录相关用例被跳过（t.Skip），其余公开接口用例仍执行，测试结果如实反映

## MODIFIED Requirements

### Requirement: deploy.sh 子命令扩展

**原状**：`deploy.sh` 仅支持 `deploy/restart/stop/status/logs`。

**修改后**：新增 `test` 子命令：`bash deploy.sh test <scope>`（scope 默认 all），流程为编译 → 测试端口启动 → 运行 `go test ./poc/predeploy/...` → 清理测试实例 → 返回测试结果（成功 0 / 失败非 0）。

### Requirement: Jenkinsfile 流水线阶段

**原状**：`Pull Code → Fix Broken Workspaces → Sync Source to Host → Build & Deploy → Verify`。

**修改后**：在 `Sync Source to Host` 与 `Build & Deploy` 之间插入 `Pre-Deploy Test` 阶段（`when { expression { params.SKIP_BUILD == false } }`），通过 SSH 在宿主机执行 `bash /opt/go-deploy/deploy.sh test all`，失败即中止。

## REMOVED Requirements

无。
