# Tasks

## 阶段一：小程序 auth-service 登录响应修复（致命问题）

- [x] Task 1: 创建 UserVO 及身份信息 VO 结构体
  - [x] SubTask 1.1: 在 `auth-service/internal/service/auth_service.go` 新建 `UserVO` 结构体（字段：`userId/roleId/identityInfo/admin/createTimeStr/updateTimeStr`）
  - [x] SubTask 1.2: 新建 `ParentIdentityVO` 结构体（字段：`userId/isAvailable/username/parentId`）
  - [x] SubTask 1.3: 新建 `TeacherIdentityVO` 结构体（字段：`userId/isAvailable/username/institutionId/teacherId/phone/isInstitutionAdmin`）
  - [x] SubTask 1.4: 新建 `AdminVO` 结构体（字段：`adminId/userId/isAvailable/username/createTimeStr/updateTimeStr`）
  - [x] SubTask 1.5: 修改 `LoginVO.User` 字段类型为 `interface{}`（已为 interface{}，确保赋值为 `*UserVO` 而非 `*entity.User`）

- [x] Task 2: 实现 IdentityService 等价的身份查询逻辑
  - [x] SubTask 2.1: 在 `auth-service/internal/mapper/` 确认/新增 `ParentMapper.SelectByUserID(userID)` 方法
  - [x] SubTask 2.2: 确认/新增 `TeacherMapper.SelectByUserID(userID)` 方法
  - [x] SubTask 2.3: 新建 `AdminMapper.SelectByUserID(userID)` 方法（查询 `c_admin` 表）
  - [x] SubTask 2.4: 在 `AuthService` 注入 `ParentMapper`/`TeacherMapper`/`AdminMapper`（如未注入）
  - [x] SubTask 2.5: 实现 `GetFullUserInfo(userID, roleID)` 方法，根据 roleID 查询对应身份表并构造 `UserVO`

- [x] Task 3: 修改 LoginByPwd 方法返回完整 UserVO
  - [x] SubTask 3.1: 在登录成功后调用 `GetFullUserInfo(userID, roleID)` 构造 `UserVO`
  - [x] SubTask 3.2: 将 `LoginVO.User` 字段赋值为 `*UserVO` 而非 `*entity.User`
  - [x] SubTask 3.3: 验证家长登录返回 `identityInfo` 为 `ParentIdentityVO`（含 `parentId`）
  - [x] SubTask 3.4: 验证教师登录返回 `identityInfo` 为 `TeacherIdentityVO`（含 `teacherId/institutionId/isInstitutionAdmin`）

- [x] Task 4: 修改 LoginByToken 方法返回完整 UserVO
  - [x] SubTask 4.1: 同 Task 3，在 Token 续登成功后调用 `GetFullUserInfo` 构造 `UserVO`

- [x] Task 5: 修正 get_open_id handler 行为
  - [x] SubTask 5.1: 修改 `auth_handler.go` 的 `GetOpenID` 方法，调用 `authService.GetOpenId(code)` 而非 `WxLogin(code)`
  - [x] SubTask 5.2: 构造 `LoginVO{AccessToken: "", RefreshToken: "", OpenID: openId, User: nil}` 返回

- [x] Task 6: 修正 refresh 和 logout 接口
  - [x] SubTask 6.1: 修改 `RefreshAccessToken` 方法，将 `RefreshToken` 字段返回空字符串 `""`（对齐 Java null）
  - [x] SubTask 6.2: 修改 `Logout` 方法返回消息为 "登出成功"（对齐 Java）

## 阶段二：Admin admin-service 登录响应修复（致命问题）

- [x] Task 7: 新增 `/crypto/public_key` 接口
  - [x] SubTask 7.1: 在 `admin-service/internal/handler/` 新建 `CryptoHandler` 或在 `admin_handler.go` 增加 `GetPublicKey` 方法
  - [x] SubTask 7.2: 注册路由 `GET /crypto/public_key`
  - [x] SubTask 7.3: 从环境变量或常量读取 SM2 公钥（与 Java `crypto.sm2.public-key` 配置一致）
  - [x] SubTask 7.4: 返回 `{ "publicKey": "<sm2-public-key>" }`

- [x] Task 8: 修改 Admin Login 方法支持 SM2 解密
  - [x] SubTask 8.1: 在 `AdminService` 注入 SM2 私钥（从环境变量或常量读取，与 Java `crypto.sm2.private-key` 一致）
  - [x] SubTask 8.2: 修改 `Login` 方法，先调用 `crypto.SM2Decrypt(req.Password, privateKey)` 解密密码
  - [x] SubTask 8.3: 解密失败时返回友好错误（`{ code: 400, message: "密码解密失败" }`）
  - [x] SubTask 8.4: 解密后使用 BCrypt 校验密码

- [x] Task 9: 新增角色查询并填充 roleIds
  - [x] SubTask 9.1: 在 `admin-service/internal/mapper/admin_mapper.go` 新增 `SelectRoleIDsByUserID(userID)` 方法，查询 `sys_user_role` 表
  - [x] SubTask 9.2: 修改 `ToSysUserVO` 函数签名，接收 `roleIds []int64` 参数
  - [x] SubTask 9.3: 修改 `Login` 方法，调用 `SelectRoleIDsByUserID` 查询角色并传入 `ToSysUserVO`
  - [x] SubTask 9.4: 修改 `GetUserInfo` 方法，同样查询角色并填充

- [x] Task 10: 修复 RefreshToken 方法
  - [x] SubTask 10.1: 修改 `RefreshToken` 方法，解析 refreshToken 后重新查询用户信息构造 `SysUserVO`（含 roleIds）
  - [x] SubTask 10.2: 校验用户状态（status != 1 时拒绝刷新）
  - [x] SubTask 10.3: 签发新的 refreshToken（而非返回原 token）
  - [x] SubTask 10.4: 返回完整 `LoginVO{AccessToken, RefreshToken, UserInfo}`

## 阶段三：编译验证

- [x] Task 11: 编译验证
  - [x] SubTask 11.1: 执行 `go build ./...` 确保所有修改编译通过
  - [x] SubTask 11.2: 检查无未使用的 import 或变量

## 阶段四：报文结构对照验证

- [x] Task 12: 验证小程序登录响应结构
  - [x] SubTask 12.1: 确认 `login_by_pwd` 响应顶层字段为 `accessToken/refreshToken/openId/user`
  - [x] SubTask 12.2: 确认 `user` 对象包含 `userId/roleId/identityInfo/admin/createTimeStr/updateTimeStr`
  - [x] SubTask 12.3: 确认家长 `identityInfo` 包含 `parentId`
  - [x] SubTask 12.4: 确认教师 `identityInfo` 包含 `teacherId/institutionId/isInstitutionAdmin`
  - [x] SubTask 12.5: 确认所有时间字段为字符串格式（非 sql.NullTime 对象）
  - [x] SubTask 12.6: 确认 `get_open_id` 仅返回 openId（不签发 Token）

- [x] Task 13: 验证 Admin 登录响应结构
  - [x] SubTask 13.1: 确认 `/admin/crypto/public_key` 返回 `{ publicKey: string }`
  - [x] SubTask 13.2: 确认 `/admin/user/login` 响应顶层字段为 `accessToken/refreshToken/userInfo`
  - [x] SubTask 13.3: 确认 `userInfo` 包含 `roleIds` 数组（非空，实际查询结果）
  - [x] SubTask 13.4: 确认 `/admin/user/refresh` 返回完整 `userInfo`（非 nil）
  - [x] SubTask 13.5: 确认 `/admin/user/refresh` 签发新的 refreshToken

# Task Dependencies

- Task 2 依赖 Task 1（需要 UserVO 结构体定义）
- Task 3、Task 4 依赖 Task 2（需要身份查询逻辑）
- Task 8 依赖 Task 7（需要 SM2 公钥配置）
- Task 9 可与 Task 8 并行
- Task 10 依赖 Task 9（需要角色查询方法）
- Task 11 依赖 Task 3-10 全部完成
- Task 12、Task 13 依赖 Task 11（编译通过后验证）
