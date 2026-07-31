---
kind: frontend_style
name: 双端前端样式体系：Element Plus + SCSS 设计令牌与 UniApp 主题变量
category: frontend_style
scope:
    - '**'
source_files:
    - class_record_admin_front/src/styles/index.scss
    - class_record_admin_front/package.json
    - class_times_record/src/uni.scss
    - class_times_record/src/static/scss/variables/_index.scss
    - class_times_record/src/static/scss/_index.scss
---

本仓库包含两个独立的前端应用，各自采用不同的 UI 风格体系，但都遵循「CSS 变量/SCSS 变量集中管理 + 组件库主题覆盖」的统一策略。

## 1. 后台管理前端（class_record_admin_front）

- **UI 框架**：Element Plus 2.x，通过 CSS 自定义属性（--el-*）全局覆盖主题色、边框、背景、文字颜色等。
- **样式语言**：SCSS，入口 src/styles/index.scss 集中声明设计令牌。
- **设计令牌**：使用 :root 下的 --color-* 系列 CSS 变量定义主色（#e8a838 金色）、成功/危险/警告语义色、表面色、阴影、滚动条等；字体令牌 --font-body / --font-heading 分别指向 DM Sans 与 Sora；布局令牌 --sidebar-width、--header-height 控制侧边栏与头部尺寸。
- **暗色模式**：通过 [data-theme="dark"] 选择器提供完整的双色板，所有 --color-* 和 --el-* 变量在暗色下重新赋值，配合 .theme-transition-disabled 类实现平滑过渡。
- **组件覆盖约定**：对 el-menu、el-button、el-table、el-dialog、el-card、el-tag、el-pagination、el-input 等常用组件逐一用 CSS 变量覆盖默认样式，保持品牌一致性。
- **页面级工具类**：page-container、page-header、page-content、search-bar、table-toolbar、pagination-wrapper、text-empty 等统一页面骨架与间距。
- **动效**：内置 fade、slide-fade 两组 Vue transition 类，以及全局 transition: background-color/border-color/color/box-shadow 0.3s ease 主题切换动画。

## 2. 课时管理小程序（class_times_record，UniApp）

- **UI 框架**：uni-ui 组件库（uni_modules/），同时保留 uni-app 官方 uni.scss 变量作为基础主题层。
- **样式语言**：SCSS，通过 uni.scss 暴露 $uni-* 变量供 uni-ui 及业务代码直接使用。
- **项目主题令牌**：src/static/scss/variables/_index.scss 定义品牌色 $theme-color（#70a9a2 青绿）及其明暗变体 $theme-color-lighter/dark/darker，以及 $default-background-color。
- **变量组织**：static/scss/_index.scss 使用 @forward 引入 variables，便于被各页面 SCSS 按需引用。
- **页面样式**：每个页面目录内自包含 index.vue + index.scss + index.d.ts，样式局部化，不污染全局。

## 3. 跨端一致性与约束

| 维度 | 后台管理前端 | 小程序 |
|------|-------------|--------|
| 主题色 | Element Plus --el-color-primary = #e8a838 | uni.scss $uni-color-primary = #007aff（可被业务覆盖） |
| 变量来源 | CSS 变量 --color-* + --el-* | SCSS 变量 $uni-* + 自定义 $theme-* |
| 暗色模式 | 原生支持，通过 data-theme 切换 | 未实现（小程序平台限制） |
| 字体 | Google Fonts（DM Sans / Sora） | 系统字体栈 |
| 构建工具 | Vite + sass 插件 | uni CLI + vite-plugin-uni + sass |

## 4. 开发者应遵循的规则

1. 新增颜色必须走令牌：后台管理前端优先使用 --color-* 变量，小程序优先使用 $theme-* 或 $uni-* 变量，禁止硬编码十六进制色值到业务组件中。
2. 覆盖 Element Plus 样式时只改 CSS 变量：通过 --el-* 变量而非直接写死颜色，确保暗色模式自动生效。
3. 页面级 SCSS 文件命名：小程序每个页面目录内统一使用 index.scss，并通过 @import 或 @use 引入 static/scss/_index.scss 获取主题变量。
4. 避免在组件内部写全局样式：将通用布局类（如 page-container、search-bar）集中在 styles/index.scss，业务组件仅使用这些类名。
5. 暗色模式兼容性：任何新增的 Element Plus 组件样式都应同时提供 [data-theme="dark"] 下的覆盖规则。