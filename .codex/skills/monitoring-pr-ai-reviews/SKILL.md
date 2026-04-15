---
name: monitoring-pr-ai-reviews
description: Use when a codetok PR exists and AI review comments need to be triaged, verified, and resolved.
---

# Monitoring PR AI Reviews

## Overview

在实现完成后把闭环补完整：开 PR、持续观察 AI review、用兼容性和正确性判断每条建议，而不是盲目同意。

**Core principle:** AI review comments are inputs to evaluate, not orders to follow.

**REQUIRED SUB-SKILLS:** `github:gh-address-comments`, `receiving-code-review`, `verification-before-completion`

## When to Use

在这些场景使用：

- 任务代码已经实现并验证
- PR 已存在，或现在就要开 PR
- Copilot 或其他 AI review 可能在首次 push 之后继续出现
- 剩余工作是 comment triage、修复、回复、重扫和 thread 收口
- 任务主要影响 `cmd/`、`provider/`、`stats/`、`cursor/` 或 `e2e/` 中的 Go 代码、CLI 输出、解析逻辑或测试契约

这些场景不要使用：

- 初始设计阶段
- 原始任务尚未实现
- 人类 review 已经改变产品方向，需要回到设计重新决策

## SOP

1. 在回复 AI review 前，先确认代码和验证已经是 fresh 状态。
2. 优先跑与改动相关的 `make fmt`、`make test`、`make vet`、`make build`，`make lint` 可用时一起跑。
3. 读取 thread-level review 上下文，连同文件锚点和 outdated 状态一起看，不只看平铺 comments。
4. 按 codetok 约束逐条判断建议，重点检查 CLI flags、stdout/stderr、provider 解析、stats 聚合、Cursor 本地/同步边界和 e2e 行为。
5. 只做有根据的最小修复；需要拒绝时，把理由写清楚。
6. 修复后重跑最窄的相关测试，再补更宽的 gate。
7. 只有 fix 或 rationale 已在 PR 中可见时，才收口 thread。

## Shared Review Loop

对每条 review thread，都按同一循环执行：

1. 先定义这条评论的真实主张是什么
2. 对照代码、契约和上下文核实它
3. 判断它是否成立
4. 修复问题或写出拒绝理由
5. 跑 fresh verification 后再回复或收口

不要凭直觉关闭 thread。

## The Flow

1. 确认任务真的 ready for PR
   - 跑 fresh verification
   - 确认任务板、workspace、验证证据一致
2. 打开或刷新 PR
   - 优先使用当前可用的 GitHub 能力
   - 不要把不存在的本地脚本当必需前置
3. 读取 review thread，而不是只看平铺 comments
   - 保留 thread、文件锚点、outdated 状态和 resolution 状态
4. 按 repo 原则评估每条 AI 建议
   - 保护 CLI 契约
   - 保护 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 相关行为
   - 优先最小修复，不扩 scope
5. 只实现 justified changes
   - 先跑最窄证明，再跑更宽 gate
6. 刷新 PR
   - push
   - 回复修复或拒绝理由
   - 只有 fix/rationale 已在 PR 可见时，才关闭 thread
7. 重扫直到没有 unresolved AI review，或只剩明确由用户持有的决策

## Evaluation Rules

接受建议，当它指出的是：

- 真实 bug、回归、race、错误处理缺失
- CLI 命令、flags、stdout/stderr、exit code 的破坏
- provider 解析、stats 聚合、Cursor 本地/同步边界或 e2e 契约漂移
- 缺少证明关键行为的验证

拒绝建议，当它主要要求：

- 为了测试单独扩大 public API
- 为未来 provider/runtime 提前抽象
- 在任务已接近完成时扩大重构范围
- 引入当前 repo 并不需要的框架、插件层或复杂 extension point

## Verification

在回应 AI review 前后，优先使用当前仓库真实存在的命令：

- `make fmt`
- `make vet`
- `make test`
- `make build`
- `make lint`，仅当本机安装了 `golangci-lint`

如果 review 触达 CLI 行为、provider 解析、stats 逻辑、Cursor 本地/同步边界或 e2e 契约，要补相关 focused test / smoke test。

## Optional Automation

如果未来仓库新增 PR review monitor workflow 或本地脚本，可以把它们接入这个 skill。

在那之前，这个 skill 的默认实现必须能在**没有额外脚本**的情况下完成闭环。

## Common Mistakes

- 把 Copilot 建议当成必须执行
- 在 fix 或 rationale 还没出现在 PR 前就关闭 thread
- 回归了 CLI / provider / stats / cursor / e2e 契约却只跑局部测试
- 为满足评论而扩大 public API
- 在本仓库没有相关脚本时仍硬引用死路径

## Next Step

当所有 justified AI review 都已处理、相关验证已重跑、thread 已正确收口后，再结束 PR 跟进周期或进入合并流程。

## Acceptance Criteria

- 每条 AI review 要么被修复，要么有清楚的拒绝理由
- 相关 PR thread 已收口，不保留未解释的 unresolved comment
- 影响到的 `cmd/`、`provider/`、`stats/`、`cursor/`、`e2e/` 改动已重跑对应验证
- 关键 CLI 或解析行为没有因为 review 修复而回归
