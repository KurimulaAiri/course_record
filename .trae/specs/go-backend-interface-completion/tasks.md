# Tasks

> **实施原则**：每个阶段完成后执行 `go build ./...` 验证编译通过。所有接口对齐 Java 后端逻辑，DTO/VO 字段命名一致，SQL 查询对齐 Java Mapper XML。
>
> **并行策略**：阶段一（auth-service）和阶段三（admin-service 系统管理）无依赖，可并行。阶段二依赖 common/entity 中新增实体，需先完成实体定义。

## 阶段一：auth-service 绑定与订阅流程（8 接口，P0）

- [x] Task 1: 实现绑定二维码生成接口
  - [ ] SubTask 1.1: 在 `auth_service.go` 新增 `GenerateBindQrcode(studentID int64)` 方法，生成 6 位随机码存入 Redis（key: `bind:code:{code}`, value: `studentID`, TTL: 10 分钟）
  - [ ] SubTask 1.2: 新增 `GenerateSubscribeQrcode(studentID int64)` 方法，生成订阅专用码
  - [ ] SubTask 1.3: 在 `auth_handler.go` 新增 `GenerateBindQrcode`、`GenerateSubscribeQrcode` handler
  - [ ] SubTask 1.4: 注册路由 `POST /auth/generate_bind_qrcode`、`POST /auth/generate_subscribe_qrcode`

- [x] Task 2: 实现绑定信息查询接口
  - [ ] SubTask 2.1: 新增 `GetBindInfo(token string)` 方法，从 Redis 读取绑定 token 对应的 studentID，查询学生信息返回
  - [ ] SubTask 2.2: 新增 `GetBindInfoByCode(code string)` 方法，从 Redis 读取绑定码对应的 studentID，查询学生信息返回（不执行绑定）
  - [ ] SubTask 2.3: 在 handler 新增 `GetBindInfo`（GET）、`GetBindInfoByCode`（GET）
  - [ ] SubTask 2.4: 注册路由 `GET /auth/get_bind_info`、`GET /auth/get_bind_info_by_code`

- [x] Task 3: 实现绑定状态检查与确认接口
  - [ ] SubTask 3.1: 新增 `CheckBindStatus(token, code string)` 方法，检查绑定码是否有效、是否已被使用
  - [ ] SubTask 3.2: 新增 `ConfirmBind(token, openId string)` 方法，确认绑定（创建 parent_student 关联，更新 wx_student_subscribe）
  - [ ] SubTask 3.3: 新增 `BindByCode(code, openId string)` 方法，按绑定码直接绑定（合并查询+绑定流程）
  - [ ] SubTask 3.4: 在 handler 新增 `CheckBindStatus`（GET）、`ConfirmBind`（POST）、`BindByCode`（POST）
  - [ ] SubTask 3.5: 注册路由 `GET /auth/check_bind_status`、`POST /auth/confirm_bind`、`POST /auth/bind_by_code`

- [x] Task 4: 实现订阅消息测试发送接口
  - [ ] SubTask 4.1: 新增 `TestSendSubscribe(openId string)` 方法，调用微信订阅消息推送 API
  - [ ] SubTask 4.2: 在 handler 新增 `TestSendSubscribe`（GET）
  - [ ] SubTask 4.3: 注册路由 `GET /auth/test_send_subscribe`

- [x] Task 5: 阶段一编译验证
  - [ ] SubTask 5.1: 执行 `go build ./...` 确保编译通过
  - [ ] SubTask 5.2: 确认 Gateway publicPaths 中 auth 相关路径全部实现

## 阶段二：business-service 写操作补全（38 接口，P1）

- [x] Task 6: 新增业务实体定义
  - [ ] SubTask 6.1: 在 `common/entity/entity.go` 新增 `Class`、`ClassSchedule`、`Course`、`CourseRecord`、`Record` 实体（含 sql.NullXxx 字段）
  - [ ] SubTask 6.2: 新增关联实体：`ClassTeacher`、`ClassStudent`（多对多关联表）
  - [ ] SubTask 6.3: 编译验证 `go build ./common/...`

- [ ] Task 7: 学生模块补全（6 接口）
  - [ ] SubTask 7.1: 在 StudentMapper 新增 `SelectByClassID`、`SelectByCourseID`、`Insert`、`Update`、`Unbind`、`CancelSubscribe` 方法
  - [ ] SubTask 7.2: 在 StudentService 新增对应 6 个方法
  - [ ] SubTask 7.3: 在 handler 注册 6 个路由：`get_by_class_id`、`get_by_course_id`、`insert`、`update`、`unbind`、`cancel_subscribe`

- [ ] Task 8: 教师模块补全（3 接口）
  - [ ] SubTask 8.1: 新增 TeacherMapper（在 business-service 内）的 `UpdateByID`、`Insert`、`Delete` 方法
  - [ ] SubTask 8.2: 在 TeacherService 新增对应 3 个方法
  - [ ] SubTask 8.3: 注册路由 `update_by_id`、`insert`、`delete`

- [ ] Task 9: 班级模块实现（8 接口，全新）
  - [ ] SubTask 9.1: 新建 `class_mapper.go`，实现 ClassMapper（查询 c_class、c_class_student、c_class_teacher）
  - [ ] SubTask 9.2: 新建 `class_service.go`，实现 ClassService 8 个方法
  - [ ] SubTask 9.3: 在 handler 注册 8 个路由

- [ ] Task 10: 课表模块实现（5 接口，全新）
  - [ ] SubTask 10.1: 新建 `class_schedule_mapper.go` 和 `class_schedule_service.go`
  - [ ] SubTask 10.2: 注册 5 个路由

- [ ] Task 11: 课程模块实现（4 接口，全新）
  - [ ] SubTask 11.1: 新建 `course_mapper.go` 和 `course_service.go`
  - [ ] SubTask 11.2: 注册 4 个路由

- [ ] Task 12: 课卡记录模块实现（9 接口，全新，含扣费双重校验）
  - [ ] SubTask 12.1: 新建 `course_record_mapper.go`，实现查询、插入、更新、扣费方法
  - [ ] SubTask 12.2: 新建 `course_record_service.go`，实现 9 个方法，扣费方法执行双重校验（Java 层校验 expire_time + SQL 层 WHERE 兜底）
  - [ ] SubTask 12.3: 注册 9 个路由（含 `GET /course_record/deduct-detail`）
  - [ ] SubTask 12.4: 确认 `deduct-detail` 在 Gateway publicPaths 中（已声明）

- [ ] Task 13: 机构与上课记录模块补全（3 接口）
  - [ ] SubTask 13.1: InstitutionMapper 新增 `Update` 方法，注册路由 `POST /institution/update`
  - [ ] SubTask 13.2: 新建 `record_mapper.go` 和 `record_service.go`，实现 `new_get`、`delete`
  - [ ] SubTask 13.3: 注册路由 `POST /record/new_get`、`POST /record/delete`

- [ ] Task 14: 阶段二编译验证
  - [ ] SubTask 14.1: 执行 `go build ./...` 确保编译通过
  - [ ] SubTask 14.2: 验证所有 business-service 路由注册无遗漏

## 阶段三：admin-service 系统管理补全（19 接口，P2，可与阶段一并行）

- [x] Task 15: 新增系统实体定义
  - [ ] SubTask 15.1: 在 `common/entity/entity.go` 新增 `SysRole`、`SysMenu`、`SysOperationLog`、`SysConfig`、`SysRoleMenu`、`SysUserRole` 实体
  - [ ] SubTask 15.2: 编译验证

- [x] Task 16: 用户管理补全（6 接口）
  - [x] SubTask 16.1: AdminUserMapper 新增 `SelectByID`、`Insert`、`Update`、`Delete`、`ResetPassword` 方法
  - [x] SubTask 16.2: AdminService 新增对应方法，`Insert`/`Update` 使用 BCrypt 哈希密码
  - [x] SubTask 16.3: 注册路由 `get_by_id`、`insert`、`update`、`delete`、`reset_password`、`get_roles`

- [x] Task 17: 角色管理实现（7 接口，全新）
  - [x] SubTask 17.1: 新建 `sys_role_mapper.go`，实现角色 CRUD 和角色菜单关联查询/保存
  - [x] SubTask 17.2: 新建 `sys_role_service.go`，实现 7 个方法
  - [x] SubTask 17.3: 注册路由 `list`、`get_by_id`、`insert`、`update`、`delete`、`get_menus`、`save_menus`

- [x] Task 18: 菜单管理实现（5 接口，全新）
  - [x] SubTask 18.1: 新建 `sys_menu_mapper.go`，实现菜单 CRUD 和树形结构查询
  - [x] SubTask 18.2: 新建 `sys_menu_service.go`，实现 5 个方法（含树形构建逻辑）
  - [x] SubTask 18.3: 注册路由 `list`、`tree`、`user_tree`、`insert`、`update`、`delete`

- [x] Task 19: 操作日志实现（3 接口，全新）
  - [x] SubTask 19.1: 新建 `sys_operation_log_mapper.go` 和 `sys_operation_log_service.go`
  - [x] SubTask 19.2: 实现操作日志记录中间件/工具函数（对齐 Java @OperationLog 注解）
  - [x] SubTask 19.3: 注册路由 `list`、`delete`、`clear`

- [x] Task 20: 阶段三编译验证
  - [x] SubTask 20.1: 执行 `go build ./...` 确保编译通过

## 阶段四：admin-service 业务管理透传（26 接口，P3）

- [x] Task 21: 机构与学生业务管理（6 接口）
  - [x] SubTask 21.1: 实现 `/business/institution/list`、`/business/institution/insert`、`/business/institution/update`
  - [x] SubTask 21.2: 实现 `/business/student/list`、`/business/student/insert`、`/business/student/update`

- [x] Task 22: 教师业务管理与账号管理（7 接口）
  - [x] SubTask 22.1: 实现 `/business/teacher/list`、`/business/teacher/insert`、`/business/teacher/update`
  - [x] SubTask 22.2: 实现 `/teacher_auth/get`、`/teacher_auth/update_account`、`/teacher_auth/update_password`、`/teacher_auth/toggle_institution_admin`

- [x] Task 23: 课程与班级业务管理（9 接口）
  - [x] SubTask 23.1: 实现 `/business/course/list`、`/business/course/insert`、`/business/course/update`
  - [x] SubTask 23.2: 实现 `/business/class/list`、`/business/class/insert`、`/business/class/update`、`/business/class/get_by_id`、`/business/class/add_student`、`/business/class/remove_student`

- [x] Task 24: 课表、课时记录、上课记录、小程序菜单管理（10 接口）
  - [x] SubTask 24.1: 实现 `/business/class_schedule/list`、`/business/class_schedule/update`
  - [x] SubTask 24.2: 实现 `/business/course_record/list`、`/business/course_record/insert`、`/business/course_record/update`
  - [x] SubTask 24.3: 实现 `/business/record/list`、`/business/record/insert`
  - [x] SubTask 24.4: 实现 `/business/mini_menu/list`、`/business/mini_menu/insert`、`/business/mini_menu/update`、`/business/mini_menu/delete`

- [x] Task 25: 阶段四编译验证
  - [x] SubTask 25.1: 执行 `go build ./...` 确保编译通过

## 阶段五：仪表盘与系统配置（7 接口，P3）

- [x] Task 26: 仪表盘实现（3 接口）
  - [x] SubTask 26.1: 新建 `dashboard_service.go`，实现汇总数据、趋势数据、机构统计查询
  - [x] SubTask 26.2: 注册路由 `/dashboard/data`、`/dashboard/trend`、`/dashboard/institution/stats`

- [x] Task 27: 系统配置实现（4 接口）
  - [x] SubTask 27.1: 新建 `sys_config_mapper.go` 和 `sys_config_service.go`
  - [x] SubTask 27.2: 注册路由 `/config/list`、`/config/insert`、`/config/update`、`/config/delete`

- [x] Task 28: 阶段五编译验证
  - [x] SubTask 28.1: 执行 `go build ./...` 确保编译通过

## 阶段六：全量接口验证

- [x] Task 29: 前后端接口对照验证
  - [x] SubTask 29.1: 验证小程序 64 个接口全部在 Go 后端注册 ✅ 全部注册
  - [x] SubTask 29.2: 验证 Admin 前端 62 个接口全部在 Go 后端注册 ✅ 全部注册
  - [x] SubTask 29.3: 验证 Gateway publicPaths 白名单中所有路径已实现 ✅ 20/20 实现
  - [x] SubTask 29.4: 验证响应结构（DTO/VO 字段）与前端类型定义一致 ⚠️ 发现 9 处不一致（见 Task 30）
  - [x] SubTask 29.5: 验证 handler/service 为真实实现（非 stub）✅ 90/91 真实实现，仅 /record/delete 路由未注册
  - [x] SubTask 29.6: 验证 `go build ./...` 编译通过 ✅ exit code 0

## 阶段七：响应结构不一致修复（9 项问题）

> **背景**：Task 29 验证发现 9 处响应结构与前端类型定义不一致，其中 3 项致命阻塞小程序核心绑定流程，3 项重要影响 Admin 仪表盘和扣费详情，3 项次要影响前端体验。

### P0 致命问题（阻塞小程序绑定流程）

- [ ] Task 30: 修复绑定二维码响应结构
  - [ ] SubTask 30.1: 修改 `auth_service.go` 中 `BindQrcodeVO`（约 1038 行）字段为前端期望的 `qrcode`（二维码 base64）、`token`（绑定 token）、`bindCode`（6 位绑定码）
  - [ ] SubTask 30.2: 同步修改 `GenerateBindQrcode` 和 `GenerateSubscribeQrcode` 方法返回值构造
  - [ ] SubTask 30.3: 对齐前端 `src/types/bind.d.ts` 的 `BindQrcodeResponse` 类型定义

- [ ] Task 31: 修复绑定信息查询响应结构
  - [ ] SubTask 31.1: 修改 `auth_service.go` 中 `BindInfoVO`（约 1047 行）补充缺失字段：`relation`（与学生关系）、`isPrimary`（是否主联系人）、`parentName`（预填家长名）、`parentPhone`（预填家长手机号）
  - [ ] SubTask 31.2: 将 `isSubscribe` 字段重命名为 `subscribeOnly`（对齐前端 `BindInfoResponse.subscribeOnly`）
  - [ ] SubTask 31.3: 修改 `GetBindInfo` 和 `GetBindInfoByCode` 方法查询 parent_student 表填充 `relation/isPrimary`，查询 parent 表填充 `parentName/parentPhone`

- [ ] Task 32: 修复绑定状态检查响应结构
  - [ ] SubTask 32.1: 修改 `auth_service.go` 中 `BindStatusVO`（约 1058 行）字段为 `{ alreadyBound: bool, hasAccount: bool }`
  - [ ] SubTask 32.2: 修改 `CheckBindStatus` 方法逻辑：`alreadyBound` 检查 parent_student 是否已存在关联；`hasAccount` 检查该学生是否已有任意家长账号（c_parent 表关联）
  - [ ] SubTask 32.3: 移除原有的 `status`/`studentInfo` 字段（或保留为内部使用，但 JSON tag 不输出）

### P1 重要问题（影响 Admin 仪表盘和扣费详情）

- [ ] Task 33: 修复仪表盘趋势和机构统计字段名
  - [ ] SubTask 33.1: 修改 `dashboard_mapper.go` 第 32 行 `DashboardTrendRow.Labels` 的 JSON tag 从 `labels` 改为 `months`
  - [ ] SubTask 33.2: 修改 `dashboard_mapper.go` 第 39 行 `InstitutionStatRow.ID` 的 JSON tag 从 `id` 改为 `institutionId`
  - [ ] SubTask 33.3: 编译验证

- [ ] Task 34: 修复扣费详情响应结构
  - [ ] SubTask 34.1: 修改 `course_record_service.go` 第 565 行 `DeductDetailVO` 补充 12 个缺失字段：`courseRecordId`、`courseName`、`courseType`、`studentName`、`deductCount`、`courseTotalTime`、`expireTime`、`classId`、`className`、`scheduleDesc`、`teacherId`、`teacherName`、`expireStatus`
  - [ ] SubTask 34.2: 修改 `GetDeductDetail` 方法，联表查询 c_course（课程名/类型/总课时/到期时间）、c_student（学生名）、c_class（班级名）、c_class_schedule（课表描述）、c_teacher（教师名）填充上述字段
  - [ ] SubTask 34.3: 将 `recordTimeStr` 重命名为 `recordTime`、`recordRemark` 重命名为 `remark`（对齐前端 `DeductDetailResponse`）
  - [ ] SubTask 34.4: 实现 `expireStatus` 字段逻辑：根据 `expire_time` 与当前时间比较返回 "normal"/"expired"/"warning"

### P2 次要问题（提升前端体验）

- [ ] Task 35: 修复 Admin 班级列表 teachers 字段结构
  - [ ] SubTask 35.1: 修改 `admin_business_mapper.go` 第 911 行 `AdminClassRow`，将 `teacherIds []int64` + `teacherNames []string` 聚合为 `teachers []TeacherResponse` 对象数组
  - [ ] SubTask 35.2: 修改 SQL 查询或 Go 层逻辑，将 ID 和名称配对为对象数组

- [ ] Task 36: 补充 Admin 家长信息 isBound 字段
  - [ ] SubTask 36.1: 在 Admin 前端 `src/types/business.d.ts` 的 `ParentInfoResponse` 类型中增加 `isBound?: boolean` 字段（Go 端已返回该字段）

- [ ] Task 37: 修复 Admin 课表时间字段命名
  - [ ] SubTask 37.1: 修改 `admin_business_mapper.go` 中 `AdminClassScheduleRow` 的 `CreateTimeStr/UpdateTimeStr` Go 字段名为 `CreateTime/UpdateTime`（与 JSON tag `createTime/updateTime` 对齐）
  - [ ] SubTask 37.2: 同步修改相关 service 层代码引用

### 阶段七编译与验证

- [ ] Task 38: 阶段七编译与全量验证
  - [ ] SubTask 38.1: 执行 `go build ./...` 确保编译通过
  - [ ] SubTask 38.2: 重新验证 Task 29 SubTask 29.4 响应结构一致性
  - [ ] SubTask 38.3: 验证小程序绑定流程端到端可用
  - [ ] SubTask 38.4: 验证 Admin 仪表盘和扣费详情页正常渲染

## 阶段八：补录缺失路由

- [ ] Task 39: 注册 `/biz/record/delete` 路由
  - [ ] SubTask 39.1: 在 `business-service/internal/handler/handler.go` 新增 `DeleteRecord` handler 方法
  - [ ] SubTask 39.2: 在 handler 注册 `POST /record/delete` 路由（Mapper 层 `RecordMapper.DeleteByID` 已实现于 `record_mapper.go` 第 277 行）
  - [ ] SubTask 39.3: 编译验证

# Task Dependencies

- Task 6（实体定义）阻塞 Task 7-13（business-service 写操作）
- Task 15（系统实体定义）阻塞 Task 16-19（admin-service 系统管理）
- Task 1-4（auth 绑定流程）无依赖，可独立进行
- Task 6-13（business）与 Task 15-20（admin 系统）无依赖，可并行
- Task 21-25（admin 业务透传）依赖 Task 6（业务实体）和 Task 19（操作日志）
- Task 26-27（仪表盘+配置）依赖 Task 15（系统实体）
- Task 29（全量验证）依赖所有前序任务完成
- Task 30-32（P0 绑定流程修复）依赖 Task 29（验证发现的问题）
- Task 33-34（P1 仪表盘+扣费详情修复）依赖 Task 29，可与 Task 30-32 并行
- Task 35-37（P2 次要修复）依赖 Task 29，可与 Task 30-34 并行
- Task 38（阶段七编译验证）依赖 Task 30-37 全部完成
- Task 39（补录 /record/delete 路由）独立，可与 Task 30-37 并行

## 并行执行建议

- **第一批并行**：Task 1-4（auth 绑定）+ Task 6（业务实体）+ Task 15（系统实体）
- **第二批并行**：Task 7-13（business 各模块）+ Task 16-19（admin 系统管理）
- **第三批并行**：Task 21-25（admin 业务透传）+ Task 26-27（仪表盘+配置）
- **第四批（验证）**：Task 29（全量验证）
- **第五批并行（修复）**：Task 30-32（P0）+ Task 33-34（P1）+ Task 35-37（P2）+ Task 39（补录路由）
- **最终**：Task 38（阶段七编译与全量验证）
