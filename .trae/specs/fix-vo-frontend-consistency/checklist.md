# Checklist — 修复前后端报文一致性问题

## 阶段一：auth-service 绑定流程（P0）

### BindQrcodeVO
- [ ] `BindQrcodeVO` 结构体字段为 `Qrcode/Token/BindCode`（非 `Code/QRContent/IsSubscribe`）
- [ ] `GenerateBindQrcode` 方法返回 `qrcode`（二维码 base64）、`token`、`bindCode`
- [ ] `GenerateSubscribeQrcode` 方法返回同上三个字段
- [ ] 响应 JSON 中不再出现 `code/qrContent/isSubscribe` 字段

### BindInfoVO
- [ ] `BindInfoVO` 包含 `relation`（字符串）、`isPrimary`（布尔）字段
- [ ] `BindInfoVO` 包含 `parentName?`、`parentPhone?` 字段
- [ ] `BindInfoVO` 使用 `subscribeOnly`（非 `isSubscribe`）
- [ ] `GetBindInfo` 方法查询 `c_parent_student` 填充 `relation/isPrimary`
- [ ] `GetBindInfo` 方法查询 `c_parent` 填充 `parentName/parentPhone`
- [ ] `GetBindInfoByCode` 方法同上填充新增字段
- [ ] Redis 中存储的绑定信息包含 `relation/isPrimary/parentName/parentPhone`

### BindStatusVO
- [ ] `BindStatusVO` 结构体字段为 `AlreadyBound/HasAccount`（非 `Status/StudentInfo/AlreadyBound`）
- [ ] `CheckBindStatus` 方法 `alreadyBound` 检查 `c_parent_student` 是否已存在关联
- [ ] `CheckBindStatus` 方法 `hasAccount` 检查该学生是否已有任意家长账号
- [ ] 响应 JSON 中不再出现 `status` 和 `studentInfo` 字段

## 阶段二：business-service 列表页层级（P0）

### ClassVO
- [ ] `ClassVO` 包含 `teachers: []TeacherVO` 对象数组（非单值 `teacherId/teacherUsername`）
- [ ] `ClassVO` 包含 `courseRecord: CourseRecordVO` 嵌套对象
- [ ] `ClassVO` 包含 `scheduleList?: []ClassScheduleVO` 数组
- [ ] `ToClassVO` 转换函数按 class_id 聚合多教师为 `Teachers` 数组
- [ ] 学生维度查询时 `CourseRecord` 填充该学生的课卡信息
- [ ] 机构/教师维度查询时 `CourseRecord` 为 `null`
- [ ] 所有维度查询时 `ScheduleList` 填充该班级的课表列表

### ClassScheduleVO
- [ ] `ClassScheduleVO` 包含 `teachers: []TeacherVO` 对象数组（非单值 `teacherId/teacherName`）
- [ ] `ClassScheduleVO` 包含 `classroom` 字符串字段
- [ ] `ClassScheduleVO` 包含 `color?` 字符串字段
- [ ] `ToClassScheduleVO` 转换函数构造 `Teachers` 数组

### CourseVO
- [ ] `CourseVO` 包含 `institution: InstitutionVO` 嵌套对象（非扁平 `institutionId`）
- [ ] `CourseVO` 包含 `currentStudentCourseRecord?: CourseRecordVO` 嵌套对象
- [ ] `InstitutionVO` 包含 `id/institutionName/institutionCode` 字段
- [ ] 机构维度查询时 `CurrentStudentCourseRecord` 为 `nil`
- [ ] 学生维度查询时 `CurrentStudentCourseRecord` 填充该学生的课卡信息

### CourseRecordVO
- [ ] `CourseRecordVO` 包含 `course: CourseVO` 嵌套对象（非扁平 `courseId/courseName/courseType`）
- [ ] `CourseRecordVO` 包含 `permissionType` 字段
- [ ] `CourseRecordVO` 包含 `expireStatus` 字段（取值 `expired`/`warning`/`valid`）
- [ ] `ExpireStatus` 计算逻辑正确（7 天内到期为 `warning`，已到期为 `expired`，其余为 `valid`）

### RecordVO
- [ ] `RecordVO` 包含 `courseRecord: CourseRecordVO` 嵌套对象
- [ ] `RecordVO` 包含 `student: StudentVO` 嵌套对象
- [ ] `RecordVO` 包含 `course: CourseVO` 嵌套对象
- [ ] `RecordVO` 包含 `operatorTeacher: TeacherVO` 嵌套对象
- [ ] 响应 JSON 中不再出现扁平的 `courseId/studentId/courseName/studentName/teacherName` 字段

## 阶段三：扣费详情（P0）

### DeductDetailVO
- [ ] `DeductDetailVO` 包含 `courseRecordId` 字段
- [ ] `DeductDetailVO` 包含 `courseName/courseType` 字段
- [ ] `DeductDetailVO` 包含 `studentName` 字段
- [ ] `DeductDetailVO` 包含 `deductCount` 字段
- [ ] `DeductDetailVO` 包含 `courseTotalTime/expireTime` 字段
- [ ] `DeductDetailVO` 包含 `classId/className` 字段
- [ ] `DeductDetailVO` 包含 `scheduleDesc` 字段
- [ ] `DeductDetailVO` 包含 `teacherId/teacherName` 字段
- [ ] `DeductDetailVO` 包含 `expireStatus` 字段（取值 `normal`/`expired`/`warning`）
- [ ] `DeductDetailVO` 使用 `recordTime`（非 `recordTimeStr`）
- [ ] `DeductDetailVO` 使用 `remark`（非 `recordRemark`）
- [ ] `GetDeductDetail` 方法 JOIN 查询 `c_course/c_student/c_class/c_class_schedule/c_teacher` 表
- [ ] 响应包含 21 个字段

## 阶段四：Admin 仪表盘（P1）

### DashboardTrendRow
- [ ] `DashboardTrendRow` JSON tag 为 `months`（非 `labels`）

### InstitutionStatRow
- [ ] `InstitutionStatRow` JSON tag 为 `institutionId`（非 `id`）

## 阶段五：编译与验证

- [ ] `go build ./...` 编译通过（exit code 0）
- [ ] 无未使用的 import 或变量
- [ ] 小程序绑定流程端到端可用（二维码生成 → 信息查询 → 状态检查 → 确认绑定）
- [ ] 小程序班级列表页正常渲染教师数组
- [ ] 小程序课表列表页正常渲染教师数组和教室信息
- [ ] 小程序课程列表页正常渲染机构嵌套对象
- [ ] 小程序课卡列表页正常渲染课程嵌套对象和到期状态
- [ ] 小程序上课记录页正常渲染 4 个嵌套对象
- [ ] 小程序扣费详情页完整渲染 21 个字段
- [ ] Admin 仪表盘趋势图 X 轴标签正常显示
- [ ] Admin 仪表盘机构统计列表正确读取机构 ID
