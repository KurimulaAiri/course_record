# Tasks

## 阶段一：P0 致命问题修复 — auth-service 绑定流程（3 项）

- [x] Task 1: 修复 BindQrcodeVO 字段对齐 ✅
- [x] Task 2: 修复 BindInfoVO 字段对齐 ✅
- [x] Task 3: 修复 BindStatusVO 结构对齐 ✅

## 阶段二：P0 致命问题修复 — business-service 列表页层级（5 项）

- [x] Task 4: 修复 ClassVO 嵌套层级 ✅
- [x] Task 5: 修复 ClassScheduleVO 嵌套层级 ✅
- [x] Task 6: 修复 CourseVO 嵌套层级 ✅

## 阶段三：P0 致命问题修复 — 课卡与上课记录（2 项）

- [x] Task 7: 修复 CourseRecordVO 嵌套层级 ✅
- [x] Task 8: 修复 RecordVO 嵌套层级 ✅

## 阶段四：P0 致命问题修复 — 扣费详情（1 项）

- [x] Task 9: 修复 DeductDetailVO 完整字段 ✅

## 阶段五：P1 重要问题修复 — Admin 仪表盘（2 项）

- [x] Task 10: 修复 DashboardTrendRow 字段名 ✅
- [x] Task 11: 修复 InstitutionStatRow 字段名 ✅

## 阶段六：编译与全量验证

- [x] Task 12: 编译验证 ✅ (go build ./... exit code 0)
- [x] Task 13: 响应结构验证 ✅ (代码审查确认所有 VO 字段对齐)
