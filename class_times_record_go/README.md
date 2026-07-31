# Course Record Go (Java → Go 迁移)

本目录为 Java 后端 (class_times_record_back) 迁移至 Go 语言的实现项目。

## 迁移目标
- 性能/资源优化：Java 微服务单实例 300-500MB → Go 30-80MB
- 启动速度优化：Java 8-15s → Go <1s
- 保持与 Java 后端完全互通（国密算法、JWT、数据库、Redis）

## 架构对齐

| Java 模块 | Go 模块 | 端口 | 状态 |
|-----------|---------|------|------|
| gateway | gateway/ | 9999 | ✅ 已完成（路由+JWT+黑名单） |
| auth-service | auth-service/ | 10002 | ✅ 已完成（登录/注册/订阅） |
| business-service | business-service/ | 10001 | ✅ 基础完成（机构/学生/教师查询） |
| admin-service | admin-service/ | 10003 | ✅ 基础完成（登录/用户管理） |
| common | common/ | — | ✅ 已完成（加密/响应/实体/Redis） |

## 模块说明

### common/ 公共模块
- `crypto/sm.go`：SM2 解密 + SM3 加盐哈希（对齐 Java SM2Util/SM3Util）
- `response/response.go`：统一响应封装 ResponseDTO（对齐 Java ResponseDTO）
- `context/user_context.go`：用户上下文（对齐 Java UserContext + GatewayUserFilter）
- `sign/sign.go`：请求签名工具（对齐 Java SignInterceptor）
- `db/mysql.go`：MySQL 连接池（对齐 Java HikariCP）
- `redis/redis.go`：Redis 客户端 + Token 黑名单（对齐 Java TokenBlacklistService）
- `entity/entity.go`：数据库实体（对齐 Java entity 包）
- `jwt/jwt.go`：JWT 工具（对齐 Java JwtUtils，HMAC-SHA256）

### gateway/ 网关
- 路由转发：/auth/** → auth-service, /biz/** → business-service, /admin/** → admin-service
- JWT 校验：HMAC-SHA256，与 Java JwtUtils 互通
- Redis 黑名单：查询 cr:token:blacklist:{token}
- 请求头注入：X-User-Id / X-User-Role / X-User-OpenId
- StripPrefix=1：转发前去除第一层路径前缀

### auth-service/ 认证服务
- 微信免密登录（getOpenId + 双 Token 签发）
- 账号密码登录（SM2 解密 + SM3 验签 + 机构过期校验）
- Token 续登 / 刷新 / 登出
- 注册（去重 + SM3 加盐哈希存储）
- 微信订阅记录与查询

### business-service/ 业务服务
- 机构查询（按ID/openId/编码/学生ID）
- 学生查询（按ID/家长ID/教师ID/机构ID）
- 教师查询（按ID/机构ID）

### admin-service/ 管理端服务
- 管理员登录（BCrypt 密码校验）
- 用户管理（CRUD）

## 运行

### 前置条件
- MySQL: 121.196.229.10:3306 (class_times_record)
- Redis: 121.196.229.10:6379 (密码: shiroko114514)
- Java 后端服务（可选，用于对比测试）

### 启动单个服务

```bash
cd class_times_record_go

# 启动 Gateway（端口 9999）
go run ./gateway

# 启动 auth-service（端口 10002）
go run ./auth-service

# 启动 business-service（端口 10001）
go run ./business-service

# 启动 admin-service（端口 10003）
go run ./admin-service
```

### 运行测试

```bash
# Gateway 联调测试（需 Java auth-service 10002 + business-service 10001 已启动）
go test ./gateway/... -v -timeout 60s

# PoC 验证测试
go test ./poc/crypto/... -v
go test ./poc/integration/... -v -timeout 120s
```

### 环境变量覆盖

```bash
# 数据库
DB_HOST=127.0.0.1
DB_PORT=3306

# Redis
REDIS_ADDR=127.0.0.1:6379

# 服务端口
AUTH_PORT=10002
BIZ_PORT=10001
ADMIN_PORT=10003
GATEWAY_PORT=9999
```

## 与 Java 后端互通验证

1. **国密算法**：Go SM2 解密 Java/前端加密的密文 ✅
2. **JWT Token**：Go 生成的 Token 可被 Java 解析，反之亦然 ✅
3. **Redis 黑名单**：Go 写入的黑名单 Token 可被 Java Gateway 识别 ✅
4. **数据库**：Go 直连 MySQL 读取 Java 写入的数据 ✅
5. **响应格式**：Go 返回的 JSON 与 Java ResponseDTO 格式一致 ✅

## 迁移进度

- [x] 阶段1：Gateway 迁移（路由+JWT+黑名单）
- [x] 阶段2：common 模块（加密/响应/实体/Redis）
- [x] 阶段3：auth-service 迁移（登录/注册/订阅）
- [x] 阶段4：business-service 迁移（基础查询）
- [x] 阶段5：admin-service 迁移（基础框架）
- [ ] 后续：完善 business-service 完整 CRUD（班级/课程/课时记录）
- [ ] 后续：完善 admin-service（角色/菜单管理）
- [ ] 后续：绑定学生流程（auth-service 核心复杂逻辑）
- [ ] 后续：Nacos 配置中心集成
- [ ] 后续：Docker 部署
