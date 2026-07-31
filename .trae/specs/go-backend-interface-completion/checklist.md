# Go 后端接口补全验证清单

## 阶段一：auth-service 绑定与订阅流程

- [ ] `POST /auth/auth/generate_bind_qrcode` 接口已实现，返回 6 位绑定码和二维码内容
- [ ] `POST /auth/auth/generate_subscribe_qrcode` 接口已实现，返回订阅专用二维码
- [ ] `GET /auth/auth/get_bind_info` 接口已实现，按 token 返回学生信息（无需登录）
- [ ] `GET /auth/auth/get_bind_info_by_code` 接口已实现，按 6 位绑定码返回学生信息（不执行绑定）
- [ ] `GET /auth/auth/check_bind_status` 接口已实现，返回绑定码状态（有效/已用/过期）
- [ ] `POST /auth/auth/confirm_bind` 接口已实现，创建 parent_student 关联记录
- [ ] `POST /auth/auth/bind_by_code` 接口已实现，按绑定码直接完成绑定流程
- [ ] `GET /auth/auth/test_send_subscribe` 接口已实现，成功调用微信订阅消息推送
- [ ] 绑定码存入 Redis，TTL=10 分钟，使用后标记为已用
- [ ] 绑定流程创建 parent_student 关联，同时更新 wx_student_subscribe
- [ ] Gateway publicPaths 中 6 个绑定相关路径全部在 Go 端实现

## 阶段二：business-service 写操作

### 学生模块
- [ ] `POST /biz/student/get_by_class_id` 返回班级学生列表
- [ ] `POST /biz/student/get_by_course_id` 返回课程选修学生列表
- [ ] `POST /biz/student/insert` 新增学生并返回新 ID
- [ ] `POST /biz/student/update` 更新学生信息
- [ ] `POST /biz/student/unbind` 解绑家长-学生关系（删除 c_parent_student 记录）
- [ ] `POST /biz/student/cancel_subscribe` 取消订阅（删除 c_wx_student_subscribe 和 c_wx_subscribe_record）

### 教师模块
- [ ] `POST /biz/teacher/update_by_id` 更新教师信息
- [ ] `POST /biz/teacher/insert` 新增教师
- [ ] `POST /biz/teacher/delete` 删除教师及关联 user_auth、user 记录

### 班级模块
- [ ] `POST /biz/class/get_classes_by_student_id` 返回学生所在班级列表
- [ ] `POST /biz/class/get_classes_by_teacher_id` 返回教师关联班级列表
- [ ] `POST /biz/class/get_classes_by_institution_id` 返回机构班级列表
- [ ] `POST /biz/class/get_class_by_id` 返回班级详情（含学生列表）
- [ ] `POST /biz/class/insert` 新增班级
- [ ] `POST /biz/class/update_by_id` 更新班级
- [ ] `POST /biz/class/add_student_to_class` 班级添加学生（写入 c_class_student）
- [ ] `POST /biz/class/remove_student_from_class` 班级移除学生（删除 c_class_student）

### 课表模块
- [ ] `POST /biz/class_schedule/get_by_class_id` 按班级查课表
- [ ] `POST /biz/class_schedule/get_by_institution_id` 按机构查课表
- [ ] `POST /biz/class_schedule/get_by_teacher_id` 按教师查课表
- [ ] `POST /biz/class_schedule/get_by_id` 按ID查课表详情
- [ ] `POST /biz/class_schedule/update_by_id` 更新课表

### 课程模块
- [ ] `POST /biz/course/get_course_by_institution_id` 按机构查课程
- [ ] `POST /biz/course/get_course_by_student_id` 按学生查课程
- [ ] `POST /biz/course/add_course` 新增课程
- [ ] `POST /biz/course/update_by_id` 更新课程

### 课卡记录模块
- [ ] `POST /biz/course_record/new_get` 查询课卡记录列表（分页）
- [ ] `POST /biz/course_record/get_by_student_id` 按学生查课卡记录
- [ ] `POST /biz/course_record/get_by_institution_id` 按机构查课卡记录
- [ ] `POST /biz/course_record/insert` 新增课卡记录
- [ ] `POST /biz/course_record/update` 更新课卡记录
- [ ] `POST /biz/course_record/deduct_by_student_id` 按学生扣课，执行双重校验（过期+余额）
- [ ] `POST /biz/course_record/deduct_by_course_id` 按课程扣课，执行双重校验
- [ ] `POST /biz/course_record/deduct_by_class_id` 按班级扣课，执行双重校验
- [ ] `GET /biz/course_record/deduct-detail` 查询扣费详情（公开接口）
- [ ] 扣费时 Java 层校验 `expire_time`，过期返回 code=1003
- [ ] 扣费时 SQL 层 WHERE 包含 `(expire_time IS NULL OR expire_time > NOW())` 兜底
- [ ] 余额不足返回 code=1001

### 机构与上课记录模块
- [ ] `POST /biz/institution/update` 更新机构信息
- [ ] `POST /biz/record/new_get` 查询上课记录列表（分页）
- [ ] `POST /biz/record/delete` 删除上课记录

## 阶段三：admin-service 系统管理

### 用户管理
- [x] `POST /admin/user/get_by_id` 按 ID 查系统用户
- [x] `POST /admin/user/insert` 新增用户（BCrypt 哈希密码）
- [x] `POST /admin/user/update` 更新用户
- [x] `POST /admin/user/delete` 删除用户
- [x] `POST /admin/user/reset_password` 重置密码
- [x] `POST /admin/user/get_roles` 查询用户角色列表

### 角色管理
- [x] `POST /admin/role/list` 角色列表（分页）
- [x] `POST /admin/role/get_by_id` 按 ID 查角色
- [x] `POST /admin/role/insert` 新增角色
- [x] `POST /admin/role/update` 更新角色
- [x] `POST /admin/role/delete` 删除角色（含关联清理）
- [x] `POST /admin/role/get_menus` 查询角色已分配菜单
- [x] `POST /admin/role/save_menus` 保存角色菜单授权（事务：删旧+插新）

### 菜单管理
- [x] `POST /admin/menu/list` 菜单扁平列表
- [x] `POST /admin/menu/tree` 完整菜单树
- [x] `POST /admin/menu/user_tree` 当前用户菜单树（按 roleIds 过滤）
- [x] `POST /admin/menu/insert` 新增菜单
- [x] `POST /admin/menu/update` 更新菜单
- [x] `POST /admin/menu/delete` 删除菜单

### 操作日志
- [x] `POST /admin/operation_log/list` 日志列表（分页，支持过滤）
- [x] `POST /admin/operation_log/delete` 删除单条日志
- [x] `POST /admin/operation_log/clear` 清空全部日志
- [x] Admin 写操作接口记录操作日志（对齐 Java @OperationLog）

## 阶段四：admin-service 业务管理透传

### 机构与学生
- [x] `POST /admin/business/institution/list` 机构分页列表
- [x] `POST /admin/business/institution/insert` 新增机构
- [x] `POST /admin/business/institution/update` 更新机构
- [x] `POST /admin/business/student/list` 学生分页列表
- [x] `POST /admin/business/student/insert` 新增学生
- [x] `POST /admin/business/student/update` 更新学生

### 教师管理与账号
- [x] `POST /admin/business/teacher/list` 教师分页列表
- [x] `POST /admin/business/teacher/insert` 新增教师
- [x] `POST /admin/business/teacher/update` 更新教师
- [x] `POST /admin/teacher_auth/get` 查询教师账号信息
- [x] `POST /admin/teacher_auth/update_account` 更新教师账号
- [x] `POST /admin/teacher_auth/update_password` 修改教师密码
- [x] `POST /admin/teacher_auth/toggle_institution_admin` 切换机构管理员身份

### 课程与班级
- [x] `POST /admin/business/course/list` 课程分页列表
- [x] `POST /admin/business/course/insert` 新增课程
- [x] `POST /admin/business/course/update` 更新课程
- [x] `POST /admin/business/class/list` 班级分页列表
- [x] `POST /admin/business/class/insert` 新增班级
- [x] `POST /admin/business/class/update` 更新班级
- [x] `POST /admin/business/class/get_by_id` 按ID查班级详情
- [x] `POST /admin/business/class/add_student` 班级添加学生
- [x] `POST /admin/business/class/remove_student` 班级移除学生

### 课表、课时记录、上课记录、小程序菜单
- [x] `POST /admin/business/class_schedule/list` 课表列表
- [x] `POST /admin/business/class_schedule/update` 更新课表
- [x] `POST /admin/business/course_record/list` 课时记录列表
- [x] `POST /admin/business/course_record/insert` 新增课时记录
- [x] `POST /admin/business/course_record/update` 更新课时记录
- [x] `POST /admin/business/record/list` 上课记录列表
- [x] `POST /admin/business/record/insert` 新增上课记录
- [x] `POST /admin/business/mini_menu/list` 小程序菜单列表
- [x] `POST /admin/business/mini_menu/insert` 新增小程序菜单
- [x] `POST /admin/business/mini_menu/update` 更新小程序菜单
- [x] `POST /admin/business/mini_menu/delete` 删除小程序菜单

## 阶段五：仪表盘与系统配置

- [x] `POST /admin/dashboard/data` 返回汇总数据
- [x] `POST /admin/dashboard/trend` 返回趋势数据（支持 range 参数）
- [x] `POST /admin/dashboard/institution/stats` 返回机构统计（支持 limit 参数）
- [x] `POST /admin/config/list` 系统配置列表
- [x] `POST /admin/config/insert` 新增系统配置
- [x] `POST /admin/config/update` 更新系统配置
- [x] `POST /admin/config/delete` 删除系统配置

## 阶段六：全量验证

- [x] 小程序前端 64 个接口全部在 Go 后端注册（对照 src/api/ 各模块）✅
- [x] Admin 前端 62 个接口全部在 Go 后端注册（对照 src/api/ 各模块）✅
- [x] Gateway publicPaths 白名单中 20 个路径全部在 Go 端实现 ✅
- [x] 所有接口响应结构顶层为 `{ code, message, data, requestTime }` ✅
- [x] 所有 VO 字段使用驼峰命名（非 sql.NullXxx 对象格式）✅
- [x] 时间字段返回格式化字符串（createTimeStr/updateTimeStr）✅
- [x] ID 字段为 int64（非 sql.NullInt64 对象）✅
- [x] 布尔字段为 bool（非 sql.NullBool 对象）✅
- [x] `go build ./...` 全量编译通过（exit code 0）✅
- [x] 各服务 main.go 中路由注册无遗漏（除 /record/delete 外）✅
- [x] auth-service 8 个绑定接口真实实现（含 Redis、DB、微信 API 调用）✅
- [x] business-service 37/38 写操作真实实现（仅 /record/delete 路由未注册）✅
- [x] admin-service 52 个接口真实实现（系统管理+业务透传+仪表盘+配置）✅
- [x] 扣费双重校验实现（Java 层 expire_time 校验 + SQL 层 WHERE 兜底）✅
- [ ] 所有接口响应结构（DTO/VO 字段）与前端类型定义完全一致 ⚠️ 发现 9 处不一致（见阶段七）

## 阶段七：响应结构不一致修复验证

### P0 致命问题（阻塞小程序绑定流程）

- [ ] `POST /auth/auth/generate_bind_qrcode` 响应字段为 `qrcode/token/bindCode`（非 `code/qrContent/isSubscribe`）
- [ ] `POST /auth/auth/generate_subscribe_qrcode` 响应字段同上
- [ ] `GET /auth/auth/get_bind_info` 响应包含 `relation/isPrimary/parentName/parentPhone` 字段
- [ ] `GET /auth/auth/get_bind_info_by_code` 响应同上
- [ ] `BindInfoVO` 的 `isSubscribe` 字段重命名为 `subscribeOnly`
- [ ] `GET /auth/auth/check_bind_status` 响应字段为 `{ alreadyBound: bool, hasAccount: bool }`
- [ ] `CheckBindStatus` 方法 `hasAccount` 逻辑正确（检查学生是否已有家长账号）

### P1 重要问题（影响 Admin 仪表盘和扣费详情）

- [ ] `POST /admin/dashboard/trend` 响应字段名为 `months`（非 `labels`）
- [ ] `POST /admin/dashboard/institution/stats` 响应字段名为 `institutionId`（非 `id`）
- [ ] `GET /biz/course_record/deduct-detail` 响应包含 12 个缺失字段（courseRecordId/courseName/courseType/studentName/deductCount/courseTotalTime/expireTime/classId/className/scheduleDesc/teacherId/teacherName/expireStatus）
- [ ] `DeductDetailVO` 的 `recordTimeStr` 重命名为 `recordTime`
- [ ] `DeductDetailVO` 的 `recordRemark` 重命名为 `remark`
- [ ] `expireStatus` 字段逻辑正确（normal/expired/warning）

### P2 次要问题（提升前端体验）

- [ ] `POST /admin/business/class/list` 的 `teachers` 字段为对象数组（非分离的 ID+名称数组）
- [ ] Admin 前端 `ParentInfoResponse` 类型增加 `isBound?: boolean` 字段
- [ ] `AdminClassScheduleRow` 的 Go 字段名 `CreateTimeStr/UpdateTimeStr` 改为 `CreateTime/UpdateTime`

### 阶段七编译与验证

- [ ] `go build ./...` 编译通过
- [ ] 小程序绑定流程端到端可用
- [ ] Admin 仪表盘页面正常渲染
- [ ] 扣费详情页完整渲染所有字段

## 阶段八：补录缺失路由

- [ ] `POST /biz/record/delete` 路由已注册
- [ ] `DeleteRecord` handler 调用 `RecordMapper.DeleteByID`
- [ ] 编译验证通过
