# Go 后端集成 Nacos（配置中心 + 服务发现）Spec

## Why

Go 后端（`feat/go-migration-poc` 分支）目前**完全未接入 Nacos**：所有配置（DB/Redis/JWT/SM2/微信/服务路由）均硬编码在 `DefaultConfig()` 或环境变量中，各服务启动时也不向 Nacos 注册。而 Java 版通过 Nacos 统一管理所有配置（`common-db.yaml`、`common-redis.yaml`、`cr-{service}.yaml` 等）并自动注册到服务发现。这导致：

1. **配置分散且不一致** — 修改 Redis 密码需要改 Go 代码或环境变量，无法在 Nacos 一处修改三服务生效；Java 与 Go 部署时若使用不同 Redis 实例需分别维护两套配置。
2. **Gateway 无法 lb 路由** — Go Gateway 当前仅支持 `http://localhost:{port}` 直连，生产环境无法通过 `lb://cr-auth-service` 走服务发现；多实例部署时无法负载均衡。
3. **服务不可观测** — Java 服务在 Nacos 控制台可见，Go 服务对运维不可见，无法监控健康状态和实例数。
4. **配置无法热更新** — Java 通过 `refresh=true` 监听 Nacos 配置变更，Go 修改配置必须重启服务。

PoC（`poc/nacos/main.go`）已验证 nacos-sdk-go v2.3.5 能连通 `nacos.kurimula-airi.top:8848` 并注册/发现服务，但生产代码未集成。

## What Changes

### 阶段一：新建 Nacos 客户端封装包（common/nacos）

- **新建 `common/nacos/client.go`**：封装 Nacos naming client 和 config client 的创建逻辑，统一连接参数（server-addr/namespace/group）
- **新建 `common/nacos/config_loader.go`**：配置中心封装
  - `LoadYAML(dataID string) (string, error)` — 拉取 Nacos 配置原文
  - `LoadAndUnmarshal(dataID string, out interface{}) error` — 拉取并 yaml.Unmarshal 到结构体
  - `ListenConfig(dataID string, onChange func(content string)) error` — 注册监听器，配置变更时回调
- **新建 `common/nacos/registry.go`**：服务注册封装
  - `RegisterInstance(serviceName, ip string, port uint64, metadata map[string]string) error` — 注册临时实例（心跳保活）
  - `DeregisterInstance(serviceName, ip string, port uint64) error` — 优雅注销
  - `SelectInstances(serviceName string, healthyOnly bool) ([]model.Instance, error)` — 服务发现查询实例列表
  - `ResolveURI(uri string) (string, error)` — 解析 `lb://{service-name}` 为 `http://{ip}:{port}`（轮询/随机选取一实例）
- **环境变量配置**：
  - `NACOS_SERVER_ADDR`（默认 `nacos.kurimula-airi.top:8848`）
  - `NACOS_NAMESPACE`（默认 `course-record`）
  - `NACOS_GROUP`（默认 `DEFAULT_GROUP`）
  - `NACOS_SCHEME`（默认 `http`，生产可选 `https`）

### 阶段二：各服务接入 Nacos 配置中心

- **BREAKING**：无（环境变量优先级最高，Nacos 不可用时降级到环境变量和默认值）
- 修改 `common/db/mysql.go` 的 `DefaultConfig()`：增加 `LoadFromNacos() *Config` 方法，从 `common-db.yaml` 加载
- 修改 `common/redis/redis.go` 的 `DefaultConfig()`：增加 `LoadFromNacos() *Config` 方法，从 `common-redis.yaml` 加载
- 修改 `common/config/config.go`：增加 `GetSM2PrivateKeyFromNacos()` 方法，从 `cr-{service}.yaml` 的 `crypto.sm2.private-key` 加载
- 修改 `gateway/internal/config.go`：增加 `LoadFromNacos() *Config` 方法，从 `cr-gateway.yaml` 加载路由和 JWT 配置
- **配置优先级**（对齐 Java `@Value` + 环境变量覆盖语义）：
  1. 环境变量（最高，部署时注入敏感凭证）
  2. Nacos 配置（运行时动态配置）
  3. 默认值（最低，本地开发兜底）
- **降级策略**：Nacos 连接失败时记录警告日志，降级到环境变量 + 默认值，服务可启动

### 阶段三：各服务接入服务注册

- **BREAKING**：无
- 修改 4 个服务的 `main.go`（gateway/auth-service/business-service/admin-service）：
  - 启动时调用 `nacos.RegisterInstance(serviceName, ip, port, metadata)`
  - 优雅关闭时调用 `nacos.DeregisterInstance`（监听 SIGINT/SIGTERM）
  - metadata 包含 `language=go`、`version=1.0`、`registered={timestamp}`
- 服务名与 Java 一致：`cr-gateway`、`cr-auth-service`、`cr-business-service`、`cr-admin-service`
- **IP 自动识别**：优先用 `NACOS_REGISTER_IP` 环境变量（容器部署用），否则用 `getLocalIP()` 获取本机局域网 IP
- **端口**：各服务从配置或命令行参数读取（gateway=9999、auth=10002、business=10001、admin=10003）

### 阶段四：Gateway 支持 lb:// 路由

- **BREAKING**：无（dev 环境继续用 localhost，prod 启用 lb）
- 修改 `gateway/internal/config.go`：路由 URI 支持 `lb://{service-name}` 格式
- 修改 `gateway/internal/server.go` 的反向代理逻辑：
  - URI 以 `lb://` 开头时，调用 `nacos.ResolveURI` 解析为真实 `http://{ip}:{port}`
  - 每次请求重新解析（实现简单负载均衡，随机选取一实例）
  - 实例列表本地缓存 10 秒（避免每次请求查 Nacos），过期重新拉取
- **dev profile 降级**：环境变量 `GATEWAY_PROFILE=dev` 或 Nacos 配置 `cr-gateway-dev.yaml` 存在时，使用 localhost 直连（对齐 Java dev profile）

### 阶段五：配置热更新（可选，P2）

- **BREAKING**：无
- 对可热更新的配置注册 Nacos 监听器
- 配置变更时：
  - Redis 连接：**仅记录警告日志提示需重启**（不重建连接池，因 Redis 客户端被多个 Service 共享，运行时替换指针不安全；P2 可选功能，收益低风险高）
  - 路由表：原子替换 `Config.Routes`
  - JWT 密钥：原子替换 `JwtUtils.secretKey`
- DB 连接池不支持热更新（连接重建代价高），配置变更时仅记录日志提示需重启

> **设计决策（2026-08-04）**：原 spec 要求 Redis 连接池重建，经评估后调整为"仅记录日志提示重启"。理由：Redis 客户端（*redis.Client）被多个 Service 共享，运行时安全重建复杂度高，且 Redis 配置变更频率极低（部署时一次配置），重启成本可接受。

## Impact

- **Affected specs**：
  - `align-go-java-vo-features` — 该 spec 第 7 项要求 Redis/DB 配置从环境变量加载，本 spec 在此基础上增加 Nacos 配置源（环境变量优先级仍最高）
  - `audit-java-interfaces-fill-gaps` — 该 spec 第 6 项要求 SysConfigService Cache-Aside 缓存，与本 spec 的 Nacos 配置缓存是不同层级的配置管理
- **Affected code**：
  - `common/nacos/` — 新建包，封装 Nacos 客户端
  - `common/config/config.go` — 增加 Nacos 加载方法
  - `common/db/mysql.go` — 增加 `LoadFromNacos` 方法
  - `common/redis/redis.go` — 增加 `LoadFromNacos` 方法
  - `gateway/internal/config.go` — 增加 `LoadFromNacos` 方法
  - `gateway/internal/server.go` — 反向代理支持 lb://
  - `gateway/main.go` — 启动时注册服务、加载 Nacos 配置
  - `auth-service/main.go` — 启动时注册服务、加载 Nacos 配置
  - `business-service/main.go` — 启动时注册服务、加载 Nacos 配置
  - `admin-service/main.go` — 启动时注册服务、加载 Nacos 配置
- **Affected deployment**：
  - 部署时需提供环境变量 `NACOS_SERVER_ADDR`、`NACOS_NAMESPACE`（可选，默认 `course-record`）
  - Docker Compose 增加环境变量注入
  - 容器内 Nacos 连接走 `host.docker.internal:8848` 或部署机 IP
- **Affected docs**：
  - `class_times_record_back/CLAUDE.md` — 补充 Go 服务 Nacos 集成说明
  - `AGENTS.md` — 如存在，补充 Nacos 相关约束

## ADDED Requirements

### Requirement: Nacos 客户端封装

系统 SHALL 在 `common/nacos` 包提供 Nacos 配置中心和服务发现的统一封装，供所有 Go 服务复用。

#### Scenario: 创建 Nacos 客户端
- **WHEN** 服务启动时调用 `nacos.NewClient()`
- **AND** 环境变量 `NACOS_SERVER_ADDR` 已设置或使用默认值 `nacos.kurimula-airi.top:8848`
- **THEN** 返回可用的 Nacos 客户端实例
- **AND** 客户端连接参数（namespace/group/scheme）从环境变量加载

#### Scenario: Nacos 不可达时降级
- **WHEN** Nacos 服务器不可达
- **THEN** 客户端创建返回错误
- **AND** 调用方记录警告日志，降级到环境变量 + 默认值
- **AND** 服务可正常启动（不阻塞）

### Requirement: 从 Nacos 加载配置

系统 SHALL 在各服务启动时从 Nacos 拉取 yaml 配置并解析为 Go 配置结构体。

#### Scenario: 加载数据库配置
- **WHEN** 服务启动调用 `db.LoadFromNacos()`
- **THEN** 从 Nacos `common-db.yaml` 拉取配置
- **AND** 解析 `spring.datasource.url` 提取 host/port/database
- **AND** 解析 `spring.datasource.username` 和 `password`
- **AND** 返回完整的 `db.Config` 结构体

#### Scenario: 加载 Redis 配置
- **WHEN** 服务启动调用 `redis.LoadFromNacos()`
- **THEN** 从 Nacos `common-redis.yaml` 拉取配置
- **AND** 解析 `spring.data.redis.host/port/password/database`
- **AND** 返回完整的 `redis.Config` 结构体

#### Scenario: 加载服务专属配置
- **WHEN** auth-service 启动调用 `config.LoadAuthServiceConfig()`
- **THEN** 从 Nacos `cr-auth-service.yaml` 拉取配置
- **AND** 解析 `crypto.sm2.private-key`、`uni-app.wx.app-id`、`uni-app.wx.secret`
- **AND** 返回完整的服务配置结构体

#### Scenario: 配置优先级
- **WHEN** 环境变量 `REDIS_PASSWORD` 已设置为 `env_value`
- **AND** Nacos `common-redis.yaml` 中 `spring.data.redis.password` 为 `nacos_value`
- **THEN** 系统使用环境变量值 `env_value`（环境变量优先级最高）

### Requirement: 服务注册到 Nacos

系统 SHALL 在各服务启动时向 Nacos 注册临时实例，并周期性发送心跳保活。

#### Scenario: 注册服务实例
- **WHEN** auth-service 启动
- **AND** 监听端口 10002
- **THEN** 调用 `nacos.RegisterInstance("cr-auth-service", ip, 10002, metadata)`
- **AND** metadata 包含 `language=go`、`version=1.0`、`registered={ISO8601 timestamp}`
- **AND** 实例类型为临时实例（`Ephemeral=true`），SDK 自动发送心跳

#### Scenario: 优雅注销
- **WHEN** 服务收到 SIGINT 或 SIGTERM 信号
- **THEN** 调用 `nacos.DeregisterInstance` 注销实例
- **AND** 关闭数据库、Redis 连接
- **AND** 退出进程

#### Scenario: 注册 IP 自动识别
- **WHEN** 环境变量 `NACOS_REGISTER_IP` 已设置
- **THEN** 使用环境变量值作为注册 IP
- **WHEN** 环境变量未设置
- **THEN** 调用 `getLocalIP()` 获取本机局域网 IP
- **AND** 获取失败时回退到 `127.0.0.1`

### Requirement: Gateway 支持 lb:// 路由

系统 SHALL 在 Gateway 反向代理时支持 `lb://{service-name}` 格式的 URI，通过 Nacos 服务发现解析真实实例地址。

#### Scenario: lb 路由解析
- **WHEN** Gateway 配置路由 URI 为 `lb://cr-auth-service`
- **AND** 请求路径匹配 `/auth/**`
- **THEN** Gateway 调用 `nacos.SelectInstances("cr-auth-service", true)` 查询健康实例
- **AND** 随机选取一实例，转发到 `http://{ip}:{port}`
- **AND** 实例列表本地缓存 10 秒，缓存命中时直接使用

#### Scenario: 无可用实例
- **WHEN** Nacos 中 `cr-auth-service` 无健康实例
- **THEN** Gateway 返回 503 错误，响应 `{"code":503,"message":"服务不可用：cr-auth-service"}`

#### Scenario: dev profile 直连
- **WHEN** 环境变量 `GATEWAY_PROFILE=dev`
- **THEN** Gateway 忽略 `lb://` 路由，使用 localhost 直连配置
- **AND** 不依赖 Nacos 服务发现即可启动

### Requirement: 配置热更新

系统 SHALL 对可热更新的配置注册 Nacos 监听器，配置变更时自动刷新。

#### Scenario: Redis 配置变更监听
- **WHEN** Nacos `common-redis.yaml` 配置变更
- **THEN** 监听器触发回调
- **AND** 记录警告日志 "Redis 配置已变更，需重启服务生效"
- **AND** 不重建连接池（Redis 客户端被多个 Service 共享，运行时替换指针不安全）
- **AND** 服务继续运行（下次重启后生效新配置）

#### Scenario: DB 配置变更提示
- **WHEN** Nacos `common-db.yaml` 配置变更
- **THEN** 监听器触发回调
- **AND** 记录警告日志 "DB 配置已变更，需重启服务生效"
- **AND** 不重建连接池（避免影响在途请求）

## MODIFIED Requirements

### Requirement: 配置加载顺序

系统 SHALL 按以下优先级顺序加载配置（从高到低）：
1. 环境变量（部署时注入敏感凭证，最高优先级）
2. Nacos 配置中心（运行时动态配置）
3. 代码默认值（本地开发兜底，最低优先级）

**修改点**：原顺序为环境变量 > 默认值，新增 Nacos 配置作为中间层。

### Requirement: Gateway 路由配置

系统 SHALL 在 Gateway 启动时从 Nacos `cr-gateway.yaml` 加载路由配置，支持 `lb://` 和 `http://` 两种 URI 格式。

**修改点**：原 `DefaultConfig()` 硬编码 localhost 路由，新增 `LoadFromNacos` 方法从 Nacos 加载。

## REMOVED Requirements

### Requirement: 硬编码服务路由
**Reason**: Go Gateway 当前硬编码 `http://localhost:10002/10001/10003` 作为服务地址，生产环境无法负载均衡
**Migration**: 改为从 Nacos `cr-gateway.yaml` 加载 `lb://` 路由，通过服务发现解析实例

### Requirement: 硬编码 Redis 密码
**Reason**: `common/redis/redis.go` 的 `DefaultConfig()` 当前硬编码 `shiroko114514` 密码（已在 `align-go-java-vo-features` spec 第 7 项部分修复，本 spec 进一步从 Nacos 加载）
**Migration**: 改为从 Nacos `common-redis.yaml` 加载，环境变量 `REDIS_PASSWORD` 优先级最高
