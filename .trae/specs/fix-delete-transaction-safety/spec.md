# 删除操作事务安全与一致性修复 Spec

## Why

当前系统中所有涉及多表级联删除的接口（DeleteTeacher/UnbindStudent/CancelStudentSubscribe/DeleteUser/DeleteRole/DeleteMenu）均**未使用事务保护**，且部分错误被显式忽略（`_, _ =`）。这会导致中间步骤失败时产生孤儿数据（如 c_user 已删但 c_teacher 还在），破坏数据一致性，且难以排查。

此外，DeleteCourseRecord 软删除 c_course_record 后不清理关联的 c_record，会产生悬挂的上课记录。

## What Changes

### 1. 为多表删除操作添加事务保护
- `DeleteTeacher`（business-service）：c_user_auth → c_user → c_teacher 三步删除包裹事务
- `UnbindStudent`（business-service）：wx_student_subscribe → parent_student → parent 三步删除包裹事务
- `CancelStudentSubscribe`（business-service）：wx_student_subscribe → wx_subscribe_record 两步删除包裹事务
- `DeleteUser`（admin-service）：sys_user 软删 + sys_user_role 硬删包裹事务
- `DeleteRole`（admin-service）：sys_role 软删 + sys_role_menu 硬删包裹事务
- `DeleteMenu`（admin-service）：sys_menu + sys_role_menu 两步删除包裹事务

### 2. 消除错误吞没（`_, _ =`）
- 所有级联删除步骤的错误必须处理：失败时回滚事务并返回错误
- 不再使用 `_, _ =` 忽略错误

### 3. 补全级联清理
- `DeleteCourseRecord`：软删除 c_course_record 时，同时硬删除关联的 c_record（按 course_record_id）
- `DeleteTeacher`：删除前检查 c_class_teacher 关联（已有），保持不变

### 4. 新增 Mapper Tx 版本方法
- 为需要事务保护的 Mapper 方法新增 `*Tx(tx, ...)` 版本，接收 `*sql.Tx` 参数
- 原 Mapper 方法保持不变（供非事务场景使用）

## Impact

- Affected specs: 无（独立修复）
- Affected code:
  - `business-service/internal/service/teacher_service.go` — DeleteTeacher 加事务
  - `business-service/internal/service/student_service.go` — UnbindStudent/CancelStudentSubscribe 加事务
  - `business-service/internal/service/course_record_service.go` — DeleteCourseRecord 补充级联清理 c_record
  - `business-service/internal/mapper/teacher_mapper.go` — 新增 DeleteByIDTx
  - `business-service/internal/mapper/user_mapper.go` — 新增 DeleteByIDTx
  - `business-service/internal/mapper/user_auth_mapper.go` — 新增 DeleteByIDTx
  - `business-service/internal/mapper/parent_mapper.go` — 新增 DeleteByIDTx/ResetUnboundTx
  - `business-service/internal/mapper/parent_student_mapper.go` — 新增 DeleteByParentAndStudentTx
  - `business-service/internal/mapper/wx_subscribe_mapper.go` — 新增 DeleteByStudentAndIsPrimaryTx/DeleteByOpenIDsTx
  - `business-service/internal/mapper/record_mapper.go` — 新增 DeleteByCourseRecordIDTx
  - `admin-service/internal/service/admin_service.go` — DeleteUser 加事务
  - `admin-service/internal/service/sys_role_service.go` — DeleteRole 加事务
  - `admin-service/internal/service/sys_menu_service.go` — DeleteMenu 加事务
  - `admin-service/internal/mapper/admin_mapper.go` — 新增 DeleteTx/DeleteUserRolesByUserIDTx
  - `admin-service/internal/mapper/sys_role_mapper.go` — 新增 DeleteTx
  - `admin-service/internal/mapper/sys_role_menu_mapper.go` — 新增 DeleteByRoleIDTx
  - `admin-service/internal/mapper/sys_menu_mapper.go` — 新增 DeleteByIDTx

## ADDED Requirements

### Requirement: 删除操作事务保护
多表级联删除操作 SHALL 在单一数据库事务中执行，任一步骤失败时整体回滚，确保数据一致性。

#### Scenario: 删除教师时 c_user 删除失败
- **WHEN** 调用 DeleteTeacher，c_user_auth 删除成功但 c_user 删除失败
- **THEN** 事务回滚，c_user_auth 恢复，返回"系统异常"错误

#### Scenario: 解绑学生时 parent 删除失败
- **WHEN** 调用 UnbindStudent，parent_student 删除成功但 parent 删除失败
- **THEN** 事务回滚，parent_student 恢复，返回"系统异常"错误

#### Scenario: 删除课卡记录时级联清理上课记录
- **WHEN** 调用 DeleteCourseRecord，c_course_record 软删除成功
- **THEN** 关联的 c_record 记录同步硬删除，无悬挂数据

### Requirement: 错误处理规范化
删除操作中的所有级联步骤 SHALL 正确处理错误，禁止使用 `_, _ =` 忽略。

#### Scenario: 级联清理失败
- **WHEN** 任一级联删除步骤返回错误
- **THEN** 事务回滚，返回"系统异常"，错误信息写入日志

## MODIFIED Requirements

### Requirement: DeleteTeacher
删除教师时 SHALL 在事务内依次执行：
1. 校验教师未关联班级（class_teacher count=0）
2. 查询教师记录获取 userID
3. 删除 c_user_auth（如有关联）
4. 删除 c_user（如有关联）
5. 删除 c_teacher

任一步骤失败时回滚事务。

### Requirement: UnbindStudent
解绑家长-学生关系时 SHALL 在事务内依次执行：
1. 查询 parent_student 获取 isPrimary
2. 删除 wx_student_subscribe（按 studentId + isPrimary）
3. 删除 parent_student 关联记录
4. 判断并处理 parent 记录（占位 parent 删除 / 已绑定 parent 无其他关联时重置）

任一步骤失败时回滚事务。

### Requirement: CancelStudentSubscribe
取消微信订阅时 SHALL 在事务内依次执行：
1. 查询 parent_student 获取 isPrimary
2. 删除 wx_student_subscribe（按 studentId + isPrimary）
3. 查询 parent 的 openId 列表
4. 删除 wx_subscribe_record（按 openId 列表）

任一步骤失败时回滚事务。

### Requirement: DeleteUser (admin)
删除管理员用户时 SHALL 在事务内依次执行：
1. 校验用户存在
2. 逻辑删除 sys_user（is_deleted=1）
3. 硬删除 sys_user_role 关联

任一步骤失败时回滚事务。

### Requirement: DeleteRole (admin)
删除角色时 SHALL 在事务内依次执行：
1. 校验角色存在
2. 逻辑删除 sys_role（is_deleted=1）
3. 硬删除 sys_role_menu 关联

任一步骤失败时回滚事务。

### Requirement: DeleteMenu (admin)
删除菜单时 SHALL 在事务内依次执行：
1. 删除 sys_menu
2. 硬删除 sys_role_menu 关联

任一步骤失败时回滚事务。

### Requirement: DeleteCourseRecord
删除课卡记录时 SHALL 依次执行：
1. 软删除 c_course_record（is_delete=1）
2. 硬删除关联的 c_record（按 course_record_id）

两步包裹在事务中，任一失败时回滚。
