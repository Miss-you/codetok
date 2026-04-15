# 面向 AI Agent 的 CLI UX 改进实施计划

> **给 Claude：** 必须使用 `superpowers:executing-plans`，按任务逐项执行本计划。

**目标：** 让 `codetok` 对人类用户和 AI agent 都更容易发现、运行和正确理解，而不需要先阅读源代码。

**架构：** 保持现有 Cobra 命令结构和本地只读报告边界。在 CLI 层改进帮助文本、参数校验、表格语义和数据来源可观察性，同时保留 provider 解析器和 token 聚合语义。

**技术栈：** Go、Cobra、现有 provider registry、现有命令测试和 e2e 测试。

---

## 背景

这份计划来自一次对 `codetok v0.4.0` 的 CLI UX 审查。审查时手动运行了：

- `codetok --help`
- `codetok daily --help`
- `codetok session --help`
- `codetok cursor --help`
- `codetok daily --provider codex --since 2026-04-15 --unit raw`
- `codetok session --provider codex --since 2026-04-15`
- 无效参数探针，例如 `--provider nope` 和 `--group-by invalid`

数值 token 统计已经和 Codex 本地 JSONL session 日志中的 `token_count` 记录交叉核对过。相对本地 Codex 数据，计数是准确的。剩下的问题不是计数准确性，而是第一次使用的人类或 AI agent 是否能只通过 CLI 理解命令面和 token 语义。

## SOP：审查面向 AI Agent 的 CLI 建议

在把 CLI UX 建议接受为产品标准前，先使用这个 SOP。

1. **像第一次使用的用户一样运行 CLI**
   - 捕获顶层帮助、命令帮助、常见默认输出、JSON 输出和代表性错误输出。
   - 不要先读源码；第一轮应该反映用户或 agent 实际看到的内容。

2. **把观察到的困惑映射到具体标准**
   - 将问题归类为：可发现性、语义清晰度、诊断信息、来源/范围透明度、副作用安全性，或向后兼容的人体工学。
   - 如果建议只是风格偏好，且不能减少真实误用或自动化失败，就拒绝。

3. **对照源码和文档验证**
   - 检查命令定义、输出格式、provider registry 行为、README 说明和测试。
   - 确认问题是真实存在、已在别处文档化，还是由旧二进制输出导致。

4. **给出裁决**
   - **保留：** 建议应按当前方向进入 agent-friendly CLI 标准。
   - **修改：** 底层问题成立，但实现需要更窄、改名或重构表述。
   - **放弃：** 建议对正确性、可发现性或自动化的提升不足以 justify 新 CLI 表面积。

5. **对标准使用独立二审**
   - 将无关建议拆给 subagent。
   - 要求每个 subagent 返回：条目、裁决、理由和验收标准。
   - 只有在把 subagent 反馈和仓库证据调和后，才合并最终决策。

6. **先写验收标准，再实现**
   - 每个接受项必须写清精确命令行为、帮助文本期望、输出列语义和测试目标。
   - 优先用测试断言行为，不要大面积 snapshot 终端输出。

7. **默认保持兼容**
   - 添加 alias，而不是重命名命令。
   - 除非明确批准 schema migration，否则保持 JSON 字段稳定。
   - 当原始字段名会造成困惑时，可以在人类输出中使用更清晰的标签。

## 标准：面向 AI Agent 的 CLI 准则

`codetok` 的 CLI 变更应满足这些标准。

1. **自发现：** `--help` 必须展示从零上下文到有效命令的最短实用途径。
2. **语义一致：** 同一个标签在人类表格、JSON、README 示例和帮助图例中必须含义一致。
3. **指标透明：** 聚合计数必须解释其组成，尤其是 cache 字段和 total 公式。
4. **可行动诊断：** 无效输入必须明确失败并列出允许值，避免把 typo 表现成空数据。
5. **机器可读等价：** JSON 输出必须暴露和人类输出相同的底层事实，即使为了可读性使用不同标签。
6. **来源透明：** 本地扫描器必须展示或文档化扫描了哪些数据根目录，以及数据不可用时的状态。
7. **副作用清晰：** 命令必须说明自己是只读本地文件，还是可能访问远程 API。
8. **快照诚实：** 对活跃本地日志的报告必须说明结果是某个时间点的快照。
9. **向后兼容的人体工学：** 只要不移除或重定义现有命令，就可以添加更自然的 alias。
10. **简洁失败模式：** 默认错误应具体而简短；完整帮助留给 `--help`。

## 二审摘要

三个 subagent 独立审查了最初的十条建议：

- **Faraday：** 审查第 1-4 项。结果：保留第 1 和第 4 项；修改第 2 和第 3 项。
- **Plato：** 审查第 5-7 项。结果：保留第 5 和第 6 项；修改第 7 项，建议用 `sources` 而不是 `doctor`。
- **Darwin：** 审查第 8-10 项。结果：全部保留。

没有建议被放弃。三个建议在接受前被收窄：日期帮助文案、输入标签语义和来源检查命令命名。

## 已审查的改进 Backlog

| # | 最终裁决 | 对应标准 | 要做什么 | 原因 | 二审结果 | 验收标准 |
|---|---|---|---|---|---|---|
| 1 | 保留 | 自发现 | 在顶层 `codetok --help` 添加短 `Examples:` 区块。包含 `daily`、`session`、JSON 和本地 Cursor 示例。 | 第一次使用的用户或 agent 不应该必须读 README 才能发现常见路径。 | Faraday：保留。顶层帮助目前较少，而 README 已有 quick-start 材料。 | `codetok --help` 展示 3-5 个可运行示例，同时不隐藏现有命令列表。 |
| 2 | 修改 | 自发现、诊断 | 将面向用户的 `format: 2006-01-02` 改为 `YYYY-MM-DD`，并给出 `2026-04-15` 这样的示例。Go 解析逻辑保持不变。 | Go reference date 在代码中正确，但对 CLI 用户不直观。 | Faraday：修改。方向正确，但应表述为人类文案，而不是 parser 行为。 | `daily --help` 和 `session --help` 将日期 flag 描述为 `YYYY-MM-DD, e.g. 2026-04-15`。 |
| 3 | 修改 | 语义一致、指标透明 | 为 input 字段定义统一的人类表格标签策略。优先使用 `Input Total`、`Input Other`、`Cache Read` 和 `Cache Create` 等明确标签；避免在含义不同时使用裸 `Input`。 | `daily` 目前把 `InputOther` 标成 `Input`，而 `session` 把 `TotalInput()` 标成 `Input`。这会让准确数据看起来可疑。 | Faraday：修改。标签不一致是真问题；修复应包含标签策略和公式文档。 | 人类表格和 README 示例使用一致标签；测试证明 `daily` 和 `session` 不会给同一标签分配不同含义。 |
| 4 | 保留 | 指标透明 | 在 `daily --help`、`session --help`、README 和 README_zh 中添加紧凑 token 字段图例。 | 用户需要知道 total 是否包含 cache read、cache creation 和 output。 | Faraday：保留。token 模型存在于 `provider.TokenUsage`，但命令帮助中可见度不足。 | 帮助解释 `input_other`、`input_cache_read`、`input_cache_creation`、`output`、`input_total` 和 `total`。 |
| 5 | 保留 | 语义一致、机器可读等价 | 默认在人类输出的 `codetok session` 中展示 cache breakdown，除非后续实现证明表格过宽，并添加显式 `--breakdown`。 | Cache 是一等数据，且可能主导 total；session 输出隐藏它会导致用户质疑数据。 | Plato：保留。README_zh 已记录类似 session 输出的 cache 列，这是可见性缺口。 | 带 cache usage 的 session fixture 在人类输出中展示 `Cache Read` 和 `Cache Create`，JSON 保持不变。 |
| 6 | 保留 | 可行动诊断 | collection 前基于 provider registry 校验 `--provider`。未知 provider 应失败并列出允许名称。 | `--provider codxe` 这样的 typo 目前看起来像“无数据”，对人和 agent 都不好。 | Plato：保留。`collectSessionsFromProviders` 会过滤成空 provider 列表并静默成功。 | `codetok daily --provider bogus` 和 `codetok session --provider bogus` 非零退出并列出有效 provider；有效 provider 但无文件仍成功返回空数据。 |
| 7 | 修改 | 来源透明、副作用清晰 | 添加 `codetok sources`，而不是 `doctor`。它应展示解析后的扫描根目录、是否存在、provider 名称和发现的文件/session 数，且不访问远程。 | 用户需要在信任 total 前回答“工具到底扫描了什么”。`sources` 比 `doctor` 更精确。 | Plato：修改。能力有价值，但 `doctor` 命名模糊。 | `codetok sources` 报告每个 provider 的本地 roots、是否存在和本地发现数量；不执行网络访问。 |
| 8 | 保留 | 向后兼容的人体工学 | 为 `session` 添加 `sessions` alias。保持 `session` 为 canonical。 | 命令报告多个 sessions，复数形式是自然猜法。 | Darwin：保留。这能提升可发现性，且不破坏现有脚本。 | `codetok sessions` 接受同样 flags 并走同一代码路径；help 将其列为 alias。 |
| 9 | 保留 | 简洁失败模式 | 配置命令错误处理，使 validation failure 默认不打印重复错误或完整 usage。 | 当输出只有一个清晰失败和一个可选下一步时，agent 更容易解析错误。 | Darwin：保留。现有 validation error 具体，但 Cobra usage 噪音会掩盖它。 | 无效值打印一行具体错误和一条简短 help hint，而不是重复错误或完整命令帮助。 |
| 10 | 保留 | 快照诚实 | 文档化 `daily` 和 `session` 运行期间 active session 可能变化。把提示放在 README 和命令帮助中，但不要主导输出。 | `codetok` 读取本地日志，而活跃 CLI 可能还在写这些日志。这是事实，不是 bug。 | Darwin：保留。现有文档提到本地文件范围，但没有说明 active-write 快照行为。 | 文档/帮助说明报告是本地时间点快照，活跃 session 可能在两次运行间变化。 |

## 实施计划

### 任务 1：帮助文本和示例

**文件：**
- 修改：`cmd/root.go`
- 修改：`cmd/daily.go`
- 修改：`cmd/session.go`
- 修改：`README.md`
- 修改：`README_zh.md`

**步骤：**
1. 在 `rootCmd.Long` 中添加顶层示例。
2. 将日期 flag 描述改为带示例的 `YYYY-MM-DD`。
3. 在 `dailyCmd.Long` 和 `sessionCmd.Long` 中添加紧凑 token 字段图例。
4. 添加 README 区块，和帮助文本保持一致且不引入新行为。
5. 使用 `go test ./cmd` 和 `go run . --help` 验证。

### 任务 2：一致的 Token 标签

**文件：**
- 修改：`cmd/daily.go`
- 修改：`cmd/session.go`
- 修改：`cmd/daily_test.go`
- 修改：`cmd/session_test.go`，如果不存在就创建
- 修改：`README.md`
- 修改：`README_zh.md`

**步骤：**
1. 定义最终人类输出标签策略。
2. 更新 `daily` share-table 表头，避免将 `InputOther` 展示为含糊的 `Input`。
3. 更新 `session` 表头和 totals，清楚展示 cache 字段。
4. 使用 fixture `provider.SessionInfo` 添加表头和 totals 测试。
5. 验证 JSON 输出保持兼容。

### 任务 3：Provider 校验和错误

**文件：**
- 修改：`cmd/collect.go`
- 修改：`cmd/collect_test.go`
- 如 Cobra 错误行为集中在根命令，则修改：`cmd/root.go`
- 如现有 e2e 覆盖期待完整 usage dump，则修改对应 e2e 测试

**步骤：**
1. 基于 `provider.Registry()` 添加 provider filter 校验。
2. filter 未知时返回清晰错误并列出有效 provider 名称。
3. 配置 Cobra，避免重复 validation error 和不必要的完整 usage dump。
4. 为未知 provider 和有效 provider 无数据两种情况添加测试。
5. 手动验证 `--group-by invalid`、`--unit invalid` 和 `--provider invalid` 输出。

### 任务 4：来源清单命令

**文件：**
- 创建：`cmd/sources.go`
- 创建：`cmd/sources_test.go`
- 只有绝对必要时才修改 provider interface
- 修改：`README.md`
- 修改：`README_zh.md`

**步骤：**
1. 围绕本地 roots、是否存在、provider name 和发现数量设计 `codetok sources` 输出。
2. 复用 provider registry 和目录 override flag 约定。
3. 保持命令只读本地，不做隐式 Cursor sync 或 auth check。
4. 使用临时 provider roots 添加测试。
5. 验证 `codetok sources`、`codetok sources --provider codex` 和缺失目录场景。

### 任务 5：Session Alias 和快照提示

**文件：**
- 修改：`cmd/session.go`
- 修改：`README.md`
- 修改：`README_zh.md`
- 如命令发现已有 e2e 覆盖，则修改对应 e2e 测试

**步骤：**
1. 给 `sessionCmd` 添加 `Aliases: []string{"sessions"}`。
2. 确认 alias help 不会作为单独命令重复文档。
3. 在 help/docs 中添加 active-session snapshot 文案。
4. 验证 `go run . sessions --help` 和 `go run . session --help`。

### 任务 6：仓库验证

**文件：**
- 以上所有修改文件

**步骤：**
1. 运行 `make fmt`。
2. 运行 `make test`。
3. 运行 `make vet`。
4. 如可用，运行 `make lint`。
5. 运行 `make build`。
6. 手动验证：
   - `./bin/codetok --help`
   - `./bin/codetok daily --help`
   - `./bin/codetok session --help`
   - `./bin/codetok sessions --help`
   - `./bin/codetok sources --help`
   - 代表性无效 flag 输出

## 非目标

- 本轮 UX pass 不改变 provider parser 中的 token 计数算法。
- 除非另行批准 schema migration，否则不改 JSON 字段名。
- 不给 `daily`、`session` 或 `sources` 添加远程 API 访问。
- 不移除现有 `session` 命令拼写。

## 实施前开放问题

1. `session` 是否默认展示所有 cache 列？如果终端宽度成为问题，是否改为紧凑默认加 `--breakdown`？
2. `sources` 应优先统计 parsed sessions 还是 discovered files？parsed sessions 更有用，但 discovered files 更能解释 parse failure。
3. 简洁错误处理应全局应用到 `rootCmd`，还是先只应用到 report commands？
