# 修复登录接口报文结构 Spec

## Why

登录成功返回的数据结构与前端期望严重不一致，导致前端无法正确加载界面。核心问题在于：
1. 小程序登录返回的 `user` 字段是 `entity.User`（仅 4 个字段，且 `sql.NullXxx` 序列化为对象格式），前端期望的是 `UserResponse`（含 `userId/roleId/identityInfo/admin/createTimeStr/updateTimeStr` 嵌套结构）
2. Admin 端缺少 `/admin/crypto/public_key` 接口，导致前端无法获取 SM2 公钥加密密码，无法登录
3. Admin 端 `/admin/user/refresh` 返回 `userInfo: nil`，Token 刷新后丢失用户信息
4. Admin 端登录响应的 `roleIds` 未实际查询填充，导致权限路由失效

## What Changes

### 小程序 auth-service（致命问题）
- **新建 `UserVO` 结构体**：对齐 Java `UserVO`，包含 `userId/roleId/identityInfo/admin/createTimeStr/updateTimeStr` 字段
- **新建 `ParentIdentityVO`**：包含 `userId/isAvailable/username/parentId` 字段
- **新建 `TeacherIdentityVO`**：包含 `userId/isAvailable/username/institutionId/teacherId/phone/isInstitutionAdmin` 字段
- **新建 `AdminVO`**：包含 `adminId/userId/isAvailable/username/createTimeStr/updateTimeStr` 字段
- **修改 `LoginByPwd`**：查询 `c_parent`/`c_teacher`/`c_admin` 表构造完整 `UserVO` 返回
- **修改 `LoginByToken`**：同 `LoginByPwd`，构造完整 `UserVO` 返回
- **修改 `GetOpenID` handler**：改为调用 `GetOpenId` 方法（仅返回 openId，不签发 Token），而非当前的 `WxLogin`
- **修改 `RefreshAccessToken`**：`RefreshToken` 字段返回空字符串（对齐 Java null 行为）
- **修改 `Logout`**：返回消息改为 "登出成功"（对齐 Java）

### Admin admin-service（致命问题）
- **新增 `/crypto/public_key` 接口**：返回 SM2 公钥 `{ "publicKey": "<sm2-public-key>" }`
- **修改 `Login` 方法**：先 SM2 解密密码，再 BCrypt 校验
- **修改 `RefreshToken` 方法**：重新查询用户信息构造 `SysUserVO` 返回（非 nil），签发新 refreshToken，校验用户状态
- **新增角色查询**：在 `AdminUserMapper` 增加 `SelectRoleIDsByUserID` 方法，查询 `sys_user_role` 表
- **修改 `ToSysUserVO`**：接收 `roleIds` 参数填充 `RoleIDs` 字段

### 通用修复
- 所有登录相关接口的 `user`/`userInfo` 字段使用 VO 对象，避免 `sql.NullXxx` 序列化为对象格式

## Impact

- Affected code:
  - `class_times_record_go/auth-service/internal/service/auth_service.go` — LoginVO、登录方法
  - `class_times_record_go/auth-service/internal/handler/auth_handler.go` — GetOpenID handler
  - `class_times_record_go/auth-service/internal/mapper/` — 需要新增 ParentMapper/TeacherMapper 查询方法
  - `class_times_record_go/admin-service/internal/service/admin_service.go` — Login/RefreshToken 方法
  - `class_times_record_go/admin-service/internal/handler/admin_handler.go` — 新增 CryptoHandler
  - `class_times_record_go/admin-service/internal/mapper/admin_mapper.go` — 新增角色查询
  - `class_times_record_go/common/crypto/sm.go` — 复用现有 SM2 解密
  - `class_times_record_go/common/entity/entity.go` — 可能需要补充 Admin 实体

## ADDED Requirements

### Requirement: SM2 公钥获取接口（Admin 端）

系统 SHALL 提供 `GET /admin/crypto/public_key` 接口，返回 SM2 公钥供前端加密密码使用。

#### Scenario: 成功获取公钥
- **WHEN** 前端发送 GET 请求到 `/admin/crypto/public_key`
- **THEN** 返回 `{ "code": 200, "data": { "publicKey": "<sm2-public-key>" } }`

### Requirement: Admin 登录密码 SM2 解密

系统 SHALL 在 Admin 登录时先使用 SM2 私钥解密前端传来的密码密文，再使用 BCrypt 校验密码。

#### Scenario: 密码正确
- **WHEN** 前端发送 SM2 加密后的密码
- **THEN** 后端 SM2 解密后 BCrypt 校验通过，返回双 Token + userInfo（含 roleIds）

#### Scenario: 密码错误
- **WHEN** SM2 解密后的密码与 BCrypt 哈希不匹配
- **THEN** 返回 `{ "code": 400, "message": "用户名或密码错误" }`

### Requirement: Admin Token 刷新返回完整用户信息

系统 SHALL 在 `/admin/user/refresh` 接口返回完整的 `userInfo`（含 roleIds），并签发新的 refreshToken。

#### Scenario: 刷新成功
- **WHEN** 前端发送有效的 refreshToken
- **THEN** 返回新的 accessToken + 新的 refreshToken + 完整 userInfo（含 roleIds）

#### Scenario: 用户已被禁用
- **WHEN** refreshToken 有效但用户 status != 1
- **THEN** 返回 `{ "code": 401, "message": "账号已被禁用" }`

### Requirement: 小程序登录返回完整 UserVO

系统 SHALL 在小程序登录接口（`login_by_pwd`/`login_by_token`）返回包含身份信息的完整 `UserVO`。

#### Scenario: 家长登录成功
- **WHEN** 家长（roleId=3）登录成功
- **THEN** `user.identityInfo` 为 `ParentIdentityVO`，包含 `parentId` 字段

#### Scenario: 教师登录成功
- **WHEN** 教师（roleId=4）登录成功
- **THEN** `user.identityInfo` 为 `TeacherIdentityVO`，包含 `teacherId/institutionId/isInstitutionAdmin` 字段
- **AND** `user.admin` 为 `AdminVO` 对象（或 null，取决于是否为机构管理员）

### Requirement: 角色ID查询

系统 SHALL 在 Admin 登录和 Token 刷新时查询 `sys_user_role` 表填充 `roleIds` 字段。

#### Scenario: 用户有角色
- **WHEN** 用户已分配角色
- **THEN** `userInfo.roleIds` 包含所有角色ID（如 `[1, 2]`）

#### Scenario: 用户无角色
- **WHEN** 用户未分配任何角色
- **THEN** `userInfo.roleIds` 为空数组 `[]`

## MODIFIED Requirements

### Requirement: 小程序登录响应结构

小程序登录接口（`get_open_id`/`login_by_pwd`/`login_by_token`/`refresh`）的响应结构修改为：

```json
{
  "code": 200,
  "data": {
    "accessToken": "string",
    "refreshToken": "string",
    "openId": "string",
    "user": {
      "userId": 123,
      "roleId": 3,
      "createTimeStr": "2026-07-31 10:00:00",
      "updateTimeStr": "2026-07-31 10:00:00",
      "identityInfo": {
        "userId": 123,
        "isAvailable": true,
        "username": "parent_name",
        "parentId": 456
      },
      "admin": null
    }
  }
}
```

### Requirement: Admin 登录响应结构

Admin 登录接口响应结构修改为：

```json
{
  "code": 200,
  "data": {
    "accessToken": "string",
    "refreshToken": "string",
    "userInfo": {
      "id": 1,
      "username": "admin",
      "nickname": "管理员",
      "phone": "13800138000",
      "email": "admin@example.com",
      "avatar": "",
      "status": 1,
      "isDeleted": 0,
      "remark": "",
      "createTimeStr": "2026-07-31 10:00:00",
      "updateTimeStr": "2026-07-31 10:00:00",
      "roleIds": [1, 2]
    }
  }
}
```

### Requirement: get_open_id 接口行为修正

`GET /auth/auth/get_open_id` 接口 SHALL 仅返回 openId，不签发 Token，不查询用户信息。

```json
{
  "code": 200,
  "data": {
    "accessToken": "",
    "refreshToken": "",
    "openId": "wx_open_id",
    "user": null
  }
}
```

## REMOVED Requirements

无移除需求。
