# Tasks

> **实施原则**：每个任务完成后执行 `go build ./...` 验证编译通过。所有修改对齐 Java 端逻辑，确保前后端契约一致。
>
> **并行策略**：Task 1（系统配置字段名）和 Task 2（UpdateUser status 指针化）互相独立，可并行。Task 3（操作日志用户信息）涉及多个 Service 和 Handler，需单独执行。

## P1 重要问题修复（3 项）

- [x] Task 1: 修复系统配置 VO 的 updateTime 字段名
  - [x] SubTask 1.1: 修改 `admin-service/internal/mapper/sys_config_mapper.go` 的 `SysConfigRow` 结构体，将 `CreateTimeStr` 的 JSON tag 从 `createTimeStr` 改为 `createTime`，将 `UpdateTimeStr` 的 JSON tag 从 `updateTimeStr` 改为 `updateTime`
  - [x] SubTask 1.2: 检查 `sys_config_service.go` 中是否有代码引用 `CreateTimeStr`/`UpdateTimeStr` 字段名（Go 字段名不变，仅 JSON tag 变），如有则同步更新
  - [x] SubTask 1.3: 验证编译通过 `go build ./...`

- [x] Task 2: 修复 UpdateUser 接口 status=0 歧义问题
  - [x] SubTask 2.1: 修改 `admin-service/internal/service/admin_service.go` 的 `UpdateUserRequest` 结构体，将 `Status int64` 改为 `Status *int64`（指针类型，nil 表示不更新）
  - [x] SubTask 2.2: 修改 `UpdateUser` 方法的 status 处理逻辑：`if req.Status != nil { updateUser.Status = *req.Status }`，移除原有的 `if req.Status == 0 { updateUser.Status = user.Status }` 逻辑
  - [x] SubTask 2.3: 检查 `admin_handler.go` 中 `UpdateUser` handler 是否需要调整（通常 JSON 解码自动处理 nil 指针，无需额外修改）
  - [x] SubTask 2.4: 验证编译通过 `go build ./...`

- [x] Task 3: 修复操作日志缺失用户信息问题
  - [x] SubTask 3.1: 修改 `admin-service/internal/service/admin_service.go` 的 `recordLog` 方法签名，增加 `operatorUserID int64, operatorUsername string` 参数
  - [x] SubTask 3.2: 修改 `admin_service.go` 中所有 `s.recordLog(...)` 调用，传入用户信息（暂时硬编码 0 和空字符串，后续由 Handler 传入）
  - [x] SubTask 3.3: 修改 `sys_role_service.go` 的 `recordLog` 方法签名，增加 `operatorUserID int64, operatorUsername string` 参数，同步修改所有调用方
  - [x] SubTask 3.4: 修改 `sys_menu_service.go` 的 `recordLog` 方法签名，增加参数，同步修改所有调用方
  - [x] SubTask 3.5: 修改 `sys_config_service.go` 的 `recordLog` 方法签名，增加参数，同步修改所有调用方
  - [x] SubTask 3.6: 修改 `teacher_auth_service.go` 的 `recordLog` 方法签名，增加参数，同步修改所有调用方
  - [x] SubTask 3.7: 修改 `admin_business_service.go` 中所有 `s.recordLog(0, "", ...)` 调用，改为传入 Service 结构体中存储的用户信息（需在 Service 结构体中增加 `operatorUserID`/`operatorUsername` 字段，或通过方法参数传递）
  - [x] SubTask 3.8: 修改 `admin_handler.go` 中所有写操作 handler，从 `commonctx.GetUserID(r.Context())` 和 `commonctx.GetUsername(r.Context())` 提取用户信息，传入 Service 方法
  - [x] SubTask 3.9: 修改 `business_handler.go` 中所有写操作 handler，同上提取用户信息传入 Service
  - [x] SubTask 3.10: 检查 `commonctx` 包是否有 `GetUsername` 方法，如无则新增（从 context 中读取 username，对齐 `GetUserID` 的实现方式）
  - [x] SubTask 3.11: 检查 `commonctxMiddleware`（admin-service/main.go 中内联实现）是否将 username 写入 context，如无则补充（需从 JWT claims 或数据库查询获取 username）
  - [x] SubTask 3.12: 验证编译通过 `go build ./...`

## 最终验证

- [ ] Task 4: 全量编译与功能验证
  - [x] SubTask 4.1: 执行 `go build ./...` 确保编译通过
  - [x] SubTask 4.2: 执行 `go vet ./...` 确保无警告
  - [ ] SubTask 4.3: 验证系统配置页面 updateTime 字段正常显示
  - [ ] SubTask 4.4: 验证用户禁用功能（status=0）正常工作
  - [ ] SubTask 4.5: 验证操作日志记录的 user_id 和 username 字段非空

# Task Dependencies

- Task 1（系统配置字段名）独立，可并行
- Task 2（UpdateUser status 指针化）独立，可并行
- Task 3（操作日志用户信息）独立，但涉及文件较多，建议单独执行
- Task 4（最终验证）依赖 Task 1/2/3 全部完成

## 并行执行建议

- **第一批并行**：Task 1 + Task 2（互相独立，改动小）
- **第二批**：Task 3（涉及 6 个 service + 2 个 handler，改动较大）
- **最终**：Task 4（全量验证）
