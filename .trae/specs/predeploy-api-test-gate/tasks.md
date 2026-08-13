# Tasks

- [x] Task 1: 搭建部署前测试包基础框架
  - [x] 新建 `poc/predeploy/` 包：`predeploy_test.go`（主入口 + 服务就绪等待）
  - [x] 配置模块：`config.go`（测试端口/基础 URL 常量，从环境变量读取测试账号）
  - [x] HTTP 客户端：`client.go`（POST/GET 封装、SM3 签名头、Bearer token、SM2 加密密码工具，复用 common/sign 与 common/crypto）
  - [x] 登录辅助：`auth.go`（管理端明文登录、小程序端 SM2 加密 + SM3 签名登录，获取 token）
  - [x] 验证：go build / go vet / go test 均通过

- [x] Task 2: auth-service 接口测试
  - [x] `auth_test.go`：覆盖 auth-service handler 注册的全部路由（get_open_id/login_no_pwd/login_by_pwd/login_by_token/logout/refresh/register/get_user_auth_info_by_teacher_id/record_subscribe/get_subscribe_status/generate_bind_qrcode/generate_subscribe_qrcode/get_bind_info/get_bind_info_by_code/check_bind_status/confirm_bind/bind_by_code/test_send_subscribe/menu/get_menu_by_role 等）
  - [x] 登录类：断言 code=200 返回 token；依赖微信 code 的接口用预期错误码断言

- [x] Task 3: business-service 接口测试
  - [x] `business_institution_test.go`：institution 5 个接口
  - [x] `business_student_test.go`：student 10 个接口（含 get_by_course_id 关键回归）
  - [x] `business_teacher_test.go`：teacher 5 个接口
  - [x] `business_class_test.go`：class 8 个接口
  - [x] `business_class_schedule_test.go`：class_schedule 5 个接口
  - [x] `business_course_test.go`：course 4 个接口
  - [x] `business_course_record_test.go`：course_record 10 个接口（含 deduct_by_* 扣课时、delete）
  - [x] `business_record_test.go`：record 4 个接口

- [x] Task 4: admin-service 接口测试
  - [x] `admin_user_test.go`：user 接口（login/refresh/info/list/get_by_id/insert/update/delete/reset_password/get_roles）
  - [x] `admin_role_test.go`：role 接口（list/get_by_id/insert/update/delete/get_menus/save_menus）
  - [x] `admin_menu_test.go`：menu 接口
  - [x] `admin_operation_log_test.go`：operation_log 接口
  - [x] `admin_business_test.go`：business/institution/student/teacher/course/class/class_schedule/course_record/record/mini_menu 透传接口（29 个）
  - [x] `admin_teacher_auth_test.go`：teacher_auth 接口
  - [x] `admin_dashboard_test.go`：dashboard 接口
  - [x] `admin_config_test.go`：config 接口 + crypto/public_key

- [x] Task 5: deploy.sh 新增 test 子命令
  - [x] 实现 `test_services` 函数：编译 4 个服务 → 以测试端口启动（环境变量覆盖端口与路由 URI）→ 轮询等待 gateway 就绪 → 运行 `cd ${SRC_DIR} && go test ./poc/predeploy/... -v -timeout 10m` → 无论结果都清理测试实例
  - [x] 在 main 入口分发逻辑中注册 `test` 命令
  - [x] 测试账号环境变量（PRE_TEST_ADMIN_USERNAME/PASSWORD、PRE_TEST_MINI_ACCOUNT/PASSWORD）透传给测试进程
  - [x] bash -n 语法验证通过（已修复 CRLF 污染）

- [x] Task 6: Jenkinsfile 集成 Pre-Deploy Test 阶段
  - [x] 在 `Sync Source to Host` 与 `Build & Deploy` 之间插入 `Pre-Deploy Test` 阶段
  - [x] `when { expression { params.SKIP_BUILD == false } }`，SSH 执行 `bash ${HOST_DEPLOY_DIR}/deploy.sh test all`
  - [x] post.failure 增加测试日志查看提示

- [x] Task 7: 本地验证
  - [x] `go build ./...` 通过
  - [x] `go vet ./...` 通过
  - [x] 本地 `go test ./poc/predeploy/...` 通过（未配置测试账号时按预期跳过登录用例）
  - [x] `bash deploy.sh test all`：宿主机环境不可用，已验证脚本语法（bash -n）+ 命令分发逻辑 + test_services 完整流程实现

- [x] Task 8: 文档更新与提交推送
  - [x] 更新 `class_times_record_back/CLAUDE.md`（部署流程：deploy.sh test 子命令、Jenkins Pre-Deploy Test 阶段、测试账号配置说明）
  - [x] 更新根目录 `AGENTS.md`（如涉及部署/测试说明）
  - [x] 提交并推送 `class_times_record_back`（master，commit e7913f0）
  - [x] 同步根仓库子模块指针并推送（master，commit 4f673ae）

# Task Dependencies

- [Task 2][Task 3][Task 4] 依赖 [Task 1]（复用框架/客户端/登录辅助）
- [Task 2][Task 3][Task 4] 相互独立，可并行
- [Task 5] 依赖 [Task 1]-[Task 4]（测试包存在后才能运行）
- [Task 6] 依赖 [Task 5]
- [Task 7] 依赖 [Task 2]-[Task 6]
- [Task 8] 依赖 [Task 7]
