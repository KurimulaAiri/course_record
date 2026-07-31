# 修复前后端报文一致性问题 Spec

## Why

Go 后端 VO 结构与前端类型定义存在系统性不一致，分为两类问题：
1. **字段缺失/命名不一致**（"缺斤少两"）：Go VO 缺少前端期望的字段，或字段名拼写不同
2. **层级格式不同**：前端期望嵌套对象（如 `course.institution.institutionName`、`teachers: [{id, name}]`），Go 返回扁平字段（如 `institutionId`、`teacherId`），导致前端按嵌套路径访问得到 `undefined`，渲染失败

共发现 **11 个问题**（9 个 P0 阻断小程序渲染 + 2 个 P1 影响 Admin 仪表盘）。

## What Changes

### P0 致命问题（9 项，阻断小程序核心流程）

#### auth-service 绑定流程（3 项）
- **修复 `BindQrcodeVO`**：字段从 `code/qrContent/isSubscribe` 改为 `qrcode/token/bindCode`
- **修复 `BindInfoVO`**：补充 `relation/isPrimary/parentName/parentPhone` 4 个字段，`isSubscribe` 重命名为 `subscribeOnly`
- **修复 `BindStatusVO`**：结构从 `{status, studentInfo, alreadyBound}` 改为 `{alreadyBound, hasAccount}`

#### business-service 列表页层级格式（5 项）
- **修复 `ClassVO`**：`teacherId/teacherUsername`（单值）→ `teachers: []TeacherVO`（对象数组）；补充 `courseRecord: CourseRecordVO`（嵌套对象）、`scheduleList?: []ScheduleVO`
- **修复 `ClassScheduleVO`**：`teacherId/teacherName`（单值）→ `teachers: []TeacherVO`（对象数组）；补充 `classroom`、`color?` 字段
- **修复 `CourseVO`**：`institutionId`（扁平）→ `institution: InstitutionVO`（嵌套对象）；`courseRecordId/courseRestTime/courseTotalTime/expireTimeStr`（扁平）→ `currentStudentCourseRecord?: CourseRecordVO`（嵌套对象）
- **修复 `CourseRecordVO`**：`courseId/courseName/courseType`（扁平）→ `course: CourseVO`（嵌套对象）；补充 `permissionType`、`expireStatus` 字段
- **修复 `RecordVO`**：扁平字段 → 4 个嵌套对象 `courseRecord: CourseRecordVO`、`student: StudentVO`、`course: CourseVO`、`operatorTeacher: TeacherVO`

#### business-service 扣费详情（1 项）
- **修复 `DeductDetailVO`**：补充 12 个缺失字段（`courseRecordId/courseName/courseType/studentName/deductCount/courseTotalTime/expireTime/classId/className/scheduleDesc/teacherId/teacherName/expireStatus`）；`recordTimeStr` → `recordTime`、`recordRemark` → `remark`

### P1 重要问题（2 项，影响 Admin 仪表盘）
- **修复 `DashboardTrendRow`**：JSON tag `labels` → `months`
- **修复 `InstitutionStatRow`**：JSON tag `id` → `institutionId`

## Impact

- Affected specs: `go-backend-interface-completion`（阶段七 Task 30-38 的超集）
- Affected code:
  - `class_times_record_go/auth-service/internal/service/auth_service.go` — 绑定流程 3 个 VO
  - `class_times_record_go/business-service/internal/service/class_service.go` — ClassVO 层级重构
  - `class_times_record_go/business-service/internal/service/class_schedule_service.go` — ClassScheduleVO 层级重构
  - `class_times_record_go/business-service/internal/service/course_service.go` — CourseVO 层级重构
  - `class_times_record_go/business-service/internal/service/course_record_service.go` — CourseRecordVO 层级重构 + DeductDetailVO 补字段
  - `class_times_record_go/business-service/internal/service/record_service.go` — RecordVO 层级重构
  - `class_times_record_go/business-service/internal/mapper/*.go` — 补充 JOIN 查询填充嵌套对象
  - `class_times_record_go/admin-service/internal/mapper/dashboard_mapper.go` — 2 个字段 JSON tag

## ADDED Requirements

### Requirement: 绑定二维码响应字段对齐

系统 SHALL 在 `POST /auth/auth/generate_bind_qrcode` 和 `POST /auth/auth/generate_subscribe_qrcode` 接口返回 `BindQrcodeResponse` 结构。

#### Scenario: 返回二维码响应
- **WHEN** 教师端调用绑定二维码生成接口
- **THEN** 响应 `data` 包含 `qrcode`（二维码 base64）、`token`（绑定 token）、`bindCode`（6 位绑定码）
- **AND** 不再返回 `code/qrContent/isSubscribe` 字段

### Requirement: 绑定信息查询响应字段对齐

系统 SHALL 在 `GET /auth/auth/get_bind_info` 和 `GET /auth/auth/get_bind_info_by_code` 接口返回完整的 `BindInfoResponse` 结构。

#### Scenario: 返回完整绑定信息
- **WHEN** 家长查询绑定信息
- **THEN** 响应包含 `studentId/studentName/sex/institutionName/relation/isPrimary/subscribeOnly/parentName?/parentPhone?` 共 9 个字段
- **AND** `subscribeOnly` 字段名不使用 `isSubscribe`
- **AND** `relation` 和 `isPrimary` 从 `c_parent_student` 表查询填充
- **AND** `parentName` 和 `parentPhone` 从 `c_parent` 表查询填充

### Requirement: 绑定状态检查响应结构对齐

系统 SHALL 在 `GET /auth/auth/check_bind_status` 接口返回简洁的 `BindStatusResponse` 结构。

#### Scenario: 返回绑定状态
- **WHEN** 家长检查绑定状态
- **THEN** 响应 `data` 为 `{ alreadyBound: boolean, hasAccount: boolean }`
- **AND** `hasAccount` 表示该学生是否已有任意家长账号（查询 `c_parent_student` 关联 `c_user_platform`）
- **AND** 不再返回 `status` 和 `studentInfo` 字段

### Requirement: 班级列表嵌套教师对象数组

系统 SHALL 在班级查询接口返回 `teachers: []TeacherVO` 对象数组，而非扁平的 `teacherId/teacherUsername` 单值。

#### Scenario: 班级有多个教师
- **WHEN** 查询班级列表，某班级关联 2 名教师
- **THEN** `teachers` 字段为 `[{teacherId: 1, teacherUsername: "张老师"}, {teacherId: 2, teacherUsername: "李老师"}]`
- **AND** 不再返回顶层 `teacherId/teacherUsername` 字段

#### Scenario: 班级无教师
- **WHEN** 查询班级列表，某班级未关联教师
- **THEN** `teachers` 字段为空数组 `[]`

### Requirement: 班级嵌套课卡和课表对象

系统 SHALL 在班级查询接口返回嵌套的 `courseRecord: CourseRecordVO` 和 `scheduleList?: []ScheduleVO` 对象。

#### Scenario: 学生维度查询班级
- **WHEN** 学生查询自己的班级列表
- **THEN** 每个班级包含 `courseRecord` 对象（含 `courseRestTime/courseTotalTime/expireTime` 等）
- **AND** 包含 `scheduleList` 数组（该班级的课表列表）

#### Scenario: 机构/教师维度查询班级
- **WHEN** 机构或教师查询班级列表
- **THEN** `courseRecord` 为 `null`（无法确定具体学生）
- **AND** `scheduleList` 为该班级的课表列表

### Requirement: 课表嵌套教师对象数组

系统 SHALL 在课表查询接口返回 `teachers: []TeacherVO` 对象数组，而非扁平的 `teacherId/teacherName` 单值。同时返回 `classroom` 和 `color` 字段。

#### Scenario: 课表有关联教师
- **WHEN** 查询课表列表
- **THEN** 每条课表包含 `teachers` 数组、`classroom` 字符串、`color` 字符串

### Requirement: 课程嵌套机构和课卡对象

系统 SHALL 在课程查询接口返回嵌套的 `institution: InstitutionVO` 和 `currentStudentCourseRecord?: CourseRecordVO` 对象。

#### Scenario: 机构维度查询课程
- **WHEN** 机构查询自己的课程列表
- **THEN** 每个课程包含 `institution` 对象（含 `id/institutionName/institutionCode` 等）
- **AND** `currentStudentCourseRecord` 为 `null`（无学生上下文）

#### Scenario: 学生维度查询课程
- **WHEN** 学生查询自己的课程列表
- **THEN** 每个课程包含 `institution` 对象
- **AND** 包含 `currentStudentCourseRecord` 对象（该学生在该课程的课卡信息）

### Requirement: 课卡嵌套课程对象

系统 SHALL 在课卡查询接口返回嵌套的 `course: CourseVO` 对象，而非扁平的 `courseId/courseName/courseType`。同时返回 `permissionType` 和 `expireStatus` 字段。

#### Scenario: 查询课卡列表
- **WHEN** 查询学生的课卡列表
- **THEN** 每条课卡包含 `course` 对象（含 `id/courseName/courseType/isAvailable`）
- **AND** 包含 `permissionType`（权限类型）和 `expireStatus`（到期状态：expired/warning/valid）

### Requirement: 上课记录嵌套对象

系统 SHALL 在上课记录查询接口返回嵌套的 `courseRecord: CourseRecordVO`、`student: StudentVO`、`course: CourseVO`、`operatorTeacher: TeacherVO` 四个对象。

#### Scenario: 查询上课记录列表
- **WHEN** 查询上课记录
- **THEN** 每条记录包含 4 个嵌套对象
- **AND** 不再返回扁平的 `courseId/studentId/courseName/studentName/teacherName` 字段

### Requirement: 扣费详情完整字段

系统 SHALL 在 `GET /biz/course_record/deduct-detail` 接口返回完整的扣费详情，包含 21 个字段。

#### Scenario: 返回完整扣费详情
- **WHEN** 查询扣费详情
- **THEN** 响应包含 `courseRecordId/recordId/courseId/courseName/courseType/studentId/studentName/deductCount/courseRestTime/restTimeAfterDeduct/courseTotalTime/expireTime/recordTime/remark/classId/className/scheduleDesc/teacherId/teacherName/deductMode/expireStatus` 共 21 个字段
- **AND** `recordTime`（非 `recordTimeStr`）、`remark`（非 `recordRemark`）
- **AND** `expireStatus` 取值为 `normal`/`expired`/`warning`
- **AND** 通过 JOIN 查询 c_course/c_student/c_class/c_class_schedule/c_teacher 表填充关联字段

### Requirement: 仪表盘趋势字段名对齐

系统 SHALL 在 `POST /admin/dashboard/trend` 接口返回 `months` 字段名。

#### Scenario: 趋势数据返回
- **WHEN** 管理员查询趋势数据
- **THEN** 响应 `data.months` 为时间刻度字符串数组（非 `labels`）

### Requirement: 机构统计字段名对齐

系统 SHALL 在 `POST /admin/dashboard/institution/stats` 接口返回 `institutionId` 字段名。

#### Scenario: 机构统计返回
- **WHEN** 管理员查询机构统计
- **THEN** 每条记录包含 `institutionId` 字段（非 `id`）

## MODIFIED Requirements

### Requirement: VO 层级格式规范

所有列表查询接口的 VO SHALL 使用嵌套对象结构，而非扁平字段。

- **教师信息**：使用 `teachers: []TeacherVO` 对象数组，而非 `teacherId/teacherName` 单值
- **课程信息**：使用 `course: CourseVO` 嵌套对象，而非 `courseId/courseName/courseType` 扁平字段
- **机构信息**：使用 `institution: InstitutionVO` 嵌套对象，而非 `institutionId` 扁平字段
- **课卡信息**：使用 `courseRecord: CourseRecordVO` 嵌套对象，而非 `courseRecordId/courseRestTime` 扁平字段
- **学生信息**：使用 `student: StudentVO` 嵌套对象，而非 `studentId/studentName` 扁平字段

### Requirement: 时间字段命名规范

所有 VO 中的时间字段 SHALL 统一使用 `xxxTime` 命名（非 `xxxTimeStr`），值为格式化字符串。

## REMOVED Requirements

无移除需求。
