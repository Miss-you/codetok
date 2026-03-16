## Context

`codetok` 当前的 provider 模型非常简单：每个 provider 只负责“发现本地文件并解析为 `SessionInfo`”，`daily` 和 `session` 再在命令层统一收集并做统计。Cursor 是现有 provider 中最特殊的一个，因为它并没有像 Claude Code 或 Codex CLI 那样稳定的本地 session JSONL；当前实现只能读取手工放入 `~/.codetok/cursor/**/*.csv` 的 dashboard 导出文件。

这带来三个实际问题。第一，用户无法在 `codetok` 内部获取 Cursor 历史 usage。第二，当前 CSV 字段语义尚未被正式固定，后续如果继续扩展，很容易在 token 口径上发生漂移。第三，Cursor 本地还有 `ai-code-tracking.db` 这类“活动归因”数据源，但它和 dashboard usage CSV 属于两种完全不同的产品语义，如果没有单独设计，很容易被误用为 token accounting。

## Goals / Non-Goals

**Goals:**

- 为 Cursor usage CSV 建立稳定、可测试、可文档化的规范化契约。
- 在不破坏现有 provider 抽象的前提下，为 Cursor 增加显式认证与 dashboard 同步能力。
- 让 `daily` / `session` 通过统一的数据收集路径消费 Cursor 的导入数据和同步缓存。
- 为未来的 Cursor `tab` / `composer` 活动归因能力预留独立能力边界，避免和 token 汇总混算。

**Non-Goals:**

- 不把 Cursor 远程同步隐式塞进 `daily` 或 `session`，避免普通报表命令带网络副作用。
- 不尝试从 `state.vscdb`、日志文件等弱信号来源反推出 Cursor token 总量。
- 不在本次设计中把 Cursor 活动归因和 token 统计合并成单一报表或单一数据结构。
- 不默认引入多账号复杂度作为第一阶段必需能力，除非后续需求明确证明必须支持。

## Decisions

### 决策 1：保持 `provider/cursor` 只做本地 CSV 解析

`provider.Provider` 接口当前只有 `CollectSessions(baseDir string)`，语义非常明确，就是从本地收集 session 数据。我们保留这个边界，让 `provider/cursor` 继续只处理本地 CSV 文件发现、格式验证和 `SessionInfo` 生成；任何 Cursor API 的认证与同步能力都放在新的 `codetok cursor ...` 子命令下。

这样做的原因是可以维持 provider 层的纯度，避免 `daily` / `session` 在运行时触发网络副作用，也避免未来为了 Cursor 一个 provider 把整个 provider 接口改造成“可选远程同步”的复杂抽象。

备选方案：
- 把 sync 逻辑直接塞进 `CollectSessions()`：实现上看起来集中，但会污染 provider 抽象，并让报表命令隐式带网络副作用。
- 在 `daily` / `session` 中单独为 Cursor 写分支：短期能跑，但命令层会复制特殊逻辑，后续更难维护。

### 决策 2：同步产物仍然落成 CSV，并复用同一条解析链路

无论 CSV 来自手工导入还是 `cursor sync` 拉取的 dashboard 导出，最终都落在 `~/.codetok/cursor/` 下，由同一套 CSV 解析与规范化逻辑消费。同步能力的职责是“拿到可靠 CSV 并原子写盘”，而不是直接产出统计结果。

这样做的原因是让网络获取和本地消费完全解耦：一旦 CSV 在磁盘上存在，报表命令就和其他 provider 一样继续走本地读取路径。测试上也更简单，sync 测网络和缓存写入，parser 测 CSV 语义，不必混成一条大链。

备选方案：
- sync 直接把 Cursor API 返回转成 `SessionInfo` 并存内部格式：可以少一次 CSV 解析，但会引入新的私有中间格式，并让手工导入与自动同步走不同路径。
- 分离“手工导入 parser”和“同步缓存 parser”：会产生双份逻辑，长期必然漂移。

### 决策 3：把 Cursor CSV 的两个 input 列作为导出中的独立类别保留

规范化层应直接保留 Cursor dashboard 导出的两个 input 维度，不对 `Input (w/ Cache Write)` 和 `Input (w/o Cache Write)` 做二次推断。也就是说，在 `codetok` 内部继续把它们映射为两个独立 token 字段，而不是在解析时先做相减推导。

这样做的原因是它和导出文件的可观测结构一致，减少人为解释层。后续如果 Cursor 官方文档给出更严格语义，变更也会集中在规范化 spec，而不是散落在 sync、parser、report 三处实现里。

备选方案：
- 仿照部分第三方实现，把 `Input (w/ Cache Write)` 减去 `Input (w/o Cache Write)` 再作为 cache write：如果解释错了，会直接改变总量口径，而且和导出表面结构不一致。
- 忽略其中一个 input 列：实现简单，但会丢失信息并导致总量不可信。

### 决策 4：为 `daily` / `session` 引入共享的 Cursor 收集编排层

当前 [cmd/daily.go](/Users/lihui/Documents/GitHub/codetok/cmd/daily.go) 和 [cmd/session.go](/Users/lihui/Documents/GitHub/codetok/cmd/session.go) 各自复制了一份 provider 过滤、目录覆盖、错误处理、session 收集逻辑。要支持 Cursor 的默认目录、同步缓存、手工导入以及将来的本地 attribution 数据，必须先抽出共享收集入口。

这个共享编排层负责统一以下事情：
- provider 过滤
- provider 目录覆盖优先级
- 默认 Cursor 根目录解析
- “报表命令只读本地，不触发 sync”的行为约束

备选方案：
- 继续让两个命令各自维护逻辑：短期改动少，但所有 Cursor 特性都要加两遍，回归风险高。
- 让 provider registry 负责更多命令编排：会让 registry 变成“带业务逻辑的调度器”，超出它当前职责。

### 决策 5：Cursor 活动归因单独建模、单独暴露

`ai-code-tracking.db` 能提供 `tab` / `composer` 的 accepted-line 归因，但它不是 token usage 源。设计上必须把这条链路和 dashboard CSV 完全分开：单独的数据模型、单独的命令或输出视图、单独的文档表述。

这样做的原因是只要两者共用 `provider.TokenUsage` 或共享 `daily/session` 总量，就会在产品语义上制造“看起来像 token 统计，实际上是行级活动归因”的误导。

备选方案：
- 直接把归因数据混进 `daily` / `session`：会破坏现有 token 统计的可信边界。
- 完全不设计 attribution：短期最省事，但会让后续想支持 `tab` / `composer` 时重新打开架构问题。

### 决策 6：第一阶段默认单活动账号，缓存目录预留扩展空间

第一阶段的 sync 能力只要求支持一个活动 Cursor 账号，但目录与缓存结构预留多账号扩展空间，例如区分 `imports/`、`synced/`、`archive/`。这样可以先用低复杂度完成端到端闭环，再根据真实需求决定是否引入命名账号管理。

备选方案：
- 一开始就做多账号：更完整，但命令、凭据、缓存冲突处理、文档复杂度都会显著上升。
- 完全不预留目录分层：最轻量，但后续做多账号或归档时会增加迁移成本。

## Risks / Trade-offs

- **[CSV 字段语义判断错误]** → 先通过规范化 spec 固定口径，并在实现阶段补充基于真实导出样本的回归测试。
- **[同步能力破坏“本地优先”产品心智]** → 明确规定只有 `codetok cursor ...` 子命令允许触发远程 API，`daily` / `session` 永远只读本地。
- **[命令层重复逻辑继续扩大]** → 在 Cursor 深入集成前先抽出共享收集编排层，减少 `daily` / `session` 的分叉实现。
- **[用户把活动归因误解为 token]** → 在 capability、命令命名、JSON 字段和文档中统一使用 activity / attribution 术语，不复用 token 字段。
- **[多账号需求晚于预期出现]** → 先设计可扩展缓存布局，第一阶段只实现单活动账号，避免前置复杂度。

## Migration Plan

1. 保持现有 `provider/cursor` 的本地 CSV 解析入口可用，确保已存在的手工导入目录不会失效。
2. 引入新的 Cursor 凭据与 sync 命令，把 dashboard CSV 缓存到 `codetok` 自己管理的目录层级中。
3. 抽取 `daily` / `session` 的共享收集逻辑，并让默认 Cursor 报表同时读取手工导入与同步缓存。
4. 更新文档，明确新的远程同步边界和“不支持 tab token 统计”的产品说明。
5. 在 token usage 能力稳定后，再单独接入本地 activity attribution 视图，避免和主链路耦合。

回滚策略：
- 如果 sync 能力需要回滚，只需停用 `codetok cursor ...` 子命令实现；已有本地 CSV 仍可继续被 parser 消费。
- 如果共享收集编排层出现问题，可以临时回退到旧命令路径，而不影响手工导入 CSV 的存在方式。

## Open Questions

- 是否需要在第一阶段就支持多命名账号，还是先只保留单活动账号并在文档中明确限制？
- 是否需要兼容 Cursor 更早期的 CSV 导出格式，还是只要求支持当前 dashboard 导出格式？
- Cursor 活动归因最终以独立命令呈现，还是以单独 JSON 视图/子输出块呈现，更符合 `codetok` 的产品风格？
- 后续如果 `codetok` 开始支持 cost 统计，是否应该直接复用 Cursor CSV 的 `Cost` 列，还是统一走未来的价格计算层？
