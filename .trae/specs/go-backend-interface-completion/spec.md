# Go 后端接口补全 Spec

## Why

Go 后端迁移已完成核心登录链路（auth-service 登录/刷新、business-service 查询、admin-service 登录），但全面核对小程序前端（65 接口）和 Admin 前端（52 接口）后发现，**Go 后端仅实现了 26 个业务接口，仍有约 91 个接口未实现**，导致前端大部分功能无法使用。

核心问题：
1. **auth-service 绑定流程完全缺失**：家长绑定学生、绑定码生成、订阅消息推送等 8 个接口未实现，小程序核心业务流程中断
2. **business-service 仅实现查询，缺失全部写操作**：学生/教师/班级/课程/课表/课时记录的增删改共 38 个接口未实现，教师端所有业务操作不可用
3. **admin-service 仅实现登录和用户列表**：角色/菜单/仪表盘/操作日志/业务管理共 45 个接口未实现，Admin 前端几乎所有页面不可用
4. **Gateway 白名单声明但未实现的路径**：7 个公开接口在 `publicPaths` 中声明但 Go 端未实现（绑定相关 6 个 + deduct-detail 1 个）

## What Changes

### 阶段一：auth-service 绑定与订阅流程补全（8 接口，P0 阻塞小程序核心流程）

- **实现绑定二维码生成**：`POST /auth/generate_bind_qrcode`、`POST /auth/generate_subscribe_qrcode`
- **实现绑定信息查询**：`GET /auth/get_bind_info`（按 token）、`GET /auth/get_bind_info_by_code`（按 6 位绑定码）
- **实现绑定状态检查与确认**：`GET /auth/check_bind_status`、`POST /auth/confirm_bind`、`POST /auth/bind_by_code`
- **实现订阅消息测试发送**：`GET /auth/test_send_subscribe`
- **复用现有 mapper**：ParentMapper、StudentMapper、ParentStudentMapper、WxSubscribeRecordMapper、WxStudentSubscribeMapper 已存在，无需新建

### 阶段二：business-service 写操作补全（38 接口，P1 小程序业务功能）

#### 学生模块（6 接口）
- `POST /student/get_by_class_id`、`POST /student/get_by_course_id`（查询补全）
- `POST /student/insert`、`POST /student/update`、`POST /student/unbind`、`POST /student/cancel_subscribe`

#### 教师模块（3 接口）
- `POST /teacher/update_by_id`、`POST /teacher/insert`、`POST /teacher/delete`

#### 班级模块（8 接口，全部）
- `POST /class/get_classes_by_student_id`、`POST /class/get_classes_by_teacher_id`、`POST /class/get_classes_by_institution_id`
- `POST /class/get_class_by_id`、`POST /class/insert`、`POST /class/update_by_id`
- `POST /class/add_student_to_class`、`POST /class/remove_student_from_class`

#### 课表模块（5 接口，全部）
- `POST /class_schedule/get_by_class_id`、`POST /class_schedule/get_by_institution_id`、`POST /class_schedule/get_by_teacher_id`
- `POST /class_schedule/get_by_id`、`POST /class_schedule/update_by_id`

#### 课程模块（4 接口，全部）
- `POST /course/get_course_by_institution_id`、`POST /course/get_course_by_student_id`
- `POST /course/add_course`、`POST /course/update_by_id`

#### 课卡记录模块（9 接口，全部）
- `POST /course_record/new_get`、`POST /course_record/get_by_student_id`、`POST /course_record/get_by_institution_id`
- `POST /course_record/insert`、`POST /course_record/update`
- `POST /course_record/deduct_by_student_id`、`POST /course_record/deduct_by_course_id`、`POST /course_record/deduct_by_class_id`
- `GET /course_record/deduct-detail`（公开接口，需加入 Gateway 白名单）

#### 机构模块（1 接口）
- `POST /institution/update`

#### 上课记录模块（2 接口）
- `POST /record/new_get`、`POST /record/delete`（页面直接调用）

### 阶段三：admin-service 系统管理补全（19 接口，P2 Admin 基础功能）

#### 用户管理（6 接口）
- `POST /user/get_by_id`、`POST /user/insert`、`POST /user/update`、`POST /user/delete`
- `POST /user/reset_password`、`POST /user/get_roles`

#### 角色管理（7 接口，全部）
- `POST /role/list`、`POST /role/get_by_id`、`POST /role/insert`、`POST /role/update`、`POST /role/delete`
- `POST /role/get_menus`、`POST /role/save_menus`

#### 菜单管理（5 接口，全部）
- `POST /menu/list`、`POST /menu/tree`、`POST /menu/user_tree`
- `POST /menu/insert`、`POST /menu/update`、`POST /menu/delete`

#### 操作日志（3 接口，全部）
- `POST /operation_log/list`、`POST /operation_log/delete`、`POST /operation_log/clear`

### 阶段四：admin-service 业务管理透传（26 接口，P3 Admin 业务功能）

#### 机构管理（3 接口）
- `POST /business/institution/list`、`POST /business/institution/insert`、`POST /business/institution/update`

#### 学生管理（3 接口）
- `POST /business/student/list`、`POST /business/student/insert`、`POST /business/student/update`

#### 教师管理（3 接口）
- `POST /business/teacher/list`、`POST /business/teacher/insert`、`POST /business/teacher/update`

#### 教师账号管理（4 接口）
- `POST /teacher_auth/get`、`POST /teacher_auth/update_account`、`POST /teacher_auth/update_password`
- `POST /teacher_auth/toggle_institution_admin`

#### 课程管理（3 接口）
- `POST /business/course/list`、`POST /business/course/insert`、`POST /business/course/update`

#### 班级管理（6 接口）
- `POST /business/class/list`、`POST /business/class/insert`、`POST /business/class/update`
- `POST /business/class/get_by_id`、`POST /business/class/add_student`、`POST /business/class/remove_student`

#### 课表管理（2 接口）
- `POST /business/class_schedule/list`、`POST /business/class_schedule/update`

#### 课时记录管理（3 接口）
- `POST /business/course_record/list`、`POST /business/course_record/insert`、`POST /business/course_record/update`

#### 上课记录管理（2 接口）
- `POST /business/record/list`、`POST /business/record/insert`

#### 小程序菜单管理（4 接口）
- `POST /business/mini_menu/list`、`POST /business/mini_menu/insert`、`POST /business/mini_menu/update`、`POST /business/mini_menu/delete`

### 阶段五：仪表盘与系统配置（7 接口，P3 辅助功能）

#### 仪表盘（3 接口）
- `POST /dashboard/data`、`POST /dashboard/trend`、`POST /dashboard/institution/stats`

#### 系统配置（4 接口）
- `POST /config/list`、`POST /config/insert`、`POST /config/update`、`POST /config/delete`

### 阶段六：全量接口验证（Task 29 已完成）

**验证结果**：
- ✅ 小程序 64 个接口全部在 Go 后端注册
- ✅ Admin 前端 62 个接口全部在 Go 后端注册
- ✅ Gateway publicPaths 20 个路径全部实现
- ✅ handler/service 均为真实实现（非 stub），仅 `/record/delete` 路由未注册
- ✅ `go build ./...` 编译通过
- ⚠️ 发现 9 处响应结构与前端类型定义不一致（见阶段七）

### 阶段七：响应结构不一致修复（9 项问题，Task 30-38）

#### P0 致命问题（3 项，阻塞小程序绑定流程）
- **修复 `BindQrcodeVO`**：字段改为 `qrcode/token/bindCode`（当前为 `code/qrContent/isSubscribe`）
- **修复 `BindInfoVO`**：补充 `relation/isPrimary/parentName/parentPhone` 字段，`isSubscribe` 重命名为 `subscribeOnly`
- **修复 `BindStatusVO`**：字段改为 `{ alreadyBound, hasAccount }`（当前为 `{ status, studentInfo, alreadyBound }`）

#### P1 重要问题（3 项，影响 Admin 仪表盘和扣费详情）
- **修复 `DashboardTrendRow`**：JSON tag `labels` 改为 `months`
- **修复 `InstitutionStatRow`**：JSON tag `id` 改为 `institutionId`
- **修复 `DeductDetailVO`**：补充 12 个缺失字段（courseName/courseType/studentName/deductCount/courseTotalTime/expireTime/classId/className/scheduleDesc/teacherId/teacherName/expireStatus），重命名 `recordTimeStr→recordTime`、`recordRemark→remark`

#### P2 次要问题（3 项，提升前端体验）
- **修复 `AdminClassRow.teachers`**：聚合为对象数组（当前为分离的 `teacherIds`+`teacherNames`）
- **补充前端 `ParentInfoResponse.isBound`**：前端类型增加 `isBound?: boolean`
- **修复 `AdminClassScheduleRow`**：Go 字段名 `CreateTimeStr/UpdateTimeStr` 改为 `CreateTime/UpdateTime`

### 阶段八：补录缺失路由（Task 39）
- **注册 `POST /biz/record/delete`**：Mapper 层 `RecordMapper.DeleteByID` 已实现，仅需补充 handler 和路由注册

## Impact

### 受影响的代码

**auth-service**（阶段一）：
- `auth-service/internal/handler/auth_handler.go` — 新增 8 个 handler 方法
- `auth-service/internal/service/auth_service.go` — 新增 8 个 service 方法（绑定/订阅流程）
- `auth-service/internal/mapper/` — 可能需要补充绑定码表 mapper
- `auth-service/main.go` — 注册新路由

**business-service**（阶段二）：
- `business-service/internal/handler/handler.go` — 新增 38 个 handler 方法
- `business-service/internal/service/service.go` — 新增 38 个 service 方法
- `business-service/internal/mapper/` — 新增 ClassMapper、ClassScheduleMapper、CourseMapper、CourseRecordMapper、RecordMapper
- `business-service/main.go` — 注册新路由
- `common/entity/entity.go` — 新增 Class、ClassSchedule、Course、CourseRecord、Record 实体

**admin-service**（阶段三、四、五）：
- `admin-service/internal/handler/admin_handler.go` — 新增 52 个 handler 方法
- `admin-service/internal/service/admin_service.go` — 新增 52 个 service 方法
- `admin-service/internal/mapper/` — 新增 SysRoleMapper、SysMenuMapper、SysOperationLogMapper、SysConfigMapper
- `admin-service/main.go` — 注册新路由
- `common/entity/entity.go` — 新增 SysRole、SysMenu、SysOperationLog、SysConfig 实体

**gateway**：
- `gateway/internal/server.go` — 确认 `publicPaths` 中所有声明路径已在 Go 端实现（阶段一完成后）

### 参考的 Java 实现

每个接口的 Go 实现应严格对齐 Java 后端逻辑：
- DTO/VO 字段命名与 Java 一致
- 业务流程（校验、事务、缓存）与 Java 对齐
- SQL 查询对齐 Java Mapper XML
- 错误码与 Java ResponseDTO 一致

## ADDED Requirements

### Requirement: 绑定二维码响应结构对齐

系统 SHALL 在 `POST /auth/generate_bind_qrcode` 和 `POST /auth/generate_subscribe_qrcode` 接口返回前端期望的字段结构。

#### Scenario: 返回正确的二维码响应
- **WHEN** 教师端调用绑定二维码生成接口
- **THEN** 响应 `data` 字段包含 `qrcode`（二维码 base64）、`token`（绑定 token）、`bindCode`（6 位绑定码）
- **AND** 不再返回 `code/qrContent/isSubscribe` 字段

### Requirement: 绑定信息查询响应结构对齐

系统 SHALL 在 `GET /auth/get_bind_info` 和 `GET /auth/get_bind_info_by_code` 接口返回完整的学生绑定信息。

#### Scenario: 返回完整绑定信息
- **WHEN** 家长查询绑定信息
- **THEN** 响应包含 `studentId/studentName/sex/institutionName/relation/isPrimary/subscribeOnly/parentName/parentPhone` 字段
- **AND** `subscribeOnly` 字段名不使用 `isSubscribe`

### Requirement: 绑定状态检查响应结构对齐

系统 SHALL 在 `GET /auth/check_bind_status` 接口返回前端期望的简洁结构。

#### Scenario: 返回绑定状态
- **WHEN** 家长检查绑定状态
- **THEN** 响应 `data` 字段为 `{ alreadyBound: boolean, hasAccount: boolean }`
- **AND** `hasAccount` 表示该学生是否已有任意家长账号

### Requirement: 仪表盘趋势字段名对齐

系统 SHALL 在 `POST /admin/dashboard/trend` 接口返回 `months` 字段名（非 `labels`）。

#### Scenario: 趋势数据返回
- **WHEN** 管理员查询趋势数据
- **THEN** 响应 `data.months` 为时间刻度字符串数组

### Requirement: 机构统计字段名对齐

系统 SHALL 在 `POST /admin/dashboard/institution/stats` 接口返回 `institutionId` 字段名（非 `id`）。

#### Scenario: 机构统计返回
- **WHEN** 管理员查询机构统计
- **THEN** 每条记录包含 `institutionId` 字段（非 `id`）

### Requirement: 扣费详情响应结构对齐

系统 SHALL 在 `GET /biz/course_record/deduct-detail` 接口返回完整的扣费详情，包含课程、学生、班级、教师等关联信息。

#### Scenario: 返回完整扣费详情
- **WHEN** 查询扣费详情
- **THEN** 响应包含 21 个字段：`courseRecordId/recordId/courseId/courseName/courseType/studentId/studentName/deductCount/courseRestTime/restTimeAfterDeduct/courseTotalTime/expireTime/recordTime/remark/classId/className/scheduleDesc/teacherId/teacherName/deductMode/expireStatus`
- **AND** `expireStatus` 取值为 `normal`/`expired`/`warning`

### Requirement: 绑定二维码生成

系统 SHALL 提供 `POST /auth/generate_bind_qrcode` 和 `POST /auth/generate_subscribe_qrcode` 接口，生成家长绑定学生/订阅的二维码。

#### Scenario: 生成绑定二维码
- **WHEN** 教师端调用 `generate_bind_qrcode` 传入 studentId
- **THEN** 生成 6 位绑定码，存入 Redis（TTL 10 分钟），返回二维码内容

### Requirement: 绑定码查询与确认

系统 SHALL 提供 `GET /auth/get_bind_info_by_code`、`GET /auth/check_bind_status`、`POST /auth/confirm_bind`、`POST /auth/bind_by_code` 接口，支持家长通过绑定码完成绑定。

#### Scenario: 按绑定码绑定学生
- **WHEN** 家长调用 `bind_by_code` 传入绑定码和 openId
- **THEN** 校验绑定码有效性，创建 parent_student 关联记录，更新 wx_student_subscribe，返回绑定结果

### Requirement: 课卡扣费双重校验

系统 SHALL 在课卡扣费（`deduct_by_student_id`/`deduct_by_course_id`/`deduct_by_class_id`）时执行双重校验：
1. Java 层校验 `expire_time`（过期抛 `COURSE_EXPIRED` code=1003）
2. SQL 层 WHERE 条件包含 `(expire_time IS NULL OR expire_time > NOW())` 兜底

#### Scenario: 课时已过期
- **WHEN** 扣费时课程 `expire_time < NOW()`
- **THEN** 返回 `{ code: 1003, message: "课程已过期" }`

#### Scenario: 课时余额不足
- **WHEN** 扣费时 `rest_time < deduct_amount`
- **THEN** 返回 `{ code: 1001, message: "课时余额不足" }`

### Requirement: Admin 角色菜单授权

系统 SHALL 提供 `POST /role/get_menus` 和 `POST /role/save_menus` 接口，管理角色与菜单的关联关系（sys_role_menu 表）。

#### Scenario: 保存角色菜单
- **WHEN** 管理员调用 `save_menus` 传入 roleId 和 menuIds
- **THEN** 删除旧关联，插入新关联（事务），记录操作日志

### Requirement: Admin 业务管理透传

系统 SHALL 在 admin-service 提供 `/admin/business/*` 路径的接口，复用 business-service 的 Mapper 直接操作业务表（非 RPC 调用），并附加操作日志记录。

#### Scenario: Admin 新增学生
- **WHEN** 管理员调用 `/admin/business/student/insert`
- **THEN** 写入 c_student 表，记录 sys_operation_log，返回新学生 ID

## MODIFIED Requirements

### Requirement: Gateway publicPaths 白名单

阶段一完成后，以下 publicPaths 声明的路径必须全部实现：
- `/auth/auth/get_bind_info`
- `/auth/auth/get_bind_info_by_code`
- `/auth/auth/check_bind_status`
- `/auth/auth/confirm_bind`
- `/auth/auth/test_send_subscribe`
- `/auth/auth/bind_by_code`
- `/biz/course_record/deduct-detail`

## REMOVED Requirements

无移除需求。所有 Java 端接口在 Go 端按原路径原样实现，保持前后端兼容。
