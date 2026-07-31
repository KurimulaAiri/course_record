---
kind: logging_system
name: 后端日志系统：SLF4J + Logback 异步输出与 AOP 操作审计日志
category: logging_system
scope:
    - '**'
source_files:
    - class_times_record_back/common/src/main/resources/logback-spring.xml
    - class_times_record_back/common/src/main/java/com/shiroko/annotation/OperationLog.java
    - class_times_record_back/admin-service/src/main/java/com/shiroko/aspect/OperationLogAspect.java
    - class_times_record_back/common/src/main/java/com/shiroko/repository/entity/SysOperationLog.java
    - class_times_record_back/admin-service/src/main/java/com/shiroko/controller/SysOperationLogController.java
---

## 1. 使用的框架与工具
- 日志门面：SLF4J（通过 `@Slf4j` Lombok 注解注入）
- 日志实现：Logback，配置文件位于 `common/src/main/resources/logback-spring.xml`
- 运行期日志：统一输出到标准输出，由容器/进程管理器收集；开发环境通过 `-Dspring.profiles.active=dev` 切换为 DEBUG 级别
- 业务审计日志：基于 Spring AOP + 自定义 `@OperationLog` 注解，将管理端关键操作持久化到 `sys_operation_log` 表，并提供查询接口

## 2. 核心文件与包
- 日志配置：`class_times_record_back/common/src/main/resources/logback-spring.xml`
- 操作日志注解：`class_times_record_back/common/src/main/java/com/shiroko/annotation/OperationLog.java`
- 操作日志切面：`class_times_record_back/admin-service/src/main/java/com/shiroko/aspect/OperationLogAspect.java`
- 操作日志实体：`class_times_record_back/common/src/main/java/com/shiroko/repository/entity/SysOperationLog.java`
- 操作日志控制器：`class_times_record_back/admin-service/src/main/java/com/shiroko/controller/SysOperationLogController.java`
- 使用示例（大量 Controller 方法标注 `@OperationLog("...")`）：`admin-service/.../controller/AdminBusinessController.java`

## 3. 架构与设计约定
- **统一输出格式**：控制台日志采用彩色结构化模板，包含时间、级别、线程、logger 名称与消息。
- **异步非阻塞**：所有 appender 均通过 `AsyncAppender` 包装，队列大小 256，`neverBlock=true`，避免日志 I/O 阻塞业务线程。
- **分层日志级别策略**：
  - dev profile：业务代码 `com.shiroko`、Spring 框架、Gateway 路由、Filter/Interceptor 均为 DEBUG；Sentinel 恢复 INFO 便于调试流控。
  - 默认 root：INFO，屏蔽高频心跳（如 SentinelHealthIndicator）。
  - 特定包降级：MyBatis Mapper 设为 DEBUG，Spring MVC DispatcherServlet 等设为 WARN，减少噪音。
- **AOP 操作审计**：
  - 在需要审计的 Controller 方法上添加 `@OperationLog("描述")`。
  - `OperationLogAspect` 自动采集：方法签名、第一个入参（截断至 2000 字符）、当前用户（来自 `UserContext`）、请求 IP（优先 `X-Forwarded-For` / `X-Real-IP`）、执行耗时、创建时间。
  - 写入失败仅记录 warn，不影响主流程。
- **前端无统一日志框架**：小程序与管理端源码中散落 `console.log`，未封装统一 logger，主要用于本地调试。

## 4. 开发者应遵循的规则
- 业务日志使用 SLF4J：通过 `@Slf4j` 注入后调用 `log.info/warn/error/debug`，禁止直接打印堆栈字符串。
- 敏感信息脱敏：参数日志最大 2000 字符，避免输出完整大对象或敏感字段。
- 审计日志规范：对增删改等关键操作在 Controller 层标注 `@OperationLog("简洁中文描述")`，确保可追溯。
- 日志级别选择：正常流程用 info，异常/降级用 warn，错误用 error；仅在开发阶段开启 debug。
- 不要修改 `logback-spring.xml` 中的全局 root 级别，如需调整请在对应 springProfile 下覆盖。