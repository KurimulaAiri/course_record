# Tasks

> **实施原则**：纯文件拆分，不修改任何业务逻辑、方法签名、API 行为。每个阶段完成后执行 `go build ./...` 和 `go vet ./...` 验证。
>
> **并行策略**：P0 三个文件互相独立，可并行拆分。P1 三个文件互相独立，可并行。P2 两个文件可并行。

## P0：超 1500 行文件拆分（3 项，优先）

- [x] Task 1: 拆分 admin_business_mapper.go (2169 行)
  - [ ] SubTask 1.1: 创建 `institution_mapper.go`，迁移 `AdminInstitutionRow` 类型 + `ListInstitutions`/`SelectInstitutionByID`/`InsertInstitution`/`UpdateInstitution` 方法
  - [ ] SubTask 1.2: 创建 `student_mapper.go`，迁移 `AdminStudentRow`/`AdminParentInfoRow` 类型 + `ListStudents`/`SelectStudentByID`/`InsertStudent`/`UpdateStudent`/`SelectParentInfoByStudentID` 方法
  - [ ] SubTask 1.3: 创建 `teacher_mapper.go`，迁移 `AdminTeacherRow` 类型 + `ListTeachers`/`SelectTeacherByID`/`InsertTeacher`/`UpdateTeacher`/`CountClassTeacherByTeacherID` 方法
  - [ ] SubTask 1.4: 创建 `course_mapper.go`，迁移 `AdminCourseRow` 类型 + `ListCourses`/`SelectCourseByID`/`InsertCourse`/`UpdateCourse` 方法
  - [ ] SubTask 1.5: 创建 `class_mapper.go`，迁移 `AdminClassRow` 类型 + `ListClasses`/`SelectClassTeachers`/`SelectClassByID`/`InsertClass`/`UpdateClass`/`InsertClassStudent`/`DeleteClassStudent`/`UpdateClassStudentCount`/`SelectStudentsByClassID` 方法（注意：`SelectStudentsByClassID` 依赖 `AdminStudentRow`，通过包名访问）
  - [ ] SubTask 1.6: 创建 `class_schedule_mapper.go`，迁移 `AdminClassScheduleRow` 类型 + `ListClassSchedules`/`SelectClassScheduleByID`/`UpdateClassSchedule` 方法
  - [ ] SubTask 1.7: 创建 `course_record_mapper.go`，迁移 `AdminCourseRecordRow` 类型 + `ListCourseRecords`/`SelectCourseRecordByID`/`InsertCourseRecord`/`UpdateCourseRecord` 方法
  - [ ] SubTask 1.8: 创建 `record_mapper.go`，迁移 `AdminRecordRow` 类型 + `ListRecords`/`InsertRecord` 方法
  - [ ] SubTask 1.9: 创建 `mini_menu_mapper.go`，迁移 `AdminMiniMenuRow` 类型 + `ListMiniMenus`/`SelectMiniMenuByID`/`InsertMiniMenu`/`UpdateMiniMenu`/`DeleteMiniMenu`/`SelectRoleIDsByMenuID`/`DeleteRoleMenuByMenuID`/`InsertRoleMenu` 方法
  - [ ] SubTask 1.10: 创建 `user_auth_mapper.go`，迁移 `AdminUserAuthRow` 类型 + `SelectUserAuthByTeacherID`/`ExistsUserAuthByInstitutionAndAccount`/`InsertUserAuth`/`UpdateUserAuthAccount`/`UpdateUserAuthPassword`/`SelectUserInstitutionID`/`UpdateTeacherInstitutionAdmin` 方法
  - [ ] SubTask 1.11: 修改主文件 `admin_business_mapper.go`，仅保留：包声明/import、`AdminBusinessMapper` 结构体、`NewAdminBusinessMapper` 构造函数、`scanInstitution` 辅助函数、`formatTimeSQL`/`formatDateSQL`/`formatTimePart` 辅助函数
  - [ ] SubTask 1.12: 每个新文件添加包注释和业务对象说明注释，删除迁移代码的主文件重复内容
  - [ ] SubTask 1.13: 验证 `go build ./...` 和 `go vet ./...` 通过

- [x] Task 2: 拆分 auth_service.go (1895 行)
  - [ ] SubTask 2.1: 创建 `auth_service_login.go`，迁移 `LoginByPwdRequest` 类型 + `GetOpenId`/`WxLogin`/`LoginByPwd`/`LoginByToken`/`RefreshAccessToken`/`Logout` 方法
  - [ ] SubTask 2.2: 创建 `auth_service_register.go`，迁移 `RegisterRequest` 类型 + `Register` 方法
  - [ ] SubTask 2.3: 创建 `auth_service_bind.go`，迁移绑定相关：`BindTokenInfo`/`BindQrcodeResponse`/`BindInfoResponse`/`BindStatusResponse`/`BindResultResponse` 类型 + 绑定常量 + `generateBindCode`/`generateBindToken` 包级函数 + `setBindInfo`/`getBindInfoByToken`/`getBindInfoByCode`/`deleteBindInfo`/`GenerateBindQrcode`/`GenerateSubscribeQrcode`/`GetBindInfo`/`GetBindInfoByCode`/`buildBindInfoResponse`/`CheckBindStatus`/`ConfirmBind`/`BindByCode`/`doBind` 方法
  - [ ] SubTask 2.4: 创建 `auth_service_subscribe.go`，迁移 `RecordSubscribeRequest` 类型 + `RecordSubscribe`/`GetSubscribeStatus`/`GetFullUserInfo`/`loadFullUserInfoFromDB`/`InvalidateUserCache`/`GetUserAuthByTeacherID`/`GetUserInfoByUserID`/`doSubscribeOnly`/`saveWxStudentSubscribe`/`TestSendSubscribe` 方法
  - [ ] SubTask 2.5: 修改主文件 `auth_service.go`，仅保留：包声明/import、常量块、6 个 VO 类型（LoginVO/UserVO/ParentIdentityVO/TeacherIdentityVO/AdminVO/RegisterVO）、`AuthService` 结构体、`NewAuthService` 构造函数
  - [ ] SubTask 2.6: 每个新文件添加包注释和业务模块说明注释
  - [ ] SubTask 2.7: 验证 `go build ./...` 和 `go vet ./...` 通过

- [x] Task 3: 拆分 business-service/mapper.go (1534 行)
  - [ ] SubTask 3.1: 创建 `institution_mapper.go`，迁移 `InstitutionMapper` 结构体 + `NewInstitutionMapper` + `SelectByID`/`SelectByOpenID`/`SelectByCode`/`SelectByStudentID`/`UpdateByID` 方法（注意：`UpdateByID` 原在 teacher 块后，需归并）
  - [ ] SubTask 3.2: 创建 `student_mapper.go`，迁移 `StudentMapper` 结构体 + `NewStudentMapper` + `StudentQueryParams` 类型 + `StudentCourseDTO` 类型 + 所有 StudentMapper 方法（含 `SelectByClassID`/`SelectByCourseID` 等扩展方法）
  - [ ] SubTask 3.3: 创建 `teacher_mapper.go`，迁移 `TeacherMapper` 结构体 + `NewTeacherMapper` + `TeacherDTO` 类型 + `teacherSelectColumns` 常量 + `scanTeacherDTO`/`scanTeacherDTOList` 辅助函数 + 所有 TeacherMapper 方法
  - [ ] SubTask 3.4: 创建 `parent_mapper.go`，迁移 `ParentMapper` 结构体 + `NewParentMapper` + `SelectByID`/`DeleteByID`/`ResetUnbound` 方法
  - [ ] SubTask 3.5: 创建 `parent_student_mapper.go`，迁移 `ParentStudentMapper` 结构体 + `NewParentStudentMapper` + `ParentStudentInfoDTO` 类型 + 所有方法
  - [ ] SubTask 3.6: 创建 `user_mapper.go`，迁移 `UserMapper` 结构体 + `NewUserMapper` + `Insert`/`DeleteByID` 方法
  - [ ] SubTask 3.7: 创建 `user_auth_mapper.go`，迁移 `UserAuthMapper` 结构体 + `NewUserAuthMapper` + 所有方法
  - [ ] SubTask 3.8: 创建 `user_platform_mapper.go`，迁移 `UserPlatformMapper` 结构体 + `NewUserPlatformMapper` + `SelectOpenIDsByUserID` 方法
  - [ ] SubTask 3.9: 创建 `wx_subscribe_mapper.go`，迁移 `WxStudentSubscribeMapper` + `WxSubscribeRecordMapper` 结构体及其所有方法（含 `NewWxStudentSubscribeMapper`/`NewWxSubscribeRecordMapper`）
  - [ ] SubTask 3.10: 修改主文件 `mapper.go`，仅保留包声明和 import（如无公共内容）
  - [ ] SubTask 3.11: 每个新文件添加包注释和业务对象说明注释
  - [ ] SubTask 3.12: 验证 `go build ./...` 和 `go vet ./...` 通过

## P1：1000-1500 行文件拆分（3 项）

- [x] Task 4: 拆分 business-service/service.go (1181 行)
  - [ ] SubTask 4.1: 创建 `institution_service.go`，迁移 `InstitutionService` 结构体 + `NewInstitutionService` + `GetInstitutionByID`/`GetInstitutionByOpenID`/`GetInstitutionByCode`/`GetInstitutionByStudentID`/`UpdateInstitution` 方法
  - [ ] SubTask 4.2: 创建 `student_service.go`，迁移 `StudentService` 结构体 + `NewStudentService` + `InsertStudentVO`/`UpdateStudentVO` 类型 + 所有 StudentService 方法（含 `toStudentVOListWithParents` 内部辅助）
  - [ ] SubTask 4.3: 创建 `teacher_service.go`，迁移 `TeacherService` 结构体 + `NewTeacherService` + `InsertTeacherVO`/`UpdateTeacherVO` 类型 + 所有 TeacherService 方法
  - [ ] SubTask 4.4: 修改主文件 `service.go`，保留：包声明/import、`QueryInstitutionVO`/`QueryStudentVO`/`QueryTeacherVO`/`UpdateResultVO` 类型、所有 VO 转换函数（`ToInstitutionVO`/`ToStudentVO` 等）、VO 类型定义（`InstitutionVO`/`StudentVO`/`ParentVO`/`TeacherVO`）
  - [ ] SubTask 4.5: 每个新文件添加包注释和业务对象说明注释
  - [ ] SubTask 4.6: 验证 `go build ./...` 和 `go vet ./...` 通过

- [x] Task 5: 拆分 business-service/handler.go (1146 行)
  - [ ] SubTask 5.1: 创建 `institution_handler.go`，迁移 5 个机构 Handler 方法
  - [ ] SubTask 5.2: 创建 `student_handler.go`，迁移 10 个学生 Handler 方法
  - [ ] SubTask 5.3: 创建 `teacher_handler.go`，迁移 5 个教师 Handler 方法
  - [ ] SubTask 5.4: 创建 `class_handler.go`，迁移 8 个班级 Handler 方法
  - [ ] SubTask 5.5: 创建 `class_schedule_handler.go`，迁移 5 个课表 Handler 方法
  - [ ] SubTask 5.6: 创建 `course_handler.go`，迁移 4 个课程 Handler 方法
  - [ ] SubTask 5.7: 创建 `course_record_handler.go`，迁移 8 个课时记录 Handler 方法
  - [ ] SubTask 5.8: 创建 `record_handler.go`，迁移 6 个上课记录 Handler 方法（含 `GetDeductDetail`/`DeleteCourseRecord`）
  - [ ] SubTask 5.9: 修改主文件 `handler.go`，保留：包声明/import、`BusinessHandler` 结构体、`NewBusinessHandler` 构造函数、`RegisterRoutes` 路由注册、`readBody`/`writeResponse`/`parseInt64` 辅助函数
  - [ ] SubTask 5.10: 每个新文件添加包注释和业务对象说明注释
  - [ ] SubTask 5.11: 验证 `go build ./...` 和 `go vet ./...` 通过

- [x] Task 6: 拆分 admin_business_service.go (1103 行)
  - [ ] SubTask 6.1: 创建 `institution_service.go`，迁移 `QueryInstitutionRequest`/`InsertInstitutionRequest`/`UpdateInstitutionRequest` DTO + `ListInstitutions`/`InsertInstitution`/`UpdateInstitution` 方法
  - [ ] SubTask 6.2: 创建 `student_service.go`，迁移学生相关 DTO + `ListStudents`/`InsertStudent`/`UpdateStudent` 方法
  - [ ] SubTask 6.3: 创建 `teacher_service.go`，迁移教师相关 DTO + `ListTeachers`/`InsertTeacher`/`UpdateTeacher` 方法
  - [ ] SubTask 6.4: 创建 `course_service.go`，迁移课程相关 DTO + `ListCourses`/`InsertCourse`/`UpdateCourse` 方法
  - [ ] SubTask 6.5: 创建 `class_service.go`，迁移班级相关 DTO + `ListClasses`/`InsertClass`/`UpdateClass`/`GetClassByID`/`AddStudentToClass`/`RemoveStudentFromClass` 方法
  - [ ] SubTask 6.6: 创建 `class_schedule_service.go`，迁移课表相关 DTO + `ListClassSchedules`/`UpdateClassSchedule` 方法
  - [ ] SubTask 6.7: 创建 `course_record_service.go`，迁移课时记录相关 DTO + `ListCourseRecords`/`InsertCourseRecord`/`UpdateCourseRecord` 方法
  - [ ] SubTask 6.8: 创建 `record_service.go`，迁移上课记录相关 DTO + `ListRecords`/`InsertRecord` 方法
  - [ ] SubTask 6.9: 创建 `mini_menu_service.go`，迁移小程序菜单相关 DTO + `ListMiniMenus`/`InsertMiniMenu`/`UpdateMiniMenu`/`DeleteMiniMenu` 方法
  - [ ] SubTask 6.10: 修改主文件 `admin_business_service.go`，保留：包声明/import、`AdminBusinessService` 结构体、`NewAdminBusinessService` 构造函数、`recordLog` 辅助方法
  - [ ] SubTask 6.11: 每个新文件添加包注释和业务对象说明注释
  - [ ] SubTask 6.12: 验证 `go build ./...` 和 `go vet ./...` 通过

## P2：600-1000 行文件拆分（2 项，可选）

- [x] Task 7: 拆分 admin_handler.go (697 行)
  - [ ] SubTask 7.1: 创建 `user_handler.go`，迁移 8 个用户管理 Handler 方法（GetUserInfo/GetUserList/GetUserByID/InsertUser/UpdateUser/DeleteUser/ResetPassword/GetUserRoles）
  - [ ] SubTask 7.2: 创建 `role_handler.go`，迁移 7 个角色管理 Handler 方法
  - [ ] SubTask 7.3: 创建 `menu_handler.go`，迁移 6 个菜单管理 Handler 方法
  - [ ] SubTask 7.4: 创建 `operation_log_handler.go`，迁移 3 个操作日志 Handler 方法
  - [ ] SubTask 7.5: 修改主文件 `admin_handler.go`，保留：包声明/import、`AdminHandler` 结构体、`NewAdminHandler` 构造函数、`RegisterRoutes`、`readBody`/`writeResponse` 辅助函数、`Login`/`RefreshToken`/`GetPublicKey` 方法
  - [ ] SubTask 7.6: 验证 `go build ./...` 和 `go vet ./...` 通过

- [x] Task 8: 拆分 business_handler.go (686 行)
  - [ ] SubTask 8.1: 创建 `business_institution_handler.go`，迁移 3 个机构 Handler 方法
  - [ ] SubTask 8.2: 创建 `business_student_handler.go`，迁移 3 个学生 Handler 方法
  - [ ] SubTask 8.3: 创建 `business_teacher_handler.go`，迁移 3 个教师 Handler 方法
  - [ ] SubTask 8.4: 创建 `business_course_handler.go`，迁移 3 个课程 Handler 方法
  - [ ] SubTask 8.5: 创建 `business_class_handler.go`，迁移 6 个班级 Handler 方法
  - [ ] SubTask 8.6: 创建 `business_class_schedule_handler.go`，迁移 2 个课表 Handler 方法
  - [ ] SubTask 8.7: 创建 `business_course_record_handler.go`，迁移 3 个课时记录 Handler 方法
  - [ ] SubTask 8.8: 创建 `business_record_handler.go`，迁移 2 个上课记录 Handler 方法
  - [ ] SubTask 8.9: 创建 `business_mini_menu_handler.go`，迁移 4 个小程序菜单 Handler 方法
  - [ ] SubTask 8.10: 创建 `business_teacher_auth_handler.go`，迁移 4 个教师账号 Handler 方法
  - [ ] SubTask 8.11: 创建 `business_dashboard_handler.go`，迁移 3 个仪表盘 Handler 方法
  - [ ] SubTask 8.12: 创建 `business_config_handler.go`，迁移 4 个系统配置 Handler 方法
  - [ ] SubTask 8.13: 修改主文件 `business_handler.go`，仅保留包声明和 import（复用 admin_handler 公共部分）
  - [ ] SubTask 8.14: 验证 `go build ./...` 和 `go vet ./...` 通过

## 最终验证

- [x] Task 9: 全量编译与格式化验证
  - [ ] SubTask 9.1: `go build ./...` 编译通过
  - [ ] SubTask 9.2: `go vet ./...` 无警告
  - [ ] SubTask 9.3: `gofmt -l .` 无格式问题
  - [ ] SubTask 9.4: 验证所有 API 路由注册无遗漏（启动服务测试或检查 RegisterRoutes）

# Task Dependencies
- Task 1/2/3（P0）互相独立，可并行
- Task 4/5/6（P1）互相独立，可并行，且与 P0 无依赖
- Task 7/8（P2）互相独立，可并行
- Task 9 依赖所有前置 Task 完成

## 并行执行建议
- **第一批并行**：Task 1 + Task 2 + Task 3（P0）
- **第二批并行**：Task 4 + Task 5 + Task 6（P1）
- **第三批并行**：Task 7 + Task 8（P2）
- **最终**：Task 9（全量验证）
