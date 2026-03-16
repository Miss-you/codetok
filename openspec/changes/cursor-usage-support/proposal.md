## Why

`codetok` 当前对 Cursor 的支持仍停留在“手工导入 CSV 后做本地解析”，这和其他 provider 相比存在明显缺口：用户无法直接通过工具获取 Cursor 历史 usage，同时现有 CSV 字段语义也还没有被固定成一个可持续维护的产品契约。除此之外，我们还需要明确区分“可信的 Cursor token 总量”和“来自本地 Cursor 跟踪库的非 token 活动归因”，避免两类数据在产品层面混淆。

## What Changes

- 定义 Cursor dashboard usage CSV 的规范化契约，包括字段语义、坏行处理、日期解析和 token 映射。
- 增加显式的 Cursor 认证与同步流程，让用户可以把 Cursor dashboard 上的 usage CSV 拉取到 `codetok` 自己管理的本地缓存目录。
- 重构报表集成方式，让 `daily` 和 `session` 通过同一条共享收集路径消费 Cursor 数据，并统一目录覆盖、回退和本地缓存行为。
- 定义单独的 Cursor 活动归因能力，基于本地 `ai-code-tracking.db` 暴露 `tab` / `composer` 相关指标，并明确它不是 token accounting。
- 更新文档与命令契约，明确普通报表命令仍然是本地文件解析，只有显式的 Cursor 子命令才允许访问远程 Cursor API。

## Capabilities

### New Capabilities
- `cursor-usage-normalization`: 定义 Cursor dashboard CSV 到 `codetok` session 与 token 字段的权威映射。
- `cursor-dashboard-sync`: 定义 Cursor 登录、状态检查、同步、登出等显式工作流，用于把 dashboard 历史 CSV 获取并缓存到本地。
- `cursor-report-integration`: 定义 `daily` 与 `session` 如何消费 Cursor 的同步缓存与手工导入数据，包括覆盖优先级和回退语义。
- `cursor-activity-attribution`: 定义如何从本地归因数据库中单独报告 Cursor 的 `tab` / `composer` 活动指标，并保证不把它们当成 token 总量。

### Modified Capabilities

无。

## Impact

- **代码范围**：`provider/cursor/`、新增的 Cursor 认证/同步命令模块、`cmd/` 或新的内部共享收集模块，以及后续读取 Cursor 本地 SQLite 归因数据的读取层。
- **CLI 接口**：新增 `codetok cursor ...` 子命令，并明确现有 `daily` / `session` 对 Cursor 的消费契约。
- **依赖与系统边界**：可能新增 HTTP 与凭据存储相关辅助能力，但普通报表命令不引入远程调用依赖。
- **文档**：`README.md`、`README_zh.md`、命令帮助文本，以及 Cursor 专项的验证说明都需要同步更新。
