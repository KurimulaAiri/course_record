# Checklist

## Task 1: 系统配置 VO updateTime 字段名修复

- [x] `SysConfigRow.CreateTimeStr` 的 JSON tag 已从 `createTimeStr` 改为 `createTime`
- [x] `SysConfigRow.UpdateTimeStr` 的 JSON tag 已从 `updateTimeStr` 改为 `updateTime`
- [x] `sys_config_service.go` 中引用 `CreateTimeStr`/`UpdateTimeStr` 的代码已同步更新（Go 字段名不变，仅 JSON tag 变）
- [x] `go build ./...` 编译通过

## Task 2: UpdateUser 接口 status=0 歧义修复

- [x] `UpdateUserRequest.Status` 字段类型已从 `int64` 改为 `*int64`
- [x] `UpdateUser` 方法的 status 处理逻辑已改为 `if req.Status != nil { updateUser.Status = *req.Status }`
- [x] 原有的 `if req.Status == 0 { updateUser.Status = user.Status }` 逻辑已移除
- [x] `go build ./...` 编译通过

## Task 3: 操作日志用户信息记录修复

### Service 层 recordLog 签名修改

- [x] `admin_service.go` 的 `recordLog` 方法已增加 `operatorUserID int64, operatorUsername string` 参数
- [x] `sys_role_service.go` 的 `recordLog` 方法已增加参数
- [x] `sys_menu_service.go` 的 `recordLog` 方法已增加参数
- [x] `sys_config_service.go` 的 `recordLog` 方法已增加参数
- [x] `teacher_auth_service.go` 的 `recordLog` 方法已增加参数
- [x] `admin_business_service.go` 的 `recordLog` 调用已传入实际用户信息（非硬编码 0 和空字符串）

### Handler 层用户信息提取

- [x] `admin_handler.go` 中所有写操作 handler 已从 context 提取 userID 和 username 传入 Service
- [x] `business_handler.go` 中所有写操作 handler 已从 context 提取 userID 和 username 传入 Service
- [x] `commonctx` 包已有 `GetUsername` 方法（或等效方式从 context 读取 username）

### Context 中间件补充

- [x] `commonctxMiddleware`（admin-service/main.go）已将 username 写入 context
- [x] username 来源：数据库查询（根据 userID 查 sys_user.username）或 Redis 缓存

### 编译验证

- [x] `go build ./...` 编译通过
- [x] `go vet ./...` 无警告

## Task 4: 全量功能验证

- [x] 系统配置页面"更新时间"列正常显示（非空）— 静态验证：JSON tag 为 `updateTime`（sys_config_mapper.go:34）
- [x] 用户管理页面"禁用用户"功能正常工作（status=0 能正确更新）— 静态验证：Status 为 *int64，req.Status 非 nil 时更新为 *req.Status（admin_service.go:618-620）
- [x] 用户管理页面"启用用户"功能正常工作（status=1 能正确更新）— 静态验证：同上逻辑覆盖 status=1 场景
- [x] 用户管理页面"不更新状态"场景正常工作（不传 status 时保持原状态）— 静态验证：req.Status 为 nil 时 updateUser.Status = user.Status 保留原值（admin_service.go:613）
- [x] 操作日志列表的"操作人"列非空（显示实际用户名）— 静态验证：recordLog 接收 operatorUsername 并传递至 RecordLog（6 个 service 均已实现）
- [x] 操作日志列表的"操作人ID"非零（显示实际 userID）— 静态验证：recordLog 接收 operatorUserID 并传递至 RecordLog（6 个 service 均已实现）

## P2 次要问题（暂不修复，记录备忘）

- [ ] ~~问题 4: GetUserRoles 响应结构与前端类型定义不匹配~~（前端已适配，功能正常）
- [ ] ~~问题 5: SysRoleVO/SysMenuVO 时间字段冗余~~（前端类型已声明两套字段，功能正常）
- [ ] ~~问题 6: 教师删除接口未实现~~（前端未调用，暂不实现）
- [ ] ~~问题 8: GetRoleMenus 响应结构与前端类型定义不匹配~~（前端已适配，功能正常）
