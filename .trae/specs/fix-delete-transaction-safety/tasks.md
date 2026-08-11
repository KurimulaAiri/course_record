# Tasks

## Task 1: business-service Mapper 新增 Tx 版本删除方法

为以下 Mapper 新增接收 `*sql.Tx` 参数的删除方法，SQL 与原方法一致，仅替换 `db` 为 `tx`：

- [x] Task 1.1: `teacher_mapper.go` — 新增 `DeleteByIDTx(tx *sql.Tx, id int64) (int64, error)`
- [x] Task 1.2: `user_mapper.go` — 新增 `DeleteByIDTx(tx *sql.Tx, id int64) (int64, error)`
- [x] Task 1.3: `user_auth_mapper.go` — 新增 `DeleteByIDTx(tx *sql.Tx, id int64) (int64, error)`
- [x] Task 1.4: `parent_mapper.go` — 新增 `DeleteByIDTx(tx *sql.Tx, id int64) (int64, error)` 和 `ResetUnboundTx(tx *sql.Tx, id int64) (int64, error)`
- [x] Task 1.5: `parent_student_mapper.go` — 新增 `DeleteByParentAndStudentTx(tx *sql.Tx, parentID, studentID int64) (int64, error)`
- [x] Task 1.6: `wx_subscribe_mapper.go` — 新增 `DeleteByStudentAndIsPrimaryTx(tx *sql.Tx, studentID int64, isPrimary bool) (int64, error)` 和 `DeleteByOpenIDsTx(tx *sql.Tx, openIDs []string) (int64, error)`
- [x] Task 1.7: `record_mapper.go` — 新增 `DeleteByCourseRecordIDTx(tx *sql.Tx, courseRecordID int64) (int64, error)`（按 course_record_id 硬删除关联上课记录）

## Task 2: business-service Service 层加事务保护

- [x] Task 2.1: `teacher_service.go` — `DeleteTeacher` 加事务：开启 tx → 查询校验 → 删 c_user_auth（Tx） → 删 c_user（Tx） → 删 c_teacher（Tx） → Commit；任一失败 Rollback 并返回错误。消除所有 `_, _ =`。
- [x] Task 2.2: `student_service.go` — `UnbindStudent` 加事务：开启 tx → 删 wx_student_subscribe（Tx） → 删 parent_student（Tx） → 处理 parent（Tx） → Commit；任一失败 Rollback 并返回错误。消除所有 `_, _ =`。
- [x] Task 2.3: `student_service.go` — `CancelStudentSubscribe` 加事务：开启 tx → 删 wx_student_subscribe（Tx） → 删 wx_subscribe_record（Tx） → Commit；任一失败 Rollback 并返回错误。消除所有 `_, _ =`。
- [x] Task 2.4: `course_record_service.go` — `DeleteCourseRecord` 加事务：软删 c_course_record（Tx） → 硬删关联 c_record（Tx，调用 Task 1.7 方法） → Commit。

## Task 3: admin-service Mapper 新增 Tx 版本删除方法

- [x] Task 3.1: `admin_mapper.go` — 新增 `DeleteTx(tx *sql.Tx, id int64) error`（逻辑删除 sys_user）和 `DeleteUserRolesByUserIDTx(tx *sql.Tx, userID int64) error`
- [x] Task 3.2: `sys_role_mapper.go` — 新增 `DeleteTx(tx *sql.Tx, id int64) error`（逻辑删除 sys_role）
- [x] Task 3.3: `sys_role_menu_mapper.go` — 新增 `DeleteByRoleIDTx(tx *sql.Tx, roleID int64) error`（实际追加到 sys_role_mapper.go，因为 SysRoleMenuMapper 定义在该文件）
- [x] Task 3.4: `sys_menu_mapper.go` — 新增 `DeleteByIDTx(tx *sql.Tx, id int64) error`

## Task 4: admin-service Service 层加事务保护

- [x] Task 4.1: `admin_service.go` — `DeleteUser` 加事务：开启 tx → 逻辑删 sys_user（Tx） → 删 sys_user_role（Tx） → Commit；任一失败 Rollback。操作日志在事务提交后记录。
- [x] Task 4.2: `sys_role_service.go` — `DeleteRole` 加事务：开启 tx → 逻辑删 sys_role（Tx） → 删 sys_role_menu（Tx） → Commit；任一失败 Rollback。操作日志在事务提交后记录。
- [x] Task 4.3: `sys_menu_service.go` — `DeleteMenu` 加事务：开启 tx → 删 sys_menu（Tx） → 删 sys_role_menu（Tx，按 menu_id） → Commit；任一失败 Rollback。

## Task 5: 编译验证与文档更新

- [x] Task 5.1: 执行 `go build ./...` 确保无编译错误
- [x] Task 5.2: 更新 `class_times_record_back/CLAUDE.md`，在"请求日志中间件"章节后新增"删除操作事务规范"说明

# Task Dependencies

- Task 2 依赖 Task 1（Service 层需要调用 Mapper Tx 方法）
- Task 4 依赖 Task 3（Service 层需要调用 Mapper Tx 方法）
- Task 1 和 Task 3 可并行
- Task 2 和 Task 4 可并行（分别对应 business-service 和 admin-service）
- Task 5 依赖 Task 1-4 全部完成
