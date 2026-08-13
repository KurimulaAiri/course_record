# 核心业务表统一软删除字段 Spec

## Why

`POST /student/get_by_course_id`（按课程查询学生）运行时抛错 `Error 1054 (42S22): Unknown column 's.is_delete' in 'on clause'`。根因是 [student_mapper.go](file:///d:/proj/kurimula-airi/course_record/class_times_record_back/business-service/internal/mapper/student_mapper.go) 的 `SelectByCourseIDWithCourseRecord` SQL 引用了 `c_student` 表的 `s.is_delete` 列，但 **`c_student` 表没有该列**。

项目软删除机制不统一：
- `c_course_record`、`c_record` 已有 `is_delete` 列 ✅
- `sys_user`、`sys_role` 使用 `is_deleted` 列 ✅
- `c_course`、`c_teacher`、`c_parent`、`c_admin`、`c_user_platform` 仅有 `is_available`（停用字段），无 `is_delete`
- `c_student`、`c_class`、`c_institution`、`c_class_schedule`、`c_subscription_plan` 无任何软删除字段 ❌

由于 SQL 与表结构不一致，运行时会抛出 500 错误，且各核心表删除机制不统一，存在同类隐患。

## What Changes

- **DDL**：为 8 个核心业务表新增 `is_delete` 列（`tinyint(1) NOT NULL DEFAULT 0`，对齐 `c_course_record`）：`c_institution`、`c_teacher`、`c_student`、`c_parent`、`c_course`、`c_class`、`c_class_schedule`、`c_subscription_plan`
- **恢复 SQL 条件**：`SelectByCourseIDWithCourseRecord` 恢复 `AND s.is_delete = 0`（当前是移除状态）
- **统一查询过滤**：三个服务（business/admin/auth）中所有核心业务表的主表查询统一补充 `is_delete = 0` 过滤，保证软删除机制生效
- **实体补字段**：`common/entity` 中对应实体（Student/Institution/Teacher/Parent/Course/Class/ClassSchedule/SubscriptionPlan）补充 `IsDelete` 字段
- **保持现状**：`c_course_record`/`c_record`/`sys_user`/`sys_role` 已有软删除字段不动；`is_available` 停用语义保留（两者并存，符合"停用 + 归档"双机制）；关联表/日志表保持硬删除

## Impact

- **Affected specs**: 核心业务数据查询与删除一致性
- **Affected code**:
  - DDL：8 张核心业务表（经 MCP `execute_db_update` + `allow_ddl=true` 执行）
  - `common/entity/entity.go`：8 个实体补 `IsDelete` 字段
  - `business-service/internal/mapper/`：student/course/class/class_schedule/institution/teacher/parent mapper 查询过滤
  - `admin-service/internal/mapper/`：student/course/class/class_schedule/institution/teacher/course_record mapper 查询过滤
  - `auth-service/internal/mapper/`：parent/teacher mapper 查询过滤
  - 各服务 dashboard 统计查询同步过滤
- **BREAKING**: 数据库 8 张表新增列（需 DDL 迁移，默认值 0，存量数据不受影响）

## ADDED Requirements

### Requirement: 核心业务表软删除字段

所有核心业务表（`c_institution`、`c_teacher`、`c_student`、`c_parent`、`c_course`、`c_class`、`c_class_schedule`、`c_subscription_plan`）SHALL 提供 `is_delete` 列，类型 `tinyint(1) NOT NULL DEFAULT 0`，`0` 表示未删除，`1` 表示已删除（对齐 `c_course_record`）。

#### Scenario: 查询课程学生列表成功
- **WHEN** 调用 `POST /student/get_by_course_id`（courseId=12）
- **THEN** 返回报读该课程且未删除（`is_delete = 0`）的学生列表，不再抛 `Unknown column` 错误

### Requirement: 核心业务表查询统一软删除过滤

三个服务（business/admin/auth）中所有核心业务表的主表查询（SELECT 列表/详情/统计）SHALL 统一携带 `is_delete = 0` 过滤条件，确保已软删除记录不出现在业务数据中。

#### Scenario: 已删除学生不出现
- **WHEN** 某学生被标记 `is_delete = 1`
- **THEN** 所有学生列表、详情、班级/课程关联查询均不再返回该学生

## MODIFIED Requirements

### Requirement: SelectByCourseIDWithCourseRecord 查询条件

**原状**：上一轮修复移除了 `s.is_delete = 0` JOIN 条件（因表无该列导致报错）。

**修改后**：`c_student` 表补充 `is_delete` 列后，恢复 `INNER JOIN c_student AS s ON s.id = cr.student_id AND s.is_delete = 0`，并更新注释说明软删除机制。

### Requirement: 实体定义

`common/entity/entity.go` 中 Student、Institution、Teacher、Parent、Course、Class、ClassSchedule、SubscriptionPlan 实体 SHALL 补充 `IsDelete sql.NullInt64` 字段（`json:"isDelete"`），与数据库列对齐，供查询使用。

## REMOVED Requirements

无。
