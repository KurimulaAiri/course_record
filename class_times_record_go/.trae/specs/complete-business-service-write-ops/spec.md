# 完成 business-service 写操作接口 Spec

## Why
Go 后端 business-service 的查询接口和大部分写操作 service/mapper 已实现，但存在三处缺口导致代码无法编译并缺少 HTTP 路由：
1. `record_service.go`（3 个上课记录接口）尚未创建
2. `handler.go` 仅注册了 8 个查询路由，38 个写操作接口的 Handler 和路由未注册
3. `main.go` 仍使用旧的 service 构造签名（如 `NewStudentService(studentMapper)` 单参数），与新签名（6 参数）不匹配，依赖注入未完成

## What Changes
- 新增 `business-service/internal/service/record_service.go`：实现 3 个上课记录接口（new_get/add/add_all）
- 重写 `business-service/internal/handler/handler.go`：注册全部 38 个写操作接口的 Handler 方法和路由
- 重写 `business-service/main.go`：初始化所有新增 Mapper 和 Service，注入到 Handler
- 执行 `go build ./...` 验证编译通过

## Impact
- Affected code:
  - `business-service/internal/service/record_service.go`（新增）
  - `business-service/internal/handler/handler.go`（重写）
  - `business-service/main.go`（重写）
- 不修改已完成的 service/mapper/entity 文件
- 不修改前端代码

## ADDED Requirements

### Requirement: 上课记录服务（RecordService）
系统 SHALL 提供 3 个上课记录接口，对齐 Java RecordController：
- `POST /record/new_get`：按机构/学生/课程名称/记录类型分页查询上课记录
- `POST /record/add`：新增单条上课记录
- `POST /record/add_all`：批量新增上课记录（同时更新课卡剩余课时）

#### Scenario: 查询上课记录列表
- WHEN 前端 POST `/biz/record/new_get` 携带 institutionId/currentPage/pageSize
- THEN 返回 `data.records`（数组）+ `data.total`

#### Scenario: 新增单条上课记录
- WHEN 前端 POST `/biz/record/add` 携带 courseRecordId/recordTime/recordType/recordChange
- THEN 返回成功消息

### Requirement: 全部 38 个写操作接口路由注册
系统 SHALL 在 handler.go 中注册全部 38 个写操作接口的 HTTP 路由，覆盖以下模块：
- 学生（6 个写操作）：insert/update/unbind/cancel_subscribe/get_by_class_id/get_by_course_id
- 教师（3 个写操作）：insert/update/delete
- 机构（1 个写操作）：update
- 班级（8 个写操作）：insert/update/add_student/remove_student/get_by_student_id/get_by_teacher_id/get_by_institution_id/get_by_id
- 课表（5 个写操作）：get_by_class_id/get_by_institution_id/get_by_teacher_id/get_by_id/update_by_id
- 课程（4 个写操作）：get_by_institution_id/get_by_student_id/insert/update
- 课卡记录（9 个写操作）：new_get/get_by_student_id/get_by_institution_id/insert/update/deduct_by_student_id/deduct_by_course_id/deduct_by_class_id/deduct-detail
- 上课记录（3 个写操作）：new_get/add/add_all

### Requirement: 依赖注入完整初始化
main.go SHALL 初始化所有 Mapper 和 Service，使用新的构造函数签名注入到 BusinessHandler。

## MODIFIED Requirements

### Requirement: BusinessHandler 构造函数
BusinessHandler 构造函数 SHALL 接受全部 7 个 Service（institution/student/teacher/class/classSchedule/course/courseRecord/record），并注册全部路由。

### Requirement: main.go 依赖装配
main.go SHALL：
1. 创建所有 Mapper 实例（含新增的 ClassMapper/ClassStudentMapper/ClassTeacherMapper/ClassScheduleMapper/CourseMapper/CourseRecordMapper/RecordMapper/ParentStudentMapper/ParentMapper/UserAuthMapper/UserMapper/WxStudentSubscribeMapper/WxSubscribeRecordMapper/UserPlatformMapper）
2. 从环境变量读取 SM2 私钥（SM2_PRIVATE_KEY），传入 TeacherService
3. 使用新签名创建所有 Service
4. 注入到 BusinessHandler 并注册路由
