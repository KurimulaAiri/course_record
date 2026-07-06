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
