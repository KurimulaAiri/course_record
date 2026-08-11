# Checklist

## business-service Mapper Tx 方法
- [x] teacher_mapper.go 新增 DeleteByIDTx 方法，接收 *sql.Tx 参数
- [x] user_mapper.go 新增 DeleteByIDTx 方法，接收 *sql.Tx 参数
- [x] user_auth_mapper.go 新增 DeleteByIDTx 方法，接收 *sql.Tx 参数
- [x] parent_mapper.go 新增 DeleteByIDTx 和 ResetUnboundTx 方法，接收 *sql.Tx 参数
- [x] parent_student_mapper.go 新增 DeleteByParentAndStudentTx 方法，接收 *sql.Tx 参数
- [x] wx_subscribe_mapper.go 新增 DeleteByStudentAndIsPrimaryTx 和 DeleteByOpenIDsTx 方法，接收 *sql.Tx 参数
- [x] record_mapper.go 新增 DeleteByCourseRecordIDTx 方法，接收 *sql.Tx 参数

## business-service Service 事务保护
- [x] DeleteTeacher 在事务内执行 c_user_auth/c_user/c_teacher 三步删除
- [x] DeleteTeacher 消除所有 `_, _ =` 错误吞没
- [x] DeleteTeacher 任一步骤失败时 Rollback 并返回"系统异常"
- [x] UnbindStudent 在事务内执行 wx_student_subscribe/parent_student/parent 三步删除
- [x] UnbindStudent 消除所有 `_, _ =` 错误吞没
- [x] UnbindStudent 任一步骤失败时 Rollback 并返回"系统异常"
- [x] CancelStudentSubscribe 在事务内执行 wx_student_subscribe/wx_subscribe_record 两步删除
- [x] CancelStudentSubscribe 消除所有 `_, _ =` 错误吞没
- [x] CancelStudentSubscribe 任一步骤失败时 Rollback 并返回"系统异常"
- [x] DeleteCourseRecord 在事务内软删 c_course_record + 硬删关联 c_record

## admin-service Mapper Tx 方法
- [x] admin_mapper.go 新增 DeleteTx 和 DeleteUserRolesByUserIDTx 方法，接收 *sql.Tx 参数
- [x] sys_role_mapper.go 新增 DeleteTx 方法，接收 *sql.Tx 参数
- [x] sys_role_mapper.go 新增 DeleteByRoleIDTx 方法（SysRoleMenuMapper），接收 *sql.Tx 参数
- [x] sys_menu_mapper.go 新增 DeleteByIDTx 方法，接收 *sql.Tx 参数

## admin-service Service 事务保护
- [x] DeleteUser 在事务内执行 sys_user 软删 + sys_user_role 硬删
- [x] DeleteUser 任一步骤失败时 Rollback 并返回"系统异常"
- [x] DeleteUser 操作日志在事务提交后记录
- [x] DeleteRole 在事务内执行 sys_role 软删 + sys_role_menu 硬删
- [x] DeleteRole 任一步骤失败时 Rollback 并返回"系统异常"
- [x] DeleteRole 操作日志在事务提交后记录
- [x] DeleteMenu 在事务内执行 sys_menu + sys_role_menu 两步删除
- [x] DeleteMenu 任一步骤失败时 Rollback 并返回"系统异常"

## 编译与文档
- [x] `go build ./...` 编译通过，无错误（go vet ./... 类型检查通过）
- [x] CLAUDE.md 新增"删除操作事务规范"说明章节
- [x] 不影响其他正常业务（原非 Tx 方法保持不变，非删除接口无改动）
