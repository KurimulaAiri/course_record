---
kind: dependency_management
name: 多语言 Monorepo 依赖管理策略
category: dependency_management
scope:
    - '**'
source_files:
    - class_times_record_back/pom.xml
    - class_times_record_back/common/pom.xml
    - class_record_admin_front/package.json
    - class_times_record/package.json
    - course_record_mcp_server/package.json
---

本仓库为前后端 + 小程序 + MCP 运维的 Monorepo，各子项目采用各自语言的包管理器独立管理依赖，未使用统一的 workspace/聚合工具。

## Java（Spring Cloud Alibaba 微服务）
- 构建系统：Maven，根 class_times_record_back/pom.xml 作为父 POM，通过 dependencyManagement 集中声明 Spring Boot、Spring Cloud、Spring Cloud Alibaba、MyBatis-Plus、Fastjson2、Hutool、MapStruct、BCrypt 等第三方版本，子模块仅引用 artifactId 不写版本号。
- 内部模块：common 作为共享 jar 被 gateway/auth-service/business-service/admin-service 四个业务模块复用，groupId 统一为 com.shiroko。
- 插件与编译器：maven-compiler-plugin 3.13.0，Java 21，注解处理器路径在父 POM 中统一管理 Lombok + MapStruct。
- 无私有 Maven 仓库配置，依赖从默认中央仓库解析。

## Vue3 后台前端（admin-frontend）
- 包管理器：同时存在 package.json、pnpm-lock.yaml、package-lock.json，表明团队可能在 pnpm 与 npm 之间切换或未统一锁定文件。
- 运行时依赖：Vue 3.5.x、Element Plus 2.14.x、Pinia 3、Axios、sm-crypto 等；开发依赖包含 Vite 8、TypeScript ~6、Vitest、Playwright、ESLint + Oxlint、Prettier。
- Node 引擎约束：engines.node 限定 ^20.19.0 || >=22.12.0。

## UniApp 小程序（class_times_record）
- 包管理器：同样并存 package.json、pnpm-lock.yaml、package-lock.json，依赖以 @dcloudio/uni-* 系列为主，版本对齐至 alpha 预发布号。
- 平台脚本：提供 dev/build 到 mp-weixin、mp-alipay、mp-harmony 等多个目标平台的命令，并集成微信开发者工具 CLI 打开产物目录。
- 运行时依赖：Vue 3.5.x、Pinia、dayjs、sm-crypto、vue-i18n。

## MCP 运维 Server（course_record_mcp_server）
- 包管理器：package.json + package-lock.json，依赖 @modelcontextprotocol/sdk、mysql2、zod。
- Dockerfile 显式 COPY package.json package-lock.json 后执行安装，Jenkinsfile 也校验 lock 文件一致性。

## 关键约定与待改进点
- 锁文件不一致：前端与小程序目录同时存在 pnpm-lock.yaml 与 package-lock.json，建议统一选定一个包管理器并在 .gitignore 中剔除另一个，避免 CI 行为不确定。
- 无私有源配置：未发现任何私有 NPM/Maven 仓库镜像或认证配置，所有依赖均拉取自公共仓库。
- 版本治理：Java 侧通过 BOM + properties 集中管控版本，值得在前端侧引入类似机制（如 pnpm overrides / npm overrides 或 monorepo workspace），当前各子项目各自维护依赖版本，存在升级不同步风险。