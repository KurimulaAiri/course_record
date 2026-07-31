// Package service business-service 业务逻辑层 - 共享 VO 定义
//
// 本文件集中定义跨模块复用的视图对象（VO），避免在多个 service 文件中重复定义同名结构。
//
// 当前共享 VO：
//   - TeacherBriefVO：教师简要信息 VO，用于 ClassVO/ClassScheduleVO/RecordVO 的嵌套字段
//   - InstitutionBriefVO：机构简要信息 VO，用于 CourseVO 的嵌套 institution 字段
//   - CourseBriefVO：课程简要信息 VO，用于 CourseRecordVO 的嵌套 course 字段（避免循环引用）
//   - StudentBriefVO：学生简要信息 VO，用于 RecordVO 的嵌套 student 字段
//
// 设计说明：
//   - 与 service.go 中的 TeacherVO/InstitutionVO/StudentVO（完整视图，对齐前端 XxxResponse）不同，
//     Brief 系列仅包含嵌套场景下需要的最小字段集合，避免在 JOIN 查询中不必要地获取过多列。
//   - 命名使用 XxxBriefVO 后缀，避免与 service.go 中已有的完整 VO（如 InstitutionVO/StudentVO）冲突。
//   - JSON 字段命名与前端类型保持一致。
//
// 循环引用处理：
//   - CourseVO 引用 *CourseRecordVO（指针），CourseRecordVO 引用 CourseBriefVO（值，非 CourseVO）
//   - 因此 CourseVO → *CourseRecordVO → CourseBriefVO，无循环，Go 编译器可接受。
package service

// TeacherBriefVO 教师简要信息 VO（用于班级/课表/上课记录嵌套）
//
// 对齐前端 TeacherResponse 中嵌套场景下实际使用的字段集合：
//   - teacherId：教师ID（主键，对应 c_teacher.id）
//   - username：教师用户名（对应 c_teacher.username）
//
// 使用场景：
//   - ClassVO.Teachers：班级的教师列表（一个班级可有多名教师）
//   - ClassScheduleVO.Teachers：课表的教师列表（继承自所属班级的教师列表）
//   - RecordVO.OperatorTeacher：上课记录的操作教师
//
// 注意：与 service.go 中的 TeacherVO（完整教师视图）不同，本结构体仅包含
// 嵌套场景下需要展示的最小字段集合，避免在查询中
// 不必要地 JOIN c_user_auth、c_institution 等表来填充 account、phone 等字段。
type TeacherBriefVO struct {
	TeacherID int64  `json:"teacherId"` // 教师ID（对应 c_teacher.id）
	Username  string `json:"username"`  // 教师用户名（对应 c_teacher.username）
}

// InstitutionBriefVO 机构简要信息 VO（用于课程嵌套）
//
// 对齐前端 CourseResponse.institution 字段（类型 InstitutionResponse）中
// 课程场景下实际使用的字段集合：
//   - id：机构ID（主键，对应 c_institution.id）
//   - institutionName：机构名称（对应 c_institution.institution_name）
//   - institutionCode：机构编码（对应 c_institution.institution_code）
//
// 使用场景：
//   - CourseVO.Institution：课程所属的机构信息
//
// 注意：与 service.go 中的 InstitutionVO（完整机构视图，含 address/status/expireTime 等）不同，
// 本结构体仅包含课程嵌套场景下需要的最小字段集合，避免在课程查询中
// 不必要地获取机构的地址、状态、订阅计划等字段。
type InstitutionBriefVO struct {
	ID              int64  `json:"id"`              // 机构ID（对应 c_institution.id）
	InstitutionName string `json:"institutionName"` // 机构名称（对应 c_institution.institution_name）
	InstitutionCode string `json:"institutionCode"` // 机构编码（对应 c_institution.institution_code）
}

// CourseBriefVO 课程简要信息 VO（用于课卡记录嵌套，避免循环引用）
//
// 对齐前端 CourseRecordResponse.course 字段（类型 CourseResponse）中
// 课卡场景下实际使用的字段集合：
//   - id：课程ID（主键，对应 c_course.id）
//   - courseName：课程名称（对应 c_course.course_name）
//   - courseType：课程类型（1=按次, 2=按天，对应 c_course.course_type）
//   - isAvailable：是否可用（对应 c_course.is_available）
//
// 使用场景：
//   - CourseRecordVO.Course：课卡记录所属的课程信息
//
// 循环引用说明：
//   - CourseVO 引用 *CourseRecordVO（CurrentStudentCourseRecord 字段）
//   - 如果 CourseRecordVO.Course 使用 CourseVO 类型，会形成 CourseVO → *CourseRecordVO → CourseVO 循环
//   - 虽然 Go 允许指针打破循环，但语义上课卡嵌套的课程不需要再包含课卡信息（无意义递归）
//   - 因此定义 CourseBriefVO 作为 CourseRecordVO.Course 的类型，彻底切断递归链路
//
// 注意：与 course_service.go 中的 CourseVO（完整课程视图，含 institution/currentStudentCourseRecord）不同，
// 本结构体仅包含课卡嵌套场景下需要的最小字段集合。
type CourseBriefVO struct {
	ID          int64 `json:"id"`          // 课程ID（对应 c_course.id）
	CourseName  string `json:"courseName"`  // 课程名称（对应 c_course.course_name）
	CourseType  int64 `json:"courseType"`   // 课程类型（1=按次, 2=按天，对应 c_course.course_type）
	IsAvailable bool  `json:"isAvailable"`  // 是否可用（对应 c_course.is_available）
}

// StudentBriefVO 学生简要信息 VO（用于上课记录嵌套）
//
// 对齐前端 RecordResponse.student 字段（类型 StudentResponse）中
// 上课记录场景下实际使用的字段集合：
//   - id：学生ID（主键，对应 c_student.id）
//   - studentName：学生姓名（对应 c_student.student_name）
//   - sex：性别（0=未知,1=男,2=女，对应 c_student.sex）
//   - avatar：头像URL（对应 c_student.avatar）
//
// 使用场景：
//   - RecordVO.Student：上课记录关联的学生信息
//
// 注意：与 service.go 中的 StudentVO（完整学生视图，含 institutionId/birthStr/school/address 等）不同，
// 本结构体仅包含上课记录嵌套场景下需要的最小字段集合。
type StudentBriefVO struct {
	ID          int64  `json:"id"`          // 学生ID（对应 c_student.id）
	StudentName string `json:"studentName"` // 学生姓名（对应 c_student.student_name）
	Sex         int64  `json:"sex"`         // 性别（0=未知,1=男,2=女，对应 c_student.sex）
	Avatar      string `json:"avatar"`      // 头像URL（对应 c_student.avatar）
}
