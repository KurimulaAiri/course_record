---
kind: configuration_system
name: 基于 Nacos 的微服务配置中心与多环境分层管理
category: configuration_system
scope:
    - '**'
source_files:
    - class_times_record_back/gateway/src/main/resources/application.yml
    - class_times_record_back/gateway/src/main/resources/application-dev.yml
    - class_times_record_back/auth-service/src/main/resources/application.yml
    - class_times_record_back/admin-service/src/main/resources/application.yml
    - class_times_record_back/business-service/src/main/resources/application.yml
    - class_times_record_back/auth-service/src/main/resources/application-dev.yml
    - class_times_record_back/admin-service/src/main/resources/application-dev.yml
    - class_times_record_back/business-service/src/main/resources/application-dev.yml
    - class_times_record_back/docs/nacos-common-redis.yaml
    - class_times_record_back/common/src/main/resources/logback-spring.xml
    - class_times_record_back/docker-compose.yml
    - class_record_admin_front/vite.config.ts
    - class_times_record/src/manifest.json
    - class_times_record/src/config/common.ts
    - course_record_mcp_server/server.ts
---

## 系统概述
本仓库采用 Spring Cloud Alibaba + Nacos 作为后端微服务的统一配置中心，配合环境变量、本地 profile 文件形成三层配置体系：Nacos 远程配置（共享/动态）→ application.yml（服务发现与导入声明）→ application-dev.yml（开发覆盖）。前端与 MCP 运维工具则通过构建期常量与环境变量注入实现轻量配置。

## 核心架构与约定
- 配置分层
  - 第一层：Nacos 远程配置。每个服务在 `application.yml` 中通过 `spring.config.import: optional:nacos:<data-id>.yaml?group=DEFAULT_GROUP&refresh=true` 声明式导入，支持运行时刷新。公共配置以 `common-*` 前缀命名（如 `common-db.yaml`、`common-redis.yaml`、`common-sentinel.yaml`），业务配置以 `<service-name>.yaml` 命名。
  - 第二层：应用内 `application.yml`。仅保留 `spring.application.name`、`spring.cloud.nacos.*` 连接参数以及 MyBatis-Plus mapper 扫描路径等不可变基础信息。
  - 第三层：`application-dev.yml`。仅在 `-Dspring.profiles.active=dev` 时生效，覆盖日志级别、MyBatis SQL 输出、Redis 远程地址、小程序码生成版本等开发期差异。
- 命名空间隔离：所有 Nacos 配置均位于 `${NACOS_NAMESPACE:course-record}` 命名空间，Group 固定为 `DEFAULT_GROUP`，通过环境变量 `NACOS_SERVER_ADDR` 切换 Nacos 服务端地址。
- 动态刷新：所有 `optional:nacos:...` 导入均带 `refresh=true`，修改 Nacos 配置后无需重启即可热更新。
- 日志配置：logback 通过 Nacos 的 `logback-spring.xml` 集中管理，`common/src/main/resources/logback-spring.xml` 提供 dev/profile 分支，默认异步控制台输出并屏蔽 Sentinel 心跳 INFO 噪音。
- 容器化注入：`docker-compose.yml` 通过 `environment` 注入 `NACOS_SERVER_ADDR`、`NACOS_NAMESPACE`、`SENTINEL_*`、HikariCP/Tomcat 线程池等运行期参数，不改动镜像内配置文件。

## 各模块配置要点
- Gateway（网关）
  - `gateway/src/main/resources/application.yml` 导入 `cr-gateway.yaml`、`common-sentinel.yaml`、`logback-spring.xml`。
  - `application-dev.yml` 使用直连 `localhost:1000x` 路由，避免本地调试走 `lb://` 命中线上未更新实例。
- Auth/Admin/Business 三个业务服务
  - 各自 `application.yml` 导入 `<service>.yaml` + `common-db.yaml` + `common-redis.yaml` + `common-sentinel.yaml` + `logback-spring.xml`。
  - `application-dev.yml` 额外导入 `common-redis-dev.yaml` 覆盖 Redis 远程地址，并开启虚拟线程与 MyBatis SQL 输出。
- 公共 Redis 配置示例见 `docs/nacos-common-redis.yaml`，需上传至 Nacos Data ID=`common-redis.yaml`。

## 前端与小程序配置
- 管理前端（Vue3 + Vite）：`class_record_admin_front/vite.config.ts` 通过 `server.proxy` 将 `/admin`、`/auth`、`/biz` 代理到本地 9999 端口；生产构建关闭 sourcemap 并按库拆分 chunk。无独立 `.env`，依赖后端网关统一前缀。
- 小程序（UniApp）：`class_times_record/src/manifest.json` 固化 appid、分包优化、平台权限等打包期配置；运行时常量集中在 `src/config/common.ts`（角色枚举、模板ID等）。
- MCP 运维 Server：`course_record_mcp_server/server.ts` 全部通过 `process.env.*` 读取 Jenkins/Nacos/Sentinel/Docker/DB 等外部系统凭据与地址，并提供 `MCP_TRANSPORT`、`MCP_PORT`、`NACOS_MCP_REGISTER` 等运行开关。

## 开发者应遵循的规则
1. **新增配置项优先放入 Nacos**：在对应 `<service>.yaml` 或 `common-*.yaml` 中维护，并通过 `${ENV_VAR:default}` 占位符暴露可覆盖点。
2. **禁止在代码中硬编码敏感值**：数据库密码、JWT 密钥、第三方 API Token 一律通过环境变量或 Nacos 配置注入。
3. **本地开发使用 `-Dspring.profiles.active=dev`**：利用 `application-dev.yml` 覆盖 Redis 地址、日志级别、SQL 输出，不要直接改 `application.yml`。
4. **公共配置加 `common-` 前缀**：跨服务复用的 DB/Redis/Sentinel/Logback 配置放在 `common-*.yaml`，由多个服务共同 import。
5. **Nacos 命名空间保持 `course-record`**：不同环境通过 `NACOS_NAMESPACE` 环境变量区分，不在代码里写死。
6. **前端/小程序不引入运行时配置中心**：打包期常量放 `vite.config.ts` / `manifest.json` / `config/*.ts`，运行时差异通过后端 API 返回。
7. **MCP Server 凭据走环境变量**：Jenkins/Nacos/Sentinel/DB 等连接信息通过 `process.env` 注入，严禁写入源码或提交到仓库。