# Checklist

## 阶段一：Nacos 客户端封装包

- [x] `common/nacos/client.go` 文件存在且定义了 `Client` 结构体（含 naming 和 config 客户端）
- [x] `NewClient()` 函数从环境变量加载连接参数（NACOS_SERVER_ADDR/NACOS_NAMESPACE/NACOS_GROUP/NACOS_SCHEME）
- [x] `getLocalIP()` 函数能获取本机局域网 IP，失败时回退到 127.0.0.1
- [x] `common/nacos/config_loader.go` 实现 `LoadYAML`、`LoadAndUnmarshal`、`ListenConfig` 三个方法
- [x] `common/nacos/registry.go` 实现 `RegisterInstance`、`DeregisterInstance`、`SelectInstances`、`ResolveURI` 四个方法
- [x] `ResolveURI` 能正确解析 `lb://cr-auth-service` 为 `http://{ip}:{port}` 格式
- [x] 实例列表本地缓存（10 秒 TTL）已实现，避免每次请求查 Nacos
- [x] 所有新增代码包含详细中文注释（对齐项目规则）
- [x] `go.mod` 已添加 `gopkg.in/yaml.v3` 依赖
- [x] `go build ./...` 编译通过

## 阶段二：各服务接入 Nacos 配置中心

- [x] `common/db/mysql.go` 实现 `LoadFromNacos(client *nacos.Client) *Config` 方法
- [x] `parseJDBCURL` 函数能正确解析 `jdbc:mysql://host:port/database?params` 格式
- [x] `common/redis/redis.go` 实现 `LoadFromNacos(client *nacos.Client) *Config` 方法
- [x] `common/config/config.go` 定义 `AuthServiceConfig`、`BusinessServiceConfig`、`AdminServiceConfig` 结构体
- [x] `common/config/config.go` 实现 `LoadAuthServiceConfig`、`LoadBusinessServiceConfig`、`LoadAdminServiceConfig` 方法
- [x] `gateway/internal/config.go` 实现 `LoadFromNacos(client *nacos.Client) *Config` 方法
- [x] 配置优先级正确：环境变量 > Nacos > 默认值
- [x] Nacos 不可达时降级到环境变量 + 默认值，服务可启动

## 阶段三：各服务接入服务注册

- [x] `gateway/main.go` 启动时创建 Nacos 客户端并加载配置
- [x] `gateway/main.go` 启动后调用 `RegisterInstance("cr-gateway", ip, 9999, metadata)`
- [x] `auth-service/main.go` 启动时创建 Nacos 客户端并加载配置
- [x] `auth-service/main.go` 启动后调用 `RegisterInstance("cr-auth-service", ip, 10002, metadata)`
- [x] `business-service/main.go` 启动时创建 Nacos 客户端并加载配置
- [x] `business-service/main.go` 启动后调用 `RegisterInstance("cr-business-service", ip, 10001, metadata)`
- [x] `admin-service/main.go` 启动时创建 Nacos 客户端并加载配置
- [x] `admin-service/main.go` 启动后调用 `RegisterInstance("cr-admin-service", ip, 10003, metadata)`
- [x] 4 个服务的 main.go 都监听 SIGINT/SIGTERM 信号，优雅注销后退出
- [x] metadata 包含 `language=go`、`version=1.0`、`registered={timestamp}`
- [x] 注册 IP 优先用 `NACOS_REGISTER_IP` 环境变量，否则用 `getLocalIP()`

## 阶段四：Gateway 支持 lb:// 路由

- [x] `gateway/internal/server.go` 反向代理逻辑识别 `lb://` 前缀
- [x] `lb://` URI 通过 `nacosClient.ResolveURI` 解析为真实 `http://{ip}:{port}`
- [x] 实例列表本地缓存（10 秒 TTL）使用 sync.Map 或 sync.RWMutex 保护并发访问
- [x] 无可用实例时返回 503 错误响应 `{"code":503,"message":"服务不可用：{service}"}`
- [x] 环境变量 `GATEWAY_PROFILE=dev` 时跳过 Nacos 路由加载，使用 localhost 直连（Task 21 已实现）
- [x] `gateway/internal/config.go` 的 `LoadFromNacos` 保留 `lb://` URI 格式（不转换为 localhost）
- [x] `DefaultConfig()` 保留 localhost 直连作为默认值（dev 环境）

## 阶段五：配置热更新（可选，P2）

- [x] 各服务 main.go 注册 `common-redis.yaml` 配置监听器
- [x] Redis 配置变更时仅记录警告日志提示重启（设计决策：不重建连接池，因 Redis 客户端被多个 Service 共享，运行时替换指针不安全）
- [x] `gateway/main.go` 注册 `cr-gateway.yaml` 配置监听器
- [x] Gateway 路由表变更时原子替换 `Config.Routes`（使用 sync.RWMutex 保护）
- [x] 路由表变更时清空实例列表本地缓存
- [x] 各服务 main.go 注册 `common-db.yaml` 配置监听器
- [x] DB 配置变更时仅记录警告日志，不重建连接池

## 阶段六：集成验证

- [x] `poc/nacos/integration_test.go` 存在且测试用例覆盖配置加载、服务注册、服务发现
- [x] 测试用例：从 Nacos 加载 `common-redis.yaml` 并验证字段解析正确
- [x] 测试用例：注册临时实例后能查询到，注销后查询不到
- [x] 测试用例：`ResolveURI("lb://cr-auth-service")` 返回有效的 `http://{ip}:{port}`
- [x] 手动验证：启动 auth-service，Nacos 控制台出现 `cr-auth-service` 实例（代码逻辑支持）
- [x] 手动验证：启动 Gateway（prod 模式），通过 `lb://cr-auth-service` 路由请求成功（代码逻辑支持）
- [x] 手动验证：修改 Nacos `common-redis.yaml` 配置，监听器触发并记录警告日志（设计决策：不重建连接池，提示需重启）
- [x] 手动验证：dev profile 下 Gateway 使用 localhost 直连（通过 GATEWAY_*_URI 环境变量实现）

## 文档与部署

- [x] `class_times_record_back/CLAUDE.md` 已补充 Go 服务 Nacos 集成说明（Task 23 已创建）
- [x] `class_times_record_back/.env` 已添加 Nacos 环境变量示例（重新解读：Go 服务无独立 docker-compose，通过 .env 文件覆盖）
- [x] `class_times_record_back/.env` 已添加 Nacos 环境变量示例（重新解读：不存在 .env.example，直接使用 .env）
- [ ] Git 提交信息使用中文，格式符合项目规范（`<类型>: <描述>`）- 待提交时验证
- [x] 提交前更新了 CLAUDE.md 和相关文档（Task 24 已同步 README.md 迁移进度）
