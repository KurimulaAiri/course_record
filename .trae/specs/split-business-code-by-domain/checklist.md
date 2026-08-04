# Checklist

## P0：超 1500 行文件拆分验证

### Task 1: admin_business_mapper.go 拆分验证

#### 文件创建验证
- [ ] `institution_mapper.go` 存在且包含 `AdminInstitutionRow` 类型 + 4 个机构方法
- [ ] `student_mapper.go` 存在且包含 `AdminStudentRow`/`AdminParentInfoRow` 类型 + 5 个学生方法
- [ ] `teacher_mapper.go` 存在且包含 `AdminTeacherRow` 类型 + 5 个教师方法
- [ ] `course_mapper.go` 存在且包含 `AdminCourseRow` 类型 + 4 个课程方法
- [ ] `class_mapper.go` 存在且包含 `AdminClassRow` 类型 + 9 个班级方法
- [ ] `class_schedule_mapper.go` 存在且包含 `AdminClassScheduleRow` 类型 + 3 个课表方法
- [ ] `course_record_mapper.go` 存在且包含 `AdminCourseRecordRow` 类型 + 4 个课时记录方法
- [ ] `record_mapper.go` 存在且包含 `AdminRecordRow` 类型 + 2 个上课记录方法
- [ ] `mini_menu_mapper.go` 存在且包含 `AdminMiniMenuRow` 类型 + 8 个小程序菜单方法
- [ ] `user_auth_mapper.go` 存在且包含 `AdminUserAuthRow` 类型 + 7 个用户认证方法

#### 主文件验证
- [ ] `admin_business_mapper.go` 仅保留：包声明/import、`AdminBusinessMapper` 结构体、`NewAdminBusinessMapper`、`scanInstitution`、`formatTimeSQL`/`formatDateSQL`/`formatTimePart`
- [ ] 主文件行数不超过 200 行

#### 跨文件依赖验证
- [ ] `class_mapper.go` 的 `SelectStudentsByClassID` 返回 `[]*AdminStudentRow`（通过包名访问 student_mapper.go 的类型）
- [ ] 所有方法接收者仍为 `*AdminBusinessMapper`

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./admin-service/...` 无警告

### Task 2: auth_service.go 拆分验证

#### 文件创建验证
- [ ] `auth_service_login.go` 存在且包含 `LoginByPwdRequest` 类型 + 6 个登录/Token 方法
- [ ] `auth_service_register.go` 存在且包含 `RegisterRequest` 类型 + `Register` 方法
- [ ] `auth_service_bind.go` 存在且包含 5 个绑定类型 + 绑定常量 + 2 个包级函数 + 13 个绑定方法
- [ ] `auth_service_subscribe.go` 存在且包含 `RecordSubscribeRequest` 类型 + 订阅/用户信息方法

#### 主文件验证
- [ ] `auth_service.go` 仅保留：包声明/import、常量块、6 个 VO 类型、`AuthService` 结构体、`NewAuthService`
- [ ] 主文件行数不超过 250 行

#### 跨文件依赖验证
- [ ] 所有方法接收者仍为 `*AuthService`
- [ ] `generateBindCode`/`generateBindToken` 包级函数在 `auth_service_bind.go` 中

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./auth-service/...` 无警告

### Task 3: business-service/mapper.go 拆分验证

#### 文件创建验证
- [ ] `institution_mapper.go` 存在且包含 `InstitutionMapper` 结构体 + 5 个方法（含 `UpdateByID`）
- [ ] `student_mapper.go` 存在且包含 `StudentMapper` 结构体 + `StudentQueryParams`/`StudentCourseDTO` 类型 + 所有方法
- [ ] `teacher_mapper.go` 存在且包含 `TeacherMapper` 结构体 + `TeacherDTO` 类型 + `teacherSelectColumns` 常量 + `scanTeacherDTO`/`scanTeacherDTOList` + 所有方法
- [ ] `parent_mapper.go` 存在且包含 `ParentMapper` 结构体 + 3 个方法
- [ ] `parent_student_mapper.go` 存在且包含 `ParentStudentMapper` 结构体 + `ParentStudentInfoDTO` 类型 + 所有方法
- [ ] `user_mapper.go` 存在且包含 `UserMapper` 结构体 + 2 个方法
- [ ] `user_auth_mapper.go` 存在且包含 `UserAuthMapper` 结构体 + 5 个方法
- [ ] `user_platform_mapper.go` 存在且包含 `UserPlatformMapper` 结构体 + 1 个方法
- [ ] `wx_subscribe_mapper.go` 存在且包含 `WxStudentSubscribeMapper` + `WxSubscribeRecordMapper` 结构体及其方法

#### 主文件验证
- [ ] `mapper.go` 仅保留包声明和 import（或删除该文件，import 分散到各业务文件）

#### 跨文件依赖验证
- [ ] `InstitutionMapper.UpdateByID` 已从 teacher 块归并到 institution_mapper.go
- [ ] `StudentMapper` 的扩展方法（`SelectByClassID` 等）已归并到 student_mapper.go
- [ ] 导出类型 `StudentQueryParams`/`TeacherDTO`/`StudentCourseDTO`/`ParentStudentInfoDTO` 可被 Service 层访问

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./business-service/...` 无警告

## P1：1000-1500 行文件拆分验证

### Task 4: business-service/service.go 拆分验证

#### 文件创建验证
- [ ] `institution_service.go` 存在且包含 `InstitutionService` 结构体 + `NewInstitutionService` + 5 个方法
- [ ] `student_service.go` 存在且包含 `StudentService` 结构体 + `NewStudentService` + `InsertStudentVO`/`UpdateStudentVO` 类型 + 所有方法
- [ ] `teacher_service.go` 存在且包含 `TeacherService` 结构体 + `NewTeacherService` + `InsertTeacherVO`/`UpdateTeacherVO` 类型 + 所有方法

#### 主文件验证
- [ ] `service.go` 保留：包声明/import、`QueryInstitutionVO`/`QueryStudentVO`/`QueryTeacherVO`/`UpdateResultVO` 类型、所有 VO 转换函数、VO 类型定义
- [ ] 主文件行数不超过 400 行

#### 跨文件依赖验证
- [ ] `UpdateResultVO` 被多个 Service 方法引用，保留在主文件
- [ ] VO 转换函数（如 `ToStudentVOFromDTO`）依赖 `mapper.StudentCourseDTO`，通过包名访问

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./business-service/...` 无警告

### Task 5: business-service/handler.go 拆分验证

#### 文件创建验证
- [ ] `institution_handler.go` 存在且包含 5 个机构 Handler 方法
- [ ] `student_handler.go` 存在且包含 10 个学生 Handler 方法
- [ ] `teacher_handler.go` 存在且包含 5 个教师 Handler 方法
- [ ] `class_handler.go` 存在且包含 8 个班级 Handler 方法
- [ ] `class_schedule_handler.go` 存在且包含 5 个课表 Handler 方法
- [ ] `course_handler.go` 存在且包含 4 个课程 Handler 方法
- [ ] `course_record_handler.go` 存在且包含 8 个课时记录 Handler 方法
- [ ] `record_handler.go` 存在且包含 6 个上课记录 Handler 方法

#### 主文件验证
- [ ] `handler.go` 保留：包声明/import、`BusinessHandler` 结构体、`NewBusinessHandler`、`RegisterRoutes`、`readBody`/`writeResponse`/`parseInt64`
- [ ] 主文件行数不超过 200 行
- [ ] `RegisterRoutes` 中所有 38 条路由注册无遗漏

#### 跨文件依赖验证
- [ ] 所有 Handler 方法接收者仍为 `*BusinessHandler`
- [ ] 新文件复用主文件的 `readBody`/`writeResponse`/`parseInt64` 辅助函数

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./business-service/...` 无警告

### Task 6: admin_business_service.go 拆分验证

#### 文件创建验证
- [ ] `institution_service.go` 存在且包含 3 个机构 DTO + 3 个方法
- [ ] `student_service.go` 存在且包含 3 个学生 DTO + 3 个方法
- [ ] `teacher_service.go` 存在且包含 3 个教师 DTO + 3 个方法
- [ ] `course_service.go` 存在且包含 3 个课程 DTO + 3 个方法
- [ ] `class_service.go` 存在且包含 4 个班级 DTO + 6 个方法
- [ ] `class_schedule_service.go` 存在且包含 2 个课表 DTO + 2 个方法
- [ ] `course_record_service.go` 存在且包含 3 个课时记录 DTO + 3 个方法
- [ ] `record_service.go` 存在且包含 2 个上课记录 DTO + 2 个方法
- [ ] `mini_menu_service.go` 存在且包含 2 个小程序菜单 DTO + 4 个方法

#### 主文件验证
- [ ] `admin_business_service.go` 保留：包声明/import、`AdminBusinessService` 结构体、`NewAdminBusinessService`、`recordLog` 辅助方法
- [ ] 主文件行数不超过 100 行

#### 跨文件依赖验证
- [ ] 所有方法接收者仍为 `*AdminBusinessService`
- [ ] `recordLog` 方法被各业务对象文件的方法调用（通过接收者访问）

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./admin-service/...` 无警告

## P2：600-1000 行文件拆分验证

### Task 7: admin_handler.go 拆分验证

#### 文件创建验证
- [ ] `user_handler.go` 存在且包含 8 个用户管理方法
- [ ] `role_handler.go` 存在且包含 7 个角色管理方法
- [ ] `menu_handler.go` 存在且包含 6 个菜单管理方法
- [ ] `operation_log_handler.go` 存在且包含 3 个操作日志方法

#### 主文件验证
- [ ] `admin_handler.go` 保留：包声明/import、`AdminHandler` 结构体、`NewAdminHandler`、`RegisterRoutes`、辅助函数、`Login`/`RefreshToken`/`GetPublicKey`
- [ ] `RegisterRoutes` 中所有 59 条路由注册无遗漏

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./admin-service/...` 无警告

### Task 8: business_handler.go 拆分验证

#### 文件创建验证
- [ ] `business_institution_handler.go` 存在且包含 3 个机构方法
- [ ] `business_student_handler.go` 存在且包含 3 个学生方法
- [ ] `business_teacher_handler.go` 存在且包含 3 个教师方法
- [ ] `business_course_handler.go` 存在且包含 3 个课程方法
- [ ] `business_class_handler.go` 存在且包含 6 个班级方法
- [ ] `business_class_schedule_handler.go` 存在且包含 2 个课表方法
- [ ] `business_course_record_handler.go` 存在且包含 3 个课时记录方法
- [ ] `business_record_handler.go` 存在且包含 2 个上课记录方法
- [ ] `business_mini_menu_handler.go` 存在且包含 4 个小程序菜单方法
- [ ] `business_teacher_auth_handler.go` 存在且包含 4 个教师账号方法
- [ ] `business_dashboard_handler.go` 存在且包含 3 个仪表盘方法
- [ ] `business_config_handler.go` 存在且包含 4 个系统配置方法

#### 跨文件依赖验证
- [ ] 新文件复用 `admin_handler.go` 的 `AdminHandler` 结构体和 `readBody`/`writeResponse` 辅助函数
- [ ] 所有方法接收者仍为 `*AdminHandler`

#### 编译验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./admin-service/...` 无警告

## 最终验证

### Task 9: 全量验证
- [ ] `go build ./...` 编译通过
- [ ] `go vet ./...` 无警告
- [ ] `gofmt -l .` 无格式问题
- [ ] 所有新文件包含包注释和业务对象说明注释
- [ ] 无业务逻辑变更（仅文件结构重组）
- [ ] 无方法签名变更
- [ ] 无 API 路由遗漏

### 文件行数验证
- [ ] 拆分后无单个文件超过 500 行（除主文件保留公共部分外）
- [ ] 主文件行数合理（admin_business_mapper.go < 200 行，auth_service.go < 250 行等）

### 代码规范验证
- [ ] 所有新文件包含 `// Package xxx` 包注释
- [ ] 所有新文件包含业务对象说明注释
- [ ] gofmt 格式规范
