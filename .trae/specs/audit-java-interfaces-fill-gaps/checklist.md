# Checklist

## P0 阻塞级修复验证

- [ ] `deduct_by_class_id` 接口能正确获取课程 ID（前端传入或从 c_class.course_id 查询），不再硬编码 0
- [ ] `course_record/insert` 接口从 UserContext 读取登录教师 ID 写入 `courseOwnerUserId`，不再硬编码 0
- [ ] `/biz/record/delete` 路由已注册，能按 recordId 删除上课记录
- [ ] `/biz/course_record/delete` 路由已注册，能逻辑删除课卡记录（is_delete=1）
- [ ] admin-service 所有写操作（用户/角色/菜单/配置/业务透传/教师账号）成功后调用 `logService.RecordLog`
- [ ] `sys_operation_log` 表在写操作后有记录插入
- [ ] `SysConfigService` 注入 Redis 客户端，读路径走 Cache-Aside（key: `sys:config:{key}`，TTL 30min）
- [ ] `UpdateConfig`/`DeleteConfig` 操作后删除对应 Redis 缓存键
- [ ] `SysConfigService` 提供 `GetConfigValue`/`GetConfigValueAsLong`/`GetConfigValueAsInt`/`GetConfigValueAsBoolean` 四个通用方法
- [ ] 通用读取方法在配置不存在时返回默认值

## P1 重要级修复验证

- [ ] `UpdateInstitution` 请求 struct 包含 `Contact` 和 `Phone` 字段声明
- [ ] admin-service `PermsMiddleware` 中间件实现，校验用户角色是否具备接口所需 perms
- [ ] auth-service `MenuService` 注入 Redis，`GetMenuByRole` 应用 Cache-Aside（key: `menu:role:{roleId}`，TTL 30min）
- [ ] auth-service `doBind` 方法用事务包裹多表写入
- [ ] auth-service `Register` 方法用事务包裹多表写入
- [ ] business-service 写操作记录到 `sys_operation_log` 表

## P2 次要级修复验证

- [ ] `SysRoleVO` 仅输出 `createTimeStr`/`updateTimeStr`（无 `createTime`/`updateTime` 冗余字段）
- [ ] `SysMenuVO` 仅输出 `createTimeStr`/`updateTimeStr`（无 `createTime`/`updateTime` 冗余字段）
- [ ] `CourseRecordVO.permissionType` 字段处理结论已记录（补回或确认无需补回）
- [ ] business-service 旧版端点（/course_record/get、/course_record/add、/record/get）评估结论已记录
- [ ] `SaveRoleMenus` 操作后失效所有 `menu:user:*` 缓存
- [ ] auth-service 缺失接口（UserController、PermissionController）评估结论已记录
- [ ] business-service VO 时间字段命名统一为 `xxxTime`（若前端类型定义要求）

## 编译与格式验证

- [ ] `go build ./...` 编译通过（无错误）
- [ ] `gofmt -l .` 无格式问题（无输出）
- [ ] `go vet ./...` 无警告

## 接口契约验证

- [ ] `deduct_by_class_id` 接口返回正常扣课结果（非"课程ID不能为空"）
- [ ] `course_record/insert` 返回的 `courseOwnerUserId` 字段为登录教师 ID（非 0）
- [ ] `/biz/record/delete` 接口返回操作成功
- [ ] `/biz/course_record/delete` 接口返回操作成功，后续查询不返回已删除记录
- [ ] admin-service 写操作后 `sys_operation_log` 表有对应记录
- [ ] 系统配置读取走 Redis 缓存（第二次读取不查库）
- [ ] 菜单查询走 Redis 缓存（第二次读取不查库）
