# Tasks

- [x] Task 1: 创建 record_service.go 实现 3 个上课记录接口
  - [x] SubTask 1.1: 创建 RecordService 结构体和构造函数（依赖 RecordMapper + CourseRecordMapper）
  - [x] SubTask 1.2: 实现 NewGetRecord 方法（按机构/学生/课程名/记录类型分页查询，返回 records+total）
  - [x] SubTask 1.3: 实现 InsertRecord 方法（新增单条上课记录）
  - [x] SubTask 1.4: 实现 InsertRecords 方法（批量新增上课记录，同时按 recordType 更新课卡剩余课时）

- [x] Task 2: 重写 handler.go 注册全部 38 个写操作接口路由
  - [x] SubTask 2.1: 扩展 BusinessHandler 结构体，新增 classService/classScheduleService/courseService/courseRecordService/recordService 字段
  - [x] SubTask 2.2: 更新 NewBusinessHandler 构造函数接受全部 8 个 Service
  - [x] SubTask 2.3: 在 RegisterRoutes 中注册全部 38 个路由
  - [x] SubTask 2.4: 实现学生模块 6 个写操作 Handler（InsertStudent/UpdateStudent/UnbindStudent/CancelStudentSubscribe/GetStudentByClassID/GetStudentByCourseID）
  - [x] SubTask 2.5: 实现教师模块 3 个写操作 Handler（InsertTeacher/UpdateTeacher/DeleteTeacher）
  - [x] SubTask 2.6: 实现机构模块 1 个写操作 Handler（UpdateInstitution）
  - [x] SubTask 2.7: 实现班级模块 8 个 Handler（含查询和写操作）
  - [x] SubTask 2.8: 实现课表模块 5 个 Handler
  - [x] SubTask 2.9: 实现课程模块 4 个 Handler
  - [x] SubTask 2.10: 实现课卡记录模块 9 个 Handler（含 3 个扣课接口和 deduct-detail）
  - [x] SubTask 2.11: 实现上课记录模块 3 个 Handler

- [x] Task 3: 重写 main.go 完成依赖注入
  - [x] SubTask 3.1: 创建所有 17 个 Mapper 实例
  - [x] SubTask 3.2: 从环境变量 SM2_PRIVATE_KEY 读取 SM2 私钥（未设置时回退到默认值）
  - [x] SubTask 3.3: 使用新签名创建所有 8 个 Service
  - [x] SubTask 3.4: 注入到 BusinessHandler 并注册路由

- [x] Task 4: 编译验证
  - [x] SubTask 4.1: 执行 go build ./... 确认编译通过
  - [x] SubTask 4.2: 修复编译错误（cr.ID 类型不匹配 + service.go 未使用的 strings 导入）

# Task Dependencies
- Task 2 依赖 Task 1（handler 引用 RecordService）
- Task 3 依赖 Task 1 和 Task 2（main.go 注入新 Service 和 Handler）
- Task 4 依赖 Task 1、Task 2、Task 3
