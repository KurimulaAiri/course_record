# Tasks

## P0 阻塞级修复（4 项）

- [x] Task 1: 提取 WeChatApiService 到 common 包
  - [x] SubTask 1.1: 在 `common/wechat/` 创建 `WeChatApiService` 结构体，包含 `GetAccessToken`、`SendSubscribeMessage`、`GenerateQrCode` 方法
  - [x] SubTask 1.2: 从环境变量 `WX_APP_ID`/`WX_APP_SECRET`/`WX_ENV_VERSION` 加载配置，移除 auth-service 中硬编码假值
  - [x] SubTask 1.3: 重构 auth-service 的 `AuthService` 使用 common 包的 `WeChatApiService`（移除私有方法 `getAccessToken`/`sendSubscribeMessage`/`generateQrCode`）
  - [x] SubTask 1.4: 验证 auth-service 的微信登录、二维码生成、测试订阅消息功能不受影响

- [x] Task 2: 补齐微信订阅 Mapper 查询方法
  - [x] SubTask 2.1: 在 `business-service/internal/mapper/mapper.go` 的 `WxStudentSubscribeMapper` 添加 `SelectByStudentID(studentID int64) ([]*entity.WxStudentSubscribe, error)` 方法
  - [x] SubTask 2.2: 在 `business-service/internal/mapper/mapper.go` 的 `WxSubscribeRecordMapper` 添加 `SelectByOpenIDsAndTemplate(openIDs []string, templateID string) ([]*entity.WxSubscribeRecord, error)` 方法
  - [x] SubTask 2.3: 在 `business-service/internal/mapper/mapper.go` 的 `WxSubscribeRecordMapper` 添加 `DecrementCount(id int64) error` 方法（永久订阅不扣减）

- [x] Task 3: 实现扣课微信通知
  - [x] SubTask 3.1: 修改 `CourseRecordService` 结构体，注入 `StudentMapper`、`CourseMapper`、`WxStudentSubscribeMapper`、`WxSubscribeRecordMapper`、`WeChatApiService`
  - [x] SubTask 3.2: 修改 `NewCourseRecordService` 构造函数，接收新依赖
  - [x] SubTask 3.3: 修改 `main.go` 的服务初始化代码，传入新依赖
  - [x] SubTask 3.4: 在 `course_record_service.go` 实现 `sendDeductNotification(studentID, courseID, courseRecordID, recordID, deductCount int64)` 方法，对齐 Java `DeductNotifyAspect.sendSingleNotification`：
    - 查学生、课程、课卡记录
    - 构建模板数据（thing1/thing8/number4/number10/number11/time5）
    - 查订阅 openId → 查授权记录 → 发送 → 成功扣减次数
  - [x] SubTask 3.5: 在 `deductOne` 方法扣课成功后（插流水之后）调用 `sendDeductNotification`，通知失败不阻塞主流程
  - [x] SubTask 3.6: 验证编译通过，gofmt 格式化

- [x] Task 4: 实现 SignInterceptor 签名校验中间件
  - [x] SubTask 4.1: 在 `common/sign/` 创建 HTTP 中间件 `SignMiddleware(publicPaths []string) func(http.Handler) http.Handler`
  - [x] SubTask 4.2: 中间件逻辑：公开路径跳过；非公开路径校验 `x-sign`/`x-timestamp`/`x-nonce` 头
  - [x] SubTask 4.3: 时间戳校验（5分钟有效期），nonce 防重放（调用 `redis.SetNonceIfAbsent`，TTL 60s）
  - [x] SubTask 4.4: 签名校验调用 `sign.VerifyRequest`，失败返回 401
  - [x] SubTask 4.5: 在 auth-service `main.go` 注册中间件（在 `commonctxMiddleware` 之前，公开路径列表对齐 Java AuthWebConfig）
  - [x] SubTask 4.6: 在 business-service `main.go` 注册中间件（公开路径列表对齐 Java WebConfig）
  - [x] SubTask 4.7: 验证编译通过

## P1 重要级修复（5 项）

- [x] Task 5: business-service 扣课事务管理
  - [x] SubTask 5.1: 在 `deductOne` 方法中用 `db.BeginTx` 开启事务
  - [x] SubTask 5.2: 将 `UpdateRestTime` 和 `recordMapper.Insert` 改为使用事务连接 `tx.Exec`/`tx.Query`
  - [x] SubTask 5.3: 两步都成功则 `tx.Commit()`，任一步失败则 `tx.Rollback()`
  - [x] SubTask 5.4: 通知发送在事务提交后执行（不在事务内）
  - [x] SubTask 5.5: 验证编译通过

- [x] Task 6: SM2 私钥统一并从环境变量加载
  - [x] SubTask 6.1: 在 `common/config/` 创建配置加载函数，从 `SM2_PRIVATE_KEY` 环境变量读取
  - [x] SubTask 6.2: 移除 auth-service `auth_service.go` 第 52 行硬编码的 SM2 私钥
  - [x] SubTask 6.3: 移除 admin-service `main.go` 第 36 行硬编码的 SM2 私钥
  - [x] SubTask 6.4: 移除 business-service `main.go` 第 26 行硬编码的 SM2 私钥
  - [x] SubTask 6.5: 三个服务统一从 `SM2_PRIVATE_KEY` 环境变量加载，使用相同私钥值
  - [x] SubTask 6.6: 验证编译通过

- [x] Task 7: Redis/DB 配置移除硬编码凭证
  - [x] SubTask 7.1: 修改 `common/redis/redis.go` 的 `DefaultConfig()`，密码从 `REDIS_PASSWORD` 环境变量读取，不再硬编码 `shiroko114514`
  - [x] SubTask 7.2: 修改 `common/db/mysql.go` 的 `DefaultConfig()`，密码从 `DB_PASSWORD` 环境变量读取，不再硬编码 `8BCnbZjTT8ZxmBj6`
  - [x] SubTask 7.3: 保留 host/port/db 等非敏感配置的默认值（便于本地开发）
  - [x] SubTask 7.4: 验证编译通过

- [x] Task 8: 缓存机制（Cache-Aside）
  - [x] SubTask 8.1: 在 `common/redis/redis.go` 添加 `GetOrSet(key string, ttl time.Duration, loader func() (string, error)) (string, error)` 函数
  - [x] SubTask 8.2: 在 `common/redis/redis.go` 添加 `Delete(key string) error` 函数
  - [x] SubTask 8.3: 在 auth-service 的 `getFullUserInfo` 应用缓存（key: `user:info:{userId}`，TTL 30min）
  - [x] SubTask 8.4: 在 admin-service 的 `getMenuByRole` 应用缓存（key: `menu:role:{roleId}`，TTL 30min）
  - [x] SubTask 8.5: 用户/菜单写操作时删除对应缓存键
  - [x] SubTask 8.6: 验证编译通过

- [x] Task 9: admin-service AdminJwtInterceptor
  - [x] SubTask 9.1: 在 `admin-service/` 创建 JWT 校验中间件 `AdminJwtMiddleware(publicPaths []string) func(http.Handler) http.Handler`
  - [x] SubTask 9.2: 校验 `Authorization: Bearer {token}` 头，验证 JWT 签名和过期时间
  - [x] SubTask 9.3: 公开路径（`/admin/user/login` 等）跳过校验
  - [x] SubTask 9.4: 在 admin-service `main.go` 注册中间件（在 `commonctxMiddleware` 之前）
  - [x] SubTask 9.5: 验证编译通过

## P2 次要级修复（3 项）

- [x] Task 10: 修复 GenerateSalt 实现
  - [x] SubTask 10.1: 修复 `common/crypto/sm.go` 的 `randomString` 函数 bug（使用 `crypto/rand` 生成随机字节）
  - [x] SubTask 10.2: `GenerateSalt` 改用 `crypto/rand` 生成 32 位十六进制盐值
  - [x] SubTask 10.3: 验证编译通过

- [x] Task 11: 确认 c_student 软删除处理
  - [x] SubTask 11.1: 查询 Java `StudentMapper.xml` 确认是否有 `is_delete` 条件
  - [x] SubTask 11.2: 若 Java 有软删除，Go 补齐 `WHERE is_delete = 0` 条件
  - [x] SubTask 11.3: 若 Java 无软删除，确认 Go 当前实现正确，记录结论

- [x] Task 12: Gateway 配置从环境变量加载（可选 lb）
  - [x] SubTask 12.1: 修改 `gateway/internal/config.go`，服务 URI 从环境变量 `GATEWAY_AUTH_URI`/`GATEWAY_BUSINESS_URI`/`GATEWAY_ADMIN_URI` 加载
  - [x] SubTask 12.2: 默认值保留 `localhost:port`，支持 `lb://{service-name}` 格式
  - [x] SubTask 12.3: 验证编译通过

## 最终验证

- [x] Task 13: 全量编译验证
  - [x] SubTask 13.1: `go build ./...` 编译通过
  - [x] SubTask 13.2: `gofmt -l .` 无格式问题
  - [x] SubTask 13.3: `go vet ./...` 无警告

# Task Dependencies

- Task 2（补齐 Mapper）→ Task 3（扣课通知）：扣课通知依赖新 Mapper 方法
- Task 1（提取 WeChatApiService）→ Task 3（扣课通知）：扣课通知依赖 common 包的 WeChatApiService
- Task 1（提取 WeChatApiService）→ Task 4（SignInterceptor）：无直接依赖，可并行
- Task 3（扣课通知）→ Task 5（事务管理）：事务包裹 deductOne 后再集成通知，避免冲突
- Task 6/7/8/9 互相独立，可并行
- Task 10/11/12 互相独立，可并行
- Task 13 依赖所有前置 Task 完成
