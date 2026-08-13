# Checklist

## DDL 迁移
- [x] 8 张核心业务表（c_institution、c_teacher、c_student、c_parent、c_course、c_class、c_class_schedule、c_subscription_plan）均已有 `is_delete` 列（tinyint(1) NOT NULL DEFAULT 0）
- [x] `SHOW COLUMNS` 验证通过，存量数据 is_delete 全部为 0

## 实体定义
- [x] `common/entity/entity.go` 中 7 个实体（Student/Institution/Teacher/Parent/Course/Class/ClassSchedule）均有 `IsDelete sql.NullInt64` 字段（json:"isDelete"）（SubscriptionPlan 无独立实体，仅 JOIN 引用）

## business-service 查询过滤
- [x] `student_mapper.go` 全部学生查询方法（含 SelectByCourseIDWithCourseRecord）均携带 `is_delete = 0` 过滤
- [x] `SelectByCourseIDWithCourseRecord` 已恢复 `INNER JOIN c_student AS s ON s.id = cr.student_id AND s.is_delete = 0`
- [x] `course_mapper.go` 课程查询携带 `is_delete = 0`
- [x] `class_mapper.go` 班级查询携带 `is_delete = 0`
- [x] `class_schedule_mapper.go` 课表查询携带 `is_delete = 0`
- [x] `institution_mapper.go` 机构查询携带 `is_delete = 0`
- [x] `teacher_mapper.go` 教师查询携带 `is_delete = 0`
- [x] `parent_mapper.go` 家长查询携带 `is_delete = 0`

## admin-service 查询过滤
- [x] `student_mapper.go` 学生列表/详情/计数携带 `is_delete = 0`
- [x] `course_mapper.go` 课程查询携带 `is_delete = 0`
- [x] `class_mapper.go` 班级查询携带 `is_delete = 0`
- [x] `class_schedule_mapper.go` 课表查询携带 `is_delete = 0`
- [x] `institution_mapper.go` 机构查询携带 `is_delete = 0`
- [x] `teacher_mapper.go` 教师查询携带 `is_delete = 0`
- [x] `dashboard_mapper.go` 学生/教师/课程/班级统计查询携带 `is_delete = 0`

## auth-service 查询过滤
- [x] `parent_mapper.go` 家长/机构/学生查询携带 `is_delete = 0`
- [x] `teacher_mapper.go` 教师查询携带 `is_delete = 0`

## 验证
- [x] `go build ./...` 通过
- [x] `go vet ./...` 通过
- [x] 通过 MCP 执行 get_by_course_id 对应 SQL（courseId=12）返回正常数据，无 `Unknown column` 错误

## 文档与提交
- [x] `class_times_record_back/CLAUDE.md` 已更新软删除机制说明
- [x] `class_times_record_back` 已提交推送（master，commit 6bb7c8b）
- [x] 根仓库子模块指针已同步推送（commit 7717bea，含 AGENTS.md 软删除说明）
