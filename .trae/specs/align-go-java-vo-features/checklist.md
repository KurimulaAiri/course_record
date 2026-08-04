# Checklist

## P0 扣课微信通知

- [x] common 包提供独立的 `WeChatApiService`（`GetAccessToken`/`SendSubscribeMessage`/`GenerateQrCode`）
- [x] `WeChatApiService` 从环境变量 `WX_APP_ID`/`WX_APP_SECRET` 加载配置，不再硬编码假值
- [x] auth-service 重构为使用 common 包的 `WeChatApiService`，移除私有方法
- [x] auth-service 微信登录（`getOpenId`）功能不受影响
- [x] auth-service 二维码生成（`generateQrCode`）功能不受影响
- [x] `WxStudentSubscribeMapper.SelectByStudentID` 方法已实现
- [x] `WxSubscribeRecordMapper.SelectByOpenIDsAndTemplate` 方法已实现
- [x] `WxSubscribeRecordMapper.DecrementCount` 方法已实现（永久订阅不扣减）
- [x] `CourseRecordService` 注入了 `StudentMapper`/`CourseMapper`/`WxStudentSubscribeMapper`/`WxSubscribeRecordMapper`/`WeChatApiService`
- [x] `deductOne` 方法扣课成功后调用通知逻辑
- [x] 通知模板数据包含 thing1/thing8/number4/number10/number11/time5 六个字段
- [x] 通知跳转页面为 `pages/main/parent/deduct-detail/index?recordId={recordId}`
- [x] 非永久订阅推送成功后 `subscribe_count - 1`
- [x] 永久订阅推送不扣减次数
- [x] 通知失败不阻塞主流程（仅记录日志）

## P0 签名校验中间件

- [x] `common/sign/` 提供 `SignMiddleware` HTTP 中间件
- [x] 中间件校验 `x-sign`/`x-timestamp`/`x-nonce` 三个请求头
- [x] 时间戳校验：5 分钟有效期
- [x] Nonce 防重放：Redis SETNX，TTL 60s
- [x] 公开路径跳过校验（对齐 Java AuthWebConfig/WebConfig 排除路径）
- [x] auth-service `main.go` 注册了 `SignMiddleware`
- [x] business-service `main.go` 注册了 `SignMiddleware`
- [x] 签名校验失败返回 401 状态码

## P1 事务管理

- [x] `deductOne` 方法用 `sql.Tx` 包裹 UPDATE + INSERT 两步操作
- [x] 两步都成功则 `tx.Commit()`
- [x] 任一步失败则 `tx.Rollback()`
- [x] 通知发送在事务提交后执行（不在事务内）
- [x] 批量扣课每个学生独立事务

## P1 配置从环境变量加载

- [x] SM2 私钥从 `SM2_PRIVATE_KEY` 环境变量加载
- [x] 三个服务（auth/business/admin）使用相同 SM2 私钥
- [x] Redis 密码从 `REDIS_PASSWORD` 环境变量加载，不再硬编码 `shiroko114514`
- [x] MySQL 密码从 `DB_PASSWORD` 环境变量加载，不再硬编码 `8BCnbZjTT8ZxmBj6`
- [x] 非敏感配置（host/port/db name）保留默认值便于本地开发

## P1 缓存机制

- [x] `common/redis/redis.go` 提供 `GetOrSet` Cache-Aside 辅助函数
- [x] `common/redis/redis.go` 提供 `Delete` 缓存删除函数
- [x] auth-service `getFullUserInfo` 应用缓存（key: `user:info:{userId}`，TTL 30min）
- [x] admin-service `getMenuByRole` 应用缓存（key: `menu:role:{roleId}`，TTL 30min）
- [x] 写操作时删除对应缓存键

## P1 AdminJwtInterceptor

- [x] admin-service 提供 `AdminJwtMiddleware` 中间件
- [x] 校验 `Authorization: Bearer {token}` 头的 JWT 签名和过期时间
- [x] 公开路径（`/admin/user/login`）跳过校验
- [x] admin-service `main.go` 注册了 `AdminJwtMiddleware`

## P2 次要修复

- [x] `common/crypto/sm.go` 的 `randomString` 函数 bug 已修复（使用 `crypto/rand`）
- [x] `GenerateSalt` 生成 32 位十六进制盐值
- [x] c_student 软删除处理已确认（Java 有/无，Go 对齐）
- [x] Gateway 配置支持从环境变量加载服务 URI

## 最终验证

- [x] `go build ./...` 编译通过
- [x] `gofmt -l .` 无格式问题
- [x] `go vet ./...` 无警告
- [ ] 部署文档说明需提供的环境变量列表（`WX_APP_ID`/`WX_APP_SECRET`/`WX_ENV_VERSION`/`SM2_PRIVATE_KEY`/`REDIS_PASSWORD`/`DB_PASSWORD`）
