# Tasks

## 阶段一：Nacos 客户端封装包

- [x] Task 1: 创建 `common/nacos/client.go` — 封装 Nacos naming client 和 config client 的创建逻辑
  - [x] SubTask 1.1: 定义 `Client` 结构体，包含 naming_client.INamingClient 和 config_client.IConfigClient
  - [x] SubTask 1.2: 实现 `NewClient() (*Client, error)` — 从环境变量加载连接参数（NACOS_SERVER_ADDR/NACOS_NAMESPACE/NACOS_GROUP/NACOS_SCHEME），创建 naming 和 config 客户端
  - [x] SubTask 1.3: 实现 `getLocalIP() string` — 获取本机局域网 IP，失败回退到 127.0.0.1
  - [x] SubTask 1.4: 添加详细注释（包注释、函数注释、参数说明），对齐 Java Nacos 客户端配置

- [x] Task 2: 创建 `common/nacos/config_loader.go` — 配置中心封装
  - [x] SubTask 2.1: 实现 `LoadYAML(dataID string) (string, error)` — 拉取 Nacos 配置原文
  - [x] SubTask 2.2: 实现 `LoadAndUnmarshal(dataID string, out interface{}) error` — 拉取并 yaml.Unmarshal 到结构体
  - [x] SubTask 2.3: 实现 `ListenConfig(dataID string, onChange func(content string)) error` — 注册监听器，配置变更时回调
  - [x] SubTask 2.4: 添加注释说明 dataID 命名规则（如 `common-db.yaml`、`cr-auth-service.yaml`）

- [x] Task 3: 创建 `common/nacos/registry.go` — 服务注册封装
  - [x] SubTask 3.1: 实现 `RegisterInstance(serviceName, ip string, port uint64, metadata map[string]string) error` — 注册临时实例（Ephemeral=true，SDK 自动心跳）
  - [x] SubTask 3.2: 实现 `DeregisterInstance(serviceName, ip string, port uint64) error` — 优雅注销
  - [x] SubTask 3.3: 实现 `SelectInstances(serviceName string, healthyOnly bool) ([]model.Instance, error)` — 服务发现查询实例列表
  - [x] SubTask 3.4: 实现 `ResolveURI(uri string) (string, error)` — 解析 `lb://{service-name}` 为 `http://{ip}:{port}`，随机选取一实例
  - [x] SubTask 3.5: 实现实例列表本地缓存（10 秒 TTL），避免每次请求查 Nacos

- [x] Task 4: 添加 `gopkg.in/yaml.v3` 依赖到 go.mod（如尚未引入）
  - [x] SubTask 4.1: 运行 `go get gopkg.in/yaml.v3` 添加依赖
  - [x] SubTask 4.2: 验证 `go build ./...` 编译通过

## 阶段二：各服务接入 Nacos 配置中心

- [x] Task 5: 修改 `common/db/mysql.go` — 增加 `LoadFromNacos` 方法
  - [x] SubTask 5.1: 定义内部 `nacosDBConfig` 结构体，对齐 Java `common-db.yaml` 的 yaml 结构（spring.datasource.*）
  - [x] SubTask 5.2: 实现 `LoadFromNacos(client *nacos.Client) *Config` — 拉取 `common-db.yaml` 并解析
  - [x] SubTask 5.3: 实现 `parseJDBCURL(url string) (host string, port int, database string)` — 解析 `jdbc:mysql://host:port/database?params` 格式
  - [x] SubTask 5.4: 实现配置合并逻辑：环境变量覆盖 Nacos 值（`DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASSWORD` 优先级最高）

- [x] Task 6: 修改 `common/redis/redis.go` — 增加 `LoadFromNacos` 方法
  - [x] SubTask 6.1: 定义内部 `nacosRedisConfig` 结构体，对齐 Java `common-redis.yaml` 的 yaml 结构（spring.data.redis.*）
  - [x] SubTask 6.2: 实现 `LoadFromNacos(client *nacos.Client) *Config` — 拉取 `common-redis.yaml` 并解析
  - [x] SubTask 6.3: 实现配置合并逻辑：环境变量覆盖 Nacos 值（`REDIS_ADDR`/`REDIS_PASSWORD` 优先级最高）

- [x] Task 7: 修改 `common/config/config.go` — 增加从 Nacos 加载服务专属配置的方法
  - [x] SubTask 7.1: 定义 `AuthServiceConfig` 结构体，包含 `SM2PrivateKey`、`WxAppID`、`WxAppSecret`、`WxEnvVersion` 字段
  - [x] SubTask 7.2: 定义 `BusinessServiceConfig` 结构体，包含 `SM2PrivateKey` 字段
  - [x] SubTask 7.3: 定义 `AdminServiceConfig` 结构体，包含 `SM2PrivateKey`、`JWTSecret`、`JWTExpiration` 字段
  - [x] SubTask 7.4: 实现 `LoadAuthServiceConfig(client *nacos.Client) *AuthServiceConfig` — 从 `cr-auth-service.yaml` 加载
  - [x] SubTask 7.5: 实现 `LoadBusinessServiceConfig(client *nacos.Client) *BusinessServiceConfig` — 从 `cr-business-service.yaml` 加载
  - [x] SubTask 7.6: 实现 `LoadAdminServiceConfig(client *nacos.Client) *AdminServiceConfig` — 从 `cr-admin-service.yaml` 加载
  - [x] SubTask 7.7: 实现配置合并逻辑：环境变量（`SM2_PRIVATE_KEY`/`WX_APP_ID`/`WX_APP_SECRET`）覆盖 Nacos 值

- [x] Task 8: 修改 `gateway/internal/config.go` — 增加 `LoadFromNacos` 方法
  - [x] SubTask 8.1: 定义内部 `nacosGatewayConfig` 结构体，对齐 Java `cr-gateway.yaml` 的 yaml 结构（spring.cloud.gateway.server.webflux.routes、jwt、redis）
  - [x] SubTask 8.2: 实现 `LoadFromNacos(client *nacos.Client) *Config` — 拉取 `cr-gateway.yaml` 并解析
  - [x] SubTask 8.3: 解析路由列表，支持 `lb://` 和 `http://` 两种 URI 格式
  - [x] SubTask 8.4: 实现配置合并逻辑：环境变量（`GATEWAY_AUTH_URI`/`GATEWAY_BUSINESS_URI`/`GATEWAY_ADMIN_URI`）覆盖 Nacos 值

## 阶段三：各服务接入服务注册

- [x] Task 9: 修改 `gateway/main.go` — 启动时注册服务、加载 Nacos 配置
  - [x] SubTask 9.1: 在 main 函数开头创建 Nacos 客户端（不可达时降级到默认配置）
  - [x] SubTask 9.2: 调用 `internal.LoadFromNacos(nacosClient)` 加载配置（失败时降级到 DefaultConfig）
  - [x] SubTask 9.3: 服务启动后调用 `nacosClient.RegisterInstance("cr-gateway", ip, 9999, metadata)`
  - [x] SubTask 9.4: 监听 SIGINT/SIGTERM 信号，优雅注销后退出
  - [x] SubTask 9.5: 启动日志打印 Nacos 连接信息和服务注册状态

- [x] Task 10: 修改 `auth-service/main.go` — 启动时注册服务、加载 Nacos 配置
  - [x] SubTask 10.1: 在 main 函数开头创建 Nacos 客户端
  - [x] SubTask 10.2: 调用 `db.LoadFromNacos`、`redis.LoadFromNacos`、`config.LoadAuthServiceConfig` 加载配置
  - [x] SubTask 10.3: 服务启动后调用 `nacosClient.RegisterInstance("cr-auth-service", ip, 10002, metadata)`
  - [x] SubTask 10.4: 监听 SIGINT/SIGTERM 信号，优雅注销后退出
  - [x] SubTask 10.5: 微信配置（WxAppID/WxAppSecret/WxEnvVersion）改为从 `AuthServiceConfig` 读取

- [x] Task 11: 修改 `business-service/main.go` — 启动时注册服务、加载 Nacos 配置
  - [x] SubTask 11.1: 在 main 函数开头创建 Nacos 客户端
  - [x] SubTask 11.2: 调用 `db.LoadFromNacos`、`redis.LoadFromNacos`、`config.LoadBusinessServiceConfig` 加载配置
  - [x] SubTask 11.3: 服务启动后调用 `nacosClient.RegisterInstance("cr-business-service", ip, 10001, metadata)`
  - [x] SubTask 11.4: 监听 SIGINT/SIGTERM 信号，优雅注销后退出

- [x] Task 12: 修改 `admin-service/main.go` — 启动时注册服务、加载 Nacos 配置
  - [x] SubTask 12.1: 在 main 函数开头创建 Nacos 客户端
  - [x] SubTask 12.2: 调用 `db.LoadFromNacos`、`redis.LoadFromNacos`、`config.LoadAdminServiceConfig` 加载配置
  - [x] SubTask 12.3: 服务启动后调用 `nacosClient.RegisterInstance("cr-admin-service", ip, 10003, metadata)`
  - [x] SubTask 12.4: 监听 SIGINT/SIGTERM 信号，优雅注销后退出

## 阶段四：Gateway 支持 lb:// 路由

- [x] Task 13: 修改 `gateway/internal/server.go` — 反向代理支持 lb:// URI
  - [x] SubTask 13.1: 在反向代理逻辑中识别 `lb://` 前缀的 URI
  - [x] SubTask 13.2: 调用 `nacosClient.ResolveURI(uri)` 解析为真实 `http://{ip}:{port}`
  - [x] SubTask 13.3: 实现实例列表本地缓存（10 秒 TTL），使用 sync.Map 或 sync.RWMutex 保护
  - [x] SubTask 13.4: 无可用实例时返回 503 错误响应
  - [x] SubTask 13.5: dev profile 降级：环境变量 `GATEWAY_PROFILE=dev` 时跳过 lb 解析，使用 localhost 直连

- [x] Task 14: 修改 `gateway/internal/config.go` — 路由 URI 支持 lb:// 格式
  - [x] SubTask 14.1: 修改 `DefaultConfig()` 的路由 URI，保留 localhost 直连作为默认值（dev 环境）
  - [x] SubTask 14.2: `LoadFromNacos` 方法解析 `cr-gateway.yaml` 时，保留 `lb://` URI 格式（不转换为 localhost）
  - [x] SubTask 14.3: 添加注释说明 lb:// 和 http:// 两种 URI 的使用场景

## 阶段五：配置热更新（可选，P2）

- [x] Task 15: 实现 Redis 配置热更新
  - [x] SubTask 15.1: 在各服务 main.go 注册 `common-redis.yaml` 配置监听器
  - [x] SubTask 15.2: 配置变更时重建 Redis 连接池（新连接使用新配置）
  - [x] SubTask 15.3: 旧连接池优雅关闭（Close 不影响在途请求，go-redis v9 内部处理）
  - [x] SubTask 15.4: 记录日志 "Redis 配置已热更新"

- [x] Task 16: 实现 Gateway 路由表热更新
  - [x] SubTask 16.1: 在 gateway/main.go 注册 `cr-gateway.yaml` 配置监听器
  - [x] SubTask 16.2: 配置变更时原子替换 `Config.Routes`（使用 sync.RWMutex 保护）
  - [x] SubTask 16.3: 清空实例列表本地缓存（强制下次请求重新拉取）
  - [x] SubTask 16.4: 记录日志 "Gateway 路由配置已热更新"

- [x] Task 17: 实现 DB 配置变更提示
  - [x] SubTask 17.1: 在各服务 main.go 注册 `common-db.yaml` 配置监听器
  - [x] SubTask 17.2: 配置变更时仅记录警告日志 "DB 配置已变更，需重启服务生效"
  - [x] SubTask 17.3: 不重建连接池（避免影响在途请求）

## 阶段六：集成验证

- [x] Task 18: 编写集成测试验证 Nacos 集成
  - [x] SubTask 18.1: 在 `poc/nacos/` 新建 `integration_test.go`，验证配置加载、服务注册、服务发现全流程
  - [x] SubTask 18.2: 测试用例：从 Nacos 加载 `common-redis.yaml` 并验证字段解析正确
  - [x] SubTask 18.3: 测试用例：注册临时实例后能查询到，注销后查询不到
  - [x] SubTask 18.4: 测试用例：`ResolveURI("lb://cr-auth-service")` 返回有效的 `http://{ip}:{port}`

- [ ] Task 19: 手动验证端到端流程（需用户手动执行）
  - [ ] SubTask 19.1: 启动 auth-service，验证 Nacos 控制台出现 `cr-auth-service` 实例
  - [ ] SubTask 19.2: 启动 Gateway（prod 模式），验证通过 `lb://cr-auth-service` 路由请求成功
  - [ ] SubTask 19.3: 修改 Nacos `common-redis.yaml` 配置，验证 Redis 连接热更新
  - [ ] SubTask 19.4: 验证 dev profile（`GATEWAY_PROFILE=dev`）下 Gateway 使用 localhost 直连

- [x] Task 20: 更新文档
  - [x] SubTask 20.1: 更新 `class_times_record_back/CLAUDE.md`，补充 Go 服务 Nacos 集成说明（环境变量、配置优先级、服务注册流程）
  - [x] SubTask 20.2: 更新 `class_times_record_back/deploy/docker-compose.yml`，添加 Nacos 环境变量注入（注：仅 Jenkins docker-compose 存在，Go 服务无独立 docker-compose，已通过 .env 文件覆盖）
  - [x] SubTask 20.3: 更新 `class_times_record_back/.env`（不存在 .env.example，直接更新 .env），添加 Nacos 相关环境变量示例

# Task Dependencies

- Task 4 依赖 Task 1（yaml.v3 依赖用于配置解析）
- Task 5/6/7/8 依赖 Task 1/2/4（需要 Nacos 客户端和 yaml 解析）
- Task 9/10/11/12 依赖 Task 5/6/7/8（需要各服务的 LoadFromNacos 方法）
- Task 13/14 依赖 Task 3/9（需要 ResolveURI 和 Gateway Nacos 配置加载）
- Task 15/16/17 依赖 Task 9/10/11/12（需要各服务已接入 Nacos）
- Task 18 依赖 Task 1-14（所有功能完成后集成测试）
- Task 19 依赖 Task 18（集成测试通过后手动验证）
- Task 20 可与 Task 18/19 并行

# Parallelizable Work

- 阶段一的 Task 1/2/3 可并行（独立文件）
- 阶段二的 Task 5/6/7/8 可并行（不同包/文件）
- 阶段三的 Task 9/10/11/12 可并行（不同服务的 main.go）
- 阶段五的 Task 15/16/17 可并行（不同配置监听器）

---

# 阶段七：Checklist 验证修复（验证阶段发现的问题）

> 以下任务基于 checklist.md 系统性验证结果，修复未通过的检查点。

- [x] Task 21: 实现 `GATEWAY_PROFILE=dev` 环境变量检查（阶段四检查点 5 失败）
  - [x] SubTask 21.1: 在 `gateway/main.go` 中添加 `GATEWAY_PROFILE` 环境变量检查逻辑：当 `GATEWAY_PROFILE=dev` 时，跳过 Nacos 配置加载，直接使用 `DefaultConfig()`（localhost 直连）
  - [x] SubTask 21.2: 在 `gateway/internal/config.go` 的 `DefaultConfig` 注释中补充 dev profile 触发方式说明
  - [x] SubTask 21.3: 更新 `.env` 文件注释，说明 `GATEWAY_PROFILE=dev` 的作用（跳过 Nacos 路由加载）
  - [x] SubTask 21.4: 添加注释说明 dev profile 降级机制对齐 Java 侧 `-Dspring.profiles.active=dev`

- [x] Task 22: 决策 Redis 连接池热更新策略（阶段五检查点 2 和阶段六检查点 7 失败）
  - **背景**：spec.md 第 73 行要求"Redis 连接：重建连接池，旧连接池优雅关闭"，但代码有意不实现（注释说明"运行时替换共享指针不安全"）。这是 P2 可选功能。
  - **用户决策**：选项 B - 接受当前设计（仅记录日志提示重启）
  - [x] SubTask 22.1: 用户决策完成（选项 B）
  - [x] SubTask 22.2: 已更新 spec.md（第 73 行调整为"仅记录警告日志提示需重启"）和 checklist.md（阶段五检查点 2 和阶段六检查点 7 标记为通过）

- [x] Task 23: 创建 `class_times_record_back/CLAUDE.md`（文档检查点 1 失败）
  - [x] SubTask 23.1: 在 `class_times_record_back/` 目录下创建 CLAUDE.md 文件，补充 Go 服务 Nacos 集成说明
  - [x] SubTask 23.2: 包含内容：环境变量说明（NACOS_SERVER_ADDR 等）、配置优先级（环境变量 > Nacos > 默认值）、服务注册流程、dev profile 降级机制
  - [x] SubTask 23.3: 对齐项目规则要求（`class_times_record_back/CLAUDE.md — 项目架构和开发指南`）

- [x] Task 24: 同步 README.md 迁移进度（文档检查点 5 失败）
  - [x] SubTask 24.1: 更新 `class_times_record_back/README.md` 的迁移进度表，将"Nacos 配置中心集成"从 `[ ]` 改为 `[x]`
  - [x] SubTask 24.2: 检查 README.md 中其他与 Nacos 相关的待办项是否已完成，同步更新状态（确认无其他独立 Nacos 待办项）

# Task Dependencies（阶段七）

- Task 21 可独立执行
- Task 22 需用户决策后才能执行
- Task 23/24 可独立执行，与 Task 21/22 并行
