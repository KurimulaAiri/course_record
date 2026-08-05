# 项目规则

## 代码规范

### 注释要求
- **所有生成的代码都必须加详细的注释**
  - 函数/方法：必须添加注释说明功能、参数含义、返回值
  - 复杂逻辑：必须添加行内注释解释关键步骤
  - 类型定义：必须说明字段含义
  - 组件：必须说明组件用途、props、events
  - API 接口：必须说明接口功能、请求参数、响应结构

### 注释示例

#### 函数/方法
```typescript
/**
 * 根据学生ID获取班级列表
 * @param params.studentId - 学生ID
 * @param params.currentPage - 当前页码
 * @param params.pageSize - 每页条数
 * @returns 班级列表分页数据
 */
const getClassListByStudentId = (params: GetClassListRequest) => { ... }
```

#### 复杂逻辑
```typescript
// 判断课程是否已到期（兼容 iOS 时间格式，将 - 替换为 /）
const expireDate = new Date(expireTimeStr.replace(/-/g, "/"));
```

#### 类型定义
```typescript
interface StudentResponse {
  /** 学生ID */
  id: number;
  /** 学生姓名 */
  studentName: string;
}
```

#### 组件
```vue
<!--
  FormPage 表单页面组件
  用途：统一渲染分组表单展示/编辑
  Props: groups - 分组配置, modelValue - 数据模型
  Events: groupTitleTap - 点击分组标题
-->
```

#### API 接口
```typescript
/**
 * 解绑学生
 * POST /biz/student/unbind
 * @param parentId - 家长ID
 * @param studentId - 学生ID
 * @returns 操作结果消息
 */
```

## Git 提交信息规范
- 使用中文
- **格式**：`<类型>: <简要中文描述>`
  - 类型与描述之间用 `: `（冒号+空格）分隔
  - 描述应简明扼要，说明"做了什么"，避免冗余上下文
- **类型取值**：
  - `feat` — 新功能、新工具、新接口
  - `fix` — 修复 bug、修复配置问题、修复部署问题
  - `refactor` — 重构（不改变功能）
  - `docs` — 文档变更
  - `debug` — 临时调试代码（上线前应移除）
  - `chore` — 构建、依赖、CI 等杂项变更
- **示例**：
  - `feat: 添加宝塔文件管理API工具(11个工具) - 目录浏览/文件读写/创建删除/复制移动/权限查询/路径检测/磁盘占用`
  - `fix: 容器内服务URL改为localhost直连,绕过Nginx反代SocketError`
  - `fix: deploy.sh改用grep逐个提取变量，彻底解决BT_API_SK解析问题`
  - `feat: trigger_jenkins_job默认分支从micro_service改为master（微服务已合并到主线）`

### Git 推送前检查清单
- **每次执行 git commit 和 push 前，必须更新以下文档**：
  - `class_times_record_back/docs/` 目录下的所有文档（如 test-cases.md、架构设计文档等）
  - `class_times_record_back/CLAUDE.md` — 项目架构和开发指南
  - `AGENTS.md`（如果存在）— AI Agent 协作规范
- 确保文档内容与最新代码变更保持一致
- 文档更新与代码提交在同一 commit 中完成

## 分支管理
- **后端 (class_times_record_back)**：主分支为 `master`，开发分支已合并到主线
- **前端小程序 (class_times_record)**：主分支为 `main`
- **前端管理面板 (class_record_admin_front)**：主分支为 `main`（Git checkout 已启用）
- Jenkinsfile 配置的 GIT_BRANCH 必须与当前主分支保持一致

## 本地开发环境

### Go 后端环境
- 项目后端已迁移到 Go 语言（位于 `class_times_record_back/`）
- 编译：`cd class_times_record_back && go build ./...`
- 配置加载：所有配置从 yml 文件加载（通过 `APP_ENV` 选择 `config.dev.yml`/`config.prod.yml`，或通过 `CONFIG_PATH` 指定路径）

### Gateway 本地开发
- Gateway 路由 URI 从 `config.dev.yml` 文件加载（`gateway.auth_uri`/`gateway.business_uri`/`gateway.admin_uri`）
- 开发环境配置为 `http://localhost:{port}` 直连格式，无需 Nacos 也可运行
- 生产环境可配置为 `lb://{service-name}` 服务发现格式（需 Nacos 可达）
- Nacos 服务注册保留：启动时注册实例，优雅关闭时注销（对齐 Java @EnableDiscoveryClient）

### Java 环境（已废弃，仅保留参考）
- 项目原要求 JDK 21（Spring Boot 4.0.4 依赖），本地路径 `D:\env\java\jdk-21.0.8.9-hotspot`
- 后端已迁移到 Go，Java 代码仅保留参考，不再维护

### 小程序前端
- 开发模式 (`NODE_ENV=development`) 使用 `http://localhost:9999`（本地 Gateway）
- 生产/预览模式使用 `https://api.kurimula-airi.top`（远程服务器）
- openId 存储在 `uni.getStorageSync("openId")`

### UniApp 条件编译规范
- **微信小程序专属代码必须使用 `#ifdef MP-WEIXIN` / `#endif` 进行条件编译隔离**
  - 涉及微信特有 API（如 `wx.getSetting`、`wx.requestSubscribeMessage`）时必须包裹
  - 非微信平台需提供 `#ifndef MP-WEIXIN` 降级逻辑
  - 示例：
    ```javascript
    // #ifdef MP-WEIXIN
    uni.getSetting({ withSubscriptions: true, ... });
    // #endif
    // #ifndef MP-WEIXIN
    resolve(false); // 非微信平台降级处理
    // #endif
    ```

## 部署说明

### Jenkins 构建
- 后端任务名：`class_time_record_back`（非参数化任务）
- 微服务任务名：`course-record-microservice`（参数化任务，支持 DEPLOY_SCOPE/SKIP_BUILD/ROLLBACK 参数）
- 前端管理面板任务名：`cr-admin-dashboard`（非参数化任务）

### Docker 容器部署
- Gateway：JAR 直跑模式（非 Docker），部署目录 `/opt/java-deploy/class_times_record_back/gateway`
- auth-service / business-service / admin-service：Docker 容器部署，目录 `/opt/java-deploy/class_times_record_docker`
- 端口映射：Gateway=9999, Business=10001, Auth=10002, Admin=10003
