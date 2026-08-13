# Tasks

- [x] Task 1: DDL 迁移 - 为 8 张核心业务表新增 is_delete 列
  - [x] 通过 MCP `execute_db_update`（`allow_ddl=true`）依次执行 `ALTER TABLE {table} ADD COLUMN is_delete tinyint(1) NOT NULL DEFAULT 0`：c_institution、c_teacher、c_student、c_parent、c_course、c_class、c_class_schedule、c_subscription_plan
  - [x] 验证：`SHOW COLUMNS` 确认 8 张表均含 is_delete 列且默认值 0

- [x] Task 2: 实体补充 IsDelete 字段
  - [x] `common/entity/entity.go` 中 Student、Institution、Teacher、Parent、Course、Class、ClassSchedule 实体补充 `IsDelete sql.NullInt64` 字段（`json:"isDelete"`），附带字段说明注释（SubscriptionPlan 无独立实体，仅 JOIN 引用）

- [x] Task 3: business-service 学生查询统一 is_delete 过滤
  - [x] `student_mapper.go`：`SelectByID`/`SelectByInstitutionID`/`SelectByParentID`/`SelectByTeacherID`/`SelectByTeacherIDWithPage`/`SelectByInstitutionIDWithPage`/`SelectByClassID`/`SelectByClassIDWithCourseRecord`/`SelectByCourseID`/`SelectByCourseIDWithCourseRecord` 补充 `is_delete = 0` 过滤
  - [x] `SelectByCourseIDWithCourseRecord` 恢复 `INNER JOIN c_student AS s ON s.id = cr.student_id AND s.is_delete = 0`，更新注释（原报错点）

- [x] Task 4: business-service 其他核心表查询过滤
  - [x] `course_mapper.go`：课程查询（列表/详情/计数）补充 `is_delete = 0`
  - [x] `class_mapper.go`：班级查询（列表/详情/计数）补充 `is_delete = 0`
  - [x] `class_schedule_mapper.go`：课表查询补充 `is_delete = 0`
  - [x] `institution_mapper.go`：机构查询补充 `is_delete = 0`
  - [x] `teacher_mapper.go`：教师查询补充 `is_delete = 0`
  - [x] `parent_mapper.go`：家长查询补充 `is_delete = 0`

- [x] Task 5: admin-service 核心表查询过滤
  - [x] `student_mapper.go`：学生列表/详情/计数补充 `is_delete = 0`
  - [x] `course_mapper.go`：课程查询补充 `is_delete = 0`
  - [x] `class_mapper.go`：班级查询补充 `is_delete = 0`
  - [x] `class_schedule_mapper.go`：课表查询补充 `is_delete = 0`
  - [x] `institution_mapper.go`：机构查询补充 `is_delete = 0`
  - [x] `teacher_mapper.go`：教师查询补充 `is_delete = 0`
  - [x] `dashboard_mapper.go`：学生/教师/课程/班级统计查询补充 `is_delete = 0`

- [x] Task 6: auth-service 核心表查询过滤
  - [x] `parent_mapper.go`：c_parent/c_institution/c_student 查询补充 `is_delete = 0`
  - [x] `teacher_mapper.go`：教师查询补充 `is_delete = 0`

- [x] Task 7: 编译与验证
  - [x] `go build ./...` 通过
  - [x] `go vet ./...` 通过
  - [x] 通过 MCP `execute_db_query` 执行 `get_by_course_id` 对应 SQL（courseId=12）验证无 `Unknown column` 错误且返回正常学生数据

- [x] Task 8: 文档更新与提交推送
  - [x] 更新 `class_times_record_back/CLAUDE.md`（软删除机制说明：核心业务表统一 is_delete）
  - [x] 更新根目录 `AGENTS.md`/`CLAUDE.md` 表字段说明（如涉及）
  - [x] 提交并推送 `class_times_record_back`（master，commit 6bb7c8b）
  - [x] 同步根仓库子模块指针并推送（commit 7717bea）

# Task Dependencies

- [Task 2] 依赖 [Task 1]（实体字段与数据库列对齐）
- [Task 3][Task 4][Task 5][Task 6] 依赖 [Task 1]（SQL 引用新列前需先建列）
- [Task 3][Task 4][Task 5][Task 6] 相互独立，可并行
- [Task 7] 依赖 [Task 2]-[Task 6]
- [Task 8] 依赖 [Task 7]
