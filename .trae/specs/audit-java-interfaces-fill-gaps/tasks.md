# Tasks

## P0 阻塞级修复（7 项）

- [x] Task 1: 修复 `deduct_by_class_id` 接口硬编码 courseID=0 bug
  - [x] SubTask 1.1: 修改 handler.go 的 `DeductByClassID` 请求 struct，补充 `CourseID int64` 字段
  - [x] SubTask 1.2: 修改 handler 调用 service 时传入 `req.CourseID`（而非硬编码 0）
  - [x] SubTask 1.3: 修改 `course_record_service.go` 的 `DeductByClassID`，当 courseID=0 时从 `c_class.course_id` 查询（注入 ClassMapper）
  - [x] SubTask 1.4: 验证编译通过

- [x] Task 2: 修复 `course_record/insert` ownerUserID 硬编码 0
  - [x] SubTask 2.1: 修改 handler.go 的 `InsertCourseRecord`，从 `r.Context().Value("userId")` 读取登录教师 ID
  - [x] SubTask 2.2: 将读取到的 userId 作为 ownerUserID 传入 service
  - [x] SubTask 2.3: 验证编译通过

- [x] Task 3: 注册 `/biz/record/delete` 路由
  - [x] SubTask 3.1: 在 handler.go 新增 `DeleteRecord` handler 方法
  - [x] SubTask 3.2: 在 record_service.go 新增 `DeleteRecord(recordID int64)` service 方法
  - [x] SubTask 3.3: 在 RegisterRoutes 注册路由 `mux.HandleFunc("POST /record/delete", h.DeleteRecord)`
  - [x] SubTask 3.4: 验证编译通过

- [x] Task 4: 实现 `/biz/course_record/delete` 逻辑删除
  - [x] SubTask 4.1: 在 course_record_mapper.go 新增 `DeleteByID(id int64) error` 方法（UPDATE is_delete=1）
  - [x] SubTask 4.2: 在 course_record_service.go 新增 `DeleteCourseRecord(id int64)` service 方法
  - [x] SubTask 4.3: 在 handler.go 新增 `DeleteCourseRecord` handler 方法
  - [x] SubTask 4.4: 在 RegisterRoutes 注册路由
  - [x] SubTask 4.5: 验证编译通过

- [x] Task 5: admin-service 操作日志自动记录
  - [x] SubTask 5.1: admin_service.go 写操作调用 `logService.RecordLog`（已存在）
  - [x] SubTask 5.2: sys_role_service.go 写操作调用 `logService.RecordLog`（已存在）
  - [x] SubTask 5.3: sys_menu_service.go 写操作调用 `recordLog`
  - [x] SubTask 5.4: sys_config_service.go 写操作调用 `recordLog`
  - [x] SubTask 5.5: admin_business_service.go 19 个写操作调用 `recordLog`
  - [x] SubTask 5.6: teacher_auth_service.go 写操作调用 `recordLog`
  - [x] SubTask 5.7: RecordLog 调用传入操作类型、参数 JSON
  - [x] SubTask 5.8: 验证编译通过

- [x] Task 6: admin-service SysConfigService Cache-Aside 缓存
  - [x] SubTask 6.1: 修改 `SysConfigService` 结构体，注入 `redisClient`
  - [x] SubTask 6.2: 修改 `NewSysConfigService` 构造函数
  - [x] SubTask 6.3: 修改 main.go 传入 redisClient
  - [x] SubTask 6.4: 新增 `getConfigFromCache(key)` 方法（先查 Redis，未命中查库回填，TTL 30min）
  - [x] SubTask 6.5: `UpdateConfig`/`DeleteConfig` 后删除 Redis 缓存键
  - [x] SubTask 6.6: 验证编译通过

- [x] Task 7: admin-service SysConfigService 通用读取方法
  - [x] SubTask 7.1: `GetConfigValue(key string) string`
  - [x] SubTask 7.2: `GetConfigValueAsLong(key string, defaultVal int64) int64`
  - [x] SubTask 7.3: `GetConfigValueAsInt(key string, defaultVal int) int`
  - [x] SubTask 7.4: `GetConfigValueAsBoolean(key string, defaultVal bool) bool`
  - [x] SubTask 7.5: 验证编译通过

## P1 重要级修复（5 项）

- [x] Task 8: business-service `UpdateInstitution` 请求体补字段
  - [x] SubTask 8.1: 请求 struct 补充 `Contact` 和 `Phone` 字段
  - [x] SubTask 8.2: 验证编译通过

- [x] Task 9: admin-service 按钮级权限校验（perms）
  - [x] SubTask 9.1-9.5: 评估后决定暂不实现（骨架中间件无实际配置意义，Java 端也仅有文档描述无具体实现证据）

- [x] Task 10: auth-service 菜单缓存
  - [x] SubTask 10.1: MenuService 注入 redis（已存在）
  - [x] SubTask 10.2: NewMenuService 接收 redisClient（已存在）
  - [x] SubTask 10.3: main.go 传入 redisClient（已存在）
  - [x] SubTask 10.4: GetMenuByRole 应用 Cache-Aside（已存在，修复了 jsonErr 作用域 bug）
  - [x] SubTask 10.5: auth-service 无菜单写操作（CRUD 在 admin-service），无需失效缓存
  - [x] SubTask 10.6: 验证编译通过

- [x] Task 11: auth-service 绑定/注册事务管理
  - [x] SubTask 11.1: AuthService 已注入 `*sql.DB`
  - [x] SubTask 11.2: doBind 用事务包裹 user/user_platform/parent/parent_student 写入
  - [x] SubTask 11.3: Register 用事务包裹 user/user_auth/user_platform/parent 写入
  - [x] SubTask 11.4: 任一步失败回滚，全部成功提交
  - [x] SubTask 11.5: 验证编译通过

- [x] Task 12: business-service 操作日志记录
  - [x] 评估后决定暂不实现（business-service 无 sys_operation_log 表的直接访问权限，且 Java 端 business-service 也无操作日志记录）

## P2 次要级修复（6 项）

- [x] Task 13: admin-service SysRoleVO/SysMenuVO 时间字段去重
  - [x] SubTask 13.1-13.4: 结论：前端依赖 createTime/updateTime 字段（admin.d.ts 和 role/index.vue 均使用），保留现状不移除

- [x] Task 14: 确认 `CourseRecordVO.permissionType` 字段
  - [x] SubTask 14.1: 确认 Java CourseRecordVO 含 permissionType 字段（.qoder 文档证据）
  - [x] SubTask 14.2: Go 已补回 PermissionType 字段（默认值 0）

- [x] Task 15: 评估 business-service 旧版端点
  - [x] SubTask 15.1: 前端有调用 `/course_record/get`（4处）、`/course_record/add`（1处）、`/record/get`（1处），但无 `/biz/` 前缀
  - [x] SubTask 15.2: 结论：这些端点无 /biz/ 前缀，与 Gateway 路由规则不符，需评估 Gateway 路由配置或前端 API 调用修正，暂不在本 spec 补齐

- [x] Task 16: admin-service SysMenuService 缓存失效范围
  - [x] SubTask 16.1: SaveRoleMenus 已调用 invalidateMenuCache（SCAN menu:user:* 并删除）
  - [x] SubTask 16.2: 验证编译通过

- [x] Task 17: 评估 auth-service 缺失接口
  - [x] SubTask 17.1: 前端无调用 /user/info、/user/password、/user/profile、/permission/by-user、/permission/by-role
  - [x] SubTask 17.2: 结论：5 个接口均无前端调用，不迁移

- [x] Task 18: business-service 时间字段命名统一
  - [x] SubTask 18.1: 前端时间字段混用（响应主要用 xxxTimeStr，请求用 xxxTime）
  - [x] SubTask 18.2: 结论：保持现状，不应统一为 xxxTime（会导致前端访问失败）

## 最终验证

- [x] Task 19: 全量编译与格式化验证
  - [x] SubTask 19.1: `go build ./...` 编译通过
  - [x] SubTask 19.2: `gofmt -w .` 格式化完成
  - [x] SubTask 19.3: `go vet ./...` 无警告

# Task Dependencies

- Task 6（配置缓存）→ Task 7（通用读取方法）：通用读取方法依赖缓存基础设施
- Task 5（操作日志）独立，可并行
- Task 1/2/3/4（business-service 修复）互相独立，可并行
- Task 9（perms 中间件）独立
- Task 10/11（auth-service）互相独立，可并行
- Task 13-18（P2）互相独立，可并行
- Task 19 依赖所有前置 Task 完成
