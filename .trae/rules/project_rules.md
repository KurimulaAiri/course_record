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

### Java 环境
- 项目要求 JDK 21（Spring Boot 4.0.4 依赖），本地路径 `D:\env\java\jdk-21.0.8.9-hotspot`
- 编译时需设置环境变量 `JAVA_HOME=D:\env\java\jdk-21.0.8.9-hotspot`

### Gateway 本地开发
- Gateway 默认使用 Nacos 的 `lb://` 路由（负载均衡到所有注册实例）
- 本地开发必须激活 dev profile：`-Dspring.profiles.active=dev`，否则会请求远程服务器
- dev profile 会用 localhost 直连路由覆盖 Nacos 的 `lb://` 路由配置
- IDEA 运行配置 `.run/GatewayApplication.run.xml` 已预设该参数

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
