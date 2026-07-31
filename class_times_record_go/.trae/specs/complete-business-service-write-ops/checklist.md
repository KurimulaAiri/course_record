# Checklist

- [x] record_service.go 已创建，包含 NewGetRecord/InsertRecord/InsertRecords 3 个方法
- [x] RecordService.NewGetRecord 返回 data.records（数组）+ data.total，对齐前端 RecordListResponse
- [x] RecordService.InsertRecords 按 recordType 正确更新课卡（type=1 减课时，type=2 加课时）
- [x] handler.go BusinessHandler 结构体包含全部 8 个 Service 字段
- [x] NewBusinessHandler 构造函数接受全部 8 个 Service 参数
- [x] RegisterRoutes 注册全部 38 个路由（含原有 8 个查询路由）
- [x] 学生模块 6 个写操作 Handler 实现正确（含参数解析和 Service 调用）
- [x] 教师模块 3 个写操作 Handler 实现正确（含 SM2 密码密文透传）
- [x] 班级模块 8 个 Handler 实现正确（含 InsertClass 的 schedules/teachers 数组解析）
- [x] 课卡记录模块 9 个 Handler 实现正确（含 3 个扣课接口的 mode 分发）
- [x] deduct-detail 接口使用 GET 方法（对齐前端 getDeductDetail）
- [x] main.go 创建全部 17 个 Mapper 实例
- [x] main.go 从环境变量 SM2_PRIVATE_KEY 读取 SM2 私钥（未设置时回退到默认值）
- [x] main.go 使用新签名创建全部 8 个 Service
- [x] go build ./... 编译通过，无错误
