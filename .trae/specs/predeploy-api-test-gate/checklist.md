# Checklist

## 测试包框架
- [ ] `poc/predeploy/` 包已创建，包含配置、HTTP 客户端（SM3 签名/SM2 加密）、登录辅助
- [ ] 测试地址使用独立测试端口（gateway=19999），非生产端口
- [ ] 测试账号从环境变量读取，未配置时登录用例 t.Skip

## auth-service 接口覆盖
- [ ] auth 全部注册路由均有测试用例（公开路径 + 登录 + 订阅 + 绑定）
- [ ] 登录成功用例断言 code=200 且返回 token

## business-service 接口覆盖
- [ ] institution 全部接口有测试
- [ ] student 全部接口有测试（含 get_by_course_id）
- [ ] teacher / class / class_schedule / course 全部接口有测试
- [ ] course_record 全部接口有测试（含 deduct_by_*、delete）
- [ ] record 全部接口有测试

## admin-service 接口覆盖
- [ ] user / role / menu 全部接口有测试
- [ ] operation_log / business 透传 / teacher_auth / dashboard / config 全部接口有测试
- [ ] 管理端登录用例通过（能获取 token 访问受保护接口）

## deploy.sh 测试子命令
- [ ] `test` 子命令已注册，语法正确
- [ ] 测试端口启动/清理逻辑正确（不与生产端口冲突）
- [ ] 测试失败时返回非 0 退出码

## Jenkinsfile 集成
- [ ] `Pre-Deploy Test` 阶段位于 Sync 与 Build & Deploy 之间
- [ ] SKIP_BUILD=true 时跳过测试阶段
- [ ] 测试失败中止流水线，不进入部署

## 验证
- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过
- [ ] `go test ./poc/predeploy/...` 通过（无测试账号时正确跳过登录用例）

## 文档与提交
- [ ] `class_times_record_back/CLAUDE.md` 已更新部署流程与测试说明
- [ ] `class_times_record_back` 已提交推送（master）
- [ ] 根仓库子模块指针已同步推送
