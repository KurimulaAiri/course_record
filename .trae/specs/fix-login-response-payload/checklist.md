# Checklist

## 小程序 auth-service 登录响应

- [x] UserVO 结构体已创建，包含 userId/roleId/identityInfo/admin/createTimeStr/updateTimeStr 字段
- [x] ParentIdentityVO 结构体已创建，包含 userId/isAvailable/username/parentId 字段
- [x] TeacherIdentityVO 结构体已创建，包含 userId/isAvailable/username/institutionId/teacherId/phone/isInstitutionAdmin 字段
- [x] AdminVO 结构体已创建，包含 adminId/userId/isAvailable/username/createTimeStr/updateTimeStr 字段
- [x] LoginByPwd 方法返回的 user 字段为 *UserVO（而非 *entity.User）
- [x] LoginByToken 方法返回的 user 字段为 *UserVO（而非 *entity.User）
- [x] 家长登录后 identityInfo 为 ParentIdentityVO，包含 parentId
- [x] 教师登录后 identityInfo 为 TeacherIdentityVO，包含 teacherId/institutionId/isInstitutionAdmin
- [x] get_open_id handler 调用 GetOpenId 方法（而非 WxLogin），仅返回 openId
- [x] RefreshAccessToken 的 refreshToken 字段返回空字符串（对齐 Java null）
- [x] Logout 返回消息为 "登出成功"
- [x] 所有时间字段为字符串格式（yyyy-MM-dd HH:mm:ss），非 sql.NullTime 对象

## Admin admin-service 登录响应

- [x] /crypto/public_key 接口已实现，返回 { publicKey: string }
- [x] Login 方法先 SM2 解密密码，再 BCrypt 校验
- [x] SM2 解密失败时返回友好错误（不抛 500）
- [x] ToSysUserVO 接收 roleIds 参数并填充 RoleIDs 字段
- [x] SelectRoleIDsByUserID 方法已实现，查询 sys_user_role 表
- [x] Login 方法调用 SelectRoleIDsByUserID 填充 roleIds
- [x] Login 响应顶层字段为 accessToken/refreshToken/userInfo
- [x] userInfo 包含 roleIds 数组（实际查询结果，非空数组）
- [x] RefreshToken 方法返回完整 userInfo（非 nil）
- [x] RefreshToken 方法签发新的 refreshToken（非原 token）
- [x] RefreshToken 方法校验用户状态（status != 1 拒绝刷新）
- [x] GetUserInfo 方法从 JWT 解析 userId（而非请求体）

## 编译验证

- [x] go build ./... 编译通过，无错误
- [x] 无未使用的 import 或变量
- [x] 无 panic 风险代码

## 报文结构对照

- [x] 小程序 login_by_pwd 响应结构与 src/types/auth.d.ts LoginResponse 一致
- [x] 小程序 user 对象结构与 src/types/user.d.ts UserResponse 一致
- [x] 小程序 ParentIdentity 与 src/types/parent.d.ts ParentIdentity 一致
- [x] 小程序 TeacherIdentity 与 src/types/teacher.d.ts TeacherIdentity 一致
- [x] Admin login 响应结构与 src/types/admin.d.ts LoginResponse 一致
- [x] Admin userInfo 与 src/types/admin.d.ts SysUserResponse 一致
- [x] Admin public_key 响应结构与 src/types/admin.d.ts GetPublicKeyResponse 一致
